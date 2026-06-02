//go:build windows

package hotkey

import (
	"errors"
	"log"
	"strings"

	"golang.design/x/hotkey"
)

var hk *hotkey.Hotkey

// Register registers a system-level global hotkey. format: "Alt+V", "Ctrl+Shift+A".
func Register(keystr string, callback func()) error {
	mods, key, err := parse(keystr)
	if err != nil {
		log.Printf("[hotkey] parse %q failed: %v", keystr, err)
		return err
	}

	log.Printf("[hotkey] Registering %q (mods=%v key=%v)", keystr, mods, key)
	hk = hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Printf("[hotkey] Register %q failed: %v", keystr, err)
		return err
	}

	log.Printf("[hotkey] %q registered successfully", keystr)
	go func() {
		for range hk.Keydown() {
			log.Println("[hotkey] Keydown triggered, calling callback")
			callback()
		}
	}()

	return nil
}

// UnregisterAll unregisters the hotkey.
func UnregisterAll() {
	log.Println("[hotkey] UnregisterAll called")
	if hk != nil {
		hk.Unregister()
		hk = nil
	}
}

func parse(s string) ([]hotkey.Modifier, hotkey.Key, error) {
	var mods []hotkey.Modifier
	var key hotkey.Key

	parts := strings.Split(s, "+")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch strings.ToLower(p) {
		case "ctrl", "control":
			mods = append(mods, hotkey.ModCtrl)
		case "alt":
			mods = append(mods, hotkey.ModAlt)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		case "win", "windows", "cmd":
			mods = append(mods, hotkey.ModWin)
		default:
			key = parseKey(p)
		}
	}

		if key == 0 {
			return nil, 0, errors.New("hotkey: invalid key")
		}
	return mods, key, nil
}

func parseKey(s string) hotkey.Key {
	s = strings.ToUpper(s)
	if len(s) == 1 {
		ch := s[0]
		if ch >= 'A' && ch <= 'Z' {
			return hotkey.Key(ch)
		}
		if ch >= '0' && ch <= '9' {
			return hotkey.Key(ch)
		}
	}
	// Function keys mapping.
	switch s {
	case "F1":
		return hotkey.KeyF1
	case "F2":
		return hotkey.KeyF2
	case "F3":
		return hotkey.KeyF3
	case "F4":
		return hotkey.KeyF4
	case "F5":
		return hotkey.KeyF5
	case "F6":
		return hotkey.KeyF6
	case "F7":
		return hotkey.KeyF7
	case "F8":
		return hotkey.KeyF8
	case "F9":
		return hotkey.KeyF9
	case "F10":
		return hotkey.KeyF10
	case "F11":
		return hotkey.KeyF11
	case "F12":
		return hotkey.KeyF12
	}
	return 0
}
