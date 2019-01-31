package main

import (
	"dnd/creature"
	"dnd/dice"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func sumIntSlice(slice []int) int {
	result := 0
	for _, n := range slice {
		result += n
	}
	return result
}

func GetPostHandler(handleGet http.HandlerFunc, handlePost http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handlePost(w, r)
		} else {
			handleGet(w, r)
		}
	})
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

func (diceServer *DiceServer) handleGet(w http.ResponseWriter, r *http.Request) {
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

func (diceServer *DiceServer) handlePost(w http.ResponseWriter, r *http.Request) {
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
	Creature  creature.Creature
	DamageURL string
}

type EncounterData struct {
	CreatureInformation                       []CreatureInformation
	NextCreatureTypeName, NextCreatureHitDice string
}

type EncounterServer struct {
	template  *template.Template
	creatures []creature.Creature
}

func (encounterServer *EncounterServer) handleGet(w http.ResponseWriter, r *http.Request) {
	cis := make([]CreatureInformation, len(encounterServer.creatures))
	cc := len(encounterServer.creatures)
	for i, c := range encounterServer.creatures {
		ci := cc - 1 - i
		cis[ci].Creature = c
		cis[ci].DamageURL = r.RequestURI + strconv.Itoa(i)
	}
	data := EncounterData{cis, "", ""}
	if cc > 0 {
		nextCreatureType := encounterServer.creatures[cc-1].Type
		data.NextCreatureTypeName = nextCreatureType.Name
		data.NextCreatureHitDice = nextCreatureType.HitDice.String()
	}
	err := encounterServer.template.Execute(w, data)
	if err != nil {
		log.Print(err)
	}
}

func (encounterServer *EncounterServer) handlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	lastSlashPos := strings.LastIndexByte(r.URL.Path, '/')
	if lastSlashPos == len(r.URL.Path)-1 { // Last characters
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
		encounterServer.creatures = append(encounterServer.creatures, newCreature)
		http.Redirect(w, r, r.RequestURI, 303)
	} else {
		urlParts := strings.Split(r.URL.Path, "/")
		redirectURL := strings.Join(urlParts[:len(urlParts)-1], "/")
		creatureToDamageString := urlParts[len(urlParts)-1]
		creatureToDamage, err := strconv.Atoi(creatureToDamageString)
		if err != nil {
			log.Printf("Error parsing int from %v - %v", creatureToDamageString, err)
			http.Redirect(w, r, redirectURL, 303)
			return
		}
		if creatureToDamage >= len(encounterServer.creatures) {
			log.Printf("Invalid creature (out of range) - %v", creatureToDamage)
			http.Redirect(w, r, redirectURL, 303)
			return
		}
		damageAmount, err := strconv.Atoi(r.Form["damageAmount"][0])
		if err != nil {
			log.Printf("Couldn't parse damage amount - %v", err)
		}
		encounterServer.creatures[creatureToDamage].DamageTaken += damageAmount
		http.Redirect(w, r, redirectURL, 303)
	}
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
		make([]creature.Creature, 0),
	}
	server := http.NewServeMux()
	server.HandleFunc("/favicon.ico", http.NotFound)
	server.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	server.Handle("/encounter/", GetPostHandler(encounterServer.handleGet, encounterServer.handlePost))
	server.Handle("/", GetPostHandler(diceServer.handleGet, diceServer.handlePost))
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
