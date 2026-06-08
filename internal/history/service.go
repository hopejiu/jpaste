package history

import (
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"jpaste/internal/model"
	"jpaste/internal/util"
)

// Service provides clipboard history queries and actions.
type Service struct {
	store      EntryStore
	clipboard  ClipboardWriter
	imageStore ImageStorer

	performPaste func()
	onEmit       func(name string, data any)
	onNotify     func(title, msg string)
}

// ClipboardWriter abstracts clipboard write operations.
type ClipboardWriter interface {
	SetText(text string) bool
	SetImage(dib []byte) bool
	SetFiles(paths []string) bool
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
func (s *Service) CaptureEntry(data model.CapturedData) (*model.Entry, bool) {
	now := nowMillis()
	tagMask := model.ComputeTagMask(data.Formats)

	// File copies (CF_HDROP) are exempt from dedup — they carry richer format
	// data that should always create a fresh entry even if the text content
	// matches a previous plain-text copy.
	hasFileFormat := false
	for _, f := range data.Formats {
		if model.IsHdropFormat(f.FormatType) {
			hasFileFormat = true
			break
		}
	}

	if !hasFileFormat {
		// Don't dedup plain-text copies of previously-copied files.
		// Existing file entries carry CF_HDROP data that text-only copies lack.
		existingHasFile, _ := s.store.HasFileFormatByHash(data.PrimaryHash)
		if !existingHasFile {
			contentLength := computeTextLength(data.Formats)
			deduped, err := s.store.UpsertDedup(data.PrimaryHash, data.SourceEXE, data.SourceTitle, tagMask, now, contentLength)
			if err != nil {
				log.Printf("[history] dedup err: %v", err)
			}
			if deduped {
				return nil, false
			}
		}
	}

	// Insert new entry.
	contentLength := computeTextLength(data.Formats)
	id, err := s.store.InsertEntry(data.PrimaryHash, data.SourceEXE, data.SourceTitle, tagMask, now, contentLength)
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
		s.onNotify("jPaste", util.TruncateBytes(entry.Content, 80))
	}

	return entry, true
}

func (s *Service) saveFormat(entryID int64, f model.CapturedFormat, today string) {
	h := util.SHA256TextOrRaw(f.Text, f.RawData)

	if f.RawData != nil && model.IsImageFormat(f.FormatType) {
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

// GetHistory returns entries with cursor-based pagination and tag filtering.
// tagMask=0 means all, afterCursor1="" means first page.
// tagMask bit 5 (value 32) triggers favorites-only filter.
// sortField: "updated_at" | "content_length", sortOrder: "asc" | "desc"
func (s *Service) GetHistory(search string, tagMask int, afterCursor1 string, afterID int64, sortField string, sortOrder string) (entries []model.Entry, hasMore bool, err error) {
	pageSize := 20 + 1 // one extra to detect hasMore

	sortField, sortOrder = resolveSortParams(sortField, sortOrder)

	rows, err := s.store.QueryHistory(search, tagMask, afterCursor1, afterID, pageSize, sortField, sortOrder)
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
		text := model.PrimaryTextFromEntries(fmts)
		entries = append(entries, model.Entry{
			ID:            r.ID,
			ContentHash:   r.ContentHash,
			Content:       text,
			SourceEXE:     r.SourceEXE,
			SourceTitle:   r.SourceTitle,
			Formats:       fmts,
			IsFavorite:    r.IsFavorite,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
			ContentLength: r.ContentLength,
		})
	}
	return entries, hasMore, nil
}

// GetHistoryRegex returns all entries matching a regex pattern.
// Loads data in batches and filters with Go regexp — no cursor needed since results
// are typically small subsets of the full history.
func (s *Service) GetHistoryRegex(pattern string, tagMask int, sortField string, sortOrder string) (entries []model.Entry, err error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	sortField, sortOrder = resolveSortParams(sortField, sortOrder)

	var cursor1 string
	cursorID := int64(0)
	batchSize := 200
	var all []model.Entry

	for {
		page, hasMore, pageErr := s.GetHistory("", tagMask, cursor1, cursorID, sortField, sortOrder)
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
		if sortField == "content_length" {
			cursor1 = fmt.Sprintf("%d", last.ContentLength)
		} else {
			cursor1 = last.UpdatedAt
		}
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
	log.Printf("[history] UseEntry: id=%d action=%s", id, action)

	// Try CF_HDROP (file paths) first — restore as proper file drop.
	hdropText, _ := s.store.QueryFormatContent(id, model.CF_HDROP)
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
	text, _ := s.store.QueryFormatContent(id, model.CF_UNICODETEXT)
	if text != "" {
		log.Printf("[history] UseEntry: calling SetText, len=%d text=%q", len(text), util.Truncate(text, 40))
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
// Favorited entries (is_favorite = 1) are exempt from cleanup.
func (s *Service) Cleanup(retainDays int) (int64, error) {
	deleted, paths, err := s.store.Cleanup(retainDays)
	if err != nil {
		log.Printf("[Cleanup] error: %v", err)
	}
	log.Printf("[Cleanup] retainDays=%d deleted=%d imagePaths=%d", retainDays, deleted, len(paths))
	if s.imageStore != nil && len(paths) > 0 {
		s.imageStore.DeleteByEntry(paths)
		s.imageStore.CleanEmptyDirs()
	}
	return deleted, err
}

// ClearAll deletes all clipboard entries and their image files.
// If keepFavorites is true, favorited entries are preserved.
func (s *Service) ClearAll(keepFavorites bool) error {
	log.Printf("[ClearAll] keepFavorites=%v", keepFavorites)
	paths, err := s.store.ClearAll(keepFavorites)
	if err != nil {
		log.Printf("[ClearAll] store error: %v", err)
		return err
	}
	log.Printf("[ClearAll] deleted, image paths=%d", len(paths))
	if s.imageStore != nil && len(paths) > 0 {
		s.imageStore.DeleteByEntry(paths)
		s.imageStore.CleanEmptyDirs()
		log.Printf("[ClearAll] image cleanup done")
	}
	return nil
}

// GetEntryContent returns the CF_UNICODETEXT content for the given entry ID.
func (s *Service) GetEntryContent(id int64) (string, error) {
	return s.store.QueryFormatContent(id, model.CF_UNICODETEXT)
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

func computeTextLength(formats []model.CapturedFormat) int {
	return model.TextLength(formats)
}

func buildEntry(id int64, hash, exe, title, createdAt, updatedAt string, formats []model.CapturedFormat) *model.Entry {
	text := model.PrimaryText(formats)
	e := &model.Entry{
		ID:          id,
		ContentHash: hash,
		Content:     text,
		SourceEXE:   exe,
		SourceTitle: title,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	for _, f := range formats {
		fe := model.FormatEntry{FormatType: f.FormatType}
		if f.RawData != nil {
			fe.Content = fmt.Sprintf("[image %d bytes]", len(f.RawData))
		} else {
			fe.Content = f.Text
		}
		e.Formats = append(e.Formats, fe)
	}
	return e
}

// resolveSortParams returns validated sort field and order, applying defaults if empty.
func resolveSortParams(sortField, sortOrder string) (field, order string) {
	if sortField == "" {
		sortField = "updated_at"
	}
	if sortOrder == "" {
		sortOrder = "DESC"
	}
	return sortField, strings.ToUpper(sortOrder)
}



