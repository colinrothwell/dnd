package party

import "dnd/creature"

// Action is a modification of the party
type Action interface {
	apply(*Party)
}

// ReversibleAction is an action that can be undone.
// undo must take the Party after applying its change, and return it as it was before apply.
type ReversibleAction interface {
	apply(*Party)
	undo(*Party)
}

// AddCreatureAction adds a creature to the encounter
type AddCreatureAction struct {
	Creature *creature.Creature
}

func (a *AddCreatureAction) apply(p *Party) {
	p.EncounterCreatures = append(p.EncounterCreatures, a.Creature)
}

func (a *AddCreatureAction) undo(p *Party) {
	p.EncounterCreatures = p.EncounterCreatures[:len(p.EncounterCreatures)-1]
}

// DamageCreatureAction subtracts a number of hitpoints from a creature
type DamageCreatureAction struct {
	ID, Amount int
}

func (a *DamageCreatureAction) apply(p *Party) {
	p.EncounterCreatures[a.ID].DamageTaken += a.Amount
}

func (a *DamageCreatureAction) undo(p *Party) {
	p.EncounterCreatures[a.ID].DamageTaken -= a.Amount
}

// DeleteCreatureAction deletes a creature
type DeleteCreatureAction struct {
	ID              int
	DeletedCreature *creature.Creature
}

// NewDeleteCreatureAction creates an action that deletes a creature.
// Restoring the creature requires keeping a reference to its whole state.
func NewDeleteCreatureAction(p *Party, ID int) *DeleteCreatureAction {
	return &DeleteCreatureAction{ID, p.EncounterCreatures[ID]}
}

func (a *DeleteCreatureAction) apply(p *Party) {
	p.EncounterCreatures = append(p.EncounterCreatures[:a.ID], p.EncounterCreatures[a.ID+1:]...)
}

func (a *DeleteCreatureAction) undo(p *Party) {
	// Operates by adding a new gap at the end, copying everything from the insertion position
	// one along to make room, then putting the deleted creature back in the same place.
	// I googled how to do this! Unit tests stopped me making a stupid mistake...
	p.EncounterCreatures = append(p.EncounterCreatures, nil)
	copy(p.EncounterCreatures[a.ID+1:], p.EncounterCreatures[a.ID:])
	p.EncounterCreatures[a.ID] = a.DeletedCreature
}
