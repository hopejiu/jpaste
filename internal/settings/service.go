package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Data holds user-configurable settings.
type Data struct {
	Hotkey         string          `json:"hotkey"`                   // "Alt+V"
	RetainDays     int             `json:"retain_days"`              // 30
	DefaultAction  string          `json:"default_action"`           // "copy" or "paste"
	AutoStart      bool            `json:"auto_start"`
	StartMinimized bool            `json:"start_minimized"`          // start minimized to tray
	NotifyEnabled  bool            `json:"notify_enabled"`           // show toast on new clipboard
	PasteOrder     string          `json:"paste_order"`              // "normal" / "stack" / "queue"
	ActionConfig   json.RawMessage `json:"action_config,omitempty"`  // frontend-managed, Go pass-through
	SortField      string          `json:"sort_field"`               // "updated_at" / "content_length"
	SortOrder      string          `json:"sort_order"`               // "asc" / "desc"
	Theme          string          `json:"theme"`                    // "a" (冷调极简) / "b" (暖调高效) / "c" (深色沉浸)
}

// SettingsReader provides read-only access to settings.
type SettingsReader interface {
	GetSettings() Data
}

// Service reads and writes settings from a JSON file.
type Service struct {
	mu                sync.RWMutex
	path              string
	data              Data
	onHotkeyChange    func(old, new string) error
	onSettingsChange  func(old, new Data)
}

// NewService creates a SettingsService. Call Load before use.
func NewService(basePath string) *Service {
	return &Service{
		path: filepath.Join(basePath, "settings.json"),
		data: Data{
			Hotkey:         "Alt+V",
			RetainDays:     30,
			DefaultAction:  "copy",
			AutoStart:      false,
			StartMinimized: false,
			NotifyEnabled:  false,
			PasteOrder:     "normal",
			ActionConfig:   json.RawMessage("{}"),
			SortField:      "updated_at",
			SortOrder:      "desc",
			Theme:          "a",
		},
	}
}

// OnHotkeyChange sets a callback invoked when the hotkey setting changes.
// The callback receives the old and new hotkey strings.
// If it returns an error, the hotkey change is rejected and SaveSettings
// returns that error to the frontend.
func (s *Service) OnHotkeyChange(fn func(old, new string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onHotkeyChange = fn
}

// OnSettingsChange sets a callback invoked when any non-hotkey setting changes.
func (s *Service) OnSettingsChange(fn func(old, new Data)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSettingsChange = fn
}

// Defaults returns a fresh settings struct with defaults.
func Defaults() Data {
	return Data{
		Hotkey:         "Alt+V",
		RetainDays:     30,
		DefaultAction:  "copy",
		AutoStart:      false,
		StartMinimized: false,
		NotifyEnabled:  false,
		PasteOrder:     "normal",
		SortField:      "updated_at",
		SortOrder:      "desc",
		Theme:          "a",
	}
}

// Load reads settings from disk, falling back to defaults on error.
func (s *Service) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = Defaults()
			return nil
		}
		return fmt.Errorf("read settings: %w", err)
	}

	loaded := Defaults()
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse settings: %w", err)
	}
	s.data = loaded
	return nil
}

// Save writes settings to disk.
func (s *Service) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

// GetSettings returns the current settings for the frontend.
func (s *Service) GetSettings() Data {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// SaveSettings updates and persists settings.
// If the hotkey has changed, the onHotkeyChange callback is invoked first.
// If the callback returns an error, the change is rejected and returned.
func (s *Service) SaveSettings(d Data) error {
	s.mu.Lock()
	oldData := s.data
	oldHotkey := s.data.Hotkey
	hkCB := s.onHotkeyChange
	scCB := s.onSettingsChange

	newHotkey := d.Hotkey
	hotkeyChanged := oldHotkey != newHotkey

	if hotkeyChanged && hkCB != nil {
		// Release lock while the callback may do I/O (system hotkey registration).
		s.mu.Unlock()
		err := hkCB(oldHotkey, newHotkey)
		s.mu.Lock()
		if err != nil {
			s.mu.Unlock()
			return err
		}
	}

	s.data = d
	s.mu.Unlock()

	// Notify general settings change for other fields.
	if scCB != nil && changedExceptHotkey(oldData, d) {
		scCB(oldData, d)
	}
	return s.Save()
}

// changedExceptHotkey returns true if any field other than Hotkey has changed.
func changedExceptHotkey(a, b Data) bool {
	return a.RetainDays != b.RetainDays ||
		a.DefaultAction != b.DefaultAction ||
		a.AutoStart != b.AutoStart ||
		a.StartMinimized != b.StartMinimized ||
		a.NotifyEnabled != b.NotifyEnabled ||
		a.PasteOrder != b.PasteOrder ||
		a.SortField != b.SortField ||
		a.SortOrder != b.SortOrder ||
		a.Theme != b.Theme
}
