package main

import (
	"dnd/dice"
	"html/template"
	"log"
	"net/http"
)

type DiceServer struct {
	Template *template.Template
	Party    *Party
}

type RollTemplateValues struct {
	HasResult      bool
	LastRoll       *dice.RollResult
	OlderRolls     chan *dice.RollResult
	LastCustomRoll *string
}

func (diceServer *DiceServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	var templateValues RollTemplateValues
	templateValues.LastCustomRoll = &diceServer.Party.LastCustomRoll
	if len(diceServer.Party.PreviousRolls) > 0 {
		templateValues.HasResult = true
		penultimateIndex := len(diceServer.Party.PreviousRolls) - 1
		templateValues.LastRoll = &diceServer.Party.PreviousRolls[penultimateIndex]
		templateValues.OlderRolls = dice.ReverseRollResultSlice(
			diceServer.Party.PreviousRolls[:penultimateIndex])
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
		diceServer.Party.LastCustomRoll = r.Form["roll"][0]
	}
	if err == nil {
		diceServer.Party.PreviousRolls = append(diceServer.Party.PreviousRolls, roll.Simulate())
	} else {
		log.Println(err)
	}
	diceServer.Party.Save()
	http.Redirect(w, r, "/", 303)
}