# jPaste — Domain Glossary

A Windows clipboard manager built with Wails v3 + React.

## Domain Terms

### Clipboard Entry
A single copy operation captured from the system clipboard. Each entry is uniquely identified by `content_hash` — the SHA-256 of the trimmed `CF_UNICODETEXT` content. For image-only copies (no text), the hash is computed from the raw image bytes instead.

Each entry records the **source application** (`source_exe`, `source_title`), its content as one or more **Clipboard Format** payloads in the format sub-table, and a **Tag Mask** (`tag_mask`) computed at capture time for list filtering.

### Clipboard Format
A format-specific payload attached to a **Clipboard Entry**. Windows supports multiple formats per clipboard operation — an entry may have `CF_UNICODETEXT` (plain text), `CF_HTML` (rich HTML), `CF_RTF`, and `CF_DIB` (image) simultaneously.

Text-based formats (`CF_UNICODETEXT`, `CF_HTML`, `CF_RTF`, `CF_HDROP`) are stored inline as TEXT. Image formats (`CF_DIB`, `CF_DIBV5`) are saved to the **Image Store** with only the file path stored in the database. Each format has its own `format_hash` for integrity.

### Clipboard Source
The application that wrote the current clipboard content. Captured via `GetClipboardOwner()` → `GetWindowThreadProcessId()` → `OpenProcess()` + `QueryFullProcessImageName()` to record the full executable path, plus `GetWindowText()` for the window title at time of copy. NULL owner (clipboard cleared or cross-session) stores empty strings.

### Paste Order
A three-state setting (`settings.paste_order`: `"normal"` / `"stack"` / `"queue"`) that controls how Ctrl+V consumes recently captured clipboard items. When non-normal, a `WH_KEYBOARD_LL` global hook intercepts user Ctrl+V, pops an item from an in-memory `container/list`, writes it to the system clipboard, and lets the original Ctrl+V pass through. Only `CF_UNICODETEXT` (plain text) is supported. Switching between modes clears the list. jPaste's own clipboard writes and simulated paste are guarded by self-write hash and self-paste timestamp flags. Setting to `"normal"` stops the hook and clears the list.

### Clipboard Stack
A **Paste Order** sub-mode (`paste_order: "stack"`). Items are consumed from the back of the list — LIFO (Last In, First Out). Copy order `1,2,3,4,5` → paste order `5,4,3,2,1`.

### Clipboard Queue
A **Paste Order** sub-mode (`paste_order: "queue"`). Items are consumed from the front of the list — FIFO (First In, First Out). Copy order `1,2,3,4,5` → paste order `1,2,3,4,5`.

### Image Store
An external file directory at `%APPDATA%/jPaste/images/{YYYY-MM-DD}/` for storing clipboard image payloads. Organized by date folders for easy cleanup — when expired entries are deleted, the corresponding date folders and image files are removed together. Images are excluded from WebDAV sync.

### Entry Tag
A classification label assigned to a **Clipboard Entry** at capture time. Tags are determined by a mixed strategy: **format-driven** (which clipboard formats are present) and **content-driven** (pattern matching on the text payload). An entry can carry multiple tags simultaneously — e.g., a browser URL copy carries both `url` and `text`.

The five capture-time tags:

| Tag | Bit | Determination |
|-----|-----|---------------|
| `text` | 1 | Has `CF_UNICODETEXT` and no image / file-path formats |
| `image` | 4 | Has `CF_DIB` or `CF_DIBV5` |
| `url` | 8 | `CF_UNICODETEXT` starts with `http://` or `https://` |
| `file` | 16 | Has `CF_HDROP`, or text matches Windows path pattern (`[A-Z]:\` or `\\`) |

### Favorite
A user-assigned marker, independent of capture-time tags. Stored as `is_favorite BOOLEAN DEFAULT 0` on `clipboard_entry` — a separate column from `tag_mask` because the user's manual choice must survive automated capture-time tag recomputation. The list page provides a **Favorite Tab** (`TAG_FAVORITE`, virtual tag bit 32) alongside the auto-classification tabs. Filtering uses `WHERE is_favorite = 1` or a dedicated backend query, not the `tag_mask` bitmask. Sync: `is_favorite` is included in the remote entry JSON and merged alongside `updated_at` — local always wins for this field to avoid remote overwrites of user intent. |

### Tag Mask
A bitmask stored on `clipboard_entry.tag_mask` (INTEGER) encoding an entry's **Entry Tags**. Multiple tags are combined via bitwise OR (e.g., a URL copy: `1 | 8 = 9`). The list page filters by passing a `tagMask` to `GetHistory`; the backend uses `tag_mask & tagMask != 0` for matching. A `tagMask` of 0 means "no filter" (show all).

### Cursor Pagination
The list page loads entries in pages of 20 via cursor-based pagination using a compound cursor `(updated_at, id)`. Each `GetHistory` call passes two cursor values from the last-seen entry. The backend queries `WHERE (updated_at < ? OR (updated_at = ? AND id < ?)) ORDER BY updated_at DESC, id DESC LIMIT 21` and returns whether more pages exist (`hasMore`). First-page requests use zero-value cursors. Timestamps are stored at **millisecond precision** (`strftime('%Y-%m-%dT%H:%M:%f', 'now')`) to minimize same-second collisions.

### Deduplication
When new clipboard content is detected, the `CF_UNICODETEXT` payload is trimmed and hashed (SHA-256). If the hash matches an existing entry, the existing entry's `updated_at` is refreshed and new format payloads are upserted — no duplicate entry inserted. For image-only copies, the image binary hash serves as the identity key. Deduplication only compares the primary format — additional format changes (e.g. same text with different HTML) do not create new entries.

### Default Action
The action executed when a user selects a clipboard entry (click or `Ctrl+Digit`). Configurable in settings: **Copy** (write to clipboard, hide window) or **Paste** (write to clipboard, switch focus to previous window, simulate `Ctrl+V`).

### Global Hotkey
A system-wide keyboard shortcut that shows/hides the jPaste window. Default: `Alt+V`. Configurable in settings.

### Retained Duration
How long clipboard entries are kept before automatic cleanup. Default: 30 days. Cleanup runs on app startup and periodically.

### JSON Viewer Window
A separate Wails window opened for structured JSON viewing and editing. When a clipboard entry is detected as valid JSON and the user clicks "查看 JSON", the `jsonviewer.Service` on the Go side stores the JSON content against a random 16-character hex token, then creates a new window at `/json-view?token=<token>`. The front-end `JsonViewPage` retrieves the content via `GetJsonViewerData(token)` (cached for 60s TTL to survive React StrictMode remounts) and renders it with the [jsoneditor](https://github.com/josdejong/jsoneditor) component in `tree` mode with optional `code` (Ace editor) mode toggle. The editor supports full CRUD operations, undo/redo, search, sort, drag-and-drop, and JSON formatting. The JSON viewer window is independent — it can stay open while the user continues using the main window.

### Toast
A native Windows notification shown in the bottom-right corner when new clipboard content is captured. Duration: 3 seconds.

### Temporary File
A `.txt` file created in `%TEMP%` with the selected entry's content, then opened in the user's preferred text editor (VS Code first, then system default).

### WebDAV Sync
Bidirectional merge of clipboard entries and settings across machines via WebDAV (e.g., 坚果云). Each machine runs a push/pull cycle independently; no central coordinator.

**Entry File**: A per-`content_hash` JSON file on WebDAV stored at `entries/{2-char-prefix}/{hash}.json`. Content: `{content, created_at, updated_at}`. Files are sharded into 256 prefix directories to avoid large flat directories.

**Merge Rule**: `content_hash` is the identity key. When a local entry and its remote counterpart exist, keep the one with the later `updated_at`. Local entries not on remote are pushed; remote entries not in local are pulled (subject to `retain_days` filter).

**Push**: On each clipboard change, the new/updated entry is `PUT` immediately. Failure triggers exponential backoff (1min → 2min → 4min → ...). During push, entries on WebDAV older than `retain_days` are deleted — since `retain_days` is shared via settings sync, both machines agree on the deletion boundary. Only text-based formats are synced; image formats are excluded.

**Pull**: Every 60 seconds, a `PROPFIND` on the `entries/` directory lists remote files with `getlastmodified`. Each is compared against local SQLite by `content_hash` + `updated_at`. Only new or changed entries are `GET`-downloaded and upserted. Settings are only pulled once at startup, not on the periodic cycle. Image formats are local-only and never pulled.

**WebDAV Credentials**: Stored in `%APPDATA%/jPaste/webdav.json` (URL, username, app password). Not synced — each machine configures its own.

**Offline**: Push failures increment a backoff counter and show a sync status indicator (green/grey/yellow/red). Pull continues on its fixed 60-second cycle regardless — if unreachable, it silently skips until the next cycle.

**Sync Status**: A small indicator in the main page header showing four states: green ✓ (synced), yellow ⟳ (syncing), grey — (not configured), red ⚠ (error, with tooltip showing last error).

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  React Frontend (WebView)                              │
│  ┌────────────┐ ┌──────────────┐ ┌──────────────────┐ │
│  │  MainPage   │ │ SettingsPage │ │ JsonViewPage (*) │ │
│  └────────────┘ └──────────────┘ └──────────────────┘ │
│         ↕ Wails Bindings         ↕ (separate window)  │
├──────────────────────────────────────────────────────┤
│  Go Backend                                            │
│  ┌──────────┐ ┌───────────────┐  ┌─────────────────┐  │
│  │Clipboard │ │ HistoryService│  │  JsonViewerSvc  │  │
│  │ Service  │ │               │  │ (token store +  │  │
│  │(lxn/win) │ └───────────────┘  │  window create) │  │
│  └──────────┘ ┌───────────────┐  └─────────────────┘  │
│               │ ImageStore    │                       │
│  ┌──────────┐ └───────────────┘                       │
│  │Settings  │ ┌───────────────┐                       │
│  │ Service  │ │ FileService   │                       │
│  └──────────┘ └───────────────┘                       │
│  ┌──────────┐                                         │
│  │  Sync    │                                         │
│  │ Service  │                                         │
│  └──────────┘                                         │
│  ┌───────────────────────────────────────────────┐    │
│  │ SQLite + settings + webdav                    │    │
│  └───────────────────────────────────────────────┘    │
│  ┌───────────────────────────────────────────────┐    │
│  │ System Tray + Global Hotkey                   │    │
│  └───────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────┘

(*) JsonViewPage runs in a separate Wails window (title: "JSON 查看"),
    created on demand by jsonviewer.Service.
```

## Storage

- **Clipboard history:** `%APPDATA%/jPaste/clipboard.db` (SQLite)
  - `clipboard_entry`: id, content_hash, source_exe, source_title, tag_mask, is_favorite, created_at, updated_at
  - `clipboard_format`: entry_id (FK), format_type, content (TEXT, nullable), file_path (nullable), format_hash
- **Images:** `%APPDATA%/jPaste/images/{YYYY-MM-DD}/{uuid}.png` — excluded from sync
- **User settings:** `%APPDATA%/jPaste/settings.json`

### Action Module
A self-contained frontend module that recognizes clipboard content and offers a contextual operation. Each module is a file in `frontend/src/actions/`. Exports: `{ id, label, icon, priority, detect(content): boolean, handler?(content): void, Component? }`. Modules are registered statically in `actions/index.js`.

**Detection** is lazy and viewport-scoped: only entries that scroll into view are tested via a shared `IntersectionObserver` (rootMargin 120px). Results are cached by entry ID for the lifetime of the entry list.

**Up to 3** highest-priority matched actions are shown as inline buttons next to Copy/Paste on each list item. Buttons show the module's lucide-react icon + label tooltip.

**Dispatch**: If the module exports a `handler` function, it is called directly when the user clicks the action button — no modal is opened. This is used for actions that open external windows (e.g., JSON viewer). Otherwise, if only a `Component` is exported, the `ActionModal` overlay renders the component.

**Action Config** lives in `settings.json` under `action_config`: `{ "moduleId": { "enabled": true, "priority": number } }`. Go passes it through as `json.RawMessage` — opaque to the backend. The Settings page provides per-module enable/disable toggles and priority adjustment via up/down buttons.

**Action Modal** is a React overlay (`components/ActionModal.jsx`) that renders the matched module's `Component` prop. The component receives `{ content, entryId, onClose }`.

### Content-Aware Actions
The six built-in action modules:

| Module    | Detection Rule                     | Behavior              | Implementation |
|-----------|------------------------------------|-----------------------|----------------|
| math      | Only digits/ops/parens, >=1 operator | Editable expression → eval in modal | Pure JS (`new Function`) |
| json      | Starts with `{` or `[`, valid JSON | Opens separate window with full JSON editor (jsoneditor) in `tree` + `code` modes | Go: `jsonviewer.Service.OpenJsonViewer()` creates a new Wails window at `/json-view?token=xxx`, front-end retrieves data via `GetJsonViewerData(token)` |
| url       | Starts with `http://` or `https://` | Open in default browser | `Browser.OpenURL()` from `@wailsio/runtime` |
| folder    | Starts with `X:\` or `\\` (Windows) | Open in Explorer | Go: `app.Env.OpenFileManager()` via `fileop.Service.OpenInExplorer()` |
| base64    | Base64 charset, length>4, mod4=0   | Editable decode in modal | `atob()` |
| unicode   | Contains `\uXXXX` pattern          | Editable decode in modal | `String.fromCharCode()` |

## Key Behaviors

- **Close button** hides to system tray, does NOT quit
- **Alt+V** shows/hides the window (Spotlight-like fade+scale animation)
- **Lose focus** → auto-hide
- **Window** starts hidden, first show on app launch
- **Clipboard monitoring** event-driven via `AddClipboardFormatListener` + `WM_CLIPBOARDUPDATE` on a message-only window (no polling)
- **List filtering** via tag tabs (全部 / 文本 / 图片 / 网址 / 文件) + keyword search, with cursor-based pagination (20 per page)
- **Cleanup** removes entries older than configured retention period
- **Clear All** — a button on the Settings page that deletes all clipboard entries and image files at once
