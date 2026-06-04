package filostack

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service implements a FILO (LIFO) clipboard stack.
// When the stack mode is enabled, each Ctrl+V pops the most recently
// captured text item. Only CF_UNICODETEXT is supported.
type Service struct {
	mu      sync.Mutex
	stack   []string
	enabled bool

	// hook management
	hookStop func()

	// self-paste guard: ignore jPaste's own keybd_event Ctrl+V
	selfPasteUntil time.Time

	// self-write guard: ignore WM_CLIPBOARDUPDATE from our own clipboard writes
	selfWriteHash string
	selfWriteTime time.Time

	// callbacks
	onStateChange func(enabled bool)
	onWriteText   func(text string) bool
}

// NewService creates a Service.
func NewService(onWriteText func(text string) bool) *Service {
	return &Service{
		onWriteText: onWriteText,
	}
}

// WithStateChange sets a callback when stack mode is toggled.
func (s *Service) WithStateChange(fn func(enabled bool)) *Service {
	s.onStateChange = fn
	return s
}

// ServiceStartup implements wails Service.
func (s *Service) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	log.Println("[filostack] ServiceStartup")
	return nil
}

// ServiceShutdown implements wails Service.
func (s *Service) ServiceShutdown() error {
	log.Println("[filostack] ServiceShutdown")
	s.stopHook()
	return nil
}

// Enabled returns whether the stack mode is active.
func (s *Service) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}

// Push adds an item to the top of the stack.
// It skips self-writes (content our own code just wrote to clipboard).
func (s *Service) Push(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled || text == "" {
		log.Printf("[filostack] push: skipped (enabled=%v text empty=%v)", s.enabled, text == "")
		return
	}
	// Skip push for self-writes (content we ourselves just placed on clipboard).
	if s.selfWriteHash != "" && time.Since(s.selfWriteTime) < 5*time.Second {
		h := contentHash(text)
		age := time.Since(s.selfWriteTime).Milliseconds()
		if h == s.selfWriteHash {
			log.Printf("[filostack] skip self-write push (hash=%s age=%dms)", h, age)
			return
		} else {
			log.Printf("[filostack] not self-write (cur=%s last=%s age=%dms)", h, s.selfWriteHash, age)
		}
	} else if s.selfWriteHash == "" {
		log.Printf("[filostack] no self-write marker, pushing")
	} else {
		log.Printf("[filostack] self-write expired (%dms ago)", time.Since(s.selfWriteTime).Milliseconds())
	}
	s.stack = append(s.stack, text)
	log.Printf("[filostack] push: stack size=%d, text=%q", len(s.stack), previewText(text))
}

// Pop removes and returns the top item. Returns false if empty.
func (s *Service) Pop() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.stack) == 0 {
		log.Println("[filostack] pop: stack EMPTY")
		return "", false
	}
	val := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	log.Printf("[filostack] pop: stack size=%d, text=%q", len(s.stack), previewText(val))
	return val, true
}

// Clear empties the stack.
func (s *Service) Clear() {
	s.mu.Lock()
	s.stack = nil
	s.selfWriteHash = ""
	s.mu.Unlock()
	log.Println("[filostack] cleared")
}

// Len returns the stack size.
func (s *Service) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.stack)
}

// SetEnabled enables or disables the stack mode.
// When enabling, starts the keyboard hook; when disabling, stops it and clears the stack.
func (s *Service) SetEnabled(enabled bool) {
	s.mu.Lock()
	if enabled == s.enabled {
		s.mu.Unlock()
		return
	}
	s.enabled = enabled
	s.mu.Unlock()

	if enabled {
		s.startHook()
		log.Println("[filostack] enabled")
	} else {
		s.stopHook()
		s.Clear()
		log.Println("[filostack] disabled (stack cleared)")
	}
	if s.onStateChange != nil {
		s.onStateChange(enabled)
	}
}

// MarkSelfWrite records jPaste's own clipboard write, so subsequent
// WM_CLIPBOARDUPDATE will be ignored by Push().
func (s *Service) MarkSelfWrite(text string) {
	s.mu.Lock()
	s.selfWriteHash = contentHash(text)
	s.selfWriteTime = time.Now()
	s.mu.Unlock()
	log.Printf("[filostack] MarkSelfWrite: hash=%q", s.selfWriteHash)
}

// SetSelfPaste marks that jPaste is about to simulate a Ctrl+V (keybd_event),
// so the keyboard hook should not intercept it.
func (s *Service) SetSelfPaste() {
	s.mu.Lock()
	s.selfPasteUntil = time.Now().Add(500 * time.Millisecond)
	s.mu.Unlock()
}

// --- hook management ---

func (s *Service) startHook() {
	s.mu.Lock()
	if s.hookStop != nil {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	stopFn := platformStartHook(s.handleHookKey)
	if stopFn == nil {
		log.Println("[filostack] WARNING: hook start returned nil (unsupported platform)")
		return
	}
	s.mu.Lock()
	s.hookStop = stopFn
	s.mu.Unlock()
}

func (s *Service) stopHook() {
	s.mu.Lock()
	stopFn := s.hookStop
	s.hookStop = nil
	s.mu.Unlock()
	if stopFn != nil {
		stopFn()
	}
}

// handleHookKey is called from the keyboard hook goroutine when Ctrl+V is pressed.
func (s *Service) handleHookKey() {
	s.mu.Lock()
	isSelfPaste := time.Now().Before(s.selfPasteUntil)
	remaining := time.Until(s.selfPasteUntil).Milliseconds()
	s.mu.Unlock()

	if isSelfPaste {
		log.Printf("[filostack] hook: self-paste guard active (%dms remaining), skipping", remaining)
		return
	}
	text, ok := s.Pop()
	if !ok {
		log.Println("[filostack] hook: stack empty, letting Ctrl+V pass through")
		return
	}
	log.Printf("[filostack] hook: writing popped text=%q, calling clipboard.WriteText", previewText(text))
	s.MarkSelfWrite(text)
	ok = s.onWriteText(text)
	log.Printf("[filostack] hook: clipboard.WriteText returned %v", ok)
}

// --- helpers ---

func contentHash(s string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return fmt.Sprintf("%x", h[:])
}

func previewText(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

// platformStartHook starts the platform-specific keyboard hook.
// Returns nil on unsupported platforms.
var platformStartHook func(onVKeyDown func()) func()
