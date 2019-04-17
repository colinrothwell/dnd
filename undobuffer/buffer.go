package undobuffer

import "fmt"

// Buffer is a stack of fixed size, stored under the hood using a circular buffer.
// If the Buffer is full, the oldest value will be overwritten.
type Buffer struct {
	data                                 []interface{}
	lowestValidElement, nextSpace, limit int
	isFull, canUnpop                     bool
}

/*
 * The isFull boolean feels inelegant to me, but I can't think of a way to avoid its use.
 * We need some way of disambguitating whether I am about to write over the value that is at the
 * bottom of the array because the buffer is full, and I'm overwriting a valid value, or because
 * the buffer is empty, and I am writing the first valid item.
 */

// NewBuffer returns a new buffer storing up to size elements
func NewBuffer(size int) *Buffer {
	return &Buffer{
		make([]interface{}, size),
		0,
		0,
		0,
		false,
		false,
	}
}

// Peek gets the last element to be pushed to the buffer
func (b *Buffer) Peek() (interface{}, error) {
	if !b.isEmpty() {
		return b.data[b.nextSpace-1], nil
	}
	return nil, fmt.Errorf("buffer has no elements")
}

// Push puts an item onto the end of the buffer, overwriting the oldest element if the buffer is full
func (b *Buffer) Push(item interface{}) {
	b.data[b.nextSpace] = item
	b.incrementNextSpace()
	b.limit = b.nextSpace
	b.canUnpop = false
}

// Pop removes and returns the last element of the buffer
func (b *Buffer) Pop() (interface{}, error) {
	if !b.isEmpty() {
		b.nextSpace--
		if b.nextSpace < 0 {
			b.nextSpace = len(b.data) - 1
		}
		b.isFull = false
		b.canUnpop = true
		return b.data[b.nextSpace], nil
	}
	return nil, fmt.Errorf("buffer has no elements")
}

// Unpop undoes the effect of removing the last element if that element was popper, and hasn't been
// overwritten
func (b *Buffer) Unpop() (interface{}, error) {
	if b.canUnpop {
		indexOfElement := b.nextSpace
		b.incrementNextSpace()
		if b.limit == b.nextSpace {
			b.canUnpop = false
		}
		return b.data[indexOfElement], nil
	}
	return nil, fmt.Errorf("no valid elements remaining on buffer")
}

/* The difficult case is when lowestValidElement is above nextSpace

    |<-------------len(b.data)---------------->|
    |------------------------------------------|
			^             ^
			|             |
		nextSpace	 lowestValidElement
			|<----------->|
		 lowestValidElement - nextSpace

So the length is len(b.data) - (lowestValidElement - nextSpace)
*/

// Len returns the number of iterms currently stored in the buffer
func (b *Buffer) Len() int {
	if b.isEmpty() {
		return 0
	}
	if b.nextSpace <= b.lowestValidElement {
		return len(b.data) - (b.lowestValidElement - b.nextSpace)
	}
	return b.nextSpace - b.lowestValidElement
}

func (b *Buffer) isEmpty() bool {
	return !(b.isFull || b.nextSpace != b.lowestValidElement)
}

/* In general, if we advance the nextSpace past the end, past lowestValidElement, we have
 * overwritten the lowestValidElement, so need to update it. It's easiest just to track this.
 */
func (b *Buffer) incrementNextSpace() {
	needToUpdateLowestValidElement := false
	if b.isFull {
		needToUpdateLowestValidElement = true
	}
	b.nextSpace++
	if b.nextSpace >= len(b.data) {
		b.nextSpace = 0
	}
	if needToUpdateLowestValidElement {
		b.lowestValidElement = b.nextSpace
	}
	if b.nextSpace == b.lowestValidElement {
		b.isFull = true
	}
}
