package main

import (
	"dnd/dice"
	"dnd/party"
	"fmt"
	"html/template"
	"net/http"
)

type DiceServer struct {
	Template *template.Template
	Party    *party.Party
}

type RollTemplateValues struct {
	HasResult      bool
	LastRoll       *dice.RollResult
	OlderRolls     chan *dice.RollResult
	LastCustomRoll *string
}

func (diceServer *DiceServer) GetTemplate() *template.Template {
	return diceServer.Template
}

func (diceServer *DiceServer) GenerateTemplateData(r *http.Request) interface{} {
	var templateValues RollTemplateValues
	templateValues.LastCustomRoll = &diceServer.Party.LastCustomRoll
	if len(diceServer.Party.PreviousRolls) > 0 {
		templateValues.HasResult = true
		penultimateIndex := len(diceServer.Party.PreviousRolls) - 1
		templateValues.LastRoll = &diceServer.Party.PreviousRolls[penultimateIndex]
		templateValues.OlderRolls = dice.ReverseRollResultSlice(
			diceServer.Party.PreviousRolls[:penultimateIndex])
	}
	return templateValues
}

func (diceServer *DiceServer) HandlePost(r *http.Request) error {
	roll, err := dice.ParseRollString(r.Form["roll"][0])
	if err != nil {
		// TODO: Handle invalid roll strings better
		return fmt.Errorf("error parsing roll string")
	}
	if len(r.Form["roll-custom"]) > 0 {
		diceServer.Party.LastCustomRoll = r.Form["roll"][0]
	}
	diceServer.Party.PreviousRolls = append(diceServer.Party.PreviousRolls, roll.Simulate())
	err = diceServer.Party.Save()
	if err != nil {
		return fmt.Errorf("error saving party - %v", err)
	}
	return nil
}
