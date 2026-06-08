package model

import "strings"

// ---------------------------------------------------------------------------
// Win32 clipboard format constants
// ---------------------------------------------------------------------------
const (
	CF_UNICODETEXT = 13
	CF_HDROP       = 15
	CF_DIB         = 8
	CF_DIBV5       = 17
	CF_TEXT        = 1
)

// ---------------------------------------------------------------------------
// Event name constants (moved from internal/events)
// ---------------------------------------------------------------------------
const (
	ClipboardUpdated  = "clipboard-updated"
	WindowShown       = "window-shown"
	WindowHiding      = "window-hiding"
	Navigate          = "navigate"
	PasteOrderChanged = "paste-order-changed"
	FrontendLog       = "frontend-log"
	ToastNotification = "toast-notification"
)

// ---------------------------------------------------------------------------
// Tag constants (bitmask)
// ---------------------------------------------------------------------------
const (
	TagText     = 1 << 0 // plain text only
	TagImage    = 1 << 2 // CF_DIB or CF_DIBV5
	TagURL      = 1 << 3 // CF_UNICODETEXT starts with http(s)://
	TagFile     = 1 << 4 // CF_HDROP or windows path pattern
	TagFavorite = 1 << 5 // virtual: used only for frontend filtering
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// Entry is the JSON-serializable clipboard history item sent to the frontend.
type Entry struct {
	ID            int64         `json:"id"`
	ContentHash   string        `json:"content_hash"`
	Content       string        `json:"content"`
	SourceEXE     string        `json:"source_exe"`
	SourceTitle   string        `json:"source_title"`
	Formats       []FormatEntry `json:"formats"`
	IsFavorite    bool          `json:"is_favorite"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	ContentLength int           `json:"content_length"`
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
	Text       string // non-empty for text formats
	RawData    []byte // non-nil for binary formats (CF_DIB / CF_DIBV5)
}

// CapturedData bundles everything captured from one clipboard change.
type CapturedData struct {
	Formats     []CapturedFormat
	SourceEXE   string
	SourceTitle string
	PrimaryHash string
}

// ---------------------------------------------------------------------------
// Tag / format helper functions
// ---------------------------------------------------------------------------

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
		case f.FormatType == CF_UNICODETEXT:
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
	if hasPlainText && !hasImage && !hasFile {
		mask |= TagText
	}
	return mask
}

// IsTextFormat reports whether f is a text clipboard format.
func IsTextFormat(f uint32) bool {
	return f == CF_UNICODETEXT || f == CF_TEXT
}

// IsImageFormat reports whether f is an image clipboard format.
func IsImageFormat(f uint32) bool {
	return f == CF_DIB || f == CF_DIBV5
}

// IsHdropFormat reports whether f is CF_HDROP.
func IsHdropFormat(f uint32) bool {
	return f == CF_HDROP
}

func isWindowsPath(s string) bool {
	if len(s) < 3 {
		return false
	}
	if s[1] == ':' && s[2] == '\\' && ((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')) {
		return true
	}
	if len(s) >= 2 && s[0] == '\\' && s[1] == '\\' {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Primary text extraction
// ---------------------------------------------------------------------------

func PrimaryText(formats []CapturedFormat) string {
	for _, f := range formats {
		if f.FormatType == CF_UNICODETEXT {
			return f.Text
		}
	}
	for _, f := range formats {
		if f.FormatType == CF_HDROP && f.Text != "" {
			return f.Text
		}
	}
	for _, f := range formats {
		if f.Text != "" {
			return f.Text
		}
	}
	return ""
}

func PrimaryTextFromEntries(formats []FormatEntry) string {
	for _, f := range formats {
		if f.FormatType == CF_UNICODETEXT {
			return f.Content
		}
	}
	for _, f := range formats {
		if f.FormatType == CF_HDROP && f.Content != "" {
			return f.Content
		}
	}
	for _, f := range formats {
		if f.Content != "" && !strings.HasPrefix(f.Content, "[image ") {
			return f.Content
		}
	}
	return ""
}

func TextLength(formats []CapturedFormat) int {
	text := PrimaryText(formats)
	return len(text)
}
