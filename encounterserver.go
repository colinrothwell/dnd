package main

import (
	"dnd/creature"
	"dnd/dice"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

type CreatureInformation struct {
	Type, Name                                                   string
	MinHealth, RolledHealth, MaxHealth, DamageTaken              int
	DamageURL, MinHealthClass, RolledHealthClass, MaxHealthClass string
}

type EncounterData struct {
	CreatureInformation                       []CreatureInformation
	NextCreatureTypeName, NextCreatureHitDice string
}

type EncounterServer struct {
	template *template.Template
	party    *Party
}

func (s *EncounterServer) GetTemplate() *template.Template {
	return s.template
}

func (s *EncounterServer) GenerateTemplateData(r *http.Request) interface{} {
	creatureCount := len(s.party.EncounterCreatures)
	creatureInformations := make([]CreatureInformation, creatureCount)
	for i, creature := range s.party.EncounterCreatures {
		var minHC, rolledHC, maxHC string
		if 2*creature.DamageTaken >= creature.RolledHealth {
			rolledHC = "damaged"
		}
		if creature.DamageTaken >= creature.Type.HitDice.Min() {
			minHC = "dead"
			if creature.DamageTaken >= creature.RolledHealth {
				rolledHC = "dead"
				if creature.DamageTaken >= creature.Type.HitDice.Max() {
					maxHC = "dead"
				}
			}
		}
		creatureInformationIndex := creatureCount - 1 - i
		creatureInformations[creatureInformationIndex] = CreatureInformation{
			creature.Type.Name,
			creature.Name,
			creature.Type.HitDice.Min(),
			creature.RolledHealth,
			creature.Type.HitDice.Max(),
			creature.DamageTaken,
			r.RequestURI + strconv.Itoa(i),
			minHC,
			rolledHC,
			maxHC}
	}
	data := EncounterData{creatureInformations, "", ""}
	if creatureCount > 0 {
		nextCreatureType := s.party.EncounterCreatures[creatureCount-1].Type
		data.NextCreatureTypeName = nextCreatureType.Name
		data.NextCreatureHitDice = nextCreatureType.HitDice.String()
	}
	return data
}

func (s *EncounterServer) HandlePost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	redirectURL, arg := getURLFunctionAndArgument(r.URL)
	if arg == "" {
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
		s.party.EncounterCreatures = append(s.party.EncounterCreatures, newCreature)
	} else {
		creatureToDamage, err := strconv.Atoi(arg)
		if err != nil {
			log.Printf("Error parsing int from %v - %v", arg, err)
			http.Redirect(w, r, redirectURL, 303)
			return
		}
		if creatureToDamage >= len(s.party.EncounterCreatures) {
			log.Printf("Invalid creature (out of range) - %v", creatureToDamage)
			http.Redirect(w, r, redirectURL, 303)
			return
		}
		damageAmount, err := strconv.Atoi(r.Form["damageAmount"][0])
		if err != nil {
			log.Printf("Couldn't parse damage amount - %v", err)
		}
		s.party.EncounterCreatures[creatureToDamage].DamageTaken += damageAmount
	}
	s.party.Save()
	http.Redirect(w, r, redirectURL, 303)
}
