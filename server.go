package main

import (
	"dnd/dice"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type DiceServer struct {
	Template       *template.Template
	StaticServer   http.Handler
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

func (diceServer *DiceServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/favicon.ico" {
		log.Print(r.RequestURI, ": Ignoring favico")
		return
	}
	if strings.HasPrefix(r.RequestURI, "/static") {
		log.Print(r.RequestURI, ": Serving static")
		diceServer.StaticServer.ServeHTTP(w, r)
		return
	}
	if r.Method == "POST" {
		diceServer.handlePost(w, r)
		return
	} else {
		diceServer.handleGet(w, r)
	}
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
		templateValues.OlderRolls = dice.ReverseDiceRollResultSlice(diceServer.PreviousRolls[:penultimateIndex])
	}
	err := diceServer.Template.Execute(w, templateValues)
	if err != nil {
		log.Print(err)
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	theTemplate, err := template.ParseFiles("templates/roll.html.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Starting dice server on localhost:1212...")
	server := DiceServer{
		theTemplate,
		http.StripPrefix("/static", http.FileServer(http.Dir("static"))),
		make([]dice.DiceRollResult, 0),
		"Custom Roll",
	}
	err = http.ListenAndServe("localhost:1212", &server)
	if err != nil {
		log.Fatal(err)
	}
}
