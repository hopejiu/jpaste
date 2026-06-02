//go:build !windows

package clipboard

// CaptureForeground is a no-op on non-Windows platforms.
func CaptureForeground() {}

// Paste is a no-op on non-Windows platforms.
func Paste() {}
