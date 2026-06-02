//go:build windows

package notify

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procCreateWindowEx  = user32.NewProc("CreateWindowExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procRegisterClassEx = user32.NewProc("RegisterClassExW")
	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
	procPostMessage     = user32.NewProc("PostMessageW")
	procGetMessage      = user32.NewProc("GetMessageW")
	procGetLastError    = kernel32.NewProc("GetLastError")
)

const (
	nimAdd     = 0x00000000
	nimModify  = 0x00000001
	nimDelete  = 0x00000002
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifInfo    = 0x00000010
	niifInfo   = 0x00000001
	wmDestroy  = 0x0002
	wmQuit     = 0x0012
)

type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uTimeout         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

var (
	msgHwnd     uintptr
	uid         uint32 = 1
	loopRunning bool
	initialized bool
)

func lastErr() uint32 {
	ret, _, _ := procGetLastError.Call()
	return uint32(ret)
}

func initWindow() {
	if initialized {
		return
	}
	initialized = true

	hInst, _, _ := procGetModuleHandle.Call(0)
	if hInst == 0 {
		return
	}

	className := syscall.StringToUTF16Ptr("jPasteNotifyClass")
	var wc struct {
		cbSize        uint32
		style         uint32
		lpfnWndProc   uintptr
		cbClsExtra    int32
		cbWndExtra    int32
		hInstance     uintptr
		hIcon         uintptr
		hCursor       uintptr
		hbrBackground uintptr
		lpszMenuName  *uint16
		lpszClassName *uint16
		hIconSm       uintptr
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = syscall.NewCallback(wndProc)
	wc.hInstance = hInst
	wc.lpszClassName = className

	atom, _, _ := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return
	}

	msgHwnd, _, _ = procCreateWindowEx.Call(
		0, uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("jPasteNotify"))),
		0, 0, 0, 0, 0,
		0, 0, hInst, 0,
	)
	if msgHwnd == 0 {
		return
	}

	// Add notification icon with default app icon (needed for balloon anchor).
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = msgHwnd
	nid.uID = uid
	nid.uFlags = nifMessage | nifIcon
	nid.uCallbackMessage = 0x8001
	// Load default application icon.
	hIcon, _, _ := user32.NewProc("LoadIconW").Call(0, 32512) // IDI_APPLICATION
	nid.hIcon = hIcon

	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))

	loopRunning = true
	go messagePump()
}

func messagePump() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var msg struct {
		hwnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      struct{ x, y int32 }
	}
	for loopRunning {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 || ret == ^uintptr(0) {
			break
		}
	}
}

func wndProc(hWnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == wmDestroy {
		loopRunning = false
		procPostMessage.Call(0, wmQuit, 0, 0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hWnd, uintptr(msg), wParam, lParam)
	return ret
}

// ShowToast displays a native Windows balloon notification from the system tray.
func ShowToast(title, message string) {
	initWindow()
	if msgHwnd == 0 {
		return
	}

	title16, _ := syscall.UTF16FromString(title)
	msg16, _ := syscall.UTF16FromString(message)

	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = msgHwnd
	nid.uID = uid
	nid.uFlags = nifInfo
	nid.dwInfoFlags = niifInfo
	nid.uTimeout = 5000

	copy(nid.szInfoTitle[:], title16)
	copy(nid.szInfo[:], msg16)

	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))

}


// Shutdown destroys the notification window and removes the tray icon.
func Shutdown() {
	if msgHwnd != 0 {
		// Remove tray icon.
		var nid notifyIconData
		nid.cbSize = uint32(unsafe.Sizeof(nid))
		nid.hWnd = msgHwnd
		nid.uID = uid
		procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))

		loopRunning = false
		procDestroyWindow.Call(msgHwnd)
		msgHwnd = 0
		initialized = false
	}
}
