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
A three-state setting (`settings.paste_order`: `"normal"` / `"stack"` / `"queue"`) that controls how Ctrl+V consumes recently captured clipboard items. When non-normal, a `WH_KEYBOARD_LL` global hook intercepts user Ctrl+V, pops an item from an in-memory `container/list`, writes it to the system clipboard, and lets the original Ctrl+V pass through. Only `CF_UNICODETEXT` (plain text) is supported. Switching to/from `"normal"` clears the list and stops/starts the hook. Switching **between** stack and queue preserves existing items — only the consumption direction changes. jPaste's own clipboard writes and simulated paste are guarded by self-write hash and self-paste timestamp flags. Setting to `"normal"` stops the hook and clears the list.

**Auto-exit on non-text capture**: When stack or queue mode is active and the user copies content containing image (`CF_DIB`/`CF_DIBV5`) or file (`CF_HDROP`) formats, the mode automatically reverts to `"normal"` (persisted to `settings.json`). The current stack/queue list is cleared and a toast notification explains the exit. This bypasses the `NotifyEnabled` toggle. Self-writes from jPaste's own clipboard operations are exempted.

**Stack/Queue visualisation**: The footer displays three mode toggle buttons (正常/栈/队列). Hovering over the active stack or queue button shows a popup listing the current items with a ▶ arrow indicating the next item to be pasted. The popup also displays the item count and a brief explanation of the mode behavior. The `filostack.Service` exposes a `GetItems()` Wails binding that returns previews of all items in the current list.

### Clipboard Stack
A **Paste Order** sub-mode (`paste_order: "stack"`). Items are consumed from the back of the list — LIFO (Last In, First Out). Copy order `1,2,3,4,5` → paste order `5,4,3,2,1`.

### Clipboard Queue
A **Paste Order** sub-mode (`paste_order: "queue"`). Items are consumed from the front of the list — FIFO (First In, First Out). Copy order `1,2,3,4,5` → paste order `1,2,3,4,5`.

### Image Store
An external file directory at `%APPDATA%/jPaste/images/{YYYY-MM-DD}/` for storing clipboard image payloads. Organized by date folders for easy cleanup — when expired entries are deleted, the corresponding date folders and image files are removed together.

### Search Sort Order
A user preference persisted in `settings.json` (`sort_field`, `sort_order`) controlling the order of clipboard history results. Two sort fields: `updated_at` (last-updated time) and `content_length` (byte length of `CF_UNICODETEXT` content, stored as a redundant column on `clipboard_entry`). Each supports `DESC` (default) and `ASC`. Image-only entries have `content_length = 0`. The sort selector is a dropdown in the search bar; changing it resets pagination and reloads from the first page.

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
A user-assigned marker, independent of capture-time tags. Stored as `is_favorite BOOLEAN DEFAULT 0` on `clipboard_entry` — a separate column from `tag_mask` because the user's manual choice must survive automated capture-time tag recomputation. The list page provides a **Favorite Tab** (`TAG_FAVORITE`, virtual tag bit 32) alongside the auto-classification tabs. Filtering uses `WHERE is_favorite = 1` or a dedicated backend query, not the `tag_mask` bitmask. |

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
A separate Wails window opened for structured JSON viewing and editing. When a clipboard entry is detected as valid JSON and the user clicks "查看 JSON", the `jsonviewer.Service` calls the create-window callback with `/json-view?id=<entryID>`. The front-end `JsonViewPage` parses the `id` query parameter and retrieves the entry content via the `HistoryService.GetEntryContent(entryId)` binding, then renders it with the [jsoneditor](https://github.com/josdejong/jsoneditor) component in `tree` mode with optional `code` (Ace editor) mode toggle. The editor supports full CRUD operations, undo/redo, search, sort, drag-and-drop, and JSON formatting. The JSON viewer window is independent — it can stay open while the user continues using the main window.

### Toast
A small frameless Wails window shown at the bottom-right of the primary screen when new clipboard content is captured. Appears with 200ms fade-in, stays for ~2.6s, exits with 150ms fade-out. IgnoreMouseEvents + no focus steal. Uses an event-driven pattern: Go emits `toast-notification` with `{title, message}`, and the pre-created toast window's frontend listens directly.

**Theme-adaptive**: The toast window loads the current theme independently via `SettingsService.GetSettings()` and renders a frosted-glass card using the CSS custom property `--toast-glass-bg` (defined per theme in `style.css`). The icon uses `--color-primary`, title uses `--color-foreground`, subtitle uses `--color-muted` — ensuring visual consistency with the user's selected theme. Runs outside the main `<App>` component tree (no MemoryRouter), rendered directly in `main.jsx` for the `/toast` route.

### Temporary File
A `.txt` file created in `%TEMP%` with the selected entry's content, then opened in the user's preferred text editor (VS Code first, then system default).

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│  React Frontend (WebView)                                  │
│  ┌────────────┐ ┌──────────────┐ ┌────────────────────┐   │
│  │  MainPage   │ │ SettingsPage │ │ JsonViewPage (*)   │   │
│  └────────────┘ └──────────────┘ └────────────────────┘   │
│         ↕ Wails Bindings         ↕ (separate window)      │
├──────────────────────────────────────────────────────────┤
│  Go Backend                                                │
│  ┌──────────┐ ┌─────────────────┐  ┌───────────────────┐  │
│  │Clipboard │ │ HistoryService  │  │  JsonViewerSvc    │  │
│  │ Watcher  │ │ (EntryStore)    │  │  ImageViewerSvc   │  │
│  │(lxn/win) │ └─────────────────┘  └───────────────────┘  │
│  └──────────┘ ┌─────────────────┐  ┌───────────────────┐  │
│               │ ImageStore      │  │  FiloStackService │  │
│               │ (ImageStorer)   │  │  (Strategy: Stack │  │
│  ┌──────────┐ └─────────────────┘  │   / Queue)       │  │
│  │Settings  │ ┌─────────────────┐  └───────────────────┘  │
│  │ Service  │ │ FileService     │                         │
│  └──────────┘ └─────────────────┘                         │
│  ┌──────────────────┐ ┌──────────────────┐                │
│  │ ToastService     │ │ System Tray +    │                │
│  │ (event-driven)   │ │ Global Hotkey    │                │
│  └──────────────────┘ └──────────────────┘                │
│  ┌───────────────────────────────────────────────────┐    │
│  │ internal/repository (single SQLite access layer)  │    │
│  │ internal/util (hashutil, textutil, dibutil)       │    │
│  └───────────────────────────────────────────────────┘    │
│  ┌───────────────────────────────────────────────────┐    │
│  │ SQLite + settings.json                            │    │
│  └───────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘

(*) JsonViewPage / ImageViewPage run in separate Wails windows,
    created on demand by jsonviewer.Service / imageviewer.Service.
```

## Design Patterns Used

| Pattern | Location | Usage |
|---------|----------|-------|
| **Repository** | `internal/repository/` | Single `Repository` struct owns ALL SQLite queries. `history.EntryStore` is a thin adapter wrapping the `Repository` instance. |
| **Strategy** | `internal/filostack/` | `PasteStrategy` interface with `StackStrategy` (LIFO) and `QueueStrategy` (FIFO) implementations. The `Service.Pop()` delegates to the strategy instead of switch-on-mode. |
| **Functional Options** | `internal/history/`, `internal/fileop/`, `internal/filostack/` | `WithPasteFunc`, `WithEmitFunc`, `WithNotifyFunc`, `WithImageStore`, `WithOpenFileManager` etc. — type-safe optional service configuration without builder boilerplate. |
| **Interface Segregation** | `internal/history/store.go` | `EntryStore` interface abstracts SQLite; tests use in-memory fakes. `ImageStorer` separates image I/O from DB. |
| **Event-Driven** | `internal/toast/`, `internal/events/` | Go emits Wails custom events (`toast-notification`, `clipboard-updated`); front-end listens via `Events.On()`. |
| **Observer** | `internal/settings/` | `OnHotkeyChange` / `OnSettingsChange` callbacks notify dependents when settings mutate. |
| **Template Method** | `internal/history/service.go` | `resolveSortParams` — shared helper extracted from duplicate sort-field resolution in `GetHistory` and `GetHistoryRegex`. |
| **Utility Module** | `internal/util/` | Consolidates SHA256 hashing (`SHA256Hex`, `SHA256String`, `SHA256TextOrRaw`), text truncation (`Truncate`, `TruncateBytes`), and DIB/BMP header prepending (`PrependBMPHeader`), eliminating cross-package code duplication. |

## Source File Layout

| File | Purpose |
|------|---------|
| `main.go` | Entry point: bootstrap, service wiring, window creation, event loop. Remains lean by delegating to helpers. |
| `main_helpers.go` | `watcherHandler`, `cleanupOrphanedWV2`, lock-file routines, `runCleanup`, `previewText` — extracted logic that reduces main.go to orchestrator-only concerns. |

## Storage

- **Clipboard history:** `%APPDATA%/jPaste/clipboard.db` (SQLite)
  - `clipboard_entry`: id, content_hash, source_exe, source_title, tag_mask, is_favorite, created_at, updated_at
  - `clipboard_format`: entry_id (FK), format_type, content (TEXT, nullable), file_path (nullable), format_hash
- **Images:** `%APPDATA%/jPaste/images/{YYYY-MM-DD}/{uuid}.png`
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
| json      | Starts with `{` or `[`, valid JSON | Opens separate window with full JSON editor (jsoneditor) in `tree` + `code` modes | Go: `jsonviewer.Service.OpenJsonViewer()` creates a new Wails window at `/json-view?id=<entryID>`, front-end retrieves data via `HistoryService.GetEntryContent(id)` |
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
