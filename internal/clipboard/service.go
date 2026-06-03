package clipboard

import (
	"context"
	"encoding/binary"
	"log"
	"strings"
	"sync"

	"github.com/lxn/win"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ---------------------------------------------------------------------------
// Tag constants (bitmask)
// ---------------------------------------------------------------------------

// Win32 clipboard format constants — exported so other packages can reference
// format types without importing lxn/win directly.
const (
	CF_UNICODETEXT = 13 // win.CF_UNICODETEXT
	CF_HDROP       = 15 // win.CF_HDROP
	CF_DIB         = 8  // win.CF_DIB
	// CFDIBV5 is defined in clipboard_windows.go (platform-specific).
)

const (
	TagText  = 1 << 0 // plain text only
	TagImage = 1 << 2 // CF_DIB or CF_DIBV5
	TagURL   = 1 << 3 // CF_UNICODETEXT starts with http(s)://
	TagFile  = 1 << 4 // CF_HDROP or windows path pattern
)

// ComputeTagMask determines tags from captured formats.
func ComputeTagMask(formats []CapturedFormat) int {
	mask := 0
	hasImage := false
	hasFile := false
	hasPlainText := false

	for _, f := range formats {
		switch {
		case IsImageFormat(f.FormatType):
			hasImage = true
		case IsHdropFormat(f.FormatType):
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
	if hasFile {
		mask |= TagFile
	}
	// text = plain text without richer formats
	if hasPlainText && !hasImage && !hasFile {
		mask |= TagText
	}

	return mask
}

// IsTextFormat reports whether f is a text clipboard format (CF_UNICODETEXT, CF_TEXT).
func IsTextFormat(f uint32) bool {
	return f == win.CF_UNICODETEXT || f == win.CF_TEXT
}

// IsImageFormat reports whether f is an image clipboard format (CF_DIB, CF_DIBV5).
func IsImageFormat(f uint32) bool {
	return f == win.CF_DIB || f == CFDIBV5
}

// IsHdropFormat reports whether f is CF_HDROP.
func IsHdropFormat(f uint32) bool {
	return f == win.CF_HDROP
}

// isWindowsPath reports whether s looks like a Windows path (C:\... or UNC).
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
	IsFavorite  bool          `json:"is_favorite"`
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

// prependBMPHeader builds a valid BMP from a DIB by prefixing BITMAPFILEHEADER.
func prependBMPHeader(dib []byte) []byte {
	headerSize := binary.LittleEndian.Uint32(dib[0:4])
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	clrUsed := binary.LittleEndian.Uint32(dib[32:36])

	var colorTableSize uint32
	if bitCount <= 8 {
		if clrUsed == 0 {
			colorTableSize = uint32(1<<bitCount) * 4
		} else {
			colorTableSize = clrUsed * 4
		}
	}

	offset := uint32(14 + headerSize + colorTableSize)
	fileSize := uint32(14 + len(dib))

	buf := make([]byte, 14+len(dib))
	binary.LittleEndian.PutUint16(buf[0:2], 0x4D42) // 'BM'
	binary.LittleEndian.PutUint32(buf[2:6], fileSize)
	binary.LittleEndian.PutUint32(buf[10:14], offset)
	copy(buf[14:], dib)
	return buf
}
