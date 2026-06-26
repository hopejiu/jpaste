package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"jpaste/internal/clipboard"
	"jpaste/internal/db"
	"jpaste/internal/fileop"
	"jpaste/internal/filostack"
	"jpaste/internal/history"
	"jpaste/internal/hotkey"
	applog "jpaste/internal/log"
	"jpaste/internal/model"
	"jpaste/internal/settings"
	"jpaste/internal/toast"
	"jpaste/internal/viewers"

	"github.com/wailsapp/wails/v3/pkg/application"
	wailsEvent "github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed logo.png
var trayIcon []byte

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

func (c clipboardImpl) SetText(text string) bool     { return clipboard.WriteText(text) }
func (c clipboardImpl) SetImage(dib []byte) bool     { return clipboard.WriteImage(dib) }
func (c clipboardImpl) SetFiles(paths []string) bool { return clipboard.WriteFilePaths(paths) }

func main() {
	appData := filepath.Join(os.Getenv("APPDATA"), "jPaste")
	if err := applog.Init(appData); err != nil {
		fmt.Fprintf(os.Stderr, "init logging: %v\n", err)
	}
	var win application.Window
	cleanupOrphanedWV2()
	if err := os.MkdirAll(appData, 0700); err != nil {
		applog.Error("create app data dir", "error", err)
		os.Exit(1)
	}

	// Bootstrap: storage + settings.
	conn := must(db.Open(appData))
	defer conn.Close()

	repo := db.NewRepository(conn)

	sett := settings.NewService(appData)
	if err := sett.Load(); err != nil {
		applog.Warn("load settings", "error", err)
	}

	handle := &AppHandle{}
	doPaste := func() {}

	// Image store for clipboard images.
	imageStore := history.NewImageStore(appData)

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
	entryStore := history.NewSQLiteStore(repo)
	histSvc := history.NewService(entryStore, clipboardImpl{},
		history.WithPasteFunc(func() { doPaste() }),
		history.WithEmitFunc(func(name string, data any) { handle.Emit(name, data) }),
		// Notification is handled in the watcher callback after Push, so count is correct.
		history.WithNotifyFunc(func(title, msg string) {}),
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

	jsonViewerSvc := viewers.NewJSONViewerService(func(path string) {
		if createJsonWindowFn != nil {
			createJsonWindowFn(path, "JSON 查看")
		}
	})

	imageViewerSvc := viewers.NewImageViewerService(func(path string) {
		if createJsonWindowFn != nil {
			createJsonWindowFn(path, "图片查看")
		}
	})

	curlViewerSvc := viewers.NewCurlViewerService(func(path string) {
		if createJsonWindowFn != nil {
			createJsonWindowFn(path, "HTTP 调试")
		}
	})

	wssViewerSvc := viewers.NewWsViewerService(func(path string) {
		if createJsonWindowFn != nil {
			createJsonWindowFn(path, "WS 调试")
		}
	})

	// Watcher — event-driven via WM_CLIPBOARDUPDATE.
	watcher := clipboard.NewWatcher(newWatcherHandler(histSvc, filoStack, sett, toastSvc))

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
					case "/image-view", "/json-view", "/settings", "/toast", "/curl-view", "/ws-view":
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
			application.NewService(jsonViewerSvc),
			application.NewService(imageViewerSvc),
			application.NewService(curlViewerSvc),
			application.NewService(wssViewerSvc),
			application.NewService(toastSvc),
			application.NewService(filoStack),
			application.NewService(&Pinner{}),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.jpaste.app",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				applog.Info("second instance launched, activating existing window")
				application.InvokeSync(func() {
					if win != nil {
						showWindow(win)
					}
				})
			},
		},
	})

	handle.Wire(app)

	// Pre-create a single toast window. Created hidden, then positioned offscreen
	// and shown. This avoids the startup center-of-screen flash while keeping
	// WebView2 continuously rendering (important — hidden windows freeze WebView2).
	applog.Info("toast: pre-creating window")
	toastOpts := application.WebviewWindowOptions{
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
		BackgroundColour:           application.NewRGBA(0, 0, 0, 0),
		BackgroundType:             application.BackgroundTypeTransparent,
		DefaultContextMenuDisabled: true,
		URL:                        "/toast",
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
			HiddenOnTaskbar:                   true,
			DisableIcon:                       true,
			Theme:                             application.Light,
			BackdropType:                      application.None,
			GeneralAutofillEnabled:            false,
			PasswordAutosaveEnabled:           false,
			ExStyle: int(w32.WS_EX_CONTROLPARENT | w32.WS_EX_TRANSPARENT |
				w32.WS_EX_NOREDIRECTIONBITMAP | w32.WS_EX_TOPMOST | w32.WS_EX_TOOLWINDOW),
		},
		CloseButtonState: application.ButtonHidden,
	}
	toastWin := app.Window.NewWithOptions(toastOpts)
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
			// WebView2 进程已崩溃，自动重建窗口。
			applog.Warn("toast: native window handle is zero, recreating")
			newWin := app.Window.NewWithOptions(toastOpts)
			if newWin == nil {
				applog.Error("toast: window recreation failed: NewWithOptions returned nil")
				return
			}
			newWin.SetPosition(-9999, -9999)
			newWin.Show()
			toastWin = newWin
			hwnd = w32.HWND(toastWin.NativeWindow())
			if hwnd == 0 {
				applog.Error("toast: recreated window still has zero handle")
				return
			}
			applog.Info("toast: window recreated successfully")
		}

		// Force popup style — Wails creates with WS_OVERLAPPEDWINDOW which
		// leaves a classic frame; convert to WS_POPUP to remove it.
		w32.SetWindowLong(hwnd, w32.GWL_STYLE, w32.WS_POPUP|w32.WS_VISIBLE)
		w32.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
			w32.SWP_FRAMECHANGED|w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)

		// Recalculate position (handles monitor / DPI changes).
		prevHwnd := w32.GetForegroundWindow()
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

		toastWin.SetPosition(targetX, targetY)
		toastWin.Show()

		// Restore focus to whatever the user was working on.
		if prevHwnd != 0 && prevHwnd != hwnd {
			w32.SetForegroundWindow(prevHwnd)
		}
		applog.Debug("toast: showToastWindow done",
			"hwnd", hwnd, "pos", fmt.Sprintf("%d,%d", targetX, targetY),
			"prevHwnd", prevHwnd)
	}

	// hideToastOffscreen moves the window offscreen instead of hiding it,
	// because WebView2 stops rendering on hidden windows, causing a white
	// flash when re-shown.
	hideToastOffscreen := func() {
		toastWin.SetPosition(-9999, -9999)
	}

	// Window is always visible (WebView2 must keep rendering) but positioned
	// offscreen at (-9999,-9999) when not active. On a notification:
	//   1. emit event → frontend renders (async)
	//   2. 30ms later → position to bottom-right of primary monitor
	//   3. 3s (real) / 5s (preview) later → position back offscreen
	// This avoids the WebView2 rendering freeze that occurs on hidden windows.
	toastEmit = func(name string, data any) {
		// Handle hide-preview separately — immediate offscreen, no timer.
		if name == model.ToastHidePreview {
			hideToastOffscreen()
			return
		}

		// Enrich ToastData with theme and opacity.
		isPreview := false
		if td, ok := data.(toast.ToastData); ok {
			settings := sett.GetSettings()
			td.Theme = settings.Theme
			isPreview = td.IsPreview
			if !isPreview {
				// Real notification: inject opacity from saved settings.
				td.Opacity = float64(settings.NotifyOpacity) / 100.0
			}
			data = td
		}

		applog.Debug("toast: Emit", "name", name, "isPreview", isPreview, "opacity", 0)

		// Emit to frontend (async delivery via IPC).
		handle.Emit(name, data)

		// After a brief delay, show the window and start the auto-hide timer.
		// The delay is long enough for React to render, short enough to feel instant.
		time.AfterFunc(30*time.Millisecond, func() {
			application.InvokeSync(func() {
				showToastWindow()

				// Reset auto-hide timer on main thread to avoid races.
				if toastHideTimer != nil {
					toastHideTimer.Stop()
				}
				hideDelay := 3 * time.Second
				if isPreview {
					hideDelay = 5 * time.Second
				}
				toastHideTimer = time.AfterFunc(hideDelay, func() {
					application.InvokeSync(func() {
						hideToastOffscreen()
					})
				})
			})
		})
	}

	// Frontend log relay — listen for Events.Emit('frontend-log', ...) from JS.
	app.Event.On(model.FrontendLog, func(event *application.CustomEvent) {
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

	// Preview toast control — frontend emits these from SettingsPage.
	app.Event.On("toast-show-preview", func(event *application.CustomEvent) {
		data, ok := event.Data.(map[string]any)
		if !ok {
			return
		}
		title, _ := data["title"].(string)
		if title == "" {
			title = "jPaste"
		}
		message, _ := data["message"].(string)
		opacity := 100.0
		if o, ok := data["opacity"].(float64); ok {
			opacity = o
		}

		td := toast.ToastData{
			Title:     title,
			Message:   message,
			Opacity:   opacity / 100.0,
			IsPreview: true,
		}
		toastEmit(model.ToastNotification, td)
	})

	app.Event.On("toast-hide-preview", func(event *application.CustomEvent) {
		toastEmit(model.ToastHidePreview, nil)
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

	win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "jPaste", Width: 480, MinWidth: 360, Height: 560, MinHeight: 300,
		Hidden: false, URL: "/",
		Frameless:                  true,
		BackgroundColour:           application.NewRGB(248, 250, 252),
		DefaultContextMenuDisabled: true,
	})

	doPaste = func() {
		if win.IsVisible() {
			win.EmitEvent(model.WindowHiding, nil)
		}
		win.Hide()
		time.Sleep(50 * time.Millisecond) // wait for WebView2 to tear down before Paste()
		filoStack.SetSelfPaste()
		clipboard.Paste()
	}

	setupSystemTray(app, win)
	hkSvc := hotkey.NewService()
	setupGlobalHotkey(win, hkSvc, sett)

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
		if pinned {
			return
		}
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
				handle.Emit(model.PasteOrderChanged, new.PasteOrder)
			}
		}
	})

	defer hkSvc.UnregisterAll()
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
		win.EmitEvent(model.Navigate, "/settings")
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

func setupGlobalHotkey(win application.Window, hkSvc *hotkey.Service, sett *settings.Service) {
	toggle := func() {
		if win.IsVisible() {
			applog.Info("hotkey: hiding")
			hideWindow(win)
		} else {
			applog.Info("hotkey: showing")
			win.EmitEvent(model.Navigate, "/")
			showWindow(win)
		}
	}

	// Initial registration (fire-and-forget at startup).
	keystr := sett.GetSettings().Hotkey
	applog.Info("setting up global hotkey", "key", keystr)
	if err := hkSvc.Register(keystr, toggle); err != nil {
		applog.Warn("register global hotkey", "key", keystr, "error", err)
	} else {
		applog.Info("global hotkey registered", "key", keystr)
	}

	// On change: try new first, swap on success, return error on failure.
	sett.OnHotkeyChange(func(_, newK string) error {
		applog.Info("hotkey change requested", "key", newK)
		if err := hkSvc.RegisterAndSwap(newK, toggle); err != nil {
			applog.Warn("register global hotkey", "key", newK, "error", err)
			return err
		}
		applog.Info("global hotkey registered", "key", newK)
		return nil
	})
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
	wasVisible := win.IsVisible()
	applog.Info("showWindow", "wasVisible", wasVisible)
	clipboard.CaptureForeground()
	win.Center()
	win.Show()
	win.Focus()
	win.EmitEvent(model.WindowShown, nil)
	applog.Debug("showWindow: done", "nowVisible", win.IsVisible())
}

func hideWindow(win application.Window) {
	if win == nil {
		applog.Info("hideWindow: win is nil")
		return
	}
	wasVisible := win.IsVisible()
	// Note: we do NOT check win.IsVisible() here — Wails' internal visibility
	// state can be stale during async window operations. Always attempt Hide();
	// hiding an already-hidden window is a safe no-op.
	applog.Info("hideWindow", "wasVisible", wasVisible)
	win.EmitEvent(model.WindowHiding, nil)
	// Move offscreen BEFORE hiding to prevent the WebView2 teardown flash
	// (transparent/semi-transparent frame that Windows shows momentarily).
	win.SetPosition(-9999, -9999)
	win.Hide()
	applog.Debug("hideWindow: done", "nowVisible", win.IsVisible())
}

func must[T any](val T, err error) T {
	if err != nil {
		applog.Error("fatal", "error", err)
		os.Exit(1)
	}
	return val
}
