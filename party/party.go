package party

import (
	"dnd/creature"
	"dnd/dice"
	"dnd/undobuffer"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// party is a structure suitable for storing as a gob that fulfils the requirements of the
// interface. It is private so that I can use public methods and get auto-gob persistence (lazy!)
type party struct {
	Filename, PartyName string
	actions             *undobuffer.Buffer

	// For dice server
	PreviousRolls  []dice.RollResult
	LastCustomRoll string

	// For encounter server
	EncounterCreatures []*creature.Creature
}

// Save the party to its Filename'd .gob file
func (p *party) Save() error {
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

// Party represents a party in a game of D&D
type Party interface {
	Name() string
	Save() error
	Apply(action Action)
	Undo() error

	RollInformation
	EncounterInformation
}

// RollInformation represents information about the rolls in a game of D&D
type RollInformation interface {
	CustomRoll() string
	SetCustomRoll(string)
	Rolls() []*dice.RollResult
	AddRoll(dice.RollResult)
}

// EncounterInformation represents information about the encounters in a game of D&D
type EncounterInformation interface {
	Creatures() []*creature.Creature
	DeleteCreatureAction(ID int) *DeleteCreatureAction
}

// New creates a new party to be saved in the given directory
func New(directory string, name string) Party {
	return &party{
		filepath.Join(directory, name+".party.gob"),
		name,
		undobuffer.NewBuffer(64),
		make([]dice.RollResult, 0),
		"",
		make([]*creature.Creature, 0)}
}

// Load party from a gob file
func Load(file *os.File) (Party, error) {
	decoder := gob.NewDecoder(file)
	// Create a party this way in order to ensure that the undobuffer gets created if necessary.
	// A bit ugly on the GC.
	party := New("", "")
	err := decoder.Decode(party)
	if err != nil {
		return nil, err
	}
	return party, nil
}

// Name returns the party's name
func (p *party) Name() string {
	if p.PartyName != "" {
		return p.PartyName
	}
	file := filepath.Base(p.Filename)
	nameGuess := file[:len(file)-len(".party.gob")]
	p.PartyName = nameGuess
	return nameGuess
}

// Apply an action to the party, adding it to the undo buffer
func (p *party) Apply(action Action) {
	p.actions.Push(action)
	action.apply(p)
}

// Undo the last action in the buffer
func (p *party) Undo() error {
	raw, err := p.actions.Pop()
	if err != nil {
		return err
	}
	action, ok := raw.(ReversibleAction)
	if !ok {
		return fmt.Errorf("undobuffer contains '%v', not a Reversible Action", raw)
	}
	action.undo(p)
	return nil
}

// CustomRoll is the last thing the user typed in the custom roll box
func (p *party) CustomRoll() string {
	return p.LastCustomRoll
}

// SetCustomRoll sets the last thing the user typed in the custom roll box
func (p *party) SetCustomRoll(roll string) {
	p.LastCustomRoll = roll
}

// Rolls returns the results of the user's rolls, most recent first
func (p *party) Rolls() []*dice.RollResult {
	r := make([]*dice.RollResult, len(p.PreviousRolls))
	for i := 0; i < len(p.PreviousRolls); i++ {
		r[len(p.PreviousRolls)-1-i] = &p.PreviousRolls[i]
	}
	return r
}

// AddRoll adds the result of rolling dice to the party
func (p *party) AddRoll(roll dice.RollResult) {
	p.PreviousRolls = append(p.PreviousRolls, roll)
}

// Creatures returns the creatures in the party's encounters
func (p *party) Creatures() []*creature.Creature {
	return p.EncounterCreatures
}

// DeleteCreatureAction creatures a delete creature action for a creature
func (p *party) DeleteCreatureAction(ID int) *DeleteCreatureAction {
	return newDeleteCreatureAction(p, ID)
}
