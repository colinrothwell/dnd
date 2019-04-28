package party

import (
	"dnd/creature"
	"dnd/dice"
	"dnd/undobuffer"
	"encoding/gob"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testingParty() *party {
	var p party
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

func TestGob(t *testing.T) {
	f, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatalf("Couldn't create temp file - %v", err)
	}
	fn := f.Name()
	defer os.Remove(fn)

	f, err = os.Create(fn)
	e := gob.NewEncoder(f)
	var encodeParty party
	encodeParty.Filename = "colin"
	encodeParty.PartyName = "is"
	err = e.Encode(encodeParty)
	if err != nil {
		t.Fatalf("Error encoding party - %v", err)
	}
	f.Close()

	fi, err := os.Stat(fn)
	t.Logf("File size: %v", fi.Size())

	f, err = os.Open(fn)
	if err != nil {
		t.Fatalf("Error opening file for reading - %v", err)
	}
	d := gob.NewDecoder(f)
	var p party
	p.Filename = "bugger"
	err = d.Decode(&p)
	if err != nil {
		t.Fatalf("Error decoding party - %v", err)
	}
	f.Close()

	if p.Filename != "colin" {
		t.Errorf("Saved party was instead '%v'", p.Filename)
	}
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

func TestRolls(t *testing.T) {
	p := testingParty()
	r1 := testDiceRoll(1).Simulate()
	r2 := testDiceRoll(2).Simulate()
	r3 := testDiceRoll(3).Simulate()
	p.AddRoll(r1)
	p.AddRoll(r2)
	p.AddRoll(r3)
	e := []*dice.RollResult{&r3, &r2, &r1}
	assert.Equal(t, e, p.Rolls())
}

func TestCustomRoll(t *testing.T) {
	p := testingParty()
	p.SetCustomRoll("colin is great!")
	assert.Equal(t, "colin is great!", p.CustomRoll())
}
