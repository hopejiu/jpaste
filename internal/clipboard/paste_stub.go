//go:build !windows

package clipboard

// CaptureForeground is a no-op on non-Windows platforms.
func CaptureForeground() {}

// GetSavedForeground returns 0 on non-Windows.
func GetSavedForeground() uintptr { return 0 }

// DebugForeground returns 0 on non-Windows.
func DebugForeground() uintptr { return 0 }

// Paste is a no-op on non-Windows platforms.
func Paste() {}
