package history

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"jpaste/internal/clipboard"

	"github.com/lxn/win"
)

// Service provides clipboard history queries and actions.
type Service struct {
	db         *sql.DB
	clipboard  ClipboardWriter
	imageStore *clipboard.ImageStore

	performPaste func()
	onEmit       func(name string, data any)
	onNotify     func(title, msg string)
	onSyncPush   func(contentHash string, formats []SyncFormat)
}

// ClipboardWriter abstracts clipboard write operations.
type ClipboardWriter interface {
	SetText(text string) bool
}

// SyncFormat is a text format payload for sync push.
type SyncFormat struct {
	FormatType uint32 `json:"t"`
	Content    string `json:"c"`
}

// Option configures the Service.
type Option func(*Service)

func WithPasteFunc(fn func()) Option {
	return func(s *Service) { s.performPaste = fn }
}
func WithEmitFunc(fn func(name string, data any)) Option {
	return func(s *Service) { s.onEmit = fn }
}
func WithNotifyFunc(fn func(title, msg string)) Option {
	return func(s *Service) { s.onNotify = fn }
}
func WithSyncPushFunc(fn func(contentHash string, formats []SyncFormat)) Option {
	return func(s *Service) { s.onSyncPush = fn }
}
func WithImageStore(is *clipboard.ImageStore) Option {
	return func(s *Service) { s.imageStore = is }
}

// NewService creates a Service.
func NewService(db *sql.DB, cw ClipboardWriter, opts ...Option) *Service {
	s := &Service{db: db, clipboard: cw}
	for _, o := range opts {
		o(s)
	}
	return s
}

// nowMillis returns the current time in SQLite-compatible millisecond format.
func nowMillis() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000")
}

// CaptureEntry persists or deduplicates a clipboard entry.
func (s *Service) CaptureEntry(data clipboard.CapturedData) (*clipboard.Entry, bool) {
	now := nowMillis()
	tagMask := clipboard.ComputeTagMask(data.Formats)

	// Try dedup: refresh timestamp if hash exists.
	result, _ := s.db.Exec(
		`UPDATE clipboard_entry SET updated_at = ?, source_exe = ?, source_title = ?, tag_mask = ? WHERE content_hash = ?`,
		now, data.SourceEXE, data.SourceTitle, tagMask, data.PrimaryHash,
	)
	n, _ := result.RowsAffected()
	if n > 0 {
		s.pushToSync(data.PrimaryHash, data.Formats)
		return nil, false
	}

	// Insert new entry.
	res, _ := s.db.Exec(
		`INSERT INTO clipboard_entry (content_hash, source_exe, source_title, tag_mask, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		data.PrimaryHash, data.SourceEXE, data.SourceTitle, tagMask, now, now,
	)
	id, _ := res.LastInsertId()

	today := time.Now().Format("2006-01-02")
	for _, f := range data.Formats {
		s.saveFormat(id, f, today)
	}

	entry := buildEntry(id, data.PrimaryHash, data.SourceEXE, data.SourceTitle, now, now, data.Formats)

	if s.onEmit != nil {
		s.onEmit("clipboard-updated", *entry)
	}
	if s.onNotify != nil {
		s.onNotify("jPaste", previewText(entry.Content))
	}
	s.pushToSync(data.PrimaryHash, data.Formats)

	return entry, true
}

func (s *Service) saveFormat(entryID int64, f clipboard.CapturedFormat, today string) {
	h := sha256Hash(f.Text, f.RawData)

	if f.RawData != nil && (f.FormatType == win.CF_DIB || f.FormatType == clipboard.CFDIBV5) {
		if s.imageStore == nil {
			return
		}
		filePath, err := s.imageStore.Save(f.RawData, today)
		if err != nil {
			return
		}
		s.db.Exec(
			`INSERT OR IGNORE INTO clipboard_format (entry_id, format_type, file_path, format_hash) VALUES (?, ?, ?, ?)`,
			entryID, f.FormatType, filePath, h,
		)
	} else {
		s.db.Exec(
			`INSERT OR IGNORE INTO clipboard_format (entry_id, format_type, content, format_hash) VALUES (?, ?, ?, ?)`,
			entryID, f.FormatType, f.Text, h,
		)
	}
}

func (s *Service) pushToSync(hash string, formats []clipboard.CapturedFormat) {
	if s.onSyncPush == nil {
		return
	}
	var sf []SyncFormat
	for _, f := range formats {
		if f.Text != "" && f.FormatType != win.CF_HDROP {
			sf = append(sf, SyncFormat{FormatType: f.FormatType, Content: f.Text})
		}
	}
	if len(sf) > 0 {
		s.onSyncPush(hash, sf)
	}
}

// GetHistory returns entries with cursor-based pagination and tag filtering.
// tagMask=0 means all, afterUpdatedAt="" means first page.
func (s *Service) GetHistory(search string, tagMask int, afterUpdatedAt string, afterID int64) (entries []clipboard.Entry, hasMore bool, err error) {
	baseSQL := `SELECT e.id, e.content_hash, e.source_exe, e.source_title, e.created_at, e.updated_at FROM clipboard_entry e`
	pageSize := 20 + 1 // one extra to detect hasMore

	var conditions []string
	var args []interface{}

	// Tag filter.
	if tagMask != 0 {
		conditions = append(conditions, `e.tag_mask & ? != 0`)
		args = append(args, tagMask)
	}

	// Search filter.
	if search != "" {
		conditions = append(conditions,
			`e.id IN (SELECT DISTINCT entry_id FROM clipboard_format WHERE content LIKE ?)`)
		args = append(args, "%"+search+"%")
	}

	// Cursor.
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
	args = append(args, pageSize)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	type raw struct {
		id          int64
		contentHash string
		sourceEXE   string
		sourceTitle string
		createdAt   string
		updatedAt   string
	}

	var rawList []raw
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.id, &r.contentHash, &r.sourceEXE, &r.sourceTitle, &r.createdAt, &r.updatedAt); err != nil {
			return nil, false, fmt.Errorf("scan entry: %w", err)
		}
		rawList = append(rawList, r)
	}

	hasMore = len(rawList) == pageSize
	if hasMore {
		rawList = rawList[:len(rawList)-1]
	}

	// Batch-load formats.
	ids := make([]int64, len(rawList))
	for i, r := range rawList {
		ids[i] = r.id
	}
	formatMap := s.loadFormats(ids)

	for _, r := range rawList {
		fmts := formatMap[r.id]
		text := ""
		for _, f := range fmts {
			if f.FormatType == win.CF_UNICODETEXT && f.Content != "" {
				text = f.Content
				break
			}
		}
		entries = append(entries, clipboard.Entry{
			ID:          r.id,
			ContentHash: r.contentHash,
			Content:     text,
			SourceEXE:   r.sourceEXE,
			SourceTitle: r.sourceTitle,
			Formats:     fmts,
			CreatedAt:   r.createdAt,
			UpdatedAt:   r.updatedAt,
		})
	}
	return entries, hasMore, nil
}

func (s *Service) loadFormats(ids []int64) map[int64][]clipboard.FormatEntry {
	if len(ids) == 0 {
		return nil
	}
	idArgs := make([]interface{}, len(ids))
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
		return nil
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
	return m
}

// DeleteEntry removes a single entry by ID.
func (s *Service) DeleteEntry(id int64) error {
	rows, err := s.db.Query(`SELECT file_path FROM clipboard_format WHERE entry_id = ? AND file_path != ''`, id)
	if err == nil {
		var paths []string
		for rows.Next() {
			var p string
			rows.Scan(&p)
			if p != "" {
				paths = append(paths, p)
			}
		}
		rows.Close()
		if s.imageStore != nil {
			s.imageStore.DeleteByEntry(paths)
		}
	}
	_, err = s.db.Exec(`DELETE FROM clipboard_entry WHERE id = ?`, id)
	return err
}

// UseEntry performs the default action and refreshes updated_at.
func (s *Service) UseEntry(id int64, action string) error {
	var text string
	err := s.db.QueryRow(
		`SELECT COALESCE(f.content, '') FROM clipboard_format f WHERE f.entry_id = ? AND f.format_type = ?`,
		id, win.CF_UNICODETEXT,
	).Scan(&text)
	if err != nil || text == "" {
		return fmt.Errorf("no text format for entry %d", id)
	}

	s.db.Exec(`UPDATE clipboard_entry SET updated_at = ? WHERE id = ?`, nowMillis(), id)
	s.clipboard.SetText(text)

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
	err := s.db.QueryRow(
		`SELECT (SELECT COUNT(*) FROM clipboard_entry), COALESCE((SELECT SUM(LENGTH(content)) FROM clipboard_format), 0)`,
	).Scan(&st.Count, &st.TotalBytes)
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	return st, nil
}

// Cleanup removes entries older than retainDays and their image files.
func (s *Service) Cleanup(retainDays int) (int64, error) {
	rows, err := s.db.Query(
		`SELECT f.file_path FROM clipboard_format f
		 JOIN clipboard_entry e ON f.entry_id = e.id
		 WHERE e.updated_at < `+millisSQL(`-`+fmt.Sprintf("%d", retainDays)+` days`)+` AND f.file_path != ''`,
	)
	if err == nil {
		var paths []string
		for rows.Next() {
			var p string
			rows.Scan(&p)
			if p != "" {
				paths = append(paths, p)
			}
		}
		rows.Close()
		if s.imageStore != nil {
			s.imageStore.DeleteByEntry(paths)
			s.imageStore.CleanEmptyDirs()
		}
	}

	result, err := s.db.Exec(
		`DELETE FROM clipboard_entry WHERE updated_at < ` + millisSQL(fmt.Sprintf("-%d days", retainDays)),
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// ClearAll deletes all clipboard entries and their image files.
func (s *Service) ClearAll() error {
	// Delete all image files first.
	if s.imageStore != nil {
		rows, err := s.db.Query(`SELECT file_path FROM clipboard_format WHERE file_path != ''`)
		if err == nil {
			var paths []string
			for rows.Next() {
				var p string
				rows.Scan(&p)
				if p != "" {
					paths = append(paths, p)
				}
			}
			rows.Close()
			s.imageStore.DeleteByEntry(paths)
			s.imageStore.CleanEmptyDirs()
		}
	}
	_, err := s.db.Exec(`DELETE FROM clipboard_entry`)
	return err
}

// millisSQL builds a strftime expression with a modifier for millisecond-precision datetime.
func millisSQL(mod string) string {
	return fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:%%M:%%f', 'now', '%s')", mod)
}

// GetImageDataURL returns a base64 data URL for the first image format of an entry.
func (s *Service) GetImageDataURL(entryID int64) (string, error) {
	log.Printf("[history] GetImageDataURL entry=%d", entryID)
	var filePath string
	err := s.db.QueryRow(
		`SELECT file_path FROM clipboard_format WHERE entry_id = ? AND file_path != '' LIMIT 1`,
		entryID,
	).Scan(&filePath)
	if err != nil {
		log.Printf("[history] GetImageDataURL entry=%d: no image format row (err=%v)", entryID, err)
		return "", fmt.Errorf("no image for entry %d", entryID)
	}
	if filePath == "" {
		log.Printf("[history] GetImageDataURL entry=%d: file_path is empty", entryID)
		return "", fmt.Errorf("no image for entry %d", entryID)
	}
	if s.imageStore == nil {
		log.Printf("[history] GetImageDataURL entry=%d: imageStore is nil", entryID)
		return "", fmt.Errorf("image store not available")
	}
	log.Printf("[history] GetImageDataURL entry=%d: reading file=%s", entryID, filePath)
	data, err := s.imageStore.ReadImage(filePath)
	if err != nil {
		log.Printf("[history] GetImageDataURL entry=%d: ReadImage failed: %v", entryID, err)
		return "", err
	}
	log.Printf("[history] GetImageDataURL entry=%d: read %d bytes", entryID, len(data))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
}

// --- helpers ---

func sha256Hash(text string, raw []byte) string {
	var input []byte
	if len(raw) > 0 {
		input = raw
	} else {
		input = []byte(text)
	}
	h := sha256.Sum256(input)
	return fmt.Sprintf("%x", h[:])
}

func buildEntry(id int64, hash, exe, title, createdAt, updatedAt string, formats []clipboard.CapturedFormat) *clipboard.Entry {
	text := ""
	for _, f := range formats {
		if f.FormatType == win.CF_UNICODETEXT {
			text = f.Text
			break
		}
	}
	e := &clipboard.Entry{
		ID:          id,
		ContentHash: hash,
		Content:     text,
		SourceEXE:   exe,
		SourceTitle: title,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	for _, f := range formats {
		fe := clipboard.FormatEntry{FormatType: f.FormatType}
		if f.RawData != nil {
			fe.Content = fmt.Sprintf("[image %d bytes]", len(f.RawData))
		} else {
			fe.Content = f.Text
		}
		e.Formats = append(e.Formats, fe)
	}
	return e
}

func previewText(s string) string {
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
