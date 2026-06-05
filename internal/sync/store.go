package sync

import (
	"database/sql"
	"fmt"

	"jpaste/internal/model"
)

// SyncStore abstracts the data access needed by sync operations.
// Defined here (consumer) rather than in history (provider) — Go interface hygiene.
type SyncStore interface {
	// GetLocalUpdatedAt returns the updated_at of a local entry, or empty string if not found.
	GetLocalUpdatedAt(hash string) (string, error)

	// UpsertEntry inserts or updates an entry by content_hash. Returns the entry ID.
	UpsertEntry(hash, dbTime string) (int64, error)

	// InsertFormat inserts a single format for an entry (INSERT OR IGNORE).
	InsertFormat(entryID int64, formatType uint32, content, formatHash string) error

	// ListLocalEntries returns up to limit local entries (hash + updated_at), newest first.
	ListLocalEntries(limit int) ([]LocalEntry, error)

	// GetTextFormats returns all non-empty text formats for a given hash.
	GetTextFormats(hash string) ([]model.SyncFormat, error)

	// CleanupExpired deletes entries older than the cutoff and returns affected image paths.
	CleanupExpired(cutoff string) (int64, []string, error)
}

// LocalEntry represents a local clipboard entry for sync comparison.
type LocalEntry struct {
	Hash      string
	UpdatedAt string
}

// sqlSyncStore implements SyncStore backed by a *sql.DB (SQLite).
type sqlSyncStore struct {
	db *sql.DB
}

// NewSQLSyncStore creates a SyncStore backed by a *sql.DB (SQLite).
func NewSQLSyncStore(db *sql.DB) SyncStore {
	return &sqlSyncStore{db: db}
}

func (s *sqlSyncStore) GetLocalUpdatedAt(hash string) (string, error) {
	var updatedAt string
	err := s.db.QueryRow(
		`SELECT updated_at FROM clipboard_entry WHERE content_hash = ?`, hash,
	).Scan(&updatedAt)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return updatedAt, err
}

func (s *sqlSyncStore) UpsertEntry(hash, dbTime string) (int64, error) {
	_, err := s.db.Exec(
		`INSERT INTO clipboard_entry (content_hash, created_at, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(content_hash) DO UPDATE SET
		   updated_at = excluded.updated_at
		   WHERE excluded.updated_at > updated_at`,
		hash, dbTime, dbTime,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert entry: %w", err)
	}
	var entryID int64
	if err := s.db.QueryRow(`SELECT id FROM clipboard_entry WHERE content_hash = ?`, hash).Scan(&entryID); err != nil {
		return 0, fmt.Errorf("get entry id: %w", err)
	}
	return entryID, nil
}

func (s *sqlSyncStore) InsertFormat(entryID int64, formatType uint32, content, formatHash string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO clipboard_format (entry_id, format_type, content, format_hash) VALUES (?, ?, ?, ?)`,
		entryID, formatType, content, formatHash,
	)
	return err
}

func (s *sqlSyncStore) ListLocalEntries(limit int) ([]LocalEntry, error) {
	rows, err := s.db.Query(`SELECT content_hash, updated_at FROM clipboard_entry ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LocalEntry
	for rows.Next() {
		var le LocalEntry
		if err := rows.Scan(&le.Hash, &le.UpdatedAt); err != nil {
			continue
		}
		entries = append(entries, le)
	}
	return entries, nil
}

func (s *sqlSyncStore) GetTextFormats(hash string) ([]model.SyncFormat, error) {
	rows, err := s.db.Query(
		`SELECT format_type, content FROM clipboard_format WHERE entry_id = (SELECT id FROM clipboard_entry WHERE content_hash = ?) AND content IS NOT NULL AND content != ''`,
		hash,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sf []model.SyncFormat
	for rows.Next() {
		var ft uint32
		var c string
		if err := rows.Scan(&ft, &c); err == nil {
			sf = append(sf, model.SyncFormat{FormatType: ft, Content: c})
		}
	}
	return sf, nil
}

func (s *sqlSyncStore) CleanupExpired(cutoff string) (int64, []string, error) {
	rows, err := s.db.Query(
		`SELECT f.file_path FROM clipboard_format f
		 JOIN clipboard_entry e ON f.entry_id = e.id
		 WHERE e.updated_at < `+cutoff+` AND e.is_favorite = 0 AND f.file_path != ''`,
	)
	var paths []string
	if err == nil {
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil && p != "" {
				paths = append(paths, p)
			}
		}
		rows.Close()
	}

	result, err := s.db.Exec(
		`DELETE FROM clipboard_entry WHERE updated_at < `+cutoff+` AND is_favorite = 0`,
	)
	if err != nil {
		return 0, nil, err
	}
	n, _ := result.RowsAffected()
	return n, paths, nil
}
