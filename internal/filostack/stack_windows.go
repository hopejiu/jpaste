//go:build windows

package filostack

import (
	"log"
	"runtime"
	"syscall"
	"unsafe"
	"time"

	"github.com/lxn/win"
)

const (
	whKeyboardLL = 13
	wmKeydown    = 0x0100
	wmQuit       = 0x0012
	vkControl    = 0x11
	vkV          = 0x56
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procPostThreadMessage   = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
)

// KBDLLHOOKSTRUCT mirrors the Windows structure.
type KBDLLHOOKSTRUCT struct {
	VKCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
)

func init() {
	platformStartHook = startWindowsHook
}

func startWindowsHook(onVKeyDown func()) func() {
	log.Println("[filostack] Starting WH_KEYBOARD_LL hook...")

	done := make(chan struct{})
	tidCh := make(chan uintptr, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		tid, _, _ := procGetCurrentThreadId.Call()
		tidCh <- tid
		log.Printf("[filostack] hook thread id=%d", tid)

		hookCallback := syscall.NewCallback(func(nCode int32, wParam, lParam uintptr) uintptr {
			if nCode >= 0 && wParam == wmKeydown {
				kbd := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
				if kbd.VKCode == vkV {
					ctrlDown, _, _ := procGetAsyncKeyState.Call(vkControl)
					if ctrlDown&0x8000 != 0 {
						log.Println("[filostack] hook: Ctrl+V intercepted")
						onVKeyDown()
					}
				}
			}
			ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
			return ret
		})

		hhk, _, err := procSetWindowsHookEx.Call(
			whKeyboardLL,
			hookCallback,
			0, 0,
		)
		if hhk == 0 {
			log.Printf("[filostack] SetWindowsHookEx failed: %d", err)
			return
		}
		log.Printf("[filostack] WH_KEYBOARD_LL hook installed (hhk=%d)", hhk)

		var msg win.MSG
		for {
			ret := win.GetMessage(&msg, 0, 0, 0)
			if ret == 0 || ret == -1 {
				log.Printf("[filostack] hook message loop exit (ret=%d)", ret)
				break
			}
			win.TranslateMessage(&msg)
			win.DispatchMessage(&msg)
		}

		procUnhookWindowsHookEx.Call(hhk)
		log.Println("[filostack] WH_KEYBOARD_LL hook removed")
		close(done)
	}()

	tid := <-tidCh

	stop := func() {
		log.Println("[filostack] stopping hook...")
		procPostThreadMessage.Call(tid, wmQuit, 0, 0)
		select {
		case <-done:
			log.Println("[filostack] hook goroutine exited")
		case <-time.After(2 * time.Second):
			log.Println("[filostack] hook goroutine did not exit in 2s")
		}
	}

	return stop
}
