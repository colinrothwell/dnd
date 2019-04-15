package main

import (
	"dnd/creature"
	"dnd/dice"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
)

type CreatureInformation struct {
	Type, Name, DamageURL, DeleteURL string
	CurrentHealth, MaxHealth         int
	CurrentHealthClass               string
}

type EncounterData struct {
	CreatureInformation                       []CreatureInformation
	NextCreatureTypeName, NextCreatureHitDice string
}

type EncounterServer struct {
	template      *template.Template
	party         *Party
	postURLRegexp *regexp.Regexp
}

// NewEncounterServer creates
func NewEncounterServer(t *template.Template, p *Party) (*EncounterServer, error) {
	r, err := regexp.Compile(`^/encounter/((?:new-creature)|(?:damage)|(?:delete))(?:/(\d+))?$`)
	if err != nil {
		return nil, fmt.Errorf("can't compile URL regex - %v", err)
	}
	return &EncounterServer{t, p, r}, nil
}

func (s *EncounterServer) GetTemplate() *template.Template {
	return s.template
}

func (s *EncounterServer) GenerateTemplateData(r *http.Request) interface{} {
	creatureCount := len(s.party.EncounterCreatures)
	creatureInformations := make([]CreatureInformation, creatureCount)
	for i, creature := range s.party.EncounterCreatures {
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
			"/encounter/damage/" + strI,
			"/encounter/delete/" + strI,
			creature.RolledHealth - creature.DamageTaken,
			creature.RolledHealth,
			hc}
	}
	data := EncounterData{creatureInformations, "", ""}
	if creatureCount > 0 {
		nextCreatureType := s.party.EncounterCreatures[creatureCount-1].Type
		data.NextCreatureTypeName = nextCreatureType.Name
		data.NextCreatureHitDice = nextCreatureType.HitDice.String()
	}
	return data
}

// HandlePost takes input in two different ways: the posted form and the url
// the form of the url path is one of
// /encounter/new-creature
// /encounter/damage/(creatureID)
// /encounter/delete/(creatureID)
func (s *EncounterServer) HandlePost(r *http.Request) error {
	args := s.postURLRegexp.FindStringSubmatch(r.URL.Path)
	if args == nil {
		return fmt.Errorf("couldn't extract arguments from URL Path '%v'", r.URL.Path)
	}
	action := args[1]
	if action == "new-creature" {
		roll, err := dice.ParseRollString(r.Form["creatureHitDice"][0])
		if err != nil {
			return fmt.Errorf("error parsing creature dice string - %v", err)
		}
		newCreature := *creature.Create(
			r.Form["creatureType"][0],
			r.Form["creatureName"][0],
			roll)
		s.party.EncounterCreatures = append(s.party.EncounterCreatures, newCreature)
	} else {
		if len(args) != 3 {
			return fmt.Errorf("unexpected number of args from regex (%d) - %#v", len(args), args)
		}
		creatureID, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("error parsing int from %v - %v", args[2], err)
		}
		if creatureID >= len(s.party.EncounterCreatures) {
			return fmt.Errorf("invalid creature (out of range) - %v", creatureID)
		}
		if action == "damage" {
			damageAmount, err := strconv.Atoi(r.Form["damageAmount"][0])
			if err != nil {
				return fmt.Errorf("couldn't parse damage amount - %v", err)
			}
			s.party.EncounterCreatures[creatureID].DamageTaken += damageAmount
		} else if action == "delete" {
			s.party.EncounterCreatures = append(s.party.EncounterCreatures[:creatureID],
				s.party.EncounterCreatures[creatureID+1:]...)
		} else {
			return fmt.Errorf("unrecognised action - %v", action)
		}
	}
	return s.party.Save()
}
