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

// New creates a new party to be saved in the given directory
func New(directory string, name string) *Party {
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

// Load party from a gob file
func Load(file *os.File) (*Party, error) {
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

// Apply an action to the party, adding it to the undo buffer
func (p *Party) Apply(action Action) {
	p.actions.Push(action)
	action.apply(p)
}

// Undo the last action in the buffer
func (p *Party) Undo() error {
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
