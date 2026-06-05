package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"jpaste/internal/events"
	"jpaste/internal/model"
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
	Formats   []model.SyncFormat `json:"formats"`
	UpdatedAt string             `json:"updated_at"`
}

// Service drives the WebDAV bidirectional sync loop.
type Service struct {
	mu            sync.Mutex
	store         SyncStore
	settingReader settings.SettingsReader
	emit          func(name string, data any)
	basePath      string
	cfg           Config
	client        WebDAVClient

	backoffUntil time.Time
	backoffCount int
	pushCh       chan PushInput

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// PushInput is a clipboard entry queued for upload.
type PushInput struct {
	ContentHash string
	Formats     []model.SyncFormat
}

// NewService creates a sync service.
func NewService(basePath string, store SyncStore, sett settings.SettingsReader, emit func(name string, data any)) *Service {
	s := &Service{
		store:         store,
		settingReader: sett,
		emit:          emit,
		basePath:      basePath,
		pushCh:        make(chan PushInput, 128),
		stopCh:        make(chan struct{}),
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

func (s *Service) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error { return nil }
func (s *Service) ServiceShutdown() error {
	close(s.stopCh)
	s.wg.Wait()
	return nil
}

func (s *Service) startWorkers() {
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

	s.wg.Add(1)
	go s.pushWorker()
}

// --- Frontend-accessible methods ---

func (s *Service) GetConfig() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.cfg
	if cfg.Password != "" {
		cfg.Password = "••••••••"
	}
	return cfg
}

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

func (s *Service) TestConnection(c Config) error {
	return newClient(c).TestConnect()
}

// --- Push ---

func (s *Service) PushEntry(input PushInput) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}
	select {
	case s.pushCh <- input:
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
		case input := <-s.pushCh:
			s.doPush(input)
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) doPush(input PushInput) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}

	payload := entryPayload{
		Formats:   input.Formats,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("sync: push marshal %s: %v", input.ContentHash[:12], err)
		return
	}

	s.emitStatus(StatusSyncing, "")

	if err := cl.PutEntry(input.ContentHash, data); err != nil {
		log.Printf("sync: push %s: %v", input.ContentHash[:12], err)
		s.backoff()
		s.mu.Lock()
		s.emitStatusLocked(StatusError, err.Error())
		s.mu.Unlock()
		select {
		case s.pushCh <- input:
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

	remoteEntries, err := cl.ListEntries()
	if err != nil {
		log.Printf("sync: pull list: %v", err)
		s.emitStatus(StatusOK, "")
		return
	}

	pulled := 0
	pushed := 0

	for _, re := range remoteEntries {
		localUpdated, err := s.store.GetLocalUpdatedAt(re.Hash)
		if err != nil {
			log.Printf("sync: query %s: %v", re.Hash[:12], err)
			continue
		}

		if localUpdated == "" {
			// Not local — pull.
			if err := s.pullEntry(cl, re); err != nil {
				log.Printf("sync: pull entry %s: %v", re.Hash[:12], err)
			} else {
				pulled++
			}
		} else {
			localT, _ := parseDBTime(localUpdated)
			if re.LastModified.After(localT) {
				if err := s.pullEntry(cl, re); err != nil {
					log.Printf("sync: pull update %s: %v", re.Hash[:12], err)
				} else {
					pulled++
				}
			}
		}
	}

	// Build remote set for push-diff.
	remoteSet := make(map[string]bool, len(remoteEntries))
	for _, re := range remoteEntries {
		remoteSet[re.Hash] = true
	}

	localEntries, err := s.store.ListLocalEntries(500)
	if err != nil {
		log.Printf("sync: query local entries: %v", err)
	} else {
		for _, le := range localEntries {
			if remoteSet[le.Hash] {
				continue
			}
			sf, fErr := s.store.GetTextFormats(le.Hash)
			if fErr != nil || len(sf) == 0 {
				continue
			}
			payload := entryPayload{Formats: sf, UpdatedAt: toRFC3339(le.UpdatedAt)}
			data, _ := json.Marshal(payload)
			if err := cl.PutEntry(le.Hash, data); err != nil {
				log.Printf("sync: push missing %s: %v", le.Hash[:12], err)
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

func (s *Service) pullEntry(cl WebDAVClient, re RemoteEntry) error {
	data, err := cl.GetEntry(re.Hash)
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

	cfg := s.settingReader.GetSettings()
	t, err := time.Parse(time.RFC3339, ep.UpdatedAt)
	if err == nil {
		cutoff := time.Now().AddDate(0, 0, -cfg.RetainDays)
		if t.Before(cutoff) {
			return nil
		}
	}

	// Convert RFC3339 to millisecond format.
	var dbTime string
	t, parseErr := time.Parse(time.RFC3339, ep.UpdatedAt)
	if parseErr == nil {
		dbTime = t.UTC().Format("2006-01-02T15:04:05.000")
	} else {
		dbTime = ep.UpdatedAt
	}

	entryID, err := s.store.UpsertEntry(re.Hash, dbTime)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}

	for _, sf := range ep.Formats {
		h := sha256hex(sf.Content)
		if err := s.store.InsertFormat(entryID, sf.FormatType, sf.Content, h); err != nil {
			log.Printf("sync: insert format for %s: %v", re.Hash[:12], err)
		}
	}

	s.emit(events.ClipboardUpdated, nil)
	return nil
}

func (s *Service) cleanupCloud() {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}

	cfg := s.settingReader.GetSettings()
	cutoff := time.Now().AddDate(0, 0, -cfg.RetainDays)

	entries, err := cl.ListEntries()
	if err != nil {
		log.Printf("sync: cleanup list: %v", err)
		return
	}

	var deleted int
	for _, re := range entries {
		if re.LastModified.Before(cutoff) {
			if err := cl.DeleteEntry(re.Hash); err != nil {
				log.Printf("sync: cleanup delete %s: %v", re.Hash[:12], err)
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

func (s *Service) PushSettings(data []byte) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return
	}
	if err := cl.PutSettings(data); err != nil {
		log.Printf("sync: push settings: %v", err)
	}
}

func (s *Service) PullSettings() ([]byte, error) {
	s.mu.Lock()
	cl := s.client
	s.mu.Unlock()
	if cl == nil {
		return nil, nil
	}
	return cl.GetSettings()
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
	t, err := time.Parse("2006-01-02T15:04:05.000", s)
	if err != nil {
		return time.Parse("2006-01-02 15:04:05", s)
	}
	return t, nil
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
