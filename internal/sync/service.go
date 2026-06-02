package sync

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"jpaste/internal/events"
	"jpaste/internal/settings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Status represents the current sync state visible in the UI.
type Status string

const (
	StatusNone    Status = "none"
	StatusSyncing Status = "syncing"
	StatusOK      Status = "ok"
	StatusError   Status = "error"
)

// StatusEvent is emitted to the frontend on every status change.
type StatusEvent struct {
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// entryPayload is the JSON format of a single entry file on WebDAV.
type entryPayload struct {
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

// Service drives the WebDAV bidirectional sync loop.
type Service struct {
	mu         sync.Mutex
	db         *sql.DB
	settingSvc *settings.Service
	emit       func(name string, data any)
	basePath   string
	cfg        Config
	client     *client

	// Push backoff state.
	backoffUntil time.Time
	backoffCount int
	pushCh       chan entryPayload

	// Lifecycle.
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewService creates a sync service. Workers start immediately.
func NewService(basePath string, db *sql.DB, sett *settings.Service, emit func(name string, data any)) *Service {
	s := &Service{
		db:         db,
		settingSvc: sett,
		emit:       emit,
		basePath:   basePath,
		pushCh:     make(chan entryPayload, 128),
		stopCh:     make(chan struct{}),
	}
	cfg, err := loadConfig(basePath)
	if err != nil {
		log.Printf("sync: load config: %v", err)
	}
	s.cfg = cfg
	if cfg.IsValid() {
		s.client = newClient(cfg)
	}
	s.startWorkers()
	return s
}

// ServiceStartup is the Wails v3 lifecycle hook (canonical interface: ServiceStartup).
func (s *Service) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error { return nil }

// ServiceShutdown is the Wails v3 lifecycle hook (canonical interface: ServiceShutdown).
func (s *Service) ServiceShutdown() error {
	close(s.stopCh)
	s.wg.Wait()
	return nil
}

func (s *Service) startWorkers() {
	// Initial full sync (if configured).
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.mu.Lock()
		if s.client == nil {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		s.fullPull()
	}()

	// Periodic pull every 60s.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				if s.client == nil {
					s.mu.Unlock()
					continue
				}
				s.mu.Unlock()
				s.fullPull()
			case <-s.stopCh:
				return
			}
		}
	}()

	// Push worker.
	s.wg.Add(1)
	go s.pushWorker()
}

// --- Frontend-accessible methods ---

// GetConfig returns the current WebDAV config (password masked).
func (s *Service) GetConfig() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.cfg
	if cfg.Password != "" {
		cfg.Password = "••••••••"
	}
	return cfg
}

// SaveConfig persists the WebDAV config and reconnects.
func (s *Service) SaveConfig(c Config) error {
	s.mu.Lock()
	existing := s.cfg
	s.mu.Unlock()

	if c.Password == "••••••••" && existing.Password != "" {
		c.Password = existing.Password
	}

	if err := saveConfig(s.basePath, c); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	s.mu.Lock()
	wasValid := s.cfg.IsValid()
	s.cfg = c
	if c.IsValid() {
		s.client = newClient(c)
		s.emitStatusLocked(StatusOK, "")
	} else {
		s.client = nil
		s.emitStatusLocked(StatusNone, "")
	}
	s.mu.Unlock()

	if c.IsValid() && !wasValid {
		go s.fullPull()
	}
	return nil
}

// TestConnection verifies the WebDAV server is reachable.
func (s *Service) TestConnection(c Config) error {
	return newClient(c).testConnect()
}

// --- Push ---

type PushInput struct {
	ContentHash string `json:"content_hash"`
	Content     string `json:"content"`
}

// PushEntry enqueues a clipboard entry for upload.
func (s *Service) PushEntry(input PushInput) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}

	payload := entryPayload{
		Content:   input.Content,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	select {
	case s.pushCh <- payload:
	default:
	}
}

func (s *Service) pushWorker() {
	defer s.wg.Done()
	for {
		s.mu.Lock()
		until := s.backoffUntil
		s.mu.Unlock()

		if time.Now().Before(until) {
			wait := time.Until(until)
			select {
			case <-time.After(wait):
				continue
			case <-s.stopCh:
				return
			}
		}

		select {
		case entry := <-s.pushCh:
			s.doPush(entry)
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) doPush(entry entryPayload) {
	hash := sha256hex(entry.Content)

	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("sync: push marshal %s: %v", hash[:12], err)
		return
	}

	s.emitStatus(StatusSyncing, "")

	if err := cl.putEntry(hash, data); err != nil {
		log.Printf("sync: push %s: %v", hash[:12], err)
		s.backoff()
		s.mu.Lock()
		s.emitStatusLocked(StatusError, err.Error())
		s.mu.Unlock()
		select {
		case s.pushCh <- entry:
		default:
		}
		return
	}

	s.mu.Lock()
	s.backoffCount = 0
	s.backoffUntil = time.Time{}
	s.mu.Unlock()

	go s.cleanupCloud()
	s.emitStatus(StatusOK, "")
}

func (s *Service) backoff() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backoffCount++
	if s.backoffCount > 6 {
		s.backoffCount = 6
	}
	d := time.Duration(1<<(s.backoffCount-1)) * time.Minute
	s.backoffUntil = time.Now().Add(d)
}

// --- Pull ---

func (s *Service) fullPull() {
	s.mu.Lock()
	if s.client == nil {
		s.mu.Unlock()
		return
	}
	cl := s.client
	s.mu.Unlock()

	s.emitStatus(StatusSyncing, "")

	remoteEntries, err := cl.listEntries()
	if err != nil {
		log.Printf("sync: pull list: %v", err)
		s.emitStatus(StatusOK, "")
		return
	}

	pulled := 0
	pushed := 0

	for _, re := range remoteEntries {
		var localUpdated string
		err := s.db.QueryRow(
			`SELECT updated_at FROM clipboard WHERE content_hash = ?`, re.hash,
		).Scan(&localUpdated)

		if err == sql.ErrNoRows {
			if err := s.pullEntry(cl, re); err != nil {
				log.Printf("sync: pull entry %s: %v", re.hash[:12], err)
			} else {
				pulled++
			}
		} else if err == nil {
			localT, _ := parseDBTime(localUpdated)
			if re.lastModified.After(localT) {
				if err := s.pullEntry(cl, re); err != nil {
					log.Printf("sync: pull update %s: %v", re.hash[:12], err)
				} else {
					pulled++
				}
			}
		}
	}

	remoteSet := make(map[string]bool, len(remoteEntries))
	for _, re := range remoteEntries {
		remoteSet[re.hash] = true
	}

	rows, err := s.db.Query(`SELECT content_hash, content, updated_at FROM clipboard ORDER BY updated_at DESC LIMIT 500`)
	if err != nil {
		log.Printf("sync: query local entries: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var hash, content, updatedAt string
			if err := rows.Scan(&hash, &content, &updatedAt); err != nil {
				continue
			}
			if remoteSet[hash] {
				continue
			}
			payload := entryPayload{Content: content, UpdatedAt: toRFC3339(updatedAt)}
			data, _ := json.Marshal(payload)
			if err := cl.putEntry(hash, data); err != nil {
				log.Printf("sync: push missing %s: %v", hash[:12], err)
			} else {
				pushed++
			}
		}
	}

	if pulled > 0 || pushed > 0 {
		log.Printf("sync: pull=%d push=%d entries", pulled, pushed)
	}
	s.emitStatus(StatusOK, "")
}

func (s *Service) pullEntry(cl *client, re remoteEntry) error {
	data, err := cl.getEntry(re.hash)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	var ep entryPayload
	if err := json.Unmarshal(data, &ep); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	cfg := s.settingSvc.GetSettings()
	t, err := time.Parse(time.RFC3339, ep.UpdatedAt)
	if err == nil {
		cutoff := time.Now().AddDate(0, 0, -cfg.RetainDays)
		if t.Before(cutoff) {
			return nil
		}
	}

	_, err = s.db.Exec(
		`INSERT INTO clipboard (content_hash, content, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(content_hash) DO UPDATE SET
		   content = excluded.content,
		   updated_at = excluded.updated_at
		   WHERE excluded.updated_at > updated_at`,
		re.hash, ep.Content, ep.UpdatedAt, ep.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}

	s.emit(events.ClipboardUpdated, nil)
	return nil
}

// cleanupCloud deletes entry files on WebDAV older than retain_days.
func (s *Service) cleanupCloud() {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}

	cfg := s.settingSvc.GetSettings()
	cutoff := time.Now().AddDate(0, 0, -cfg.RetainDays)

	entries, err := cl.listEntries()
	if err != nil {
		log.Printf("sync: cleanup list: %v", err)
		return
	}

	var deleted int
	for _, re := range entries {
		if re.lastModified.Before(cutoff) {
			if err := cl.deleteEntry(re.hash); err != nil {
				log.Printf("sync: cleanup delete %s: %v", re.hash[:12], err)
			} else {
				deleted++
			}
		}
	}
	if deleted > 0 {
		log.Printf("sync: cleaned up %d expired entries (retain=%d days)", deleted, cfg.RetainDays)
	}
}

// --- Settings sync ---

// PushSettings uploads settings to WebDAV.
func (s *Service) PushSettings(data []byte) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}
	if err := cl.putSettings(data); err != nil {
		log.Printf("sync: push settings: %v", err)
	}
}

// PullSettings downloads settings from WebDAV.
func (s *Service) PullSettings() ([]byte, error) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return nil, nil
	}
	return cl.getSettings()
}

// --- Helpers ---

func (s *Service) emitStatus(st Status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitStatusLocked(st, errMsg)
}

func (s *Service) emitStatusLocked(st Status, errMsg string) {
	s.emit(events.SyncStatus, StatusEvent{Status: st, Error: errMsg})
}

func parseDBTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s)
}

func toRFC3339(s string) string {
	t, err := parseDBTime(s)
	if err != nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return t.UTC().Format(time.RFC3339)
}

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
