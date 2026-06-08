package util

// FormatInt converts int64 to string without importing fmt.
// This avoids importing the fmt package purely for number formatting
// in packages that otherwise have no need for it.
func FormatInt(i int64) string {
	if i == 0 {
		return "0"
	}
	negative := false
	if i < 0 {
		negative = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

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
