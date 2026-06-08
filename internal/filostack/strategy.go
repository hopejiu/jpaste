package filostack

import "container/list"

// PasteStrategy defines the consumption behavior for a paste order mode.
type PasteStrategy interface {
	// Pop returns the next element to remove from the list.
	Pop(l *list.List) *list.Element

	// ModeName returns a human-readable name for the strategy (e.g. "栈", "队列").
	ModeName() string

	// NextItemIndex returns the index of the next item to paste (for UI display).
	// -1 means last item (for stack LIFO), 0 means first item (for queue FIFO).
	NextItemIndex(total int) int
}

// StackStrategy implements LIFO (last in, first out).
type StackStrategy struct{}

func (StackStrategy) Pop(l *list.List) *list.Element { return l.Back() }

func (StackStrategy) ModeName() string { return "栈" }

func (StackStrategy) NextItemIndex(total int) int { return total - 1 }

// QueueStrategy implements FIFO (first in, first out).
type QueueStrategy struct{}

func (QueueStrategy) Pop(l *list.List) *list.Element { return l.Front() }

func (QueueStrategy) ModeName() string { return "队列" }

func (QueueStrategy) NextItemIndex(total int) int { return 0 }

// newStrategy returns the PasteStrategy for the given mode string.
func newStrategy(mode string) PasteStrategy {
	switch mode {
	case ModeStack:
		return StackStrategy{}
	case ModeQueue:
		return QueueStrategy{}
	default:
		return nil
	}
}
