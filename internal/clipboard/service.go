package clipboard

import (
	"context"
	"log"
	"sync"
	"time"

	"jpaste/internal/model"
	"jpaste/internal/util"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ---------------------------------------------------------------------------
// Self-write tracking
// ---------------------------------------------------------------------------

var (
	selfWriteMu    sync.Mutex
	selfWriteHash  string
	selfWriteTime  time.Time
)

// MarkSelfWrite records that jPaste wrote text to the clipboard, so subsequent
// WM_CLIPBOARDUPDATE callbacks can distinguish our own writes from user copies.
func MarkSelfWrite(text string) {
	selfWriteMu.Lock()
	selfWriteHash = util.SHA256String(text)
	selfWriteTime = time.Now()
	selfWriteMu.Unlock()
	log.Printf("[clipboard] MarkSelfWrite: hash=%s text=%q", selfWriteHash[:12], util.Truncate(text, 40))
}

// IsSelfWrite reports whether the captured data matches the last text
// written by jPaste itself. Used to avoid pushing self-writes onto the stack.
func IsSelfWrite(data model.CapturedData) bool {
	selfWriteMu.Lock()
	defer selfWriteMu.Unlock()
	if selfWriteHash == "" {
		log.Printf("[clipboard] IsSelfWrite: no self-write marker set")
		return false
	}
	age := time.Since(selfWriteTime).Milliseconds()
	if age > 5000 {
		log.Printf("[clipboard] IsSelfWrite: marker expired (%dms ago)", age)
		return false
	}
	for _, f := range data.Formats {
		if model.IsTextFormat(f.FormatType) {
			hx := util.SHA256String(f.Text)
			match := hx == selfWriteHash
			log.Printf("[clipboard] IsSelfWrite: captured=%s last=%s age=%dms match=%v", hx, selfWriteHash, age, match)
			return match
		}
	}
	log.Printf("[clipboard] IsSelfWrite: no text format found in captured data")
	return false
}

// ClearSelfWrite clears the self-write marker.
func ClearSelfWrite() {
	selfWriteMu.Lock()
	selfWriteHash = ""
	selfWriteMu.Unlock()
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
