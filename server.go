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

type DndServer struct {
	DiceServer      DiceServer
	EncounterServer EncounterServer
	StaticServer    http.Handler
}

type DiceServer struct {
	Template       *template.Template
	PreviousRolls  []dice.DiceRollResult
	LastCustomRoll string
}

type RollTemplateValues struct {
	HasResult      bool
	LastRoll       dice.DiceRollResult
	OlderRolls     chan dice.DiceRollResult
	LastCustomRoll string
}

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

func (diceServer *DiceServer) handlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	roll, err := dice.ParseDiceRollString(r.Form["roll"][0])
	if len(r.Form["roll-custom"]) > 0 {
		diceServer.LastCustomRoll = r.Form["roll"][0]
	}
	if err == nil {
		diceServer.PreviousRolls = append(diceServer.PreviousRolls, roll.SimulateDiceRolls())
	} else {
		log.Println(err)
	}
	http.Redirect(w, r, "/", 303)
}

func (diceServer *DiceServer) handleGet(w http.ResponseWriter, r *http.Request) {
	var templateValues RollTemplateValues
	templateValues.LastCustomRoll = diceServer.LastCustomRoll
	if len(diceServer.PreviousRolls) > 0 {
		templateValues.HasResult = true
		penultimateIndex := len(diceServer.PreviousRolls) - 1
		templateValues.LastRoll = diceServer.PreviousRolls[penultimateIndex]
		templateValues.OlderRolls = dice.ReverseDiceRollResultSlice(
			diceServer.PreviousRolls[:penultimateIndex])
	}
	err := diceServer.Template.Execute(w, templateValues)
	if err != nil {
		log.Print(err)
	}
}

type EncounterServer struct {
	Template  *template.Template
	Creatures []creature.Creature
}

func (encounterServer *EncounterServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := encounterServer.Template.Execute(w, struct{}{})
	if err != nil {
		log.Print(err)
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
		make([]dice.DiceRollResult, 0),
		"Custom Roll",
	}
	encounterServer := EncounterServer{
		theTemplate.Lookup("encounter.html.tmpl"),
		make([]creature.Creature, 0),
	}
	server := http.NewServeMux()
	server.HandleFunc("/favicon.ico", http.NotFound)
	server.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	server.HandleFunc("/encounter/", encounterServer.ServeHTTP)
	server.Handle("/", GetPostHandler(diceServer.handleGet, diceServer.handlePost))
	err = http.ListenAndServe("localhost:1212", server)
	if err != nil {
		log.Fatal(err)
	}
}
