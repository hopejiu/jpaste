package clipboard

import (
	"context"
	"log"
	"sync"

	"jpaste/internal/model"
	"jpaste/internal/util"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ---------------------------------------------------------------------------
// Self-write tracking
// ---------------------------------------------------------------------------

var selfWriteTracker util.SelfWriteTracker

// MarkSelfWrite records that jPaste wrote text to the clipboard, so subsequent
// WM_CLIPBOARDUPDATE callbacks can distinguish our own writes from user copies.
func MarkSelfWrite(text string) {
	selfWriteTracker.Mark(text)
	log.Printf("[clipboard] MarkSelfWrite: hash=%s text=%q", selfWriteTracker.Hash()[:12], util.Truncate(text, 40))
}

// IsSelfWrite reports whether the captured data matches the last text
// written by jPaste itself. Used to avoid pushing self-writes onto the stack.
func IsSelfWrite(data model.CapturedData) bool {
	for _, f := range data.Formats {
		if model.IsTextFormat(f.FormatType) {
			match := selfWriteTracker.IsSelfWrite(f.Text)
			log.Printf("[clipboard] IsSelfWrite: match=%v age=%dms", match, 0)
			return match
		}
	}
	log.Printf("[clipboard] IsSelfWrite: no text format found in captured data")
	return false
}

// ClearSelfWrite clears the self-write marker.
func ClearSelfWrite() {
	selfWriteTracker.Clear()
}

// OnCapture is called when new clipboard content is detected.
type OnCapture func(model.CapturedData)

// ---------------------------------------------------------------------------
// Watcher — event-driven clipboard monitoring
// ---------------------------------------------------------------------------

// platformStart starts the platform-specific monitoring.
// Returns a stop function that shuts down the monitor cleanly.
var platformStart func(onCapture OnCapture) (stop func(), err error)

// Watcher monitors the system clipboard via AddClipboardFormatListener.
// It is a Wails v3 Service.
type Watcher struct {
	onCapture OnCapture
	stopFn    func()
	mu        sync.Mutex
	started   bool
}

// NewWatcher creates a clipboard Watcher.
func NewWatcher(onCapture OnCapture) *Watcher {
	return &Watcher{onCapture: onCapture}
}

// ServiceStartup begins event-driven clipboard monitoring.
func (w *Watcher) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		log.Println("[clipboard] ServiceStartup called but already started")
		return nil
	}
	log.Println("[clipboard] ServiceStartup called")
	if platformStart == nil {
		log.Println("[clipboard] platformStart is nil (non-Windows stub)")
		return nil
	}
	stop, err := platformStart(w.onCapture)
	if err != nil {
		log.Printf("[clipboard] ERROR starting monitor: %v", err)
		return err
	}
	w.stopFn = stop
	w.started = true
	log.Println("[clipboard] ServiceStartup completed")
	return nil
}

// ServiceShutdown stops monitoring.
func (w *Watcher) ServiceShutdown() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopFn != nil {
		w.stopFn()
		w.stopFn = nil
	}
	w.started = false
	return nil
}
