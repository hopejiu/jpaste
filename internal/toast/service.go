package toast

import (
	"jpaste/internal/model"

	applog "jpaste/internal/log"
)

// ToastData holds the content for a single toast notification.
// Theme is injected by main.go at emit time so the toast window
// (a separate Wails window) can stay in sync after theme changes.
type ToastData struct {
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	Theme     string  `json:"theme,omitempty"`
	Opacity   float64 `json:"opacity,omitempty"`    // 0.0-1.0, used by ToastPage for CSS opacity
	IsPreview bool    `json:"is_preview,omitempty"` // if true, main.go skips auto-hide timer (preview mode)
}

// Service manages toast notifications via an event-driven pattern.
// The Go side emits a "toast-notification" event with ToastData payload,
// and the pre-created toast window's frontend listens for it directly.
type Service struct {
	emitFunc func(name string, data any)
}

// NewService creates a new toast service.
// emitFunc is a callback that emits an app-level event (e.g. app.Event.Emit).
func NewService(emitFunc func(name string, data any)) *Service {
	return &Service{emitFunc: emitFunc}
}

// ShowToast emits a "toast-notification" event to the pre-created toast window.
// The window's Go-side listener handles positioning, showing, and auto-hiding.
func (s *Service) ShowToast(title, message string) {
	applog.Info("toast: show", "title", title, "message", message)
	s.emitFunc(model.ToastNotification, ToastData{Title: title, Message: message})
}
