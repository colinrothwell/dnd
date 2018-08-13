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
	PreviousRolls []int
}

type RollTemplateValues struct {
	HasResult bool
	DiceRolled dice.DiceRoll
	Result int
	PreviousRolls chan int
}

func reverse(lst []int) chan int {
	ret := make(chan int)
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
	log.Print(r.RequestURI, ": Dice roll")
	roll, err := dice.ParseDiceRollString(r.RequestURI[1:])
	if err == nil {
		var templateValues RollTemplateValues
		if r.RequestURI != "/" {
			templateValues.HasResult = true
			templateValues.DiceRolled = roll
			templateValues.Result = roll.SimulateValue()
		}
		templateValues.PreviousRolls = reverse(diceServer.PreviousRolls)
		diceServer.Template.Execute(w, templateValues)
		if templateValues.HasResult {
			diceServer.PreviousRolls = append(diceServer.PreviousRolls, templateValues.Result)
		}
	} else {
		log.Println(err)
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
		make([]int, 0),
	}
	err = http.ListenAndServe("localhost:1212", &server)
	if err != nil {
		log.Fatal(err)
	}
}
