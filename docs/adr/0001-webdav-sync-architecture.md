# WebDAV sync: per-hash entry files with bidirectional merge

jPaste needs multi-machine clipboard sync via WebDAV (坚果云). We chose per-`content_hash` JSON files with bidirectional merge (latest `updated_at` wins), incremental detection via PROPFIND, and shared `retain_days` for cloud cleanup — rejecting a single JSON export, a separate manifest, and isolated cloud retention settings.

## Context

jPaste stores clipboard entries in local SQLite (`clipboard.db`) and user settings in `settings.json`. Two or more Windows machines share one 坚果云 WebDAV account. The sync must handle: new entries appearing on either machine, entries deleted by local cleanup, configuration drift, and offline periods.

## Decision

### 1. Per-hash entry files (not single JSON export)

Each clipboard entry is an individual JSON file on WebDAV: `entries/{hash[0:2]}/{hash}.json`. Contents: `{content, created_at, updated_at}`.

**Why not a single file**: With 5000+ entries, re-uploading 1MB on every change is wasteful and creates merge conflicts. Per-hash files deliver true incremental sync — one 200-byte PUT per new entry. Sharding by hash prefix avoids directory limits.

### 2. No separate sync manifest — SQLite is the ground truth

Pull uses `PROPFIND` to list remote files with `getlastmodified`, then compares each hash against the local SQLite `updated_at`. No separate `sync_manifest.json`.

**Why not a manifest**: SQLite already holds `updated_at` per entry. A manifest duplicates this, creating a consistency problem if they diverge. The PROPFIND round-trip is cheap for the expected entry count (~5000).

### 3. Settings: push on change, pull on startup only

`settings.json` is pushed to WebDAV immediately on save (with backoff). It is pulled only once at app startup, not on the 60-second cycle.

Settings change rarely. Startup-only pull covers the common case (machine B was configured while machine A was running). Avoiding a 60-second poll eliminates unnecessary HTTP requests.

### 4. Cloud cleanup uses shared `retain_days`

During push, entries on WebDAV older than `retain_days` are deleted. Since `settings.json` is synced, both machines share a single `retain_days` value — no split-brain on deletion boundary.

**Why not separate `cloud_retain_days`**: Adding a second retention setting is confusing (users already have `retain_days`). Settings sync guarantees consistency without extra config.

## Considered Alternatives

- **Single JSON dump**: Rejected — full re-upload per change, conflicts on concurrent edits.
- **Upload SQLite directly**: Rejected — WAL journal makes the file unsafe to copy while open; multi-writer corruption.
- **Separate sync manifest**: Rejected — duplicates SQLite's `updated_at`, creates maintenance burden.
- **Separate `cloud_retain_days`**: Rejected — unnecessary config when settings sync already solves the consistency problem.

## Consequences

- WebDAV root will contain 256 subdirectories under `entries/`. Expected file count scales linearly with entries.
- First sync on a pre-existing machine triggers a full PROPFIND + diff against local DB. One-time cost.
- Push failure backoff means entries captured while offline may take a few minutes to appear on the other machine after connectivity returns.
- No conflict resolution UX — `updated_at` comparison is automatic. Entries with identical content but different metadata keep the later timestamp.
