package history

import (
	"database/sql"
	"fmt"
	"log"

	"jpaste/internal/clipboard"
)

// EntryStore abstracts the persistence layer for clipboard entries.
// The production adapter is sqliteStore; tests use an in-memory fake.
type EntryStore interface {
	// QueryHistory returns a page of entry rows sorted by updated_at DESC, id DESC.
	// Returns pageSize+1 rows to enable hasMore detection.
	QueryHistory(search string, tagMask int, afterUpdatedAt string, afterID int64, limit int) ([]EntryRow, error)

	// LoadFormats returns all formats for the given entry IDs, keyed by entry ID.
	LoadFormats(ids []int64) (map[int64][]clipboard.FormatEntry, error)

	// UpsertDedup tries to dedup an existing entry by hash. Returns true if deduped.
	UpsertDedup(hash, sourceEXE, sourceTitle string, tagMask int, now string) (deduped bool, err error)

	// InsertEntry inserts a new entry, returning the auto-generated ID.
	InsertEntry(hash, sourceEXE, sourceTitle string, tagMask int, now string) (id int64, err error)

	// InsertFormat inserts a format row for an entry.
	InsertFormat(entryID int64, formatType uint32, content, filePath, formatHash string) error

	// QueryFormatContent returns the text content for a specific format of an entry.
	QueryFormatContent(entryID int64, formatType uint32) (string, error)

	// QueryImageFilePath returns the first image file path for an entry.
	QueryImageFilePath(entryID int64) (string, error)

	// UpdateTimestamp refreshes the updated_at of an entry.
	UpdateTimestamp(id int64, now string) error

	// DeleteEntry removes an entry, returning associated image file paths.
	DeleteEntry(id int64) (imagePaths []string, err error)

	// ToggleFavorite sets the is_favorite flag on an entry.
	ToggleFavorite(id int64, value bool) error

	// GetStats returns aggregate clipboard statistics.
	GetStats() (Stats, error)

	// Cleanup removes expired entries, returning count and associated image paths.
	Cleanup(retainDays int) (deleted int64, imagePaths []string, err error)

	// ClearAll removes all entries, returning associated image paths.
	ClearAll() (imagePaths []string, err error)

	// HasFileFormatByHash checks if an entry (by content_hash) has CF_HDROP formats.
	HasFileFormatByHash(hash string) (bool, error)

	// QueryImageEntryIDs returns all entry IDs that have image formats, filtered by tag/search.
	QueryImageEntryIDs(tagMask int, search string) ([]int64, error)
}

// EntryRow is a single row from the clipboard_entry table.
type EntryRow struct {
	ID          int64
	ContentHash string
	SourceEXE   string
	SourceTitle string
	IsFavorite  bool
	CreatedAt   string
	UpdatedAt   string
}

// sqliteStore implements EntryStore backed by SQLite.
type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates an EntryStore backed by the given SQLite connection.
func NewSQLiteStore(db *sql.DB) EntryStore {
	return &sqliteStore{db: db}
}

func (s *sqliteStore) QueryHistory(search string, tagMask int, afterUpdatedAt string, afterID int64, limit int) ([]EntryRow, error) {
	baseSQL := `SELECT e.id, e.content_hash, e.source_exe, e.source_title, e.is_favorite, e.created_at, e.updated_at FROM clipboard_entry e`

	var conditions []string
	var args []any

	if tagMask&32 != 0 {
		conditions = append(conditions, `e.is_favorite = 1`)
		tagMask &^= 32
	}
	if tagMask != 0 {
		conditions = append(conditions, `e.tag_mask & ? != 0`)
		args = append(args, tagMask)
	}
	if search != "" {
		conditions = append(conditions, `e.id IN (SELECT DISTINCT entry_id FROM clipboard_format WHERE content LIKE ?)`)
		args = append(args, "%"+search+"%")
	}
	if afterUpdatedAt != "" {
		conditions = append(conditions, `(e.updated_at < ? OR (e.updated_at = ? AND e.id < ?))`)
		args = append(args, afterUpdatedAt, afterUpdatedAt, afterID)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	query := baseSQL + where + ` ORDER BY e.updated_at DESC, e.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var result []EntryRow
	for rows.Next() {
		var r EntryRow
		if err := rows.Scan(&r.ID, &r.ContentHash, &r.SourceEXE, &r.SourceTitle, &r.IsFavorite, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		result = append(result, r)
	}
	return result, nil
}

func (s *sqliteStore) LoadFormats(ids []int64) (map[int64][]clipboard.FormatEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idArgs := make([]any, len(ids))
	placeholders := make([]byte, 0, len(ids)*3)
	placeholders = append(placeholders, '?')
	idArgs[0] = ids[0]
	for i := 1; i < len(ids); i++ {
		placeholders = append(placeholders, ',', '?')
		idArgs[i] = ids[i]
	}

	rows, err := s.db.Query(
		`SELECT entry_id, format_type, COALESCE(content, ''), COALESCE(file_path, '') FROM clipboard_format WHERE entry_id IN (`+string(placeholders)+`)`,
		idArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int64][]clipboard.FormatEntry)
	for rows.Next() {
		var eid int64
		var ft clipboard.FormatEntry
		if err := rows.Scan(&eid, &ft.FormatType, &ft.Content, &ft.FilePath); err != nil {
			continue
		}
		m[eid] = append(m[eid], ft)
	}
	return m, nil
}

func (s *sqliteStore) UpsertDedup(hash, sourceEXE, sourceTitle string, tagMask int, now string) (bool, error) {
	result, err := s.db.Exec(
		`UPDATE clipboard_entry SET updated_at = ?, source_exe = ?, source_title = ?, tag_mask = ? WHERE content_hash = ?`,
		now, sourceEXE, sourceTitle, tagMask, hash,
	)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *sqliteStore) InsertEntry(hash, sourceEXE, sourceTitle string, tagMask int, now string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO clipboard_entry (content_hash, source_exe, source_title, tag_mask, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		hash, sourceEXE, sourceTitle, tagMask, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *sqliteStore) InsertFormat(entryID int64, formatType uint32, content, filePath, formatHash string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO clipboard_format (entry_id, format_type, content, file_path, format_hash) VALUES (?, ?, ?, ?, ?)`,
		entryID, formatType, content, filePath, formatHash,
	)
	return err
}

func (s *sqliteStore) QueryFormatContent(entryID int64, formatType uint32) (string, error) {
	var content string
	err := s.db.QueryRow(
		`SELECT COALESCE(f.content, '') FROM clipboard_format f WHERE f.entry_id = ? AND f.format_type = ?`,
		entryID, formatType,
	).Scan(&content)
	return content, err
}

func (s *sqliteStore) QueryImageFilePath(entryID int64) (string, error) {
	var filePath string
	err := s.db.QueryRow(
		`SELECT COALESCE(f.file_path, '') FROM clipboard_format f WHERE f.entry_id = ? AND f.file_path != '' LIMIT 1`,
		entryID,
	).Scan(&filePath)
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", fmt.Errorf("no image for entry %d", entryID)
	}
	return filePath, nil
}

func (s *sqliteStore) UpdateTimestamp(id int64, now string) error {
	_, err := s.db.Exec(`UPDATE clipboard_entry SET updated_at = ? WHERE id = ?`, now, id)
	return err
}

func (s *sqliteStore) DeleteEntry(id int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT file_path FROM clipboard_format WHERE entry_id = ? AND file_path != ''`, id)
	if err == nil {
		defer rows.Close()
		var paths []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil && p != "" {
				paths = append(paths, p)
			}
		}
	}
	_, err = s.db.Exec(`DELETE FROM clipboard_entry WHERE id = ?`, id)
	return nil, err
}

func (s *sqliteStore) ToggleFavorite(id int64, value bool) error {
	v := 0
	if value {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE clipboard_entry SET is_favorite = ? WHERE id = ?`, v, id)
	return err
}

func (s *sqliteStore) GetStats() (Stats, error) {
	var st Stats
	err := s.db.QueryRow(
		`SELECT (SELECT COUNT(*) FROM clipboard_entry), COALESCE((SELECT SUM(LENGTH(content)) FROM clipboard_format), 0)`,
	).Scan(&st.Count, &st.TotalBytes)
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	return st, nil
}

func (s *sqliteStore) Cleanup(retainDays int) (int64, []string, error) {
	rows, err := s.db.Query(
		`SELECT f.file_path FROM clipboard_format f
		 JOIN clipboard_entry e ON f.entry_id = e.id
		 WHERE e.updated_at < `+
			fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:%%M:%%f', 'now', '-%d days')", retainDays)+
			` AND f.file_path != ''`,
	)
	if err == nil {
		defer rows.Close()
		var paths []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil && p != "" {
				paths = append(paths, p)
			}
		}
	}

	result, err := s.db.Exec(
		`DELETE FROM clipboard_entry WHERE updated_at < `+
			fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:%%M:%%f', 'now', '-%d days')", retainDays),
	)
	if err != nil {
		return 0, nil, err
	}
	n, _ := result.RowsAffected()
	return n, rowsPathOrNil(rows), nil
}

func (s *sqliteStore) ClearAll() ([]string, error) {
	rows, err := s.db.Query(`SELECT file_path FROM clipboard_format WHERE file_path != ''`)
	if err == nil {
		defer rows.Close()
		var paths []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil && p != "" {
				paths = append(paths, p)
			}
		}
	}
	_, err = s.db.Exec(`DELETE FROM clipboard_entry`)
	return nil, err
}

func (s *sqliteStore) HasFileFormatByHash(hash string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM clipboard_format WHERE entry_id = (SELECT id FROM clipboard_entry WHERE content_hash = ?) AND format_type = 15`,
		hash,
	).Scan(&count)
	return count > 0, err
}

func (s *sqliteStore) QueryImageEntryIDs(tagMask int, search string) ([]int64, error) {
	baseSQL := `SELECT e.id FROM clipboard_entry e
		JOIN clipboard_format f ON f.entry_id = e.id AND f.format_type IN (8, 17)
		WHERE 1=1`
	var args []any

	if tagMask&32 != 0 {
		baseSQL += ` AND e.is_favorite = 1`
		tagMask &^= 32
	}
	if tagMask != 0 {
		baseSQL += ` AND e.tag_mask & ? != 0`
		args = append(args, tagMask)
	}
	if search != "" {
		baseSQL += ` AND e.id IN (SELECT DISTINCT entry_id FROM clipboard_format WHERE content LIKE ?)`
		args = append(args, "%"+search+"%")
	}

	baseSQL += ` GROUP BY e.id ORDER BY e.updated_at DESC`
	rows, err := s.db.Query(baseSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// rowsPathOrNil extracts file paths from a *sql.Rows if available.
func rowsPathOrNil(rows *sql.Rows) []string {
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil && p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// logErr is a helper to silently handle expected errors.
func logErr(err error) {
	if err != nil {
		log.Printf("[store] error: %v", err)
	}
}
