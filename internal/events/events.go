// Package events defines the canonical event names shared between Go and JS.
//
// These constants replace raw string literals in both codebases. Changing a name
// here will break at compile time in Go and at import time in JS — no silent failures.
package events

// Clipboard lifecycle
const (
	ClipboardUpdated = "clipboard-updated"
)

// Sync
const (
	SyncStatus = "sync-status"
)

// Window visibility
const (
	WindowShown  = "window-shown"
	WindowHiding = "window-hiding"
)

// Navigation
const (
	Navigate = "navigate"
)

// Paste order (stack / queue / normal)
const (
	PasteOrderChanged = "paste-order-changed"
)

// Frontend log relay
const (
	FrontendLog = "frontend-log"
)
