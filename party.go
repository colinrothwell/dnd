package main

import "dnd/creature"

type Party struct {
	Filename, Name     string
	EncounterCreatures []creature.Creature
}
