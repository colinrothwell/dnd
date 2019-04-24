package party

import (
	"dnd/creature"
	"dnd/dice"
	"dnd/undobuffer"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testingParty() *Party {
	var p Party
	p.actions = undobuffer.NewBuffer(4)
	return &p
}

func testDiceRoll(maxHealth int) *dice.Roll {
	d, err := dice.ParseRollString(strconv.Itoa(maxHealth))
	if err != nil {
		panic(err)
	}
	return d
}

func TestBasicUndo(t *testing.T) {
	p := testingParty()
	c := creature.Create("colinis", "great", testDiceRoll(50))
	action := &AddCreatureAction{c}
	p.Apply(action)
	assert.Equal(t, []*creature.Creature{c}, p.EncounterCreatures)
	p.Undo()
	assert.Equal(t, []*creature.Creature{}, p.EncounterCreatures)

}
