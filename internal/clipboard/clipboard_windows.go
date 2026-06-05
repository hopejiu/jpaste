//go:build windows

package clipboard

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"runtime"
	"strings"
	"time"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/win"
)

func init() {
	platformStart = startWindowsMonitor
}

const (
	gmemMoveable      = 0x0002
	wmClipboardUpdate = 0x031D
	wmDestroy         = 0x0002
	wmQuit            = 0x0012
	processQueryLimit = 0x1000

	CFDIBV5 = 17
)

// hwndMessage is HWND_MESSAGE (-3).
var hwndMessage = win.HWND(^uintptr(2)) // -3

var (
	kernel32                          = syscall.NewLazyDLL("kernel32.dll")
	user32                            = syscall.NewLazyDLL("user32.dll")
	procEnumClipboardFormats          = user32.NewProc("EnumClipboardFormats")
	procGlobalSize                    = kernel32.NewProc("GlobalSize")
	procGetClipboardOwner             = user32.NewProc("GetClipboardOwner")
	procOpenProcess                   = kernel32.NewProc("OpenProcess")
	procGetWindowText                 = user32.NewProc("GetWindowTextW")
	procRemoveClipboardFormatListener = user32.NewProc("RemoveClipboardFormatListener")
	procQueryFullProcessImageName     = kernel32.NewProc("QueryFullProcessImageNameW")
	procKeybdEvent                    = user32.NewProc("keybd_event")
	procGetForeground                 = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	procGetCurrentThreadId            = kernel32.NewProc("GetCurrentThreadId")
	procGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput             = user32.NewProc("AttachThreadInput")
	procPostMessage                   = user32.NewProc("PostMessageW")
)

func isTextFormat(f uint32) bool {
	switch f {
	case win.CF_UNICODETEXT, win.CF_TEXT:
		return true
	}
	return false
}

func isImageFormat(f uint32) bool {
	return f == win.CF_DIB || f == CFDIBV5
}

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
			if msg.Message == wmQuit {
				log.Println("[clipboard] WM_QUIT received, exiting loop")
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
		win.PostMessage(hwnd, wmQuit, 0, 0)
		select {
		case <-done:
			log.Println("[clipboard] Monitor stopped")
		case <-time.After(3 * time.Second):
			log.Println("[clipboard] Monitor stop timed out")
		}
	}

	return stop, nil
}

func clipboardWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func captureAll() CapturedData {
	if !win.OpenClipboard(0) {
		log.Println("[clipboard] OpenClipboard failed for capture")
		return CapturedData{}
	}
	defer win.CloseClipboard()

	formats := enumFormats()
	log.Printf("[clipboard] EnumFormats count=%d", len(formats))
	var cf []CapturedFormat
	var textContent string

	for _, f := range formats {
		if f == win.CF_HDROP {
			txt := readClipboardHDROP()
			if txt != "" {
				log.Printf("[clipboard] CF_HDROP parsed: paths=%q", txt)
				cf = append(cf, CapturedFormat{FormatType: f, Text: txt})
			}
		} else if isTextFormat(f) {
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

	log.Printf("[clipboard] Captured %d formats, textContent.len=%d", len(cf), len(textContent))

	exe, title := getClipboardSource()

	var hashInput []byte
	if textContent != "" {
		hashInput = []byte(textContent)
	} else {
		for _, f := range cf {
			if isImageFormat(f.FormatType) {
				hashInput = imagePixelHash(f.RawData)
				break
			}
		}
	}
	// Fallback for CF_HDROP-only (no CF_UNICODETEXT, no image).
	if len(hashInput) == 0 {
		for _, f := range cf {
			if f.FormatType == win.CF_HDROP && f.Text != "" {
				hashInput = []byte(f.Text)
				break
			}
		}
	}
	if len(hashInput) == 0 {
		log.Println("[clipboard] No hashable content, returning empty")
		return CapturedData{}
	}

	h := sha256.Sum256(hashInput)
	hashStr := fmt.Sprintf("%x", h[:])

	// File entries (CF_HDROP) get a distinct hash prefix to coexist
	// with plain-text copies of the same path content.
	if hasHdropFormat(cf) {
		hashStr = "hdrop:" + hashStr
	}

	log.Printf("[clipboard] Capture success: hash=%s source=%q", hashStr[:8], exe)
	return CapturedData{
		Formats:     cf,
		SourceEXE:   exe,
		SourceTitle: title,
		PrimaryHash: hashStr,
	}
}

func hasHdropFormat(formats []CapturedFormat) bool {
	for _, f := range formats {
		if f.FormatType == win.CF_HDROP {
			return true
		}
	}
	return false
}

// imagePixelHash decodes DIB raw bytes to image.Image, converts to NRGBA
// for consistent byte layout, and returns the pixel bytes for hashing.
// Falls back to raw DIB bytes if decoding fails.
func imagePixelHash(dib []byte) []byte {
	if len(dib) < 40 {
		return dib
	}
	bmpData := prependBMPHeader(dib)
	img, _, err := image.Decode(bytes.NewReader(bmpData))
	if err != nil {
		return dib
	}
	bounds := img.Bounds()
	rgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba.Pix
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

// readClipboardHDROP parses the CF_HDROP (DROPFILES) format and returns
// a newline-separated list of file paths.
func readClipboardHDROP() string {
	hData := win.GetClipboardData(win.CF_HDROP)
	if hData == 0 {
		return ""
	}
	hMem := win.HGLOBAL(hData)
	size, _, _ := procGlobalSize.Call(uintptr(hMem))
	if size == 0 || size < 20 { // DROPFILES struct is at least 20 bytes
		return ""
	}
	ptr := win.GlobalLock(hMem)
	if ptr == nil {
		return ""
	}
	defer win.GlobalUnlock(hMem)

	raw := unsafe.Slice((*byte)(ptr), size)

	// DROPFILES: offset 16 is fWide (BOOL, 4 bytes in 64-bit).
	// struct layout: pFiles(uint32=4), pt(POINT=8), fNC(uint32=4), fWide(uint32=4) = 20 bytes
	pFiles := binary.LittleEndian.Uint32(raw[0:4])
	fWide := binary.LittleEndian.Uint32(raw[16:20])

	if pFiles >= uint32(size) {
		return ""
	}

	paths := raw[pFiles:]
	if fWide != 0 {
		// Unicode paths: each is uint16 array, double-null terminated.
		var result []string
		for i := 0; i < len(paths)-1; i += 2 {
			if paths[i] == 0 && paths[i+1] == 0 {
				break
			}
			// Find end of this string (null terminator).
			end := i
			for end+1 < len(paths) && !(paths[end] == 0 && paths[end+1] == 0) {
				end += 2
			}
			u16 := make([]uint16, (end-i)/2)
			for j := range u16 {
				u16[j] = binary.LittleEndian.Uint16(paths[i+j*2 : i+j*2+2])
			}
			result = append(result, string(utf16.Decode(u16)))
			i = end + 2 // skip past the null terminator
		}
		return strings.Join(result, "\n")
	} else {
		// ANSI paths: each is byte array, double-null terminated.
		var result []string
		start := 0
		for start < len(paths) {
			if paths[start] == 0 {
				break
			}
			end := start
			for end < len(paths) && paths[end] != 0 {
				end++
			}
			result = append(result, string(paths[start:end]))
			start = end + 1
		}
		return strings.Join(result, "\n")
	}
}

// WriteFilePaths writes file paths to the system clipboard as CF_HDROP format.
func WriteFilePaths(paths []string) bool {
	// Build CF_HDROP data: DROPFILES struct + Unicode file list.
	const dropFilesSize = 20
	var fileList []uint16
	for _, p := range paths {
		u16 := utf16.Encode([]rune(p))
		fileList = append(fileList, u16...)
		fileList = append(fileList, 0) // null terminator
	}
	fileList = append(fileList, 0) // double-null

	dataSize := dropFilesSize + len(fileList)*2
	hMem := win.GlobalAlloc(gmemMoveable, uintptr(dataSize))
	if hMem == 0 {
		return false
	}

	ptr := win.GlobalLock(hMem)
	if ptr == nil {
		win.GlobalFree(hMem)
		return false
	}

	dst := unsafe.Slice((*byte)(ptr), dataSize)
	// DROPFILES header.
	binary.LittleEndian.PutUint32(dst[0:4], dropFilesSize) // pFiles
	// pt (8 bytes) — zero
	// fNC (4 bytes) — zero
	binary.LittleEndian.PutUint32(dst[16:20], 1) // fWide = TRUE

	// File list.
	for i, v := range fileList {
		binary.LittleEndian.PutUint16(dst[dropFilesSize+i*2:], v)
	}

	win.GlobalUnlock(hMem)

	if !win.OpenClipboard(0) {
		win.GlobalFree(hMem)
		return false
	}
	defer win.CloseClipboard()

	win.EmptyClipboard()
	win.SetClipboardData(win.CF_HDROP, win.HANDLE(hMem))
	log.Printf("[clipboard] WriteFilePaths: wrote %d paths to CF_HDROP", len(paths))
	return true
}

func WriteText(text string) bool {
	MarkSelfWrite(text)
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
