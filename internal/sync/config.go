package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds WebDAV connection credentials.
// Stored in %APPDATA%/jPaste/webdav.json — not synced.
type Config struct {
	URL      string `json:"url"`      // e.g. "https://dav.jianguoyun.com/dav/"
	Username string `json:"username"` // 坚果云 account
	Password string `json:"password"` // 坚果云 app password (not login password)
	Enabled  bool   `json:"enabled"`  // user-toggleable enable/disable
}

// IsValid returns true if all required fields are non-empty and enabled.
func (c Config) IsValid() bool {
	return c.URL != "" && c.Username != "" && c.Password != "" && c.Enabled
}

// loadConfig reads webdav.json from basePath, or returns an empty config.
func loadConfig(basePath string) (Config, error) {
	p := filepath.Join(basePath, "webdav.json")
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read webdav config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse webdav config: %w", err)
	}
	return c, nil
}

// saveConfig writes Config to webdav.json.
func saveConfig(basePath string, c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal webdav config: %w", err)
	}
	p := filepath.Join(basePath, "webdav.json")
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("write webdav config: %w", err)
	}
	return nil
}
