package filostack

import "container/list"

// PasteStrategy defines the consumption behavior for a paste order mode.
type PasteStrategy interface {
	// Pop returns the next element to remove from the list.
	Pop(l *list.List) *list.Element

	// ModeName returns a human-readable name for the strategy.
	ModeName() string

	// NextItemIndex returns the index of the next item to paste (for UI display).
	NextItemIndex(total int) int
}

// QueueStrategy implements FIFO (first in, first out).
type QueueStrategy struct{}

func (QueueStrategy) Pop(l *list.List) *list.Element { return l.Front() }

func (QueueStrategy) ModeName() string { return "队列" }

func (QueueStrategy) NextItemIndex(total int) int { return 0 }

// newStrategy returns the PasteStrategy for the given mode string.
func newStrategy(mode string) PasteStrategy {
	if mode == ModeQueue {
		return QueueStrategy{}
	}
	return nil
}
