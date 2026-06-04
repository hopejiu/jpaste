package filostack

import (
	"container/list"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	ModeNormal = "normal"
	ModeStack  = "stack"
	ModeQueue  = "queue"
)

// Service implements a paste-order controller: normal / stack (FILO) / queue (FIFO).
// Only CF_UNICODETEXT is supported.
type Service struct {
	mu    sync.Mutex
	items *list.List
	mode  string

	// hook management
	hookStop func()

	// self-paste guard: ignore jPaste's own keybd_event Ctrl+V
	selfPasteUntil time.Time

	// self-write guard: ignore WM_CLIPBOARDUPDATE from our own clipboard writes
	selfWriteHash string
	selfWriteTime time.Time

	// callbacks
	onWriteText func(text string) bool
}

// NewService creates a Service.
func NewService(onWriteText func(text string) bool) *Service {
	return &Service{
		onWriteText: onWriteText,
		items:       list.New(),
		mode:        ModeNormal,
	}
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

// Mode returns the current paste order mode.
func (s *Service) Mode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// Enabled returns whether a non-normal mode is active.
func (s *Service) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode != ModeNormal
}

// Push adds an item to the back of the list.
// It skips self-writes (content our own code just wrote to clipboard).
func (s *Service) Push(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == ModeNormal || text == "" {
		return
	}
	// Skip push for self-writes.
	if s.selfWriteHash != "" && time.Since(s.selfWriteTime) < 5*time.Second {
		h := contentHash(text)
		if h == s.selfWriteHash {
			log.Printf("[filostack] skip self-write push (hash=%s)", h)
			return
		}
	}
	s.items.PushBack(text)
	log.Printf("[filostack] push: size=%d, mode=%s, text=%q", s.items.Len(), s.mode, previewText(text))
}

// Pop removes an item from the list. Direction depends on mode:
//   - stack: removes from the back (LIFO)
//   - queue: removes from the front (FIFO)
//   - normal: always returns false
func (s *Service) Pop() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items.Len() == 0 {
		log.Printf("[filostack] pop: empty (mode=%s)", s.mode)
		return "", false
	}
	var e *list.Element
	var direction string
	switch s.mode {
	case ModeStack:
		e = s.items.Back()
		direction = "back (stack)"
	case ModeQueue:
		e = s.items.Front()
		direction = "front (queue)"
	default:
		return "", false
	}
	val := e.Value.(string)
	s.items.Remove(e)
	log.Printf("[filostack] pop: size=%d, from=%s, text=%q", s.items.Len(), direction, previewText(val))
	return val, true
}

// Clear empties the list.
func (s *Service) Clear() {
	s.mu.Lock()
	s.items.Init()
	s.selfWriteHash = ""
	s.mu.Unlock()
	log.Println("[filostack] cleared")
}

// Len returns the number of items.
func (s *Service) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.items.Len()
}

// SetMode changes the paste order mode.
// Switching to normal stops the hook; switching between stack/queue clears the list.
func (s *Service) SetMode(mode string) {
	s.mu.Lock()
	if mode == s.mode {
		s.mu.Unlock()
		return
	}
	oldMode := s.mode
	s.mode = mode
	s.mu.Unlock()

	log.Printf("[filostack] SetMode: %s → %s", oldMode, mode)

	if mode == ModeNormal {
		s.stopHook()
		s.Clear()
	} else {
		// Clear list when switching from any mode to a different non-normal mode.
		s.Clear()
		// Start hook if not already running (first time or after normal).
		// No need to restart if already running from a previous non-normal mode.
		// But we stopped + cleared, so we need to start.
		s.startHook()
	}
}

// MarkSelfWrite records jPaste's own clipboard write.
func (s *Service) MarkSelfWrite(text string) {
	s.mu.Lock()
	s.selfWriteHash = contentHash(text)
	s.selfWriteTime = time.Now()
	s.mu.Unlock()
	log.Printf("[filostack] MarkSelfWrite: hash=%s", s.selfWriteHash)
}

// SetSelfPaste marks that jPaste is about to simulate a Ctrl+V (keybd_event).
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
		log.Printf("[filostack] hook: self-paste guard active (%dms), skipping", remaining)
		return
	}
	text, ok := s.Pop()
	if !ok {
		log.Println("[filostack] hook: no item, letting Ctrl+V pass through")
		return
	}
	log.Printf("[filostack] hook: writing text=%q", previewText(text))
	s.MarkSelfWrite(text)
	ok = s.onWriteText(text)
	log.Printf("[filostack] hook: WriteText returned %v", ok)
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
var platformStartHook func(onVKeyDown func()) func()
