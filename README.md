# jPaste

A modern clipboard manager for Windows, built with Wails v3 + React.

> 一个轻量级 Windows 剪贴板管理器，事件驱动监听，支持文本/图片/文件/URL 分类和即时搜索。

## Features

- **Event-driven capture** — `WM_CLIPBOARDUPDATE` message-only window, no polling
- **Multi-format support** — text (`CF_UNICODETEXT`), images (`CF_DIB`), file paths (`CF_HDROP`)
- **Source tracking** — records which application produced each clipboard entry
- **Tag-based filtering** — 全部 / 文本 / 图片 / 网址 / 文件 / 收藏
- **Instant search** — full-text search across clipboard history
- **Content-aware actions** — JSON viewer, math evaluator, URL opener, Base64 decoder, etc.
- **Global hotkey** — `Alt+V` shows/hides the window
- **Configurable** — copy or paste-on-select, retention period, auto-start

## Download

Download the latest `jpaste.exe` from [Releases](https://github.com/wangxianyu/jPaste/releases). Portable — no installation required.

## Development

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# Run in dev mode (hot-reload)
wails3 dev

# Build production binary
wails3 build
# Output: bin/jpaste.exe
```

### App Icon

Source: `jpaste-logo.svg`. Generate all icon assets with one command:

```powershell
.\scripts\update-icon-svg.ps1
```

Output files:
- `paste.png` — tray icon, embedded by `main.go` via `//go:embed`
- `build/windows/icon.ico` — multi-resolution (16–256px) app icon
- `wails_windows_amd64.syso` — Windows build resource

Rasterization is pure Go (`scripts/rasterize-logo/`), no external dependencies. Gradients in the SVG are flattened to solid colors for compatibility with `oksvg`.

## Tech Stack

- **Backend**: Go 1.25 + Wails v3 + SQLite (via `modernc.org/sqlite`)
- **Frontend**: React 18 + Vite + React Router + Lucide Icons
- **Clipboard**: `lxn/win` — raw Win32 API for format enumeration, owner detection, and paste simulation
- **Storage**: `%APPDATA%/jPaste/clipboard.db` (SQLite) + `%APPDATA%/jPaste/images/` (PNG/DIB)

## License

MIT
