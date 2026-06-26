// Package log provides structured logging to daily-hourly files in appdata.
// It also redirects the standard log package output to the same file.
package log

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu             sync.Mutex
	curFile        *os.File
	curName        string
	logDir         string
	terminalOutput bool // true in dev mode (build tag !production), false in production builds
)

// Init sets up slog + standard log to write to appdata/jPaste/jpaste-{date}-{hour}.log.
// Starts a goroutine that rotates logs hourly and cleans up files older than 12 hours.
func Init(dir string) error {
	logDir = dir
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return err
	}
	if err := switchLog(); err != nil {
		return err
	}
	go func() {
		for range time.Tick(1 * time.Minute) {
			rotateIfNeeded()
			cleanOldLogs()
		}
	}()
	return nil
}

func logFileName() string {
	now := time.Now()
	return filepath.Join(logDir, "jpaste-"+now.Format("2006-01-02-15")+".log")
}

func switchLog() error {
	mu.Lock()
	defer mu.Unlock()
	name := logFileName()
	if name == curName {
		return nil
	}
	if curFile != nil {
		curFile.Close()
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	curFile = f
	curName = name

	// MultiWriter: log always goes to file; stderr only in dev mode.
	var writers []io.Writer
	writers = append(writers, f)
	if terminalOutput {
		writers = append(writers, os.Stderr)
	}
	mw := io.MultiWriter(writers...)

	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	h := slog.NewTextHandler(mw, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(h))
	return nil
}

func rotateIfNeeded() {
	if err := switchLog(); err != nil {
		log.Printf("rotate log: %v", err)
	}
}

func cleanOldLogs() {
	mu.Lock()
	defer mu.Unlock()
	cutoff := time.Now().Add(-12 * time.Hour)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".log" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			p := filepath.Join(logDir, e.Name())
			if p != curName {
				os.Remove(p)
			}
		}
	}
}

func Info(msg string, args ...any)  { slog.Info(msg, args...) }
func Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }
func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
