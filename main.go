package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"jpaste/internal/clipboard"
	"jpaste/internal/db"
	"jpaste/internal/events"
	"jpaste/internal/fileop"
	"jpaste/internal/history"
	"jpaste/internal/hotkey"
	"jpaste/internal/imageviewer"
	"jpaste/internal/jsonviewer"
	applog "jpaste/internal/log"
	"jpaste/internal/notify"
	"jpaste/internal/settings"
	"jpaste/internal/sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	wailsEvent "github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed paste.png
var trayIcon []byte

var lockFilePath string
var quitting bool

// AppHandle bundles app-level dependencies that services need.
type AppHandle struct{ app *application.App }

func (h *AppHandle) Emit(name string, data any) { h.app.Event.Emit(name, data) }
func (h *AppHandle) Wire(a *application.App)    { h.app = a }

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
	syncSvc := sync.NewService(appData, conn, sett, func(name string, data any) {
		if handle.app != nil {
			handle.Emit(name, data)
		}
	})

	// History service with capture pipeline hooks.
	entryStore := history.NewSQLiteStore(conn)
	histSvc := history.NewService(entryStore, clipboardImpl{},
		history.WithPasteFunc(func() { doPaste() }),
		history.WithEmitFunc(func(name string, data any) { handle.Emit(name, data) }),
		history.WithNotifyFunc(func(title, msg string) {
			if sett.GetSettings().NotifyEnabled {
				notify.ShowToast(title, msg)
			}
		}),
		history.WithSyncPushFunc(func(hash string, formats []history.SyncFormat) {
			syncSvc.PushEntry(sync.PushInput{ContentHash: hash, Formats: formats})
		}),
		history.WithImageStore(imageStore),
	)

	// File-manager function (wired after app creation).
	var openFileManagerFn func(string, bool) error
	fileSvc := fileop.NewService(func(id int64) (string, error) {
		c, err := entryStore.QueryFormatContent(id, clipboard.CF_UNICODETEXT)
		if err != nil || c == "" {
			c, err = entryStore.QueryFormatContent(id, clipboard.CF_HDROP)
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
	watcher := clipboard.NewWatcher(func(data clipboard.CapturedData) {
		applog.Info("capture callback", "formats", len(data.Formats), "hash", data.PrimaryHash[:12], "source", data.SourceEXE)
		entry, isNew := histSvc.CaptureEntry(data)
		if isNew {
			applog.Info("new clipboard entry", "id", entry.ID, "text", previewText(entry.Content), "source", entry.SourceEXE)
		} else {
			applog.Info("dedup entry", "hash", data.PrimaryHash[:12])
		}
	})

	// Create Wails app.
	app := application.New(application.Options{
		Name:        "jPaste",
		Description: "A modern clipboard manager for Windows",
		Assets:      application.AssetOptions{Handler: application.BundledAssetFileServer(assets)},
		Services: []application.Service{
			application.NewService(watcher),
			application.NewService(histSvc),
			application.NewService(sett),
			application.NewService(fileSvc),
			application.NewService(syncSvc),
			application.NewService(jsonViewerSvc),
			application.NewService(imageViewerSvc),
		},
	})

	handle.Wire(app)

	openFileManagerFn = func(path string, selectFile bool) error {
		return app.Env.OpenFileManager(path, selectFile)
	}

	createJsonWindowFn = func(path, title string) {
		applog.Info("secondary window", "title", title)
		win := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:            title,
			Width:            1200,
			Height:           800,
			MinWidth:         600,
			MinHeight:        400,
			URL:              path,
			BackgroundColour: application.NewRGB(248, 250, 252),
		})
		win.Show()
		applog.Info("secondary window shown", "title", title)
	}

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "jPaste", Width: 480, MinWidth: 360, Height: 560, MinHeight: 300,
		Hidden: false, URL: "/",
		BackgroundColour: application.NewRGB(248, 250, 252),
	})

	doPaste = func() {
		if win.IsVisible() {
			win.EmitEvent(events.WindowHiding, nil)
		}
		time.Sleep(200 * time.Millisecond)
		win.Hide()
		time.Sleep(50 * time.Millisecond)
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
		if data, err := json.Marshal(new); err == nil {
			syncSvc.PushSettings(data)
		}
	})

	defer hotkey.UnregisterAll()
	defer notify.Shutdown()
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
	register := func(keystr string) {
		applog.Info("setting up global hotkey", "key", keystr)
		hotkey.UnregisterAll()
		if err := hotkey.Register(keystr, toggle); err != nil {
			applog.Warn("register global hotkey", "key", keystr, "error", err)
		} else {
			applog.Info("global hotkey registered", "key", keystr)
		}
	}
	register(sett.GetSettings().Hotkey)
	sett.OnHotkeyChange(func(_, newK string) { register(newK) })
}

func runCleanup(histSvc *history.Service, sett *settings.Service) {
	cfg := sett.GetSettings()
	if n, err := histSvc.Cleanup(cfg.RetainDays); err != nil {
		applog.Warn("cleanup", "error", err)
	} else if n > 0 {
		applog.Info("cleaned up old entries", "count", n)
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
	if !win.IsVisible() {
		applog.Info("hideWindow: already hidden")
		return
	}
	applog.Info("hideWindow: hiding")
	win.EmitEvent(events.WindowHiding, nil)
	time.AfterFunc(200*time.Millisecond, func() { win.Hide() })
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
