package main

import (
	"dnd/creature"
	"dnd/dice"
	"html/template"
	"log"
	"math/rand"
	"net/http"
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

type EncounterValues struct {
	Creatures        []creature.Creature
	NextCreatureType *creature.Type
}

type EncounterServer struct {
	template *template.Template
	values   *EncounterValues
}

func (encounterServer *EncounterServer) handleGet(w http.ResponseWriter, r *http.Request) {
	err := encounterServer.template.Execute(w, encounterServer.values)
	if err != nil {
		log.Print(err)
	}
}

func (encounterServer *EncounterServer) handlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
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
	// TODO: Fix this if/when it becomes a problem
	encounterServer.values.Creatures = append([]creature.Creature{newCreature}, encounterServer.values.Creatures...)
	encounterServer.values.NextCreatureType = newCreature.Type
	http.Redirect(w, r, r.RequestURI, 303)
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
		&EncounterValues{make([]creature.Creature, 0), &creature.Type{HitDice: &dice.Roll{}}},
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
