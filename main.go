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

// lockFilePath is set by acquireSingleton(); cleaned on exit via releaseLockFile().
var lockFilePath string

// AppHandle bundles all app-level dependencies that services need.
// Wire() is called once after the Wails app is created.
type AppHandle struct{ app *application.App }

func (h *AppHandle) ReadClipboard() (string, bool) { return h.app.Clipboard.Text() }
func (h *AppHandle) SetText(s string) bool          { return h.app.Clipboard.SetText(s) }
func (h *AppHandle) Emit(name string, data any)    { h.app.Event.Emit(name, data) }
func (h *AppHandle) Wire(a *application.App)       { h.app = a }

func main() {
	// Single-instance guard using lock file + PID check.
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
	doPaste := func() {} // wired after window creation

	// Sync service (WebDAV) — declared first so histSvc closure can capture it.
	syncSvc := sync.NewService(appData, conn, sett, func(name string, data any) {
		if handle.app != nil {
			handle.Emit(name, data)
		}
	})

	// Build history service with capture pipeline hooks.
	histSvc := history.NewService(conn, handle,
		history.WithPasteFunc(func() { doPaste() }),
		history.WithEmitFunc(func(name string, data any) { handle.Emit(name, data) }),
		history.WithNotifyFunc(func(title, msg string) {
			if sett.GetSettings().NotifyEnabled {
				notify.ShowToast(title, msg)
			}
		}),
		history.WithSyncPushFunc(func(hash, content string) {
			syncSvc.PushEntry(sync.PushInput{ContentHash: hash, Content: content})
		}),
	)

	// File-manager function wired after app creation.
	var openFileManagerFn func(string, bool) error
	fileSvc := fileop.NewService(func(id int64) (string, error) {
		var c string
		err := conn.QueryRow(`SELECT content FROM clipboard WHERE id = ?`, id).Scan(&c)
		return c, err
	}, fileop.WithOpenFileManager(func(path string, selectFile bool) error {
		if openFileManagerFn == nil {
			return fmt.Errorf("file manager not wired")
		}
		return openFileManagerFn(path, selectFile)
	}))

	// Pull remote settings on startup. Only apply if different from local.
	if remoteSettings, err := syncSvc.PullSettings(); err == nil && remoteSettings != nil {
		var remote settings.Data
		if err := json.Unmarshal(remoteSettings, &remote); err == nil {
			// Compare: marshal local to detect real differences (avoids RawMessage compare issue).
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

	// Watcher callback delegates capture to history.Service.
	watcher := clipboard.NewWatcher(handle.ReadClipboard, func(text, hash string) {
		entry, isNew := histSvc.CaptureEntry(text, hash)
		if isNew {
			log.Printf("new clipboard entry: %q", previewText(entry.Content))
		}
	})

	// Create Wails app with all services.
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

	// Wire app handle now that app exists.
	handle.Wire(app)

	// Wire file-manager function (needs app handle).
	openFileManagerFn = func(path string, selectFile bool) error {
		return app.Env.OpenFileManager(path, selectFile)
	}

	// Create window (default visible for good UX).
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "jPaste", Width: 480, MinWidth: 360, Height: 560, MinHeight: 300,
		Hidden: sett.GetSettings().StartMinimized, URL: "/",
		BackgroundColour: application.NewRGB(248, 250, 252),
	})

	// Wire paste function (needs win reference).
	doPaste = func() {
		hideWindow(win)
		time.Sleep(150 * time.Millisecond)
		clipboard.Paste()
	}

	// System tray.
	setupSystemTray(app, win)

	// Global hotkey.
	setupGlobalHotkey(win, sett)

	// Window hooks.
	win.RegisterHook(wailsEvent.Common.WindowClosing, func(e *application.WindowEvent) {
		hideWindow(win)
		e.Cancel()
	})
	win.OnWindowEvent(wailsEvent.Common.WindowLostFocus, func(e *application.WindowEvent) {
		hideWindow(win)
	})

	// Startup tasks.
	runCleanup(histSvc, sett)
	if !sett.GetSettings().StartMinimized {
		showWindow(win)
	}
	setupAutostart(app, sett)

	// Real-time settings handlers.
	sett.OnSettingsChange(func(old, new settings.Data) {
		if new.AutoStart {
			app.Autostart.Enable()
		} else {
			app.Autostart.Disable()
		}
		// Run local cleanup immediately when retain_days changes.
		if old.RetainDays != new.RetainDays {
			if n, err := histSvc.Cleanup(new.RetainDays); err != nil {
				log.Printf("warn: cleanup on retain change: %v", err)
			} else if n > 0 {
				log.Printf("cleaned up %d old entries (retain_days changed to %d)", n, new.RetainDays)
			}
		}
		// Push settings to WebDAV.
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
			win.EmitEvent(events.Navigate, "/") // always go to list page
			showWindow(win)
		}
	}
	register := func(keystr string) {
		hotkey.UnregisterAll()
		if err := hotkey.Register(keystr, toggle); err != nil {
			log.Printf("warn: register global hotkey %q: %v", keystr, err)
		} else {
			log.Printf("global hotkey registered: %s", keystr)
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

// --- Window helpers ---

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

// --- Helpers ---

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

// acquireLock creates a lock file with the current PID.
// If a lock file already exists, checks whether the owning process is still alive.
// Returns true if successfully acquired.
func acquireLock(appData string) bool {
	lockFilePath = filepath.Join(appData, "instance.lock")

	// Ensure dir exists.
	os.MkdirAll(appData, 0700)

	// Try to read existing lock.
	data, err := os.ReadFile(lockFilePath)
	if err == nil {
		if pid, parseErr := strconv.Atoi(string(data)); parseErr == nil && pid > 0 && isProcessAlive(pid) {
			return false // lock is valid, another instance is running
		}
		// Stale lock — remove and retry.
		os.Remove(lockFilePath)
	}

	// Create exclusive lock file.
	pid := os.Getpid()
	if writeErr := os.WriteFile(lockFilePath, []byte(strconv.Itoa(pid)), 0600); writeErr != nil {
		log.Printf("warn: write lock file: %v", writeErr)
		return true // allow to run even if lock fails
	}
	return true
}

// releaseLock removes the lock file.
func releaseLock() {
	if lockFilePath != "" {
		os.Remove(lockFilePath)
	}
}

// isProcessAlive checks if a Windows process with the given PID exists.
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
	return exitCode == 259 // STILL_ACTIVE
}
