//go:build windows

package clipboard

import (
	"time"
)

const (
	VK_CONTROL      = 0x11
	VK_V            = 0x56
	KEYEVENTF_KEYUP = 0x0002
)

// _foregroundBeforeShow saves the foreground window handle before jPaste appears.
var _foregroundBeforeShow uintptr

// CaptureForeground captures the current foreground window BEFORE jPaste shows.
func CaptureForeground() {
	hwnd, _, _ := procGetForeground.Call()
	_foregroundBeforeShow = hwnd
}

// Paste sends Ctrl+V to whatever window currently has focus.
// Caller must ensure jPaste window is already hidden.
func Paste() {
	time.Sleep(100 * time.Millisecond)

	// Send Ctrl+V.
	procKeybdEvent.Call(VK_CONTROL, 0, 0, 0)
	procKeybdEvent.Call(VK_V, 0, 0, 0)
	time.Sleep(20 * time.Millisecond)
	procKeybdEvent.Call(VK_V, 0, KEYEVENTF_KEYUP, 0)
	time.Sleep(20 * time.Millisecond)
	procKeybdEvent.Call(VK_CONTROL, 0, KEYEVENTF_KEYUP, 0)
}
