package history

import (
	"database/sql"
	"fmt"
	"time"

	"jpaste/internal/clipboard"
)

// Service provides clipboard history queries and actions.
type Service struct {
	db           *sql.DB
	clipboard    ClipboardWriter
	performPaste func()
	// Capture pipeline hooks — injected at construction.
	onEmit     func(name string, data any)
	onNotify   func(title, msg string)
	onSyncPush func(contentHash, content string)
}

// ClipboardWriter abstracts the clipboard write operation.
type ClipboardWriter interface {
	SetText(text string) bool
}

// Option configures the Service.
type Option func(*Service)

// WithPasteFunc sets the platform-specific paste function.
func WithPasteFunc(fn func()) Option {
	return func(s *Service) { s.performPaste = fn }
}

// WithEmitFunc sets the event emitter for capture pipeline events.
func WithEmitFunc(fn func(name string, data any)) Option {
	return func(s *Service) { s.onEmit = fn }
}

// WithNotifyFunc sets the toast notification function.
func WithNotifyFunc(fn func(title, msg string)) Option {
	return func(s *Service) { s.onNotify = fn }
}

// WithSyncPushFunc sets the sync push function.
func WithSyncPushFunc(fn func(contentHash, content string)) Option {
	return func(s *Service) { s.onSyncPush = fn }
}

// NewService creates a Service.
func NewService(db *sql.DB, cw ClipboardWriter, opts ...Option) *Service {
	s := &Service{db: db, clipboard: cw}
	for _, o := range opts {
		o(s)
	}
	return s
}

// CaptureEntry persists or deduplicates a clipboard entry.
// Returns the new entry (if inserted) and whether a new row was created.
func (s *Service) CaptureEntry(text, hash string) (*clipboard.Entry, bool) {
	// Try dedup: refresh timestamp if hash exists.
	result, _ := s.db.Exec(`UPDATE clipboard SET updated_at = datetime('now') WHERE content_hash = ?`, hash)
	n, _ := result.RowsAffected()
	if n > 0 {
		// Existing entry refreshed, still push to sync.
		if s.onSyncPush != nil {
			s.onSyncPush(hash, text)
		}
		return nil, false
	}

	// Insert new entry.
	res, _ := s.db.Exec(
		`INSERT INTO clipboard (content_hash, content, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))`,
		hash, text,
	)
	id, _ := res.LastInsertId()

	now := time.Now().Format("2006-01-02 15:04:05")
	entry := &clipboard.Entry{
		ID:          id,
		ContentHash: hash,
		Content:     text,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Emit event.
	if s.onEmit != nil {
		s.onEmit("clipboard-updated", *entry)
	}

	// Show toast notification.
	if s.onNotify != nil {
		s.onNotify("jPaste", previewText(text))
	}

	// Push to sync.
	if s.onSyncPush != nil {
		s.onSyncPush(hash, text)
	}

	return entry, true
}

// GetHistory returns clipboard entries matching the optional search string,
// ordered by updated_at descending. Limit 200.
func (s *Service) GetHistory(search string) ([]clipboard.Entry, error) {
	var rows *sql.Rows
	var err error

	if search == "" {
		rows, err = s.db.Query(
			`SELECT id, content_hash, content, created_at, updated_at
			 FROM clipboard ORDER BY updated_at DESC LIMIT 200`,
		)
	} else {
		like := "%" + search + "%"
		rows, err = s.db.Query(
			`SELECT id, content_hash, content, created_at, updated_at
			 FROM clipboard WHERE content LIKE ? ORDER BY updated_at DESC LIMIT 200`,
			like,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var entries []clipboard.Entry
	for rows.Next() {
		var e clipboard.Entry
		if err := rows.Scan(&e.ID, &e.ContentHash, &e.Content, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// DeleteEntry removes a single entry by ID.
func (s *Service) DeleteEntry(id int64) error {
	_, err := s.db.Exec(`DELETE FROM clipboard WHERE id = ?`, id)
	return err
}

// UseEntry performs the default action (copy or paste) and refreshes updated_at.
func (s *Service) UseEntry(id int64, action string) error {
	var content string
	err := s.db.QueryRow(`SELECT content FROM clipboard WHERE id = ?`, id).Scan(&content)
	if err != nil {
		return fmt.Errorf("get content: %w", err)
	}

	// Refresh timestamp.
	s.db.Exec(`UPDATE clipboard SET updated_at = datetime('now') WHERE id = ?`, id)

	// Write to clipboard.
	s.clipboard.SetText(content)

	// If paste mode, hide window and simulate Ctrl+V.
	if action == "paste" && s.performPaste != nil {
		s.performPaste()
	}

	return nil
}

// Stats holds aggregate clipboard statistics.
type Stats struct {
	Count      int64 `json:"count"`
	TotalBytes int64 `json:"total_bytes"`
}

// GetStats returns count and total content size.
func (s *Service) GetStats() (Stats, error) {
	var st Stats
	err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(LENGTH(content)), 0) FROM clipboard`).Scan(&st.Count, &st.TotalBytes)
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	return st, nil
}

// Cleanup removes entries older than the specified number of days.
func (s *Service) Cleanup(retainDays int) (int64, error) {
	result, err := s.db.Exec(
		`DELETE FROM clipboard WHERE updated_at < datetime('now', ?)`,
		fmt.Sprintf("-%d days", retainDays),
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func previewText(s string) string {
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
