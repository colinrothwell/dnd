package party

import (
	"dnd/creature"
	"dnd/dice"
	"dnd/undobuffer"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Encounter creature represents the information about a creature for the purpose of one
// specific encounter
type EncounterCreature struct {
	Name           string
	InitiativeDice dice.Roll
	Initiative     int
}

// party is a structure suitable for storing as a gob that fulfils the requirements of the
// interface. It is private so that I can use public methods and get auto-gob persistence (lazy!)
type party struct {
	Filename, PartyName string
	actions             *undobuffer.Buffer
	Players             []*Player

	// For dice server
	PreviousRolls  []dice.RollResult
	LastCustomRoll string

	// For encounter server
	EncounterCreatures []*creature.Creature

	// For initiative server
	PlayerHasInitiatives      []bool
	PlayerInitiativeRolls     []int
	CurrentEncounterCreatures []*EncounterCreature
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
	Apply(action Action) error
	Undo() error
	CanUndo() bool
	Redo() error
	CanRedo() bool

	RollInformation
	EncounterInformation
	InitiativeInformation
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

// InitiativeInformation is the information about a creature's initiative
type CreatureInitiative struct {
	Name          string
	HasInitiative bool
	Initiative    int
}

// InitiativeInformation represents information about the initiative in the current combat
type InitiativeInformation interface {
	PlayerInitiatives() []*CreatureInitiative
}

// New creates a new party to be saved in the given directory
func New(directory string, name string) Party {
	return &party{
		filepath.Join(directory, name+".party.gob"),
		name,
		undobuffer.NewBuffer(64),
		make([]*Player, 0),
		make([]dice.RollResult, 0),
		"",
		make([]*creature.Creature, 0),
		make([]bool, 0),
		make([]int, 0),
		make([]*EncounterCreature, 0)}
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

// Apply an action to the party, adding it to the undo buffer. Returns an error if action is nil
func (p *party) Apply(action Action) error {
	if action == nil {
		return errors.New("can't apply a nil action")
	}
	p.actions.Push(action)
	action.apply(p)
	return nil
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

func (p *party) CanUndo() bool {
	return p.actions.CanPop()
}

func (p *party) Redo() error {
	raw, err := p.actions.Unpop()
	if err != nil {
		return fmt.Errorf("error redoing: %v", err)
	}
	action, ok := raw.(ReversibleAction)
	if !ok {
		return fmt.Errorf("undobuffer contains '%v', not a Reversible Action", raw)
	}
	action.apply(p)
	return nil
}

func (p *party) CanRedo() bool {
	return p.actions.CanUnpop()
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

// PlayerInitiatives gets the information about the players initiatives
func (p *party) PlayerInitiatives() []*CreatureInitiative {
	r := make([]*CreatureInitiative, len(p.Players))
	for i, player := range p.Players {
		r[i] = &CreatureInitiative{
			player.Name,
			p.PlayerHasInitiatives[i],
			p.PlayerInitiativeRolls[i]}
	}
	return r
}
