package jsonviewer

import (
	applog "jpaste/internal/log"
)

// CreateWindowFunc is a callback that opens a new window at a given URL path.
type CreateWindowFunc func(path string)

// Service manages JSON viewer windows.
type Service struct {
	createWin CreateWindowFunc
}

// NewService creates a new JSON viewer service.
func NewService(createWin CreateWindowFunc) *Service {
	return &Service{createWin: createWin}
}

// OpenJsonViewer opens a new Wails window at /json-view?id=<entryID>.
// The JSON viewer page fetches the entry content by ID from the database,
// eliminating the need for in-memory token storage.
func (s *Service) OpenJsonViewer(id int64) {
	path := "/json-view?id=" + itoa(id)
	applog.Info("jsonviewer: open", "id", id, "path", path)
	s.createWin(path)
}

// itoa converts int64 to string without importing fmt.
func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	negative := false
	if i < 0 {
		negative = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
