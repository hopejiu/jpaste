package util

import (
	"sync"
	"time"
)

// SelfWriteTracker tracks whether a recent clipboard write was performed
// by the application itself (self-write), using a hash + 5-second TTL pattern.
type SelfWriteTracker struct {
	mu       sync.Mutex
	hash     string
	writTime time.Time
}

// Mark records the hash of the text being written as a self-write.
func (t *SelfWriteTracker) Mark(text string) {
	t.mu.Lock()
	t.hash = SHA256String(text)
	t.writTime = time.Now()
	t.mu.Unlock()
}

// IsSelfWrite checks if a captured text matches the most recent self-write.
// Returns true only if the hash matches and the write occurred within 5 seconds.
func (t *SelfWriteTracker) IsSelfWrite(text string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.hash == "" {
		return false
	}
	if time.Since(t.writTime).Milliseconds() > 5000 {
		return false
	}
	return SHA256String(text) == t.hash
}

// IsExpired returns true if the tracker has no recent self-write or the TTL has elapsed.
func (t *SelfWriteTracker) IsExpired() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hash == "" || time.Since(t.writTime).Milliseconds() > 5000
}

// Clear resets the tracker.
func (t *SelfWriteTracker) Clear() {
	t.mu.Lock()
	t.hash = ""
	t.mu.Unlock()
}

// Hash returns the tracked hash (empty if none).
func (t *SelfWriteTracker) Hash() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hash
}
