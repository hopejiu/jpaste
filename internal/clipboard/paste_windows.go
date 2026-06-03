//go:build windows

package clipboard

import (
	"time"
	"unsafe"

	"github.com/lxn/win"
)

const (
	VK_CONTROL      = 0x11
	VK_V            = 0x56
	KEYEVENTF_KEYUP = 0x0002
	WM_PASTE        = 0x0302
)

// _foregroundBeforeShow saves the foreground window handle before jPaste appears.
var _foregroundBeforeShow uintptr

// CaptureForeground captures the current foreground window BEFORE jPaste shows.
func CaptureForeground() {
	hwnd, _, _ := procGetForeground.Call()
	_foregroundBeforeShow = hwnd
}

// WriteImage puts raw DIB bytes onto the system clipboard as CF_DIB.
func WriteImage(dibData []byte) bool {
	byteLen := len(dibData)
	hMem := win.GlobalAlloc(gmemMoveable, uintptr(byteLen))
	if hMem == 0 {
		return false
	}

	ptr := win.GlobalLock(hMem)
	if ptr == nil {
		win.GlobalFree(hMem)
		return false
	}
	dst := unsafe.Slice((*byte)(ptr), byteLen)
	copy(dst, dibData)
	win.GlobalUnlock(hMem)

	if !win.OpenClipboard(0) {
		win.GlobalFree(hMem)
		return false
	}
	defer win.CloseClipboard()

	win.EmptyClipboard()
	win.SetClipboardData(win.CF_DIB, win.HANDLE(hMem))
	return true
}

// Paste uses PostMessage(WM_PASTE) to the target window, then falls back to
// Ctrl+V via keybd_event for apps that don't handle WM_PASTE.
func Paste() {
	if _foregroundBeforeShow != 0 {
		procPostMessage.Call(_foregroundBeforeShow, WM_PASTE, 0, 0)
		time.Sleep(30 * time.Millisecond)
	}
	procKeybdEvent.Call(VK_CONTROL, 0, 0, 0)
	procKeybdEvent.Call(VK_V, 0, 0, 0)
	time.Sleep(20 * time.Millisecond)
	procKeybdEvent.Call(VK_V, 0, KEYEVENTF_KEYUP, 0)
	time.Sleep(20 * time.Millisecond)
	procKeybdEvent.Call(VK_CONTROL, 0, KEYEVENTF_KEYUP, 0)
}
