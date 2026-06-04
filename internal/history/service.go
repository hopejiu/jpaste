package history

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"jpaste/internal/clipboard"
)

// Service provides clipboard history queries and actions.
type Service struct {
	store      EntryStore
	clipboard  ClipboardWriter
	imageStore ImageStorer

	performPaste func()
	onEmit       func(name string, data any)
	onNotify     func(title, msg string)
	onSyncPush   func(contentHash string, formats []SyncFormat)
}

// ClipboardWriter abstracts clipboard write operations.
type ClipboardWriter interface {
	SetText(text string) bool
	SetImage(dib []byte) bool
	SetFiles(paths []string) bool
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
func WithImageStore(is ImageStorer) Option {
	return func(s *Service) { s.imageStore = is }
}

// NewService creates a Service.
func NewService(store EntryStore, cw ClipboardWriter, opts ...Option) *Service {
	s := &Service{store: store, clipboard: cw}
	for _, o := range opts {
		o(s)
	}
	return s
}

// nowMillis returns the current local time in SQLite-compatible millisecond format.
func nowMillis() string {
	return time.Now().Format("2006-01-02T15:04:05.000")
}

// CaptureEntry persists or deduplicates a clipboard entry.
func (s *Service) CaptureEntry(data clipboard.CapturedData) (*clipboard.Entry, bool) {
	now := nowMillis()
	tagMask := clipboard.ComputeTagMask(data.Formats)

	// File copies (CF_HDROP) are exempt from dedup — they carry richer format
	// data that should always create a fresh entry even if the text content
	// matches a previous plain-text copy.
	hasFileFormat := false
	for _, f := range data.Formats {
		if clipboard.IsHdropFormat(f.FormatType) {
			hasFileFormat = true
			break
		}
	}

	if !hasFileFormat {
		// Don't dedup plain-text copies of previously-copied files.
		// Existing file entries carry CF_HDROP data that text-only copies lack.
		existingHasFile, _ := s.store.HasFileFormatByHash(data.PrimaryHash)
		if !existingHasFile {
			deduped, err := s.store.UpsertDedup(data.PrimaryHash, data.SourceEXE, data.SourceTitle, tagMask, now)
			if err != nil {
				log.Printf("[history] dedup err: %v", err)
			}
			if deduped {
				s.pushToSync(data.PrimaryHash, data.Formats)
				return nil, false
			}
		}
	}

	// Insert new entry.
	id, err := s.store.InsertEntry(data.PrimaryHash, data.SourceEXE, data.SourceTitle, tagMask, now)
	if err != nil {
		log.Printf("[history] insert entry err: %v", err)
		return nil, false
	}

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

	if f.RawData != nil && clipboard.IsImageFormat(f.FormatType) {
		if s.imageStore == nil {
			return
		}
		filePath, err := s.imageStore.Save(f.RawData, today)
		if err != nil {
			return
		}
		s.store.InsertFormat(entryID, f.FormatType, "", filePath, h)
	} else {
		s.store.InsertFormat(entryID, f.FormatType, f.Text, "", h)
	}
}

func (s *Service) pushToSync(hash string, formats []clipboard.CapturedFormat) {
	if s.onSyncPush == nil {
		return
	}
	var sf []SyncFormat
	for _, f := range formats {
		if f.Text != "" && !clipboard.IsHdropFormat(f.FormatType) {
			sf = append(sf, SyncFormat{FormatType: f.FormatType, Content: f.Text})
		}
	}
	if len(sf) > 0 {
		s.onSyncPush(hash, sf)
	}
}

// GetHistory returns entries with cursor-based pagination and tag filtering.
// tagMask=0 means all, afterUpdatedAt="" means first page.
// tagMask bit 5 (value 32) triggers favorites-only filter.
func (s *Service) GetHistory(search string, tagMask int, afterUpdatedAt string, afterID int64) (entries []clipboard.Entry, hasMore bool, err error) {
	pageSize := 20 + 1 // one extra to detect hasMore

	rows, err := s.store.QueryHistory(search, tagMask, afterUpdatedAt, afterID, pageSize)
	if err != nil {
		return nil, false, fmt.Errorf("query history: %w", err)
	}

	hasMore = len(rows) == pageSize
	if hasMore {
		rows = rows[:len(rows)-1]
	}

	// Batch-load formats.
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	formatMap, _ := s.store.LoadFormats(ids)

	for _, r := range rows {
		fmts := formatMap[r.ID]
		text := ""
		for _, f := range fmts {
			if f.FormatType == clipboard.CF_UNICODETEXT && f.Content != "" {
				text = f.Content
				break
			}
		}
		// Fallback: CF_HDROP path list when CF_UNICODETEXT is absent.
		if text == "" {
			for _, f := range fmts {
				if f.FormatType == clipboard.CF_HDROP && f.Content != "" {
					text = f.Content
					break
				}
			}
		}
		entries = append(entries, clipboard.Entry{
			ID:          r.ID,
			ContentHash: r.ContentHash,
			Content:     text,
			SourceEXE:   r.SourceEXE,
			SourceTitle: r.SourceTitle,
			Formats:     fmts,
			IsFavorite:  r.IsFavorite,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	return entries, hasMore, nil
}

// GetHistoryRegex returns all entries matching a regex pattern.
// Loads data in batches and filters with Go regexp — no cursor needed since results
// are typically small subsets of the full history.
func (s *Service) GetHistoryRegex(pattern string, tagMask int) (entries []clipboard.Entry, err error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	cursorAt := ""
	cursorID := int64(0)
	batchSize := 200
	var all []clipboard.Entry

	for {
		page, hasMore, pageErr := s.GetHistory("", tagMask, cursorAt, cursorID)
		if pageErr != nil {
			return nil, pageErr
		}
		for _, e := range page {
			if re.MatchString(e.Content) {
				all = append(all, e)
			}
		}
		if !hasMore || len(page) == 0 {
			break
		}
		last := page[len(page)-1]
		cursorAt = last.UpdatedAt
		cursorID = last.ID
		// Safety limit: don't scan more than 5000 entries.
		if len(all) > 5000 || len(page) >= batchSize*10 {
			break
		}
	}
	return all, nil
}

// DeleteEntry removes a single entry by ID.
func (s *Service) DeleteEntry(id int64) error {
	paths, err := s.store.DeleteEntry(id)
	if s.imageStore != nil && len(paths) > 0 {
		s.imageStore.DeleteByEntry(paths)
	}
	return err
}

// UseEntry performs the default action and refreshes updated_at.
func (s *Service) UseEntry(id int64, action string) error {
	// Try CF_HDROP (file paths) first — restore as proper file drop.
	hdropText, _ := s.store.QueryFormatContent(id, clipboard.CF_HDROP)
	if hdropText != "" {
		paths := strings.Split(hdropText, "\n")
		s.store.UpdateTimestamp(id, nowMillis())
		s.clipboard.SetFiles(paths)
		if action == "paste" && s.performPaste != nil {
			s.performPaste()
		}
		return nil
	}

	// Try text.
	text, _ := s.store.QueryFormatContent(id, clipboard.CF_UNICODETEXT)
	if text != "" {
		s.store.UpdateTimestamp(id, nowMillis())
		s.clipboard.SetText(text)
		if action == "paste" && s.performPaste != nil {
			s.performPaste()
		}
		return nil
	}

	// Fallback: try image.
	filePath, err := s.store.QueryImageFilePath(id)
	if err != nil || s.imageStore == nil {
		return fmt.Errorf("no pasteable format for entry %d", id)
	}

	dib, err := s.imageStore.ReadDIB(filePath)
	if err != nil {
		return fmt.Errorf("read image for entry %d: %w", id, err)
	}

	s.store.UpdateTimestamp(id, nowMillis())
	s.clipboard.SetImage(dib)
	if action == "paste" && s.performPaste != nil {
		s.performPaste()
	}
	return nil
}

// ToggleFavorite toggles the is_favorite flag for an entry.
func (s *Service) ToggleFavorite(id int64, value bool) error {
	return s.store.ToggleFavorite(id, value)
}

// Stats holds aggregate clipboard statistics.
type Stats struct {
	Count      int64 `json:"count"`
	TotalBytes int64 `json:"total_bytes"`
}

// GetStats returns count and total content size (text + image files).
func (s *Service) GetStats() (Stats, error) {
	st, err := s.store.GetStats()
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	// Add image file sizes from disk.
	if s.imageStore != nil {
		imgBytes, _ := s.imageStore.TotalImageBytes()
		st.TotalBytes += imgBytes
	}
	return st, nil
}

// Cleanup removes entries older than retainDays and their image files.
func (s *Service) Cleanup(retainDays int) (int64, error) {
	deleted, paths, err := s.store.Cleanup(retainDays)
	if s.imageStore != nil && len(paths) > 0 {
		s.imageStore.DeleteByEntry(paths)
		s.imageStore.CleanEmptyDirs()
	}
	return deleted, err
}

// ClearAll deletes all clipboard entries and their image files.
func (s *Service) ClearAll() error {
	paths, err := s.store.ClearAll()
	if s.imageStore != nil && len(paths) > 0 {
		s.imageStore.DeleteByEntry(paths)
		s.imageStore.CleanEmptyDirs()
	}
	return err
}

// GetEntryContent returns the CF_UNICODETEXT content for the given entry ID.
func (s *Service) GetEntryContent(id int64) (string, error) {
	return s.store.QueryFormatContent(id, clipboard.CF_UNICODETEXT)
}

// GetImageList returns all image entry IDs matching the given tag mask and search,
// ordered by updated_at DESC. Used by the image viewer for prev/next navigation.
func (s *Service) GetImageList(tagMask int, search string) ([]int64, error) {
	return s.store.QueryImageEntryIDs(tagMask, search)
}

// GetImageDataURL returns a base64 data URL for the first image format of an entry.
func (s *Service) GetImageDataURL(entryID int64) (string, error) {
	log.Printf("[history] GetImageDataURL entry=%d", entryID)
	filePath, err := s.store.QueryImageFilePath(entryID)
	if err != nil {
		log.Printf("[history] GetImageDataURL entry=%d: no image format row (err=%v)", entryID, err)
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
		if f.FormatType == clipboard.CF_UNICODETEXT {
			text = f.Text
			break
		}
	}
	// Fallback for CF_HDROP-only entries.
	if text == "" {
		for _, f := range formats {
			if f.FormatType == clipboard.CF_HDROP && f.Text != "" {
				text = f.Text
				break
			}
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
