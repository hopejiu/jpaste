package wssviewer

import (
	applog "jpaste/internal/log"
)

// CreateWindowFunc is a callback that opens a new Wails window at a given URL path.
type CreateWindowFunc func(path string)

// Service manages WebSocket viewer windows.
type Service struct {
	createWin CreateWindowFunc
}

// NewService creates a new WebSocket viewer service.
func NewService(createWin CreateWindowFunc) *Service {
	return &Service{createWin: createWin}
}

// OpenWsViewer opens a new Wails window at /ws-view?id=<entryID>.
func (s *Service) OpenWsViewer(id int64) {
	path := "/ws-view?id=" + formatInt(id)
	applog.Info("wssviewer: open", "id", id, "path", path)
	s.createWin(path)
}

// formatInt converts int64 to string without importing fmt.
func formatInt(i int64) string {
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
