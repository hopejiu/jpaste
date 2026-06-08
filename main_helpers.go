package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"jpaste/internal/clipboard"
	"jpaste/internal/filostack"
	"jpaste/internal/history"
	"jpaste/internal/model"
	"jpaste/internal/settings"
	"jpaste/internal/toast"
	"jpaste/internal/util"

	applog "jpaste/internal/log"
)

// watcherHandler processes clipboard capture events.
// It is called from the clipboard watcher on each WM_CLIPBOARDUPDATE.
type watcherHandler struct {
	histSvc   *history.Service
	filoStack *filostack.Service
	sett      *settings.Service
	toastSvc  *toast.Service
}

func newWatcherHandler(histSvc *history.Service, filoStack *filostack.Service, sett *settings.Service, toastSvc *toast.Service) clipboard.OnCapture {
	h := &watcherHandler{
		histSvc:   histSvc,
		filoStack: filoStack,
		sett:      sett,
		toastSvc:  toastSvc,
	}
	return h.handle
}

func (h *watcherHandler) handle(data model.CapturedData) {
	applog.Info("capture callback", "formats", len(data.Formats), "hash", data.PrimaryHash[:12], "source", data.SourceEXE)

	// Detect self-writes early so we can override the clipboard owner EXE.
	// jPaste writes to the clipboard via Wails' WebView2 (msedgewebview2.exe),
	// so getClipboardSource() returns the WebView2 process instead of jPaste.
	isSelfWrite := clipboard.IsSelfWrite(data)
	applog.Info("self-write check", "isSelfWrite", isSelfWrite, "source", data.SourceEXE)
	if isSelfWrite {
		data.SourceEXE = "jPaste"
		applog.Info("self-write detected, overriding source to jPaste")
	}

	entry, isNew := h.histSvc.CaptureEntry(data)
	if isNew {
		applog.Info("new clipboard entry", "id", entry.ID, "text", previewText(entry.Content), "source", entry.SourceEXE)
	} else {
		applog.Info("dedup entry", "hash", data.PrimaryHash[:12])
	}

	// Push text to FILO stack when stack mode is enabled.
	stackEnabled := h.filoStack.Enabled()
	textToPush := model.PrimaryText(data.Formats)
	applog.Info("stack decision",
		"stackEnabled", stackEnabled,
		"isSelfWrite", isSelfWrite,
		"text", previewText(textToPush),
		"hash", data.PrimaryHash[:12],
	)
	if stackEnabled && !isSelfWrite && textToPush != "" {
		h.filoStack.Push(textToPush)
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
			modeLabel := h.filoStack.ModeName()
			itemCount := h.filoStack.Len()

			newSettings := h.sett.GetSettings()
			newSettings.PasteOrder = "normal"
			h.sett.SaveSettings(newSettings)

			h.toastSvc.ShowToast("jPaste",
				fmt.Sprintf("检测到非文本内容，已退出%s模式（剩余 %d 项已清空）", modeLabel, itemCount))
		}
	}

	// Notify（无论是否新内容，重复也弹）。
	if h.sett.GetSettings().NotifyEnabled {
		previewSource := textToPush
		if entry != nil {
			previewSource = entry.Content
		}
		contentPreview := util.Truncate(previewSource, 10)
		if contentPreview == "" {
			// Image-only entry: show a meaningful label.
			for _, f := range data.Formats {
				if model.IsImageFormat(f.FormatType) {
					contentPreview = "[图片]"
					break
				}
			}
		}
		if h.filoStack.Enabled() {
			modeLabel := h.filoStack.ModeName()
			h.toastSvc.ShowToast("jPaste",
				fmt.Sprintf("剪贴板写入: %s, 当前%s已有: %d 个", contentPreview, modeLabel, h.filoStack.Len()))
		} else {
			h.toastSvc.ShowToast("jPaste", "剪贴板写入: "+contentPreview)
		}
	}
}

// cleanupOrphanedWV2 kills orphaned msedgewebview2 processes — WebView2 child
// processes whose parent (previous jPaste instance) no longer exists.
func cleanupOrphanedWV2() {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer syscall.CloseHandle(snapshot)

	alive := make(map[uint32]bool)
	type procInfo struct {
		name      string
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
			continue
		}
		if alive[pi.parentPid] {
			continue
		}
		h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, pid)
		if err != nil {
			continue
		}
		syscall.TerminateProcess(h, 0)
		syscall.CloseHandle(h)
		applog.Info("cleaned orphaned WebView2 process", "pid", pid, "parent", pi.parentPid)
	}
}

func runCleanup(histSvc *history.Service, sett *settings.Service) {
	cfg := sett.GetSettings()
	if n, err := histSvc.Cleanup(cfg.RetainDays); err != nil {
		applog.Warn("cleanup", "error", err)
	} else if n > 0 {
		applog.Info("cleaned up old entries", "count", n)
	}

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

func previewText(s string) string {
	return util.TruncateBytes(s, 80)
}
