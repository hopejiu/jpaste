//go:build !windows

package notify

// ShowToast is a no-op on non-Windows platforms.
func ShowToast(title, message string) {}

// Shutdown is a no-op on non-Windows platforms.
func Shutdown() {}
