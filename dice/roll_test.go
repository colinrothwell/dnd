package dice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringFaceCountMapResults(t *testing.T) {
	assert.Equal(t, "4", StringFaceCountMapResults([][]uint{[]uint{4}}))
	assert.Equal(t, "1 + 2", StringFaceCountMapResults([][]uint{[]uint{1, 2}}))
	assert.Equal(t, "1 + 2", StringFaceCountMapResults([][]uint{[]uint{1}, []uint{2}}))
	assert.Equal(t, "(1 + 2) + 3", StringFaceCountMapResults([][]uint{[]uint{1, 2}, []uint{3}))
	assert.Equal(t, "1 + (2 + 3)", StringFaceCountMapResults([][]uint{[]uint{1}, []uint{2, 3}}))
}
