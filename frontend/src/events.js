// Canonical event names shared between Go and JS.
// Keep in sync with internal/events/events.go.

export const EVENTS = {
  // Clipboard lifecycle
  CLIPBOARD_UPDATED: 'clipboard-updated',

  // Sync
  SYNC_STATUS: 'sync-status',

  // Window visibility
  WINDOW_SHOWN:  'window-shown',
  WINDOW_HIDING: 'window-hiding',

// Navigation
  NAVIGATE: 'navigate',

// Paste order (stack / queue / normal)
  PASTE_ORDER_CHANGED: 'paste-order-changed',
}
