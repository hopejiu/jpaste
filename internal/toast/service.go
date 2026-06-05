package toast

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"jpaste/internal/events"

	applog "jpaste/internal/log"
)

// ToastData holds the content for a single toast notification.
type ToastData struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// Service manages toast notifications via an event-driven pattern.
// The Go side emits a "toast-notification" event with ToastData payload,
// and the pre-created toast window's frontend listens for it directly.
// Tokens are still kept for backward compatibility of the GetToastData binding.
type Service struct {
	store    map[string]ToastData
	mu       sync.RWMutex
	emitFunc func(name string, data any)
}

// NewService creates a new toast service.
// emitFunc is a callback that emits an app-level event (e.g. app.Event.Emit).
func NewService(emitFunc func(name string, data any)) *Service {
	return &Service{
		store:    make(map[string]ToastData),
		emitFunc: emitFunc,
	}
}

// ShowToast stores the toast data and emits a "toast-notification" event
// to the pre-created toast window. The window's Go-side listener handles
// positioning, showing, and auto-hiding.
func (s *Service) ShowToast(title, message string) {
	token := generateToken()

	s.mu.Lock()
	s.store[token] = ToastData{Title: title, Message: message}
	s.mu.Unlock()

	// Fallback cleanup in case the window close event is missed.
	time.AfterFunc(10*time.Second, func() {
		s.mu.Lock()
		delete(s.store, token)
		s.mu.Unlock()
	})

	applog.Info("toast: show", "title", title, "token", token)
	s.emitFunc(events.ToastNotification, ToastData{Title: title, Message: message})
}

// GetToastData retrieves toast data by token.
// Exposed as a Wails binding for the front-end to call (legacy path).
func (s *Service) GetToastData(token string) *ToastData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.store[token]
	if !ok {
		return nil
	}
	return &data
}

func generateToken() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		b = []byte("fallback00000000")
	}
	return hex.EncodeToString(b)
}
