package main

import (
	"dnd/creature"
	"dnd/dice"
	"dnd/party"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type CreatureInformation struct {
	Type, Name, DamageName, DeleteURL string
	CurrentHealth, MaxHealth          int
	CurrentHealthClass                string
}

type EncounterData struct {
	CreatureInformation                       []CreatureInformation
	NextCreatureTypeName, NextCreatureHitDice string
}

type EncounterServer struct {
	template      *template.Template
	postURLRegexp *regexp.Regexp
}

// NewEncounterServer creates
func NewEncounterServer(t *template.Template) (*EncounterServer, error) {
	r, err := regexp.Compile(`^/encounter/((?:new-creature)|(?:damage)|(?:delete))(?:/(\d+))?$`)
	if err != nil {
		return nil, fmt.Errorf("can't compile URL regex - %v", err)
	}
	return &EncounterServer{t, r}, nil
}

func (s *EncounterServer) GetTemplate() *template.Template {
	return s.template
}

func (s *EncounterServer) GenerateTemplateData(r *http.Request, p party.Party) interface{} {
	creatureCount := len(p.Creatures())
	creatureInformations := make([]CreatureInformation, creatureCount)
	for i, creature := range p.Creatures() {
		var hc string
		if 2*creature.DamageTaken >= creature.RolledHealth {
			hc = "damaged"
		}
		if creature.DamageTaken >= creature.RolledHealth {
			hc = "dead"
		}
		creatureInformationIndex := creatureCount - 1 - i
		strI := strconv.Itoa(i)
		creatureInformations[creatureInformationIndex] = CreatureInformation{
			creature.Type.Name,
			creature.Name,
			"damageAmount" + strI,
			"/encounter/delete/" + strI,
			creature.RolledHealth - creature.DamageTaken,
			creature.RolledHealth,
			hc}
	}
	data := EncounterData{creatureInformations, "", ""}
	if creatureCount > 0 {
		nextCreatureType := p.Creatures()[creatureCount-1].Type
		data.NextCreatureTypeName = nextCreatureType.Name
		data.NextCreatureHitDice = nextCreatureType.HitDice.String()
	}
	return data
}

// HandlePost takes input in two different ways: the posted form and the url
// the form of the url path is one of
// /encounter/new-creature
// /encounter/damage
// /encounter/delete/(creatureID)
func (s *EncounterServer) HandlePost(r *http.Request, p party.Party) (party.ReversibleAction, error) {
	args := s.postURLRegexp.FindStringSubmatch(r.URL.Path)
	if args == nil {
		return nil, fmt.Errorf("couldn't extract arguments from URL Path '%v'", r.URL.Path)
	}
	action := args[1]
	if action == "new-creature" {
		roll, err := dice.ParseRollString(r.Form["creatureHitDice"][0])
		if err != nil {
			return nil, fmt.Errorf("error parsing creature dice string - %v", err)
		}
		return &party.AddCreatureAction{creature.Create(
			r.Form["creatureType"][0],
			r.Form["creatureName"][0],
			roll)}, nil
	} // else
	if action == "damage" {
		actions := make([]party.DamageCreatureAction, 0)
		for k, v := range r.Form {
			if strings.HasPrefix(k, "damageAmount") {
				creatureID, err := strconv.Atoi(k[len("damageAmount"):])
				if err != nil {
					return nil, fmt.Errorf("critical error deriving id: %v", err)
				}
				if v[0] == "Amount" {
					continue
				}
				damageAmount, err := strconv.Atoi(v[0])
				if err != nil {
					return nil, fmt.Errorf("couldn't parse damage amount: %v", err)
				}
				if damageAmount != 0 {
					actions = append(actions, party.DamageCreatureAction{
						creatureID, damageAmount})
				}
			}
		}
		if len(actions) == 0 {
			return nil, errors.New("no creatures damaged in action")
		} else if len(actions) == 1 {
			return &actions[0], nil
		} else {
			return party.DamageMultipleCreaturesAction(actions), nil
		}
	} // else
	if len(args) != 3 {
		return nil, fmt.Errorf("unexpected number of args from regex (%d) - %#v", len(args), args)
	} // else
	creatureID, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, fmt.Errorf("error parsing int from %v - %v", args[2], err)
	}
	if action != "delete" {
		return nil, fmt.Errorf("unrecognised action - %v", action)
	}
	return p.DeleteCreatureAction(creatureID), nil
}
