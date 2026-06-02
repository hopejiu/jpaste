package clipboard

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Entry represents a clipboard history item.
type Entry struct {
	ID          int64  `json:"id"`
	ContentHash string `json:"content_hash"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Reader reads text from the system clipboard.
type Reader func() (string, bool)

// OnCapture is called when new (non-duplicate) text is detected.
type OnCapture func(text string, hash string)

// Watcher polls the system clipboard and reports new text via OnCapture.
// It has no knowledge of the database or event system — those are wired at startup.
type Watcher struct {
	readCB   Reader
	onNew    OnCapture
	lastHash string
	ctx      context.Context
}

// NewWatcher creates a clipboard Watcher.
func NewWatcher(readCB Reader, onNew OnCapture) *Watcher {
	return &Watcher{readCB: readCB, onNew: onNew}
}

// ServiceStartup begins the 1s polling loop.
func (w *Watcher) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	w.ctx = ctx
	go w.pollLoop()
	return nil
}

// ServiceShutdown is called when the application exits.
func (w *Watcher) ServiceShutdown() error {
	return nil
}

func (w *Watcher) pollLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.check()
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Watcher) check() {
	if w.readCB == nil || w.onNew == nil {
		return
	}
	text, ok := w.readCB()
	if !ok {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	h := sha256sum(text)
	if h == w.lastHash {
		return
	}
	w.lastHash = h
	w.onNew(text, h)
}

func sha256sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}
