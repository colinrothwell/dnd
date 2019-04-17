package undobuffer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonWrappingBehaviour(t *testing.T) {
	b := NewBuffer(8)
	el, err := b.Peek()
	assert.Error(t, err)
	assert.Equal(t, 0, b.Len())
	b.Push(8)
	assert.Equal(t, 1, b.Len())
	el, err = b.Peek()
	assert.NoError(t, err)
	assert.Equal(t, 8, el)
	assert.Equal(t, 1, b.Len())
	el, err = b.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 8, el)
	assert.Equal(t, 0, b.Len())
	_, err = b.Pop()
	assert.Error(t, err)
	el, err = b.Unpop()
	assert.NoError(t, err)
	assert.Equal(t, 8, el)
	assert.Equal(t, 1, b.Len())
}

func TestUnpopOnceThenReadd(t *testing.T) {
	b := NewBuffer(4)
	b.Push(0)
	b.Push(1)
	el, err := b.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 1, el)
	el, err = b.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 0, el)
	el, err = b.Unpop()
	assert.NoError(t, err)
	assert.Equal(t, 0, el)
	b.Push(1337)
	el, err = b.Peek()
	assert.NoError(t, err)
	assert.Equal(t, 1337, el)
	_, err = b.Unpop()
	assert.Error(t, err)
}

func TestWrapping(t *testing.T) {
	b := NewBuffer(3)
	b.Push(0)
	b.Push(1)
	b.Push(2)
	b.Push(3)
	for _, expected := range []int{3, 2, 1} {
		el, err := b.Pop()
		assert.NoError(t, err)
		assert.Equal(t, expected, el)
		assert.Equal(t, expected-1, b.Len())
	}
	_, err := b.Pop()
	assert.Error(t, err)
	b.Push(1337)
	el, err := b.Peek()
	assert.NoError(t, err)
	assert.Equal(t, el, 1337)
	assert.Equal(t, 1, b.Len())
	el, err = b.Pop()
	assert.NoError(t, err)
	assert.Equal(t, el, 1337)
	el, err = b.Unpop()
	assert.NoError(t, err)
	assert.Equal(t, el, 1337)
}

func TestWrapOverTwice(t *testing.T) {
	b := NewBuffer(10)
	for i := 0; i <= 25; i++ {
		b.Push(i)
	}
	length := 10
	assert.Equal(t, length, b.Len())
	for i := 25; i > 15; i-- {
		el, err := b.Pop()
		assert.NoError(t, err)
		assert.Equal(t, el, i)
		length--
		assert.Equal(t, length, b.Len())
	}
	_, err := b.Pop()
	assert.Error(t, err)
	length = 0
	assert.Equal(t, length, b.Len())
	for i := 16; i <= 25; i++ {
		fmt.Println(i)
		el, err := b.Unpop()
		assert.NoError(t, err)
		assert.Equal(t, i, el)
	}
	_, err = b.Unpop()
	assert.Error(t, err)
	for i := 0; i < 5; i++ {
		_, err = b.Pop()
		assert.NoError(t, err)
	}
	for i := 0; i < 2; i++ {
		_, err = b.Unpop()
		assert.NoError(t, err)
	}
	b.Push(1337)
	_, err = b.Unpop()
	assert.Error(t, err)
}
