package imageviewer

import (
	"fmt"
	applog "jpaste/internal/log"
)

// CreateWindowFunc is a callback that opens a new Wails window at the given path.
type CreateWindowFunc func(path string)

// Service manages image viewer windows.
type Service struct {
	createWin CreateWindowFunc
}

// NewService creates a new image viewer service.
func NewService(createWin CreateWindowFunc) *Service {
	return &Service{createWin: createWin}
}

// OpenImageViewer opens a new Wails window for viewing an image entry.
// The window path includes the entry ID, tag mask and search for navigation.
func (s *Service) OpenImageViewer(id int64, tagMask int, search string) {
	path := fmt.Sprintf("/image-view?id=%d&tag=%d&search=%s", id, tagMask, search)
	applog.Info("imageviewer: open", "id", id, "tag", tagMask, "search", search)
	s.createWin(path)
}
