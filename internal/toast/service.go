package toast

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	applog "jpaste/internal/log"
)

// ToastData holds the content for a single toast notification.
type ToastData struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// Service manages toast notifications via a token-store pattern.
// The Go side stores {title, message} keyed by a random hex token,
// the window URL is /toast?token=X, and the front-end retrieves
// the payload via GetToastData(token).
type Service struct {
	store     map[string]ToastData
	mu        sync.RWMutex
	createWin func(path string)
}

// NewService creates a new toast service.
// createWin is a callback that opens a Wails window at a given URL path.
func NewService(createWin func(path string)) *Service {
	return &Service{
		store:     make(map[string]ToastData),
		createWin: createWin,
	}
}

// ShowToast stores the toast data and opens a toast window.
// The window auto-closes after 3 seconds; the token is cleaned up
// after a 10-second fallback timer.
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
	s.createWin("/toast?token=" + token)
}

// GetToastData retrieves toast data by token.
// Exposed as a Wails binding for the front-end to call.
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
		// Fallback: not cryptographically secure but acceptable for UI tokens.
		b = []byte("fallback00000000")
	}
	return hex.EncodeToString(b)
}
