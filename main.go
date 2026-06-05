package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"jpaste/internal/clipboard"
	"jpaste/internal/db"
	"jpaste/internal/events"
	"jpaste/internal/filostack"
	"jpaste/internal/fileop"
	"jpaste/internal/history"
	"jpaste/internal/hotkey"
	"jpaste/internal/imageviewer"
	"jpaste/internal/jsonviewer"
	applog "jpaste/internal/log"
	"jpaste/internal/model"
	"jpaste/internal/settings"
	"jpaste/internal/sync"
	"jpaste/internal/toast"

	"github.com/wailsapp/wails/v3/pkg/application"
	wailsEvent "github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed paste.png
var trayIcon []byte

var lockFilePath string
var quitting bool
var pinned bool

// AppHandle bundles app-level dependencies that services need.
type AppHandle struct{ app *application.App }

func (h *AppHandle) Emit(name string, data any) { h.app.Event.Emit(name, data) }
func (h *AppHandle) Wire(a *application.App)    { h.app = a }

// Pinner controls whether the main window stays visible on focus loss.
type Pinner struct{}

func (p *Pinner) SetPinned(val bool) {
	pinned = val
	applog.Info("pin", "pinned", val)
}

func (p *Pinner) IsPinned() bool {
	return pinned
}

// clipboardImpl delegates to clipboard package functions.
type clipboardImpl struct{}

func (c clipboardImpl) SetText(text string) bool    { return clipboard.WriteText(text) }
func (c clipboardImpl) SetImage(dib []byte) bool     { return clipboard.WriteImage(dib) }
func (c clipboardImpl) SetFiles(paths []string) bool { return clipboard.WriteFilePaths(paths) }

func main() {
	appData := filepath.Join(os.Getenv("APPDATA"), "jPaste")
	if err := applog.Init(appData); err != nil {
		fmt.Fprintf(os.Stderr, "init logging: %v\n", err)
	}
	if !acquireLock(appData) {
		applog.Info("another instance is already running, exiting")
		return
	}
	defer releaseLock()
	cleanupOrphanedWV2()
	if err := os.MkdirAll(appData, 0700); err != nil {
		applog.Error("create app data dir", "error", err)
		os.Exit(1)
	}

	// Bootstrap: storage + settings.
	conn := must(db.Open(appData))
	defer conn.Close()

	sett := settings.NewService(appData)
	if err := sett.Load(); err != nil {
		applog.Warn("load settings", "error", err)
	}

	handle := &AppHandle{}
	doPaste := func() {}

	// Image store for clipboard images.
	imageStore := history.NewImageStore(appData)

	// Sync service (WebDAV).
	syncStore := sync.NewSQLSyncStore(conn)
	syncSvc := sync.NewService(appData, syncStore, sett, func(name string, data any) {
		if handle.app != nil {
			handle.Emit(name, data)
		}
	})

	// Helper: truncate text to at most n runes for display.
	previewShort := func(s string, n int) string {
		runes := []rune(s)
		if len(runes) > n {
			return string(runes[:n]) + "..."
		}
		return s
	}

	// Toast service — pre-created frameless window for clipboard notifications.
	var toastEmit func(name string, data any)
	toastSvc := toast.NewService(func(name string, data any) {
		if toastEmit != nil {
			toastEmit(name, data)
		}
	})

	notify := func(title, msg string) {
		if sett.GetSettings().NotifyEnabled {
			toastSvc.ShowToast(title, msg)
		}
	}

	// FILO clipboard stack service.
	filoStack := filostack.NewService(clipboard.WriteText,
		filostack.WithNotifyFunc(func(title, msg string) {
			notify(title, msg)
		}),
	)

	// History service with capture pipeline hooks.
	entryStore := history.NewSQLiteStore(conn)
	histSvc := history.NewService(entryStore, clipboardImpl{},
		history.WithPasteFunc(func() { doPaste() }),
		history.WithEmitFunc(func(name string, data any) { handle.Emit(name, data) }),
		// Notification is handled in the watcher callback after Push, so count is correct.
		history.WithNotifyFunc(func(title, msg string) {}),
		history.WithSyncPushFunc(func(hash string, formats []model.SyncFormat) {
			syncSvc.PushEntry(sync.PushInput{ContentHash: hash, Formats: formats})
		}),
		history.WithImageStore(imageStore),
	)

	// File-manager function (wired after app creation).
	var openFileManagerFn func(string, bool) error
	fileSvc := fileop.NewService(func(id int64) (string, error) {
		c, err := entryStore.QueryFormatContent(id, model.CF_UNICODETEXT)
		if err != nil || c == "" {
			c, err = entryStore.QueryFormatContent(id, model.CF_HDROP)
		}
		return c, err
	}, fileop.WithOpenFileManager(func(path string, selectFile bool) error {
		if openFileManagerFn == nil {
			return fmt.Errorf("file manager not wired")
		}
		return openFileManagerFn(path, selectFile)
	}))

	// JSON viewer — opens a separate window for structured JSON viewing.
	var createJsonWindowFn func(path, title string)

	jsonViewerSvc := jsonviewer.NewService(func(path string) {
		if createJsonWindowFn != nil {
			createJsonWindowFn(path, "JSON 查看")
		}
	})

	imageViewerSvc := imageviewer.NewService(func(path string) {
		if createJsonWindowFn != nil {
			createJsonWindowFn(path, "图片查看")
		}
	})

	// Pull remote settings on startup.
	if remoteSettings, err := syncSvc.PullSettings(); err == nil && remoteSettings != nil {
		var remote settings.Data
		if err := json.Unmarshal(remoteSettings, &remote); err == nil {
			localJSON, _ := json.Marshal(sett.GetSettings())
			if !bytes.Equal(remoteSettings, localJSON) {
				sett.SaveSettings(remote)
				applog.Info("sync: applied remote settings on startup", "retain", remote.RetainDays)
			} else {
				applog.Info("sync: remote settings identical, skipped")
			}
		}
	} else if err != nil {
		applog.Warn("sync: pull settings startup", "error", err)
	}

	// Watcher — event-driven via WM_CLIPBOARDUPDATE.
	watcher := clipboard.NewWatcher(func(data model.CapturedData) {
		applog.Info("capture callback", "formats", len(data.Formats), "hash", data.PrimaryHash[:12], "source", data.SourceEXE)
		entry, isNew := histSvc.CaptureEntry(data)
		if isNew {
			applog.Info("new clipboard entry", "id", entry.ID, "text", previewText(entry.Content), "source", entry.SourceEXE)
		} else {
			applog.Info("dedup entry", "hash", data.PrimaryHash[:12])
		}

		// Push text to FILO stack when stack mode is enabled.
		stackEnabled := filoStack.Enabled()
		isSelfWrite := clipboard.IsSelfWrite(data)
		var textToPush string
		for _, f := range data.Formats {
			if model.IsTextFormat(f.FormatType) && f.Text != "" {
				textToPush = f.Text
				break
			}
		}
		applog.Info("stack decision",
			"stackEnabled", stackEnabled,
			"isSelfWrite", isSelfWrite,
			"text", previewText(textToPush),
			"hash", data.PrimaryHash[:12],
		)
		if stackEnabled && !isSelfWrite && textToPush != "" {
			filoStack.Push(textToPush)
		}

		// When in stack/queue mode, detect non-text captures (images, files)
		// and auto-exit the mode with a toast.
		if stackEnabled && !isSelfWrite {
			hasNonText := false
			for _, f := range data.Formats {
				if model.IsImageFormat(f.FormatType) || model.IsHdropFormat(f.FormatType) {
					hasNonText = true
					break
				}
			}
			if hasNonText {
				modeLabel := map[string]string{filostack.ModeStack: "栈", filostack.ModeQueue: "队列"}[filoStack.Mode()]
				itemCount := filoStack.Len()

				newSettings := sett.GetSettings()
				newSettings.PasteOrder = "normal"
				sett.SaveSettings(newSettings)

				toastSvc.ShowToast("jPaste",
					fmt.Sprintf("检测到非文本内容，已退出%s模式（剩余 %d 项已清空）", modeLabel, itemCount))
			}
		}

		// Notify（无论是否新内容，重复也弹）。
		if sett.GetSettings().NotifyEnabled {
			// CaptureEntry 在去重时返回 nil entry，此时用 data.Formats 中的纯文本兜底。
			previewSource := textToPush
			if entry != nil {
				previewSource = entry.Content
			}
			contentPreview := previewShort(previewSource, 10)
			if filoStack.Enabled() {
				modeLabel := map[string]string{filostack.ModeStack: "栈", filostack.ModeQueue: "队列"}[filoStack.Mode()]
				toastSvc.ShowToast("jPaste",
					fmt.Sprintf("剪贴板写入: %s, 当前%s已有: %d 个", contentPreview, modeLabel, filoStack.Len()))
			} else {
				toastSvc.ShowToast("jPaste", "剪贴板写入: "+contentPreview)
			}
		}
	})

	// Create Wails app.
	app := application.New(application.Options{
		Name:        "jPaste",
		Description: "A modern clipboard manager for Windows",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					// SPA fallback: only rewrite known SPA routes to serve index.html.
					// Avoid rewriting other extension-less paths (e.g. /wails/*, Vite internal paths).
					switch req.URL.Path {
					case "/image-view", "/json-view", "/settings", "/toast":
						req.URL.Path = "/"
					}
					next.ServeHTTP(rw, req)
				})
			},
		},
		Services: []application.Service{
			application.NewService(watcher),
			application.NewService(histSvc),
			application.NewService(sett),
			application.NewService(fileSvc),
			application.NewService(syncSvc),
			application.NewService(jsonViewerSvc),
			application.NewService(imageViewerSvc),
			application.NewService(toastSvc),
			application.NewService(filoStack),
			application.NewService(&Pinner{}),
		},
	})

	handle.Wire(app)

	// Pre-create a single toast window. Created hidden, then positioned offscreen
	// and shown. This avoids the startup center-of-screen flash while keeping
	// WebView2 continuously rendering (important — hidden windows freeze WebView2).
	applog.Info("toast: pre-creating window")
	toastWin := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      "",
		Width:                      360,
		Height:                     80,
		MinWidth:                   360,
		MinHeight:                  80,
		MaxWidth:                   360,
		MaxHeight:                  80,
		Frameless:                  true,
		AlwaysOnTop:                true,
		DisableResize:              true,
		Hidden:                     true,
		IgnoreMouseEvents:          true,
		BackgroundColour:           application.NewRGB(255, 255, 255),
		DefaultContextMenuDisabled: true,
		URL:                        "/toast",
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
			HiddenOnTaskbar:                   true,
		},
	})
	// Position offscreen while still hidden, THEN show — no center flash,
	// and WebView2 starts rendering immediately at the offscreen location.
	toastWin.SetPosition(-9999, -9999)
	toastWin.Show()
	applog.Info("toast: window pre-created and offscreened")

	var toastHideTimer *time.Timer

	// showToastWindow positions the window at bottom-right of the primary monitor.
	// Called after the frontend has had time to receive the event and render.
	showToastWindow := func() {
		hwnd := w32.HWND(toastWin.NativeWindow())
		if hwnd == 0 {
			applog.Warn("toast: native window handle is zero")
			return
		}
		prevHwnd := w32.GetForegroundWindow()

		// Recalculate position (handles monitor / DPI changes).
		hMonitor := w32.MonitorFromPoint(0, 0, w32.MONITOR_DEFAULTTOPRIMARY)
		var mi w32.MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		if !w32.GetMonitorInfo(hMonitor, &mi) {
			return
		}
		workRect := mi.RcWork
		workW := int(workRect.Right - workRect.Left)
		workH := int(workRect.Bottom - workRect.Top)
		dpi := w32.GetDpiForWindow(hwnd)
		scale := float64(dpi) / 96.0
		padding := 10
		targetX := int(float64(workW)/scale) - 360 - padding
		targetY := int(float64(workH)/scale) - 80 - padding
		applog.Info("toast: positioning", "x", targetX, "y", targetY, "workW", workW, "workH", workH, "scale", scale)

		toastWin.SetPosition(targetX, targetY)

		// Ensure the window is visible after being offscreen.
		toastWin.Show()

		// Restore focus to whatever the user was working on.
		if prevHwnd != 0 && prevHwnd != hwnd {
			w32.SetForegroundWindow(prevHwnd)
		}
		style := uint32(w32.GetWindowLong(hwnd, w32.GWL_STYLE))
		applog.Info("toast style", "style", fmt.Sprintf("0x%08X", style))
	}

	// hideToastOffscreen moves the window offscreen instead of hiding it,
	// because WebView2 stops rendering on hidden windows, causing a white
	// flash when re-shown.
	hideToastOffscreen := func() {
		toastWin.SetPosition(-9999, -9999)
		applog.Info("toast: window offscreened")
	}

	// Window is always visible (WebView2 must keep rendering) but positioned
	// offscreen at (-9999,-9999) when not active. On a notification:
	//   1. emit event → frontend renders (async)
	//   2. 30ms later → position to bottom-right of primary monitor
	//   3. 3s later → position back offscreen
	// This avoids the WebView2 rendering freeze that occurs on hidden windows.
	toastEmit = func(name string, data any) {
		applog.Info("toast: emit start") // T0

		// Emit to frontend (async delivery via IPC).
		handle.Emit(name, data)
		applog.Info("toast: emit done, scheduling show") // T1

		// After a brief delay, show the window and start the auto-hide timer.
		// The delay is long enough for React to render, short enough to feel instant.
		time.AfterFunc(30*time.Millisecond, func() {
			applog.Info("toast: 30ms elapsed, invoking show") // T2
			application.InvokeSync(func() {
				applog.Info("toast: InvokeSync enter") // T3
				showToastWindow()
				applog.Info("toast: window shown, resetting timer") // T4

				// Reset auto-hide timer on main thread to avoid races.
				if toastHideTimer != nil {
					toastHideTimer.Stop()
				}
				toastHideTimer = time.AfterFunc(3*time.Second, func() {
					application.InvokeSync(func() {
						hideToastOffscreen()
					})
				})
				applog.Info("toast: timer reset done") // T5
			})
		})
	}

	// Frontend log relay — listen for Events.Emit('frontend-log', ...) from JS.
	app.Event.On(events.FrontendLog, func(event *application.CustomEvent) {
		data, ok := event.Data.(map[string]any)
		if !ok {
			return
		}
		component, _ := data["component"].(string)
		msg, _ := data["msg"].(string)
		level, _ := data["level"].(string)
		switch level {
		case "debug":
			applog.Debug(msg, "component", component)
		case "warn":
			applog.Warn(msg, "component", component)
		case "error":
			applog.Error(msg, "component", component)
		default:
			applog.Info(msg, "component", component)
		}
	})

	// F12 打开开发者工具（调试用，生产构建也需要保留以便排查问题）。
	app.KeyBinding.Add("F12", func(window application.Window) {
		applog.Info("F12 pressed, opening DevTools")
		window.OpenDevTools()
	})

	openFileManagerFn = func(path string, selectFile bool) error {
		return app.Env.OpenFileManager(path, selectFile)
	}

	createJsonWindowFn = func(path, title string) {
		applog.Info("secondary window", "title", title)
		win := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:                      title,
			Width:                      1200,
			Height:                     800,
			MinWidth:                   600,
			MinHeight:                  400,
			URL:                        path,
			BackgroundColour:           application.NewRGB(248, 250, 252),
			DefaultContextMenuDisabled: true,
		})
		win.Show()
		applog.Info("secondary window shown", "title", title)
	}

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "jPaste", Width: 480, MinWidth: 360, Height: 560, MinHeight: 300,
		Hidden: false, URL: "/",
		BackgroundColour:           application.NewRGB(248, 250, 252),
		DefaultContextMenuDisabled: true,
	})

	doPaste = func() {
		if win.IsVisible() {
			win.EmitEvent(events.WindowHiding, nil)
		}
		win.Hide()
		time.Sleep(50 * time.Millisecond) // wait for WebView2 to tear down before Paste()
		filoStack.SetSelfPaste()
		clipboard.Paste()
	}

	setupSystemTray(app, win)
	setupGlobalHotkey(win, sett)

	win.RegisterHook(wailsEvent.Common.WindowClosing, func(e *application.WindowEvent) {
		if quitting {
			applog.Info("WindowClosing: quitting=true, allowing close")
			return
		}
		applog.Info("WindowClosing: hiding to tray")
		hideWindow(win)
		e.Cancel()
	})
	win.OnWindowEvent(wailsEvent.Common.WindowLostFocus, func(e *application.WindowEvent) {
		if pinned { return }
		applog.Info("WindowLostFocus: hiding")
		hideWindow(win)
	})

	runCleanup(histSvc, sett)
	startMinimized := sett.GetSettings().StartMinimized
	applog.Info("startup", "start_minimized", startMinimized)
	if startMinimized {
		time.AfterFunc(500*time.Millisecond, func() {
			applog.Info("start_minimized: hiding window")
			hideWindow(win)
		})
	}
	setupAutostart(app, sett)

	sett.OnSettingsChange(func(old, new settings.Data) {
		if new.AutoStart {
			app.Autostart.Enable()
		} else {
			app.Autostart.Disable()
		}
		if old.RetainDays != new.RetainDays {
			if n, err := histSvc.Cleanup(new.RetainDays); err != nil {
				applog.Warn("cleanup on retain change", "error", err)
			} else if n > 0 {
				applog.Info("cleaned up old entries", "count", n, "retain_days", new.RetainDays)
			}
		}
		if old.PasteOrder != new.PasteOrder {
			applog.Info("paste order change", "order", new.PasteOrder)
			filoStack.SetMode(new.PasteOrder)
			if handle.app != nil {
				handle.Emit(events.PasteOrderChanged, new.PasteOrder)
			}
		}
		if data, err := json.Marshal(new); err == nil {
			syncSvc.PushSettings(data)
		}
	})

	defer hotkey.UnregisterAll()
	defer filoStack.ServiceShutdown()

	// Activate paste order if stored from a previous session.
	if order := sett.GetSettings().PasteOrder; order != "normal" {
		applog.Info("initial paste order from saved settings", "order", order)
		filoStack.SetMode(order)
	}

	if err := app.Run(); err != nil {
		applog.Error("app.Run", "error", err)
		os.Exit(1)
	}
	applog.Info("app.Run returned, process exiting")
}

// --- Phase helpers ---

func setupSystemTray(app *application.App, win application.Window) {
	tray := app.SystemTray.New()
	tray.SetLabel("jPaste")
	tray.SetIcon(trayIcon)
	menu := app.Menu.New()
	menu.Add("显示").OnClick(func(ctx *application.Context) {
		applog.Info("tray: 显示")
		showWindow(win)
	})
	menu.Add("设置").OnClick(func(ctx *application.Context) {
		applog.Info("tray: 设置")
		win.EmitEvent(events.Navigate, "/settings")
		showWindow(win)
	})
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(ctx *application.Context) {
		applog.Info("tray: 退出")
		quitting = true
		app.Quit()
	})
	tray.SetMenu(menu)
	tray.OnClick(func() {
		if win.IsVisible() {
			applog.Info("tray click: hiding")
			hideWindow(win)
		} else {
			applog.Info("tray click: showing")
			showWindow(win)
		}
	})
	tray.AttachWindow(win)
}

func setupGlobalHotkey(win application.Window, sett *settings.Service) {
	toggle := func() {
		if win.IsVisible() {
			applog.Info("hotkey: hiding")
			hideWindow(win)
		} else {
			applog.Info("hotkey: showing")
			win.EmitEvent(events.Navigate, "/")
			showWindow(win)
		}
	}

	// Initial registration (fire-and-forget at startup).
	keystr := sett.GetSettings().Hotkey
	applog.Info("setting up global hotkey", "key", keystr)
	if err := hotkey.Register(keystr, toggle); err != nil {
		applog.Warn("register global hotkey", "key", keystr, "error", err)
	} else {
		applog.Info("global hotkey registered", "key", keystr)
	}

	// On change: try new first, swap on success, return error on failure.
	sett.OnHotkeyChange(func(_, newK string) error {
		applog.Info("hotkey change requested", "key", newK)
		if err := hotkey.RegisterAndSwap(newK, toggle); err != nil {
			applog.Warn("register global hotkey", "key", newK, "error", err)
			return err
		}
		applog.Info("global hotkey registered", "key", newK)
		return nil
	})
}

func runCleanup(histSvc *history.Service, sett *settings.Service) {
	cfg := sett.GetSettings()
	if n, err := histSvc.Cleanup(cfg.RetainDays); err != nil {
		applog.Warn("cleanup", "error", err)
	} else if n > 0 {
		applog.Info("cleaned up old entries", "count", n)
	}

	// Clean up orphaned temp files from previous sessions.
	tmpDir := filepath.Join(os.Getenv("TEMP"), "jPaste")
	if entries, err := os.ReadDir(tmpDir); err == nil {
		cleaned := 0
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(tmpDir, e.Name())); err == nil {
				cleaned++
			}
		}
		if cleaned > 0 {
			applog.Info("cleaned up temp dir", "dir", tmpDir, "count", cleaned)
		}
	}
}

func setupAutostart(app *application.App, sett *settings.Service) {
	if sett.GetSettings().AutoStart {
		if err := app.Autostart.Enable(); err != nil {
			applog.Warn("autostart", "error", err)
		}
	}
}

func showWindow(win application.Window) {
	if win == nil {
		applog.Info("showWindow: win is nil")
		return
	}
	applog.Info("showWindow: capturing foreground + showing")
	clipboard.CaptureForeground()
	win.Center()
	win.Show()
	win.Focus()
	win.EmitEvent(events.WindowShown, nil)
}

func hideWindow(win application.Window) {
	if win == nil {
		applog.Info("hideWindow: win is nil")
		return
	}
	// Note: we do NOT check win.IsVisible() here — Wails' internal visibility
	// state can be stale during async window operations. Always attempt Hide();
	// hiding an already-hidden window is a safe no-op.
	applog.Info("hideWindow: hiding")
	win.EmitEvent(events.WindowHiding, nil)
	// Move offscreen BEFORE hiding to prevent the WebView2 teardown flash
	// (transparent/semi-transparent frame that Windows shows momentarily).
	win.SetPosition(-9999, -9999)
	win.Hide()
}

func must[T any](val T, err error) T {
	if err != nil {
		applog.Error("fatal", "error", err)
		os.Exit(1)
	}
	return val
}

func previewText(s string) string {
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}

// cleanupOrphanedWV2 kills orphaned msedgewebview2 processes — WebView2 child
// processes whose parent (previous jPaste instance) no longer exists.
func cleanupOrphanedWV2() {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer syscall.CloseHandle(snapshot)

	// Build: alive PID set, and process info map.
	alive := make(map[uint32]bool)
	type procInfo struct {
		name     string
		parentPid uint32
	}
	info := make(map[uint32]procInfo)

	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for ok := syscall.Process32First(snapshot, &pe); ok == nil; ok = syscall.Process32Next(snapshot, &pe) {
		pid := pe.ProcessID
		name := syscall.UTF16ToString(pe.ExeFile[:])
		alive[pid] = true
		info[pid] = procInfo{name: name, parentPid: pe.ParentProcessID}
	}

	selfPID := uint32(os.Getpid())

	for pid, pi := range info {
		if !strings.EqualFold(pi.name, "msedgewebview2.exe") {
			continue
		}
		if pi.parentPid == selfPID {
			continue // belongs to current instance
		}
		if alive[pi.parentPid] {
			continue // parent still alive — not orphaned
		}
		// Orphaned — terminate it.
		h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, pid)
		if err != nil {
			continue
		}
		syscall.TerminateProcess(h, 0)
		syscall.CloseHandle(h)
		applog.Info("cleaned orphaned WebView2 process", "pid", pid, "parent", pi.parentPid)
	}
}

func acquireLock(appData string) bool {
	lockFilePath = filepath.Join(appData, "instance.lock")
	os.MkdirAll(appData, 0700)
	data, err := os.ReadFile(lockFilePath)
	if err == nil {
		if pid, parseErr := strconv.Atoi(string(data)); parseErr == nil && pid > 0 && isProcessAlive(pid) {
			return false
		}
		os.Remove(lockFilePath)
	}
	pid := os.Getpid()
	if writeErr := os.WriteFile(lockFilePath, []byte(strconv.Itoa(pid)), 0600); writeErr != nil {
		applog.Warn("write lock file", "error", writeErr)
		return true
	}
	return true
}

func releaseLock() {
	if lockFilePath != "" {
		os.Remove(lockFilePath)
	}
}

func isProcessAlive(pid int) bool {
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	h, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var exitCode uint32
	err = syscall.GetExitCodeProcess(h, &exitCode)
	if err != nil {
		return false
	}
	return exitCode == 259
}
