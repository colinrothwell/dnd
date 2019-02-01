package main

import (
	"dnd/creature"
	"dnd/dice"
	"encoding/gob"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"
)

func getURLFunctionAndArgument(u *url.URL) (function string, argument string) {
	urlParts := strings.Split(u.Path, "/")
	if len(urlParts) == 0 {
		return "", ""
	} else if len(urlParts) == 1 {
		return "", urlParts[0]
	} else {
		return strings.Join(urlParts[:len(urlParts)-1], "/"), urlParts[len(urlParts)-1]
	}
}

func sumIntSlice(slice []int) int {
	result := 0
	for _, n := range slice {
		result += n
	}
	return result
}

type GetPostHandler interface {
	HandleGet(w http.ResponseWriter, r *http.Request)
	HandlePost(w http.ResponseWriter, r *http.Request)
}

func StandardGetPostHandler(h GetPostHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h.HandlePost(w, r)
		} else {
			h.HandleGet(w, r)
		}
	})
}

type Party struct {
	name               string
	encounterCreatures []creature.Creature
}

type DndServer struct {
	DiceServer      DiceServer
	EncounterServer EncounterServer
	StaticServer    http.Handler
}

type DiceServer struct {
	Template       *template.Template
	PreviousRolls  []dice.RollResult
	LastCustomRoll string
}

type RollTemplateValues struct {
	HasResult      bool
	LastRoll       *dice.RollResult
	OlderRolls     chan *dice.RollResult
	LastCustomRoll *string
}

func (diceServer *DiceServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	var templateValues RollTemplateValues
	templateValues.LastCustomRoll = &diceServer.LastCustomRoll
	if len(diceServer.PreviousRolls) > 0 {
		templateValues.HasResult = true
		penultimateIndex := len(diceServer.PreviousRolls) - 1
		templateValues.LastRoll = &diceServer.PreviousRolls[penultimateIndex]
		templateValues.OlderRolls = dice.ReverseRollResultSlice(
			diceServer.PreviousRolls[:penultimateIndex])
	}
	err := diceServer.Template.Execute(w, templateValues)
	if err != nil {
		log.Print(err)
	}
}

func (diceServer *DiceServer) HandlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	roll, err := dice.ParseRollString(r.Form["roll"][0])
	if len(r.Form["roll-custom"]) > 0 {
		diceServer.LastCustomRoll = r.Form["roll"][0]
	}
	if err == nil {
		diceServer.PreviousRolls = append(diceServer.PreviousRolls, roll.Simulate())
	} else {
		log.Println(err)
	}
	http.Redirect(w, r, "/", 303)
}

type CreatureInformation struct {
	Type, Name                                                   string
	MinHealth, RolledHealth, MaxHealth, DamageTaken              int
	DamageURL, MinHealthClass, RolledHealthClass, MaxHealthClass string
}

type EncounterData struct {
	CreatureInformation                       []CreatureInformation
	NextCreatureTypeName, NextCreatureHitDice string
}

type EncounterServer struct {
	template *template.Template
	party    *Party
}

func (encounterServer *EncounterServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	creatures := encounterServer.party.encounterCreatures
	creatureCount := len(creatures)
	creatureInformations := make([]CreatureInformation, creatureCount)
	for i, creature := range creatures {
		var minHC, rolledHC, maxHC string
		if 2*creature.DamageTaken >= creature.RolledHealth {
			rolledHC = "damaged"
		}
		if creature.DamageTaken >= creature.Type.HitDice.Min() {
			minHC = "dead"
			if creature.DamageTaken >= creature.RolledHealth {
				rolledHC = "dead"
				if creature.DamageTaken >= creature.Type.HitDice.Max() {
					maxHC = "dead"
				}
			}
		}
		creatureInformationIndex := creatureCount - 1 - i
		creatureInformations[creatureInformationIndex] = CreatureInformation{
			creature.Type.Name,
			creature.Name,
			creature.Type.HitDice.Min(),
			creature.RolledHealth,
			creature.Type.HitDice.Max(),
			creature.DamageTaken,
			r.RequestURI + strconv.Itoa(i),
			minHC,
			rolledHC,
			maxHC}
	}
	data := EncounterData{creatureInformations, "", ""}
	if creatureCount > 0 {
		nextCreatureType := creatures[creatureCount-1].Type
		data.NextCreatureTypeName = nextCreatureType.Name
		data.NextCreatureHitDice = nextCreatureType.HitDice.String()
	}
	err := encounterServer.template.Execute(w, data)
	if err != nil {
		log.Print(err)
	}
}

func (encounterServer *EncounterServer) HandlePost(w http.ResponseWriter, r *http.Request) {
	creatures := encounterServer.party.encounterCreatures
	r.ParseForm()
	redirectURL, arg := getURLFunctionAndArgument(r.URL)
	if arg == "" {
		roll, err := dice.ParseRollString(r.Form["creatureHitDice"][0])
		if err != nil {
			log.Printf("Error parsing creature dice string - %v", err)
			http.Redirect(w, r, r.RequestURI, 303)
			return
		}
		newCreature := *creature.Create(
			r.Form["creatureType"][0],
			r.Form["creatureName"][0],
			roll)
		creatures = append(creatures, newCreature)
		http.Redirect(w, r, r.RequestURI, 303)
	} else {
		creatureToDamage, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("Error parsing int from %v - %v", arg, err)
			http.Redirect(w, r, redirectURL, 303)
			return
		}
		if creatureToDamage >= len(creatures) {
			log.Printf("Invalid creature (out of range) - %v", creatureToDamage)
			http.Redirect(w, r, redirectURL, 303)
			return
		}
		damageAmount, err := strconv.Atoi(r.Form["damageAmount"][0])
		if err != nil {
			log.Printf("Couldn't parse damage amount - %v", err)
		}
		creatures[creatureToDamage].DamageTaken += damageAmount
		http.Redirect(w, r, redirectURL, 303)
	}
}

type PartyInitialisationData struct {
	Name, URL string
}

type initialisationServer struct {
	template               *template.Template
	partyNames             []string
	InitialisationComplete bool
}

func newInitialisationServer(t *template.Template) (*initialisationServer, error) {
	s := initialisationServer{t, make([]string, 0), false}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := path.Join(usr.HomeDir, "encounters")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("encounters directory '%s' does not exist, creating...", dataDir)
		os.Mkdir(dataDir, 0755)
	}
	contents, err := ioutil.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("Error when reading data directory - %v", err)
	}
	i := 0
	for _, fileInfo := range contents {
		if !fileInfo.IsDir() && strings.HasSuffix(fileInfo.Name(), ".party.gob") {
			fileName := path.Join(dataDir, fileInfo.Name())
			file, err := os.Open(fileName)
			if err != nil {
				log.Printf("Error opening .party.gob file - %v", err)
			}
			defer file.Close()
			decoder := gob.NewDecoder(file)
			var party *Party
			err = decoder.Decode(party)
			if err != nil {
				log.Printf("Error decoding party - %v", err)
				continue
			}
			s.partyNames = append(s.partyNames, party.name)
			i++
		}
	}
	return &s, nil
}

// Returns true if initialisation is complete
func (s *initialisationServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	err := s.template.Execute(w, nil)
	if err != nil {
		log.Fatalf("Error rendering initialisation template - %v", err)
	}
}

func (s *initialisationServer) HandlePost(w http.ResponseWriter, r *http.Request) {
	log.Fatalf("I DON'T ACTUALLY KNOW HOW TO PICK A PARTY WAAAA!!!")
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	theTemplate, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Starting dice server on localhost:1212...")
	diceServer := DiceServer{
		theTemplate.Lookup("roll.html.tmpl"),
		make([]dice.RollResult, 0),
		"Custom Roll",
	}
	encounterServer := EncounterServer{
		theTemplate.Lookup("encounter.html.tmpl"),
		&Party{"FCT34", make([]creature.Creature, 0)},
	}

	// This is a bit messy to always get the same static handling, but stateful otherwise.
	// Its possible all this chaining and calls is inefficient, but it shouldn't be a very hot path.
	// Maybe writing a local application while using exclusively server-side logic is an unusual,
	// bad, and not-well-supported pattern. Who knew?
	server := http.NewServeMux()
	server.HandleFunc("/favicon.ico", http.NotFound)
	server.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))

	initialisationHandler, err := newInitialisationServer(theTemplate.Lookup("choosegroup.html.tmpl"))
	if err != nil {
		log.Fatalf("Catacylsmic error initialising - %v", err)
	}

	logicServer := http.NewServeMux()
	logicServer.Handle("/encounter/", StandardGetPostHandler(&encounterServer))
	logicServer.Handle("/", StandardGetPostHandler(&diceServer))

	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if initialisationHandler.InitialisationComplete {
			logicServer.ServeHTTP(w, r)
		} else {
			StandardGetPostHandler(initialisationHandler).ServeHTTP(w, r)
		}
	})
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
