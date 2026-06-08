package history

import (
	"jpaste/internal/db"
	"jpaste/internal/model"
)

// EntryStore abstracts the persistence layer for clipboard entries.
// The production adapter is repoAdapter backed by repository.Repository.
type EntryStore interface {
	QueryHistory(search string, tagMask int, afterCursor1 string, afterID int64, limit int, sortField string, sortOrder string) ([]EntryRow, error)
	LoadFormats(ids []int64) (map[int64][]model.FormatEntry, error)
	UpsertDedup(hash, sourceEXE, sourceTitle string, tagMask int, now string, contentLength int) (deduped bool, err error)
	InsertEntry(hash, sourceEXE, sourceTitle string, tagMask int, now string, contentLength int) (id int64, err error)
	InsertFormat(entryID int64, formatType uint32, content, filePath, formatHash string) error
	QueryFormatContent(entryID int64, formatType uint32) (string, error)
	QueryImageFilePath(entryID int64) (string, error)
	UpdateTimestamp(id int64, now string) error
	DeleteEntry(id int64) (imagePaths []string, err error)
	ToggleFavorite(id int64, value bool) error
	GetStats() (Stats, error)
	Cleanup(retainDays int) (deleted int64, imagePaths []string, err error)
	ClearAll(keepFavorites bool) (imagePaths []string, err error)
	HasFileFormatByHash(hash string) (bool, error)
	QueryImageEntryIDs(tagMask int, search string) ([]int64, error)
}

// EntryRow is a single row from the clipboard_entry table.
type EntryRow = db.EntryRow

// repoAdapter adapts *db.Repository to EntryStore.
type repoAdapter struct {
	repo *db.Repository
}

// NewSQLiteStore creates an EntryStore backed by *db.Repository.
func NewSQLiteStore(repo *db.Repository) EntryStore {
	return &repoAdapter{repo: repo}
}

func (a *repoAdapter) QueryHistory(search string, tagMask int, afterCursor1 string, afterID int64, limit int, sortField, sortOrder string) ([]EntryRow, error) {
	return a.repo.QueryHistory(search, tagMask, afterCursor1, afterID, limit, sortField, sortOrder)
}

func (a *repoAdapter) LoadFormats(ids []int64) (map[int64][]model.FormatEntry, error) {
	return a.repo.LoadFormats(ids)
}

func (a *repoAdapter) UpsertDedup(hash, sourceEXE, sourceTitle string, tagMask int, now string, contentLength int) (bool, error) {
	return a.repo.UpsertDedup(hash, sourceEXE, sourceTitle, tagMask, now, contentLength)
}

func (a *repoAdapter) InsertEntry(hash, sourceEXE, sourceTitle string, tagMask int, now string, contentLength int) (int64, error) {
	return a.repo.InsertEntry(hash, sourceEXE, sourceTitle, tagMask, now, contentLength)
}

func (a *repoAdapter) InsertFormat(entryID int64, formatType uint32, content, filePath, formatHash string) error {
	return a.repo.InsertFormat(entryID, formatType, content, filePath, formatHash)
}

func (a *repoAdapter) QueryFormatContent(entryID int64, formatType uint32) (string, error) {
	return a.repo.QueryFormatContent(entryID, formatType)
}

func (a *repoAdapter) QueryImageFilePath(entryID int64) (string, error) {
	return a.repo.QueryImageFilePath(entryID)
}

func (a *repoAdapter) UpdateTimestamp(id int64, now string) error {
	return a.repo.UpdateTimestamp(id, now)
}

func (a *repoAdapter) DeleteEntry(id int64) ([]string, error) {
	return a.repo.DeleteEntry(id)
}

func (a *repoAdapter) ToggleFavorite(id int64, value bool) error {
	return a.repo.ToggleFavorite(id, value)
}

func (a *repoAdapter) GetStats() (Stats, error) {
	st, err := a.repo.GetStats()
	if err != nil {
		return Stats{}, err
	}
	return Stats(st), nil
}

func (a *repoAdapter) Cleanup(retainDays int) (int64, []string, error) {
	return a.repo.Cleanup(retainDays)
}

func (a *repoAdapter) ClearAll(keepFavorites bool) ([]string, error) {
	return a.repo.ClearAll(keepFavorites)
}

func (a *repoAdapter) HasFileFormatByHash(hash string) (bool, error) {
	return a.repo.HasFileFormatByHash(hash)
}

func (a *repoAdapter) QueryImageEntryIDs(tagMask int, search string) ([]int64, error) {
	return a.repo.QueryImageEntryIDs(tagMask, search)
}
