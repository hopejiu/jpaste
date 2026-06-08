//go:build !windows

package hotkey

// Service manages system-level global hotkeys (stub for non-Windows).
type Service struct{}

// NewService creates a new hotkey service (no-op on non-Windows).
func NewService() *Service {
	return &Service{}
}

// Register is a no-op on non-Windows platforms.
func (s *Service) Register(keystr string, callback func()) error {
	return nil
}

// UnregisterAll is a no-op on non-Windows platforms.
func (s *Service) UnregisterAll() {}

// RegisterAndSwap is a no-op on non-Windows platforms.
func (s *Service) RegisterAndSwap(keystr string, callback func()) error {
	return nil
}
