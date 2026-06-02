# jPaste — Domain Glossary

A Windows clipboard manager built with Wails v3 + React.

## Domain Terms

### Clipboard Entry
A single text record copied to the system clipboard. Uniquely identified by `content_hash` (SHA-256 of trimmed content). Fields: `id`, `content_hash`, `content`, `created_at`, `updated_at`.

### Deduplication
When a new clipboard content is detected, it is hashed. If the hash matches an existing entry, the existing entry's `updated_at` is refreshed — no duplicate inserted. The entry moves to the top of the history list.

### Default Action
The action executed when a user selects a clipboard entry (click or `Ctrl+Digit`). Configurable in settings: **Copy** (write to clipboard, hide window) or **Paste** (write to clipboard, switch focus to previous window, simulate `Ctrl+V`).

### Global Hotkey
A system-wide keyboard shortcut that shows/hides the jPaste window. Default: `Alt+V`. Configurable in settings.

### Retained Duration
How long clipboard entries are kept before automatic cleanup. Default: 30 days. Cleanup runs on app startup and periodically.

### Toast
A native Windows notification shown in the bottom-right corner when new clipboard content is captured. Duration: 3 seconds.

### Temporary File
A `.txt` file created in `%TEMP%` with the selected entry's content, then opened in the user's preferred text editor (VS Code first, then system default).

### WebDAV Sync
Bidirectional merge of clipboard entries and settings across machines via WebDAV (e.g., 坚果云). Each machine runs a push/pull cycle independently; no central coordinator.

**Entry File**: A per-`content_hash` JSON file on WebDAV stored at `entries/{2-char-prefix}/{hash}.json`. Content: `{content, created_at, updated_at}`. Files are sharded into 256 prefix directories to avoid large flat directories.

**Merge Rule**: `content_hash` is the identity key. When a local entry and its remote counterpart exist, keep the one with the later `updated_at`. Local entries not on remote are pushed; remote entries not in local are pulled (subject to `retain_days` filter).

**Push**: On each clipboard change, the new/updated entry is `PUT` immediately. Failure triggers exponential backoff (1min → 2min → 4min → ...). During push, entries on WebDAV older than `retain_days` are deleted — since `retain_days` is shared via settings sync, both machines agree on the deletion boundary.

**Pull**: Every 60 seconds, a `PROPFIND` on the `entries/` directory lists remote files with `getlastmodified`. Each is compared against local SQLite by `content_hash` + `updated_at`. Only new or changed entries are `GET`-downloaded and upserted. Settings are only pulled once at startup, not on the periodic cycle.

**WebDAV Credentials**: Stored in `%APPDATA%/jPaste/webdav.json` (URL, username, app password). Not synced — each machine configures its own.

**Offline**: Push failures increment a backoff counter and show a sync status indicator (green/grey/yellow/red). Pull continues on its fixed 60-second cycle regardless — if unreachable, it silently skips until the next cycle.

**Sync Status**: A small indicator in the main page header showing four states: green ✓ (synced), yellow ⟳ (syncing), grey — (not configured), red ⚠ (error, with tooltip showing last error).

## Architecture

```
┌──────────────────────────────────┐
│  React Frontend (WebView)         │
│  ┌────────────┐ ┌──────────────┐ │
│  │  MainPage   │ │ SettingsPage │ │
│  └────────────┘ └──────────────┘ │
│         ↕ Wails Bindings         │
├──────────────────────────────────┤
│  Go Backend                       │
│  ┌──────────┐ ┌───────────────┐  │
│  │Clipboard │ │ HistoryService│  │
│  │ Service  │ │               │  │
│  └──────────┘ └───────────────┘  │
│  ┌──────────┐ ┌───────────────┐  │
│  │Settings  │ │ FileService   │  │
│  │ Service  │ │               │  │
│  └──────────┘ └───────────────┘  │
│  ┌──────────┐                    │
│  │  Sync    │                    │
│  │ Service  │                    │
│  └──────────┘                    │
│  ┌──────────────────────────────┐ │
│  │ SQLite + settings + webdav   │ │
│  └──────────────────────────────┘ │
│  ┌──────────────────────────────┐ │
│  │ System Tray + Global Hotkey  │ │
│  └──────────────────────────────┘ │
└──────────────────────────────────┘
```

## Storage

- **Clipboard history:** `%APPDATA%/jPaste/clipboard.db` (SQLite)
- **User settings:** `%APPDATA%/jPaste/settings.json`

### Action Module
A self-contained frontend module that recognizes clipboard content and offers a contextual operation. Each module is a file in `frontend/src/actions/`. Exports: `{ id, label, icon, priority, detect(content): boolean, Component }`. Modules are registered statically in `actions/index.js`.

**Detection** is lazy and viewport-scoped: only entries that scroll into view are tested via a shared `IntersectionObserver` (rootMargin 120px). Results are cached by entry ID for the lifetime of the entry list.

**Up to 3** highest-priority matched actions are shown as inline buttons next to Copy/Paste on each list item. Buttons show the module's lucide-react icon + label tooltip.

**Action Config** lives in `settings.json` under `action_config`: `{ "moduleId": { "enabled": true, "priority": number } }`. Go passes it through as `json.RawMessage` — opaque to the backend. The Settings page provides per-module enable/disable toggles and priority adjustment via up/down buttons.

**Action Modal** is a React overlay (`components/ActionModal.jsx`) that renders the matched module's `Component` prop. The component receives `{ content, entryId, onClose }`.

### Content-Aware Actions
The six built-in action modules:

| Module    | Detection Rule                     | Behavior              | Implementation |
|-----------|------------------------------------|-----------------------|----------------|
| math      | Only digits/ops/parens, >=1 operator | Editable expression → eval in modal | Pure JS (`new Function`) |
| json      | Starts with `{` or `[`, valid JSON | Collapsible tree viewer | Custom React component |
| url       | Starts with `http://` or `https://` | Open in default browser | `Browser.OpenURL()` from `@wailsio/runtime` |
| folder    | Starts with `X:\` or `\\` (Windows) | Open in Explorer | Go: `app.Env.OpenFileManager()` via `fileop.Service.OpenInExplorer()` |
| base64    | Base64 charset, length>4, mod4=0   | Editable decode in modal | `atob()` |
| unicode   | Contains `\uXXXX` pattern          | Editable decode in modal | `String.fromCharCode()` |

## Key Behaviors

- **Close button** hides to system tray, does NOT quit
- **Alt+V** shows/hides the window (Spotlight-like fade+scale animation)
- **Lose focus** → auto-hide
- **Window** starts hidden, first show on app launch
- **Clipboard polling** every 1 second, active regardless of window visibility
- **Cleanup** removes entries older than configured retention period
