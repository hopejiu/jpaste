package db

import (
	"database/sql"
	"fmt"

	"jpaste/internal/model"
)

// Repository is the single source of truth for all SQLite data access.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a Repository backed by the given SQLite connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Entry queries
// ---------------------------------------------------------------------------

// EntryRow is a single row from the clipboard_entry table.
type EntryRow struct {
	ID            int64
	ContentHash   string
	SourceEXE     string
	SourceTitle   string
	IsFavorite    bool
	CreatedAt     string
	UpdatedAt     string
	ContentLength int
}

func (r *Repository) QueryHistory(search string, tagMask int, afterCursor1 string, afterID int64, limit int, sortField, sortOrder string) ([]EntryRow, error) {
	baseSQL := `SELECT e.id, e.content_hash, e.source_exe, e.source_title, e.is_favorite, e.created_at, e.updated_at, e.content_length FROM clipboard_entry e`
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
	if afterCursor1 != "" {
		if sortOrder == "DESC" {
			conditions = append(conditions, fmt.Sprintf("(e.%s < ? OR (e.%s = ? AND e.id < ?))", sortField, sortField))
		} else {
			conditions = append(conditions, fmt.Sprintf("(e.%s > ? OR (e.%s = ? AND e.id > ?))", sortField, sortField))
		}
		args = append(args, afterCursor1, afterCursor1, afterID)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			where += " AND " + conditions[i]
		}
	}

	query := baseSQL + where + fmt.Sprintf(" ORDER BY e.%s %s, e.id %s LIMIT ?", sortField, sortOrder, sortOrder)
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var result []EntryRow
	for rows.Next() {
		var row EntryRow
		if err := rows.Scan(&row.ID, &row.ContentHash, &row.SourceEXE, &row.SourceTitle, &row.IsFavorite, &row.CreatedAt, &row.UpdatedAt, &row.ContentLength); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		result = append(result, row)
	}
	return result, nil
}

func (r *Repository) LoadFormats(ids []int64) (map[int64][]model.FormatEntry, error) {
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
	rows, err := r.db.Query(
		`SELECT entry_id, format_type, COALESCE(content, ''), COALESCE(file_path, '') FROM clipboard_format WHERE entry_id IN (`+string(placeholders)+`)`,
		idArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int64][]model.FormatEntry)
	for rows.Next() {
		var eid int64
		var ft model.FormatEntry
		if err := rows.Scan(&eid, &ft.FormatType, &ft.Content, &ft.FilePath); err != nil {
			continue
		}
		m[eid] = append(m[eid], ft)
	}
	return m, nil
}

func (r *Repository) UpsertDedup(hash, sourceEXE, sourceTitle string, tagMask int, now string, contentLength int) (bool, error) {
	result, err := r.db.Exec(
		`UPDATE clipboard_entry SET updated_at = ?, source_exe = ?, source_title = ?, tag_mask = ?, content_length = ? WHERE content_hash = ?`,
		now, sourceEXE, sourceTitle, tagMask, contentLength, hash,
	)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (r *Repository) InsertEntry(hash, sourceEXE, sourceTitle string, tagMask int, now string, contentLength int) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO clipboard_entry (content_hash, source_exe, source_title, tag_mask, content_length, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		hash, sourceEXE, sourceTitle, tagMask, contentLength, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) InsertFormat(entryID int64, formatType uint32, content, filePath, formatHash string) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO clipboard_format (entry_id, format_type, content, file_path, format_hash) VALUES (?, ?, ?, ?, ?)`,
		entryID, formatType, content, filePath, formatHash,
	)
	return err
}

func (r *Repository) QueryFormatContent(entryID int64, formatType uint32) (string, error) {
	var content string
	err := r.db.QueryRow(
		`SELECT COALESCE(f.content, '') FROM clipboard_format f WHERE f.entry_id = ? AND f.format_type = ?`,
		entryID, formatType,
	).Scan(&content)
	return content, err
}

func (r *Repository) QueryImageFilePath(entryID int64) (string, error) {
	var filePath string
	err := r.db.QueryRow(
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

func (r *Repository) UpdateTimestamp(id int64, now string) error {
	_, err := r.db.Exec(`UPDATE clipboard_entry SET updated_at = ? WHERE id = ?`, now, id)
	return err
}

func (r *Repository) DeleteEntry(id int64) ([]string, error) {
	rows, err := r.db.Query(`SELECT file_path FROM clipboard_format WHERE entry_id = ? AND file_path != ''`, id)
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
	_, err = r.db.Exec(`DELETE FROM clipboard_entry WHERE id = ?`, id)
	return nil, err
}

func (r *Repository) ToggleFavorite(id int64, value bool) error {
	v := 0
	if value {
		v = 1
	}
	_, err := r.db.Exec(`UPDATE clipboard_entry SET is_favorite = ? WHERE id = ?`, v, id)
	return err
}

type Stats struct {
	Count      int64 `json:"count"`
	TotalBytes int64 `json:"total_bytes"`
}

func (r *Repository) GetStats() (Stats, error) {
	var st Stats
	err := r.db.QueryRow(
		`SELECT (SELECT COUNT(*) FROM clipboard_entry), COALESCE((SELECT SUM(LENGTH(content)) FROM clipboard_format), 0)`,
	).Scan(&st.Count, &st.TotalBytes)
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	return st, nil
}

func (r *Repository) Cleanup(retainDays int) (int64, []string, error) {
	cutoff := fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:%%M:%%f', 'now', '-%d days')", retainDays)
	var paths []string
	rows, err := r.db.Query(
		`SELECT f.file_path FROM clipboard_format f
		 JOIN clipboard_entry e ON f.entry_id = e.id
		 WHERE e.updated_at < `+cutoff+` AND e.is_favorite = 0 AND f.file_path != ''`,
	)
	if err == nil {
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil && p != "" {
				paths = append(paths, p)
			}
		}
		rows.Close()
	}
	result, err := r.db.Exec(
		`DELETE FROM clipboard_entry WHERE updated_at < `+cutoff+` AND is_favorite = 0`,
	)
	if err != nil {
		return 0, nil, err
	}
	n, _ := result.RowsAffected()
	return n, paths, nil
}

func (r *Repository) ClearAll(keepFavorites bool) ([]string, error) {
	var paths []string
	{
		var query string
		if keepFavorites {
			query = `SELECT f.file_path FROM clipboard_format f
				JOIN clipboard_entry e ON f.entry_id = e.id
				WHERE e.is_favorite = 0 AND f.file_path != ''`
		} else {
			query = `SELECT file_path FROM clipboard_format WHERE file_path != ''`
		}
		rows, err := r.db.Query(query)
		if err == nil {
			for rows.Next() {
				var p string
				if err := rows.Scan(&p); err == nil && p != "" {
					paths = append(paths, p)
				}
			}
			rows.Close()
		}
	}
	if keepFavorites {
		_, err := r.db.Exec(`DELETE FROM clipboard_entry WHERE is_favorite = 0`)
		return paths, err
	}
	_, err := r.db.Exec(`DELETE FROM clipboard_entry`)
	return paths, err
}

func (r *Repository) HasFileFormatByHash(hash string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM clipboard_format WHERE entry_id = (SELECT id FROM clipboard_entry WHERE content_hash = ?) AND format_type = 15`,
		hash,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) QueryImageEntryIDs(tagMask int, search string) ([]int64, error) {
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
	rows, err := r.db.Query(baseSQL, args...)
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
