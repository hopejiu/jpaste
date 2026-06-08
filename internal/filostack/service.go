package filostack

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"jpaste/internal/util"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	ModeNormal = "normal"
	ModeStack  = "stack"
	ModeQueue  = "queue"
)
type Service struct {
	mu    sync.Mutex
	items *list.List
	mode  string

	// strategy defines the consumption behavior (stack / queue).
	strategy PasteStrategy

	// hook management
	hookStop func()

	selfTracker     util.SelfWriteTracker
	selfPasteUntil time.Time

	onWriteText func(text string) bool
	onNotify    func(title, msg string)
}

// Option configures Service.
type Option func(*Service)

// WithNotifyFunc sets a callback invoked when an item is pasted.
func WithNotifyFunc(fn func(title, msg string)) Option {
	return func(s *Service) { s.onNotify = fn }
}

// NewService creates a Service.
func NewService(onWriteText func(text string) bool, opts ...Option) *Service {
	s := &Service{
		onWriteText: onWriteText,
		items:       list.New(),
		mode:        ModeNormal,
	}
	s.updateStrategy()
	for _, opt := range opts {
		opt(s)
	}
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

// Mode returns the current paste order mode.
func (s *Service) Mode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// ModeName returns a human-readable name for the current paste order mode.
func (s *Service) ModeName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strategy != nil {
		return s.strategy.ModeName()
	}
	return ""
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
	if !s.selfTracker.IsExpired() && s.selfTracker.IsSelfWrite(text) {
		log.Printf("[filostack] skip self-write push (hash=%s)", s.selfTracker.Hash())
		return
	}
	s.items.PushBack(text)
	log.Printf("[filostack] push: size=%d, mode=%s, text=%q", s.items.Len(), s.mode, util.TruncateBytes(text, 40))
}

// Pop removes an item from the list. Direction depends on strategy:
//   - stack: removes from the back (LIFO)
//   - queue: removes from the front (FIFO)
//   - normal: always returns false
func (s *Service) Pop() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items.Len() == 0 || s.strategy == nil {
		log.Printf("[filostack] pop: empty (mode=%s)", s.mode)
		return "", false
	}
	e := s.strategy.Pop(s.items)
	if e == nil {
		return "", false
	}
	val := e.Value.(string)
	s.items.Remove(e)
	log.Printf("[filostack] pop: size=%d, from=%s, text=%q", s.items.Len(), s.strategy.ModeName(), util.TruncateBytes(val, 40))
	return val, true
}

// Clear empties the list.
func (s *Service) Clear() {
	s.mu.Lock()
	s.items.Init()
	s.selfTracker.Clear()
	s.mu.Unlock()
	log.Println("[filostack] cleared")
}

// Len returns the number of items.
func (s *Service) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.items.Len()
}

// GetItems returns previews of all items in the current list (newest first for stack).
// Returns nil when mode is normal. Exposed to frontend via Wails binding.
func (s *Service) GetItems() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == ModeNormal || s.items.Len() == 0 {
		return nil
	}

	// Stack: iterate from back (LIFO — next to paste is last item).
	// Queue: iterate from front (FIFO — next to paste is first item).
	// Frontend renders next-item indicator regardless.
	items := make([]string, 0, s.items.Len())
	for e := s.items.Front(); e != nil; e = e.Next() {
		text := e.Value.(string)
		items = append(items, util.TruncateBytes(text, 40))
	}
	return items
}

// SetMode changes the paste order mode.
// Switching to normal stops the hook and clears the list.
// Switching between stack/queue preserves existing items.
func (s *Service) SetMode(mode string) {
	s.mu.Lock()
	if mode == s.mode {
		s.mu.Unlock()
		return
	}
	oldMode := s.mode
	s.mode = mode
	s.strategy = newStrategy(mode)
	s.mu.Unlock()

	log.Printf("[filostack] SetMode: %s → %s (strategy=%v)", oldMode, mode, s.strategy != nil)

	if mode == ModeNormal {
		s.stopHook()
		s.Clear()
	} else if oldMode == ModeNormal {
		// Switching from normal to stack/queue: start fresh.
		s.Clear()
		s.startHook()
	} else {
		// Switching between stack/queue: preserve items, hook is already running.
		log.Printf("[filostack] preserved %d items", s.Len())
	}
}

// updateStrategy sets the strategy based on the current mode.
func (s *Service) updateStrategy() {
	s.strategy = newStrategy(s.mode)
}

// MarkSelfWrite records jPaste's own clipboard write.
func (s *Service) MarkSelfWrite(text string) {
	s.mu.Lock()
	s.selfTracker.Mark(text)
	hash := s.selfTracker.Hash()
	s.mu.Unlock()
	log.Printf("[filostack] MarkSelfWrite: hash=%s", hash)
}

// SetSelfPaste marks that jPaste is about to simulate a Ctrl+V (keybd_event).
func (s *Service) SetSelfPaste() {
	s.mu.Lock()
	s.selfPasteUntil = time.Now().Add(500 * time.Millisecond)
	s.mu.Unlock()
}

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
	log.Printf("[filostack] hook: writing text=%q", util.TruncateBytes(text, 40))
	s.MarkSelfWrite(text)
	ok = s.onWriteText(text)
	log.Printf("[filostack] hook: WriteText returned %v", ok)
	if ok && s.onNotify != nil {
		// Pop happened above — s.Len() is the remaining count after this paste.
		s.onNotify("jPaste", fmt.Sprintf("成功粘贴, 当前%s已有: %d 个", s.ModeName(), s.Len()))
		return
	}
}

// platformStartHook starts the platform-specific keyboard hook.
var platformStartHook func(onVKeyDown func()) func()
