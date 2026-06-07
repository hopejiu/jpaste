package util

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// SHA256Hex returns the hex-encoded SHA-256 of the given data.
func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])
}

// SHA256String returns the hex-encoded SHA-256 of the trimmed string.
func SHA256String(s string) string {
	return SHA256Hex([]byte(strings.TrimSpace(s)))
}

// SHA256TextOrRaw hashes text if raw is empty, otherwise hashes raw bytes.
func SHA256TextOrRaw(text string, raw []byte) string {
	if len(raw) > 0 {
		return SHA256Hex(raw)
	}
	return SHA256Hex([]byte(text))
}
