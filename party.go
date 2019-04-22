package main

import (
	"dnd/creature"
	"dnd/dice"
	"dnd/undobuffer"
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
)

// PartyAction is a modification of the party
type PartyAction interface {
	apply(*Party)
}

// ReversiblePartyAction is an action that can be undone.
// Undo must take the Party after applying its change, and return it as it was before Apply.
type ReversiblePartyAction interface {
	apply(*Party)
	undo(*Party)
}

// AddCreatureAction adds a creature to the encounter
type AddCreatureAction struct {
	creature *creature.Creature
}

func (a *AddCreatureAction) apply(p *Party) {
	p.EncounterCreatures = append(p.EncounterCreatures, a.creature)
}

func (a *AddCreatureAction) undo(p *Party) {
	p.EncounterCreatures = p.EncounterCreatures[:len(p.EncounterCreatures)-1]
}

// DamageCreatureAction subtracts a number of hitpoints from a creature
type DamageCreatureAction struct {
	id, amount int
}

func (a *DamageCreatureAction) apply(p *Party) {
	p.EncounterCreatures[a.id].DamageTaken += a.amount
}

func (a *DamageCreatureAction) undo(p *Party) {
	p.EncounterCreatures[a.id].DamageTaken -= a.amount
}

// DeleteCreatureAction deletes a creature
type DeleteCreatureAction struct {
	id              int
	deletedCreature *creature.Creature
}

func NewDeleteCreatureAction(p *Party, id int) *DeleteCreatureAction {
	return &DeleteCreatureAction{id, p.EncounterCreatures[id]}
}

func (a *DeleteCreatureAction) apply(p *Party) {
	p.EncounterCreatures = append(p.EncounterCreatures[:a.id], p.EncounterCreatures[a.id+1:]...)
}

func (a *DeleteCreatureAction) undo(p *Party) {
	firstPart := p.EncounterCreatures[:a.id-1]
	secondPart := p.EncounterCreatures[a.id:]
	p.EncounterCreatures = append(firstPart, a.deletedCreature)
	p.EncounterCreatures = append(p.EncounterCreatures, secondPart...)
}

// Party represents all of the information about a party that should be persisted
type Party struct {
	Filename, Name string
	actions        *undobuffer.Buffer

	// For dice server
	PreviousRolls  []dice.RollResult
	LastCustomRoll string

	// For encounter server
	EncounterCreatures []*creature.Creature
}

// NewParty creates a new party to be saved in the given directory
func NewParty(directory string, name string) *Party {
	return &Party{
		filepath.Join(directory, name+".party.gob"),
		name,
		undobuffer.NewBuffer(64),
		make([]dice.RollResult, 0),
		"",
		make([]*creature.Creature, 0)}
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

// Load a party from a gob file
func LoadParty(file *os.File) (*Party, error) {
	decoder := gob.NewDecoder(file)
	// Create a party this way in order to ensure that the undobuffer gets created if necessary.
	// A bit ugly on the GC.
	party := NewParty("", "")
	err := decoder.Decode(party)
	if err != nil {
		return nil, err
	}
	return party, nil
}
