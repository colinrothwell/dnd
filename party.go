package main

import (
	"dnd/creature"
	"dnd/dice"
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
)

// Party represents all of the information about a party that should be persisted
type Party struct {
	Filename, Name string

	// For dice server
	PreviousRolls  []dice.RollResult
	LastCustomRoll string

	// For encounter server
	EncounterCreatures []creature.Creature
}

// NewParty creates a new party to be saved in the given directory
func NewParty(directory string, name string) *Party {
	return &Party{
		filepath.Join(directory, name+".party.gob"),
		name,
		make([]dice.RollResult, 0),
		"",
		make([]creature.Creature, 0)}
}

// Save the party to its Filename'd .gob file
func (p *Party) Save() error {
	file, err := os.Create(p.Filename)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing party file - %v", err)
		}
	}()
	encoder := gob.NewEncoder(file)
	return encoder.Encode(p)
}
