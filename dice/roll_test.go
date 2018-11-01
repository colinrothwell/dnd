package dice

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiceUnitToString(t *testing.T) {
	assert.Equal(t, "d6", (&diceUnit{1, 6}).String())
	assert.Equal(t, "2d12", (&diceUnit{2, 12}).String())
}

func TestDiceUnitResult(t *testing.T) {
	rand.Seed(0)
	var frequencies [7]int
	d6 := &diceUnit{1, 6}
	for i := 0; i < 600; i++ {
		frequencies[d6.SimulateValue()[0]]++
	}
	assert.Equal(t, 0, frequencies[0])
	assert.Equal(t, 129, frequencies[1])
	assert.Equal(t, 89, frequencies[2])
	assert.Equal(t, 110, frequencies[3])
	assert.Equal(t, 83, frequencies[4])
	assert.Equal(t, 91, frequencies[5])
	assert.Equal(t, 98, frequencies[6])
	// These do some to 600, as we'd hope

	var fLeft [3]int
	var fRight [3]int

	twoD2 := &diceUnit{2, 2}
	for i := 0; i < 200; i++ {
		r := twoD2.SimulateValue()
		fLeft[r[0]]++
		fRight[r[1]]++
	}
	assert.Equal(t, 0, fLeft[0])
	assert.Equal(t, 0, fRight[0])
	assert.Equal(t, 102, fLeft[1])
	assert.Equal(t, 99, fRight[1])
	assert.Equal(t, 98, fLeft[2])
	assert.Equal(t, 101, fRight[2])
}
