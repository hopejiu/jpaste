package jsonviewer

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	applog "jpaste/internal/log"
)

// entryTTL is how long token→content mappings live before cleanup.
const entryTTL = 60 * time.Second

// entry holds cached JSON content with an expiration time.
type entry struct {
	content string
	expires time.Time
}

// CreateWindowFunc is a callback that opens a new window with the given URL.
type CreateWindowFunc func(url string)

// Service manages JSON viewer windows. It stores JSON content indexed by
// a short random token, so the new window can retrieve it via GetJsonViewerData.
type Service struct {
	mu        sync.Mutex
	entries   map[string]entry
	createWin CreateWindowFunc
}

// NewService creates a new JSON viewer service.
func NewService(createWin CreateWindowFunc) *Service {
	s := &Service{
		entries:   make(map[string]entry),
		createWin: createWin,
	}
	// Background cleanup of expired entries.
	go s.cleanupLoop()
	return s
}

// OpenJsonViewer stores the given JSON content against a random token,
// then opens a new Wails window at /json-view?token=<token>.
func (s *Service) OpenJsonViewer(content string) {
	token := make([]byte, 8)
	rand.Read(token)
	tokenStr := hex.EncodeToString(token)

	applog.Info("jsonviewer: open called", "token", tokenStr, "content_len", len(content))

	s.mu.Lock()
	s.entries[tokenStr] = entry{
		content: content,
		expires: time.Now().Add(entryTTL),
	}
	s.mu.Unlock()

	// Delay window creation so the entry is committed before the
	// new front-end tries to fetch it.
	time.AfterFunc(50*time.Millisecond, func() {
		url := "http://wails.localhost/json-view?token=" + tokenStr
		applog.Info("jsonviewer: creating window", "url", url)
		s.createWin(url)
	})
}

// GetJsonViewerData returns the JSON content for the given token.
// The entry is NOT deleted on first call so that React StrictMode
// double-mounts in dev mode don't lose data. Entries expire after
// entryTTL and are cleaned up by a background goroutine.
func (s *Service) GetJsonViewerData(token string) string {
	applog.Info("jsonviewer: fetch called", "token", token)

	s.mu.Lock()
	e, ok := s.entries[token]
	s.mu.Unlock()

	if !ok || time.Now().After(e.expires) {
		applog.Warn("jsonviewer: token not found or expired", "token", token, "expired", ok)
		return ""
	}

	applog.Info("jsonviewer: fetch OK", "token", token, "bytes", len(e.content))
	return e.content
}

// cleanupLoop periodically removes expired entries.
func (s *Service) cleanupLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		s.cleanup()
	}
}

func (s *Service) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, e := range s.entries {
		if now.After(e.expires) {
			applog.Debug("jsonviewer: cleanup expired", "token", token)
			delete(s.entries, token)
		}
	}
}
