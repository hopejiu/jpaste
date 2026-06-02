//go:build windows

package clipboard

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/win"
)

func init() {
	platformStart = startWindowsMonitor
}

// ---------------------------------------------------------------------------
// Constants not in lxn/win
// ---------------------------------------------------------------------------

const (
	gmemMoveable      = 0x0002
	wmClipboardUpdate = 0x031D
	wmDestroy         = 0x0002
	processQueryLimit = 0x1000

	// CFDIBV5 is the DIB v5 clipboard format constant.
	CFDIBV5 = 17
)

// hwndMessage is HWND_MESSAGE (-3).
var hwndMessage = win.HWND(^uintptr(2)) // -3

// ---------------------------------------------------------------------------
// Syscall procs for functions not in lxn/win
// ---------------------------------------------------------------------------

var (
	kernel32                          = syscall.NewLazyDLL("kernel32.dll")
	user32                            = syscall.NewLazyDLL("user32.dll")
	procRegisterClipboardFormat       = user32.NewProc("RegisterClipboardFormatW")
	procEnumClipboardFormats          = user32.NewProc("EnumClipboardFormats")
	procGlobalSize                    = kernel32.NewProc("GlobalSize")
	procGetClipboardOwner             = user32.NewProc("GetClipboardOwner")
	procOpenProcess                   = kernel32.NewProc("OpenProcess")
	procGetWindowText                 = user32.NewProc("GetWindowTextW")
	procRemoveClipboardFormatListener = user32.NewProc("RemoveClipboardFormatListener")
	procQueryFullProcessImageName     = kernel32.NewProc("QueryFullProcessImageNameW")
	procKeybdEvent                    = user32.NewProc("keybd_event")
	procGetForeground                 = user32.NewProc("GetForegroundWindow")
)

// ---------------------------------------------------------------------------
// Registered format IDs
// ---------------------------------------------------------------------------

var (
	cfHTML uint32
	cfRTF  uint32
)

func init() {
	cfHTML = registerClipboardFormat("HTML Format")
	cfRTF = registerClipboardFormat("Rich Text Format")
}

func registerClipboardFormat(name string) uint32 {
	ptr, _ := syscall.UTF16PtrFromString(name)
	r, _, _ := procRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(ptr)))
	return uint32(r)
}

func isTextFormat(f uint32) bool {
	switch f {
	case win.CF_UNICODETEXT, win.CF_TEXT, win.CF_HDROP, cfHTML, cfRTF:
		return true
	}
	return false
}

func isImageFormat(f uint32) bool {
	return f == win.CF_DIB || f == CFDIBV5
}

// ---------------------------------------------------------------------------
// Event-driven monitoring via AddClipboardFormatListener
// ---------------------------------------------------------------------------

func startWindowsMonitor(onCapture OnCapture) (func(), error) {
	log.Println("[clipboard] Starting WM_CLIPBOARDUPDATE monitor...")

	instance := win.GetModuleHandle(nil)
	className, _ := syscall.UTF16PtrFromString("jPasteClipMon")

	var wc win.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(clipboardWndProc)
	wc.HInstance = instance
	wc.LpszClassName = className
	atom := win.RegisterClassEx(&wc)
	log.Printf("[clipboard] RegisterClassEx atom=%d", atom)
	if atom == 0 {
		return nil, fmt.Errorf("RegisterClassEx failed: %d", win.GetLastError())
	}

	ready := make(chan win.HWND, 1)
	errCh := make(chan error, 1)
	dataCh := make(chan CapturedData, 64)
	done := make(chan struct{})

	// Window creation + message pump on a single locked OS thread.
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hwnd := win.CreateWindowEx(
			0, className, nil,
			win.WS_POPUP, 0, 0, 0, 0,
			hwndMessage, 0, instance, nil,
		)
		log.Printf("[clipboard] CreateWindowEx hwnd=%v", hwnd)
		if hwnd == 0 {
			errCh <- fmt.Errorf("CreateWindowEx failed: %d", win.GetLastError())
			return
		}
		ready <- hwnd

		if !win.AddClipboardFormatListener(hwnd) {
			win.DestroyWindow(hwnd)
			errCh <- fmt.Errorf("AddClipboardFormatListener failed: %d", win.GetLastError())
			return
		}
		log.Println("[clipboard] Monitor listening on message-only window")

		var msg win.MSG
		for {
			ret := win.GetMessage(&msg, hwnd, 0, 0)
			if ret == 0 || ret == -1 {
				log.Printf("[clipboard] Message loop exit (ret=%d)", ret)
				break
			}
			if msg.Message == wmClipboardUpdate {
				log.Println("[clipboard] WM_CLIPBOARDUPDATE received")
				data := captureAll()
				if len(data.Formats) > 0 {
					select {
					case dataCh <- data:
					default:
						log.Println("[clipboard] WARNING: data channel full")
					}
				}
			}
		}
		close(done)
	}()

	// Wait for window creation.
	var hwnd win.HWND
	select {
	case hwnd = <-ready:
		log.Printf("[clipboard] Window ready, hwnd=%v", hwnd)
	case err := <-errCh:
		return nil, err
	}

	// Data processor goroutine.
	go func() {
		for {
			select {
			case data := <-dataCh:
				log.Printf("[clipboard] Processing capture: hash=%s formats=%d", data.PrimaryHash[:12], len(data.Formats))
				onCapture(data)
			case <-done:
				return
			}
		}
	}()

	stop := func() {
		log.Println("[clipboard] Stopping monitor...")
		procRemoveClipboardFormatListener.Call(uintptr(hwnd))
		win.PostMessage(hwnd, wmDestroy, 0, 0)
		<-done
		log.Println("[clipboard] Monitor stopped")
	}

	return stop, nil
}

func clipboardWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

// ---------------------------------------------------------------------------
// Capture — read all formats + source info from system clipboard
// ---------------------------------------------------------------------------

func captureAll() CapturedData {
	if !win.OpenClipboard(0) {
		return CapturedData{}
	}
	defer win.CloseClipboard()

	formats := enumFormats()
	var cf []CapturedFormat
	var textContent string

	for _, f := range formats {
		if isTextFormat(f) {
			txt := readClipboardText(f)
			if txt != "" {
				cf = append(cf, CapturedFormat{FormatType: f, Text: txt})
				if f == win.CF_UNICODETEXT {
					textContent = txt
				}
			}
		} else if isImageFormat(f) {
			raw := readClipboardBytes(f)
			if len(raw) > 0 {
				cf = append(cf, CapturedFormat{FormatType: f, RawData: raw})
			}
		}
	}

	exe, title := getClipboardSource()

	var hashInput []byte
	if textContent != "" {
		hashInput = []byte(textContent)
	} else {
		for _, f := range cf {
			if isImageFormat(f.FormatType) {
				hashInput = f.RawData
				break
			}
		}
	}
	if len(hashInput) == 0 {
		return CapturedData{}
	}

	h := sha256.Sum256(hashInput)
	return CapturedData{
		Formats:     cf,
		SourceEXE:   exe,
		SourceTitle: title,
		PrimaryHash: fmt.Sprintf("%x", h[:]),
	}
}

func enumFormats() []uint32 {
	var formats []uint32
	f := uint32(0)
	for {
		r, _, _ := procEnumClipboardFormats.Call(uintptr(f))
		f = uint32(r)
		if f == 0 {
			break
		}
		formats = append(formats, f)
	}
	return formats
}

func readClipboardText(format uint32) string {
	hData := win.GetClipboardData(format)
	if hData == 0 {
		return ""
	}
	hMem := win.HGLOBAL(hData)
	size, _, _ := procGlobalSize.Call(uintptr(hMem))
	if size == 0 {
		return ""
	}
	ptr := win.GlobalLock(hMem)
	if ptr == nil {
		return ""
	}
	defer win.GlobalUnlock(hMem)

	if format == win.CF_UNICODETEXT {
		return utf16BytesToString(unsafe.Slice((*byte)(ptr), size))
	}
	return string(bytesFromPtr(ptr, int(size)))
}

func readClipboardBytes(format uint32) []byte {
	hData := win.GetClipboardData(format)
	if hData == 0 {
		return nil
	}
	hMem := win.HGLOBAL(hData)
	size, _, _ := procGlobalSize.Call(uintptr(hMem))
	if size == 0 {
		return nil
	}
	ptr := win.GlobalLock(hMem)
	if ptr == nil {
		return nil
	}
	defer win.GlobalUnlock(hMem)

	return bytesFromPtr(ptr, int(size))
}

func bytesFromPtr(ptr unsafe.Pointer, size int) []byte {
	data := make([]byte, size)
	copy(data, unsafe.Slice((*byte)(ptr), size))
	return data
}

func utf16BytesToString(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	for len(data) >= 2 && data[len(data)-2] == 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-2]
	}
	if len(data) == 0 {
		return ""
	}
	u16 := make([]uint16, len(data)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(u16))
}

// ---------------------------------------------------------------------------
// Source application identification
// ---------------------------------------------------------------------------

func getClipboardSource() (exe, title string) {
	hwndRaw, _, _ := procGetClipboardOwner.Call()
	hwnd := win.HWND(hwndRaw)
	if hwnd == 0 {
		return "", ""
	}

	var pid uint32
	win.GetWindowThreadProcessId(hwnd, &pid)
	if pid == 0 {
		return "", ""
	}

	hProcessRaw, _, _ := procOpenProcess.Call(processQueryLimit, 0, uintptr(pid))
	hProcess := win.HANDLE(hProcessRaw)
	if hProcess == 0 {
		return "", ""
	}
	defer win.CloseHandle(hProcess)

	exe = queryFullProcessImageName(hProcess)

	var buf [260]uint16
	r, _, _ := procGetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n := int(r); n > 0 {
		title = syscall.UTF16ToString(buf[:n])
	}

	return exe, title
}

func queryFullProcessImageName(hProcess win.HANDLE) string {
	var buf [win.MAX_PATH]uint16
	size := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageName.Call(
		uintptr(hProcess), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

// ---------------------------------------------------------------------------
// Write to system clipboard
// ---------------------------------------------------------------------------

func WriteText(text string) bool {
	u16, err := syscall.UTF16FromString(text)
	if err != nil {
		return false
	}
	byteLen := len(u16) * 2
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
	src := unsafe.Slice((*byte)(unsafe.Pointer(&u16[0])), byteLen)
	copy(dst, src)
	win.GlobalUnlock(hMem)

	if !win.OpenClipboard(0) {
		win.GlobalFree(hMem)
		return false
	}
	defer win.CloseClipboard()

	win.EmptyClipboard()
	win.SetClipboardData(win.CF_UNICODETEXT, win.HANDLE(hMem))
	return true
}
