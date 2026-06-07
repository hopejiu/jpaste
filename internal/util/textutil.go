package util

// Truncate returns the first n runes of s, appending "..." if truncated.
func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n]) + "..."
	}
	return s
}

// TruncateBytes returns the first n bytes of s, appending "..." if truncated.
func TruncateBytes(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
