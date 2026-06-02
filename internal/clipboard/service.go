package clipboard

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/lxn/win"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ---------------------------------------------------------------------------
// Tag constants (bitmask)
// ---------------------------------------------------------------------------

const (
	TagText     = 1 << 0 // plain text only
	TagRichText = 1 << 1 // CF_HTML or CF_RTF
	TagImage    = 1 << 2 // CF_DIB or CF_DIBV5
	TagURL      = 1 << 3 // CF_UNICODETEXT starts with http(s)://
	TagFile     = 1 << 4 // CF_HDROP or windows path pattern
)

// ComputeTagMask determines tags from captured formats.
func ComputeTagMask(formats []CapturedFormat) int {
	mask := 0
	hasImage := false
	hasRichText := false
	hasFile := false
	hasPlainText := false

	for _, f := range formats {
		switch {
		case f.FormatType == win.CF_DIB || f.FormatType == CFDIBV5:
			hasImage = true
		case f.FormatType == cfHTML || f.FormatType == cfRTF:
			hasRichText = true
		case f.FormatType == win.CF_HDROP:
			hasFile = true
		case f.FormatType == win.CF_UNICODETEXT:
			hasPlainText = true
			txt := strings.TrimSpace(f.Text)
			if strings.HasPrefix(txt, "http://") || strings.HasPrefix(txt, "https://") {
				mask |= TagURL
			}
			if isWindowsPath(txt) {
				hasFile = true
			}
		}
	}

	if hasImage {
		mask |= TagImage
	}
	if hasRichText {
		mask |= TagRichText
	}
	if hasFile {
		mask |= TagFile
	}
	// text = plain text without richer formats
	if hasPlainText && !hasImage && !hasRichText && !hasFile {
		mask |= TagText
	}

	return mask
}

func isWindowsPath(s string) bool {
	if len(s) < 3 {
		return false
	}
	// C:\... or c:\...
	if s[1] == ':' && s[2] == '\\' && ((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')) {
		return true
	}
	// UNC path: \\...
	if len(s) >= 2 && s[0] == '\\' && s[1] == '\\' {
		return true
	}
	return false
}



// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// Entry is the JSON-serializable clipboard history item sent to the frontend.
type Entry struct {
	ID          int64         `json:"id"`
	ContentHash string        `json:"content_hash"`
	Content     string        `json:"content"` // CF_UNICODETEXT for backward compat
	SourceEXE   string        `json:"source_exe"`
	SourceTitle string        `json:"source_title"`
	Formats     []FormatEntry `json:"formats"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}

// FormatEntry is one format payload within a clipboard entry.
type FormatEntry struct {
	FormatType uint32 `json:"format_type"`
	Content    string `json:"content"`
	FilePath   string `json:"file_path"`
}

// CapturedFormat is a raw format payload read from the system clipboard.
type CapturedFormat struct {
	FormatType uint32
	Text       string // non-empty for text formats (CF_UNICODETEXT, CF_HTML, etc.)
	RawData    []byte // non-nil for binary formats (CF_DIB / CF_DIBV5)
}

// CapturedData bundles everything captured from one clipboard change.
type CapturedData struct {
	Formats     []CapturedFormat
	SourceEXE   string
	SourceTitle string
	PrimaryHash string // SHA-256 of CF_UNICODETEXT, or image bytes if text is absent
}

// OnCapture is called when new clipboard content is detected.
type OnCapture func(CapturedData)

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
