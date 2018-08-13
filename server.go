package main

import (
	"net/http"
	"log"
	"math/rand"
	"time"
	"dnd/dice"
	"html/template"
	"strings"
)

type DiceServer struct {
	Template *template.Template
	StaticServer http.Handler
	PreviousRolls []dice.DiceRolls
}

type RollTemplateValues struct {
	HasResult bool
	DiceRolled dice.DiceRoll
	Result int
	PreviousRolls chan dice.DiceRolls
}

func reverse(lst []dice.DiceRolls) chan dice.DiceRolls {
	ret := make(chan dice.DiceRolls)
	go func() {
		for i, _ := range lst {
			ret <- lst[len(lst)-1-i]
		}
		close(ret)
	}()
	return ret
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
		diceServer.handlePost(w, r);
		return
	} else {
		diceServer.handleGet(w, r);
	}
}

func (diceServer *DiceServer) handlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	roll, err := dice.ParseDiceRollString(r.Form["roll"][0])
	if err == nil {
		diceServer.PreviousRolls = append(diceServer.PreviousRolls, roll)
	} else {
		log.Println(err)
	}
	http.Redirect(w, r, "/", 303)
}

func (diceServer *DiceServer) handleGet(w http.ResponseWriter, r *http.Request) {
	var templateValues RollTemplateValues
	if len(diceServer.PreviousRolls) > 0 {
		templateValues.HasResult = true
		templateValues.DiceRolled = diceServer.PreviousRolls[0]
		templateValues.PreviousRolls = reverse(diceServer.PreviousRolls[1:])
	}
	diceServer.Template.Execute(w, templateValues)
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
		make([]dice.DiceRolls, 0),
	}
	err = http.ListenAndServe("localhost:1212", &server)
	if err != nil {
		log.Fatal(err)
	}
}
