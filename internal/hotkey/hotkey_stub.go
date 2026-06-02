//go:build !windows

package hotkey

// Register is a no-op on non-Windows platforms.
func Register(keystr string, callback func()) error {
	return nil
}

// UnregisterAll is a no-op on non-Windows platforms.
func UnregisterAll() {}
