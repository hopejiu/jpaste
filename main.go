package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log"
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

// AppHandle bundles app-level dependencies that services need.
type AppHandle struct{ app *application.App }

func (h *AppHandle) Emit(name string, data any) { h.app.Event.Emit(name, data) }
func (h *AppHandle) Wire(a *application.App)    { h.app = a }

// clipboardImpl delegates to clipboard package functions.
type clipboardImpl struct{}

func (c clipboardImpl) SetText(text string) bool   { return clipboard.WriteText(text) }
func (c clipboardImpl) SetImage(dib []byte) bool    { return clipboard.WriteImage(dib) }
func (c clipboardImpl) SetFiles(paths []string) bool { return clipboard.WriteFilePaths(paths) }

func main() {
	appData := filepath.Join(os.Getenv("APPDATA"), "jPaste")
	if !acquireLock(appData) {
		log.Println("another instance is already running, exiting")
		return
	}
	defer releaseLock()
	if err := os.MkdirAll(appData, 0700); err != nil {
		log.Fatalf("create app data dir: %v", err)
	}

	// Bootstrap: storage + settings.
	conn := must(db.Open(appData))
	defer conn.Close()

	sett := settings.NewService(appData)
	if err := sett.Load(); err != nil {
		log.Printf("warn: load settings: %v", err)
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
		var c string
		// Try CF_UNICODETEXT first, then CF_HDROP for file entries.
		err := conn.QueryRow(
			`SELECT COALESCE(f.content, '') FROM clipboard_format f WHERE f.entry_id = ? AND f.format_type = 13`,
			id,
		).Scan(&c)
		if err != nil {
			err = conn.QueryRow(
				`SELECT COALESCE(f.content, '') FROM clipboard_format f WHERE f.entry_id = ? AND f.format_type = 15`,
				id,
			).Scan(&c)
		}
		return c, err
	}, fileop.WithOpenFileManager(func(path string, selectFile bool) error {
		if openFileManagerFn == nil {
			return fmt.Errorf("file manager not wired")
		}
		return openFileManagerFn(path, selectFile)
	}))

	// Pull remote settings on startup.
	if remoteSettings, err := syncSvc.PullSettings(); err == nil && remoteSettings != nil {
		var remote settings.Data
		if err := json.Unmarshal(remoteSettings, &remote); err == nil {
			localJSON, _ := json.Marshal(sett.GetSettings())
			if !bytes.Equal(remoteSettings, localJSON) {
				sett.SaveSettings(remote)
				log.Printf("sync: applied remote settings on startup (retain=%d)", remote.RetainDays)
			} else {
				log.Println("sync: remote settings identical, skipped")
			}
		}
	} else if err != nil {
		log.Printf("sync: pull settings startup: %v", err)
	}

	// Watcher — event-driven via WM_CLIPBOARDUPDATE.
	watcher := clipboard.NewWatcher(func(data clipboard.CapturedData) {
		log.Printf("[main] Capture callback: formats=%d hash=%s source=%q", len(data.Formats), data.PrimaryHash[:12], data.SourceEXE)
		entry, isNew := histSvc.CaptureEntry(data)
		if isNew {
			log.Printf("[main] new clipboard entry: id=%d text=%q source=%s", entry.ID, previewText(entry.Content), entry.SourceEXE)
		} else {
			log.Printf("[main] dedup entry (hash=%s)", data.PrimaryHash[:12])
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
		},
	})

	handle.Wire(app)

	openFileManagerFn = func(path string, selectFile bool) error {
		return app.Env.OpenFileManager(path, selectFile)
	}

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "jPaste", Width: 480, MinWidth: 360, Height: 560, MinHeight: 300,
		Hidden: sett.GetSettings().StartMinimized, URL: "/",
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
		hideWindow(win)
		e.Cancel()
	})
	win.OnWindowEvent(wailsEvent.Common.WindowLostFocus, func(e *application.WindowEvent) {
		hideWindow(win)
	})

	runCleanup(histSvc, sett)
	if !sett.GetSettings().StartMinimized {
		showWindow(win)
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
				log.Printf("warn: cleanup on retain change: %v", err)
			} else if n > 0 {
				log.Printf("cleaned up %d old entries (retain_days changed to %d)", n, new.RetainDays)
			}
		}
		if data, err := json.Marshal(new); err == nil {
			syncSvc.PushSettings(data)
		}
	})

	defer hotkey.UnregisterAll()
	defer notify.Shutdown()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// --- Phase helpers ---

func setupSystemTray(app *application.App, win application.Window) {
	tray := app.SystemTray.New()
	tray.SetLabel("jPaste")
	tray.SetIcon(trayIcon)
	menu := app.Menu.New()
	menu.Add("显示").OnClick(func(ctx *application.Context) { showWindow(win) })
	menu.Add("设置").OnClick(func(ctx *application.Context) {
		win.EmitEvent(events.Navigate, "/settings")
		showWindow(win)
	})
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(ctx *application.Context) { app.Quit() })
	tray.SetMenu(menu)
	tray.OnClick(func() {
		if win.IsVisible() {
			hideWindow(win)
		} else {
			showWindow(win)
		}
	})
	tray.AttachWindow(win)
}

func setupGlobalHotkey(win application.Window, sett *settings.Service) {
	toggle := func() {
		if win.IsVisible() {
			hideWindow(win)
		} else {
			win.EmitEvent(events.Navigate, "/")
			showWindow(win)
		}
	}
	register := func(keystr string) {
		log.Printf("[main] Setting up global hotkey: %s", keystr)
		hotkey.UnregisterAll()
		if err := hotkey.Register(keystr, toggle); err != nil {
			log.Printf("[main] WARN: register global hotkey %q: %v", keystr, err)
		} else {
			log.Printf("[main] global hotkey registered: %s", keystr)
		}
	}
	register(sett.GetSettings().Hotkey)
	sett.OnHotkeyChange(func(_, newK string) { register(newK) })
}

func runCleanup(histSvc *history.Service, sett *settings.Service) {
	cfg := sett.GetSettings()
	if n, err := histSvc.Cleanup(cfg.RetainDays); err != nil {
		log.Printf("warn: cleanup: %v", err)
	} else if n > 0 {
		log.Printf("cleaned up %d old entries", n)
	}
}

func setupAutostart(app *application.App, sett *settings.Service) {
	if sett.GetSettings().AutoStart {
		if err := app.Autostart.Enable(); err != nil {
			log.Printf("warn: autostart: %v", err)
		}
	}
}

func showWindow(win application.Window) {
	if win == nil {
		return
	}
	clipboard.CaptureForeground()
	win.Center()
	win.Show()
	win.Focus()
	win.EmitEvent(events.WindowShown, nil)
}

func hideWindow(win application.Window) {
	if win == nil || !win.IsVisible() {
		return
	}
	win.EmitEvent(events.WindowHiding, nil)
	time.AfterFunc(200*time.Millisecond, func() { win.Hide() })
}

func must[T any](val T, err error) T {
	if err != nil {
		log.Fatal(err)
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
		log.Printf("warn: write lock file: %v", writeErr)
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
