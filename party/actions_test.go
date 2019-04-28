package party

import (
	"dnd/creature"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddCreatureAction(t *testing.T) {
	p := testingParty()
	c := creature.Create("test", "colin", testDiceRoll(1337))
	action := &AddCreatureAction{c}
	action.apply(p)
	assert.Equal(t, []*creature.Creature{c}, p.EncounterCreatures)
	action.undo(p)
	assert.Equal(t, []*creature.Creature{}, p.EncounterCreatures)
}

func TestDamageCreatureAction(t *testing.T) {
	p := testingParty()
	c := creature.Create("test", "foo", testDiceRoll(50))
	p.Apply(&AddCreatureAction{c})
	assert.Equal(t, 0, p.EncounterCreatures[0].DamageTaken)
	a := &DamageCreatureAction{0, 20}
	a.apply(p)
	assert.Equal(t, 20, p.EncounterCreatures[0].DamageTaken)
	a.undo(p)
	assert.Equal(t, 0, p.EncounterCreatures[0].DamageTaken)
}

func TestDeleteCreatureAction(t *testing.T) {
	p := testingParty()
	c := creature.Create("baz", "bar", testDiceRoll(1337))
	p.Apply(&AddCreatureAction{c})
	a := newDeleteCreatureAction(p, 0)
	a.apply(p)
	assert.Equal(t, []*creature.Creature{}, p.EncounterCreatures)
	a.undo(p)
	assert.Equal(t, []*creature.Creature{c}, p.EncounterCreatures)
}

func TestDeleteOneOfMultipleCreatures(t *testing.T) {
	p := testingParty()
	cs := make([]*creature.Creature, 4)
	for i := 0; i < 4; i++ {
		cs[i] = creature.Create("test", "cret", testDiceRoll(i))
		p.Apply(&AddCreatureAction{cs[i]})
	}
	a := newDeleteCreatureAction(p, 2)
	a.apply(p)
	assert.Equal(t, []*creature.Creature{cs[0], cs[1], cs[3]}, p.EncounterCreatures)
	a.undo(p)
	assert.Equal(t, cs, p.EncounterCreatures)
}
