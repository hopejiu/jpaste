package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database and runs migrations.
func Open(basePath string) (*sql.DB, error) {
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(basePath, "clipboard.db")
	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)

	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return conn, nil
}

const nowMillis = "strftime('%Y-%m-%dT%H:%M:%f', 'now')"

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS clipboard_entry (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			content_hash TEXT NOT NULL UNIQUE,
			source_exe   TEXT NOT NULL DEFAULT '',
			source_title TEXT NOT NULL DEFAULT '',
			tag_mask     INTEGER NOT NULL DEFAULT 0,
			is_favorite  INTEGER NOT NULL DEFAULT 0,
			created_at   TEXT NOT NULL DEFAULT (` + nowMillis + `),
			updated_at   TEXT NOT NULL DEFAULT (` + nowMillis + `)
		);
		CREATE INDEX IF NOT EXISTS idx_entry_updated_at ON clipboard_entry(updated_at DESC, id DESC);

		CREATE TABLE IF NOT EXISTS clipboard_format (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			entry_id    INTEGER NOT NULL REFERENCES clipboard_entry(id) ON DELETE CASCADE,
			format_type INTEGER NOT NULL,
			content     TEXT,
			file_path   TEXT,
			format_hash TEXT NOT NULL,
			UNIQUE(entry_id, format_type)
		);
		CREATE INDEX IF NOT EXISTS idx_format_entry ON clipboard_format(entry_id);

		PRAGMA foreign_keys = ON;
	`)
	if err != nil {
		return err
	}

	// Migration: add is_favorite column to existing databases.
	db.Exec(`ALTER TABLE clipboard_entry ADD COLUMN is_favorite INTEGER NOT NULL DEFAULT 0`)

	return nil
}
