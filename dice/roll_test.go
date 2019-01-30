package dice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringRoll(t *testing.T) {
	emptyFaceCountMap := createFaceCountMap()
	d20 := createFaceCountMap()
	d20.add(1, 20)
	positiveD20Roll := &Roll{d20, emptyFaceCountMap, 0}
	assert.Equal(t, "d20", positiveD20Roll.String())
	negativeD20Roll := &Roll{emptyFaceCountMap, d20, 0}
	assert.Equal(t, "-d20", negativeD20Roll.String())
	positiveAndNegativeD20Roll := &Roll{d20, d20, 0}
	assert.Equal(t, "d20 - d20", positiveAndNegativeD20Roll.String())
	positiveOffsetRoll := &Roll{emptyFaceCountMap, emptyFaceCountMap, 1337}
	assert.Equal(t, "1337", positiveOffsetRoll.String())
	negativeOffsetRoll := &Roll{emptyFaceCountMap, emptyFaceCountMap, -420}
	assert.Equal(t, "-420", negativeOffsetRoll.String())
	positiveRollWithOffset := &Roll{d20, emptyFaceCountMap, -420}
	assert.Equal(t, "d20 - 420", positiveRollWithOffset.String())
	negativeRollWithOffset := &Roll{emptyFaceCountMap, d20, 1337}
	assert.Equal(t, "-d20 + 1337", negativeRollWithOffset.String())
}

func TestStringFaceCountMapResults(t *testing.T) {
	assert.Equal(t, "4", StringFaceCountMapResults([][]uint{[]uint{4}}))
	assert.Equal(t, "1 + 2", StringFaceCountMapResults([][]uint{[]uint{1, 2}}))
	assert.Equal(t, "1 + 2", StringFaceCountMapResults([][]uint{[]uint{1}, []uint{2}}))
	assert.Equal(t, "(1 + 2) + 3", StringFaceCountMapResults([][]uint{[]uint{1, 2}, []uint{3}}))
	assert.Equal(t, "1 + (2 + 3)", StringFaceCountMapResults([][]uint{[]uint{1}, []uint{2, 3}}))
}
