package fileop

import "strings"

const sniffSize = 8 * 1024 // 8KB

// detectFormat sniffs the file format from content by examining the first 8KB.
// Returns the file extension without dot (e.g., "json", "xml"), or "" if
// the format cannot be determined (caller should fall back to ".txt").
func detectFormat(content string) string {
	if content == "" {
		return ""
	}

	// Limit sample to first 8KB
	sample := content
	if len(sample) > sniffSize {
		sample = sample[:sniffSize]
	}

	trimmed := strings.TrimSpace(sample)
	if trimmed == "" {
		return ""
	}

	first := trimmed[0]

	// 1. JSON: starts with { or [
	if first == '{' || first == '[' {
		return "json"
	}

	// 2. HTML (before XML to avoid <html being misclassified)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
		return "html"
	}

	// 3. XML: starts with <
	if first == '<' {
		return "xml"
	}

	// 4. SQL: statement keywords at the start
	upper := strings.ToUpper(trimmed)
	sqlKeywords := []string{
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ",
		"CREATE ", "ALTER ", "DROP ", "TRUNCATE ",
	}
	for _, kw := range sqlKeywords {
		if strings.HasPrefix(upper, kw) {
			return "sql"
		}
	}

	// 5. Markdown: heading markers
	if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") ||
		strings.HasPrefix(trimmed, "### ") {
		return "md"
	}

	// 6. INI / TOML section: [section]
	if first == '[' && strings.Contains(trimmed, "]") {
		return "ini"
	}

	// 7. CSV: multiple rows with consistent comma count
	lines := splitLines(trimmed)
	if len(lines) >= 2 {
		firstCommas := strings.Count(lines[0], ",")
		if firstCommas > 0 {
			match := true
			checked := 0
			for i := 1; i < len(lines) && checked < 10; i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}
				checked++
				if strings.Count(line, ",") != firstCommas {
					match = false
					break
				}
			}
			if match && checked > 0 {
				return "csv"
			}
		}
	}

	// 8. YAML: key: value on multiple non-comment lines
	var yamlLines int
	for _, line := range splitLines(trimmed) {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if strings.Contains(line, ": ") && !strings.HasPrefix(line, "-") {
			yamlLines++
		}
	}
	if yamlLines >= 2 {
		return "yaml"
	}

	// 9. TOML: key = value on multiple lines (skip comment and section lines)
	var tomlLines int
	for _, line := range splitLines(trimmed) {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || line[0] == '[' {
			continue
		}
		if strings.Contains(line, " = ") {
			tomlLines++
		}
	}
	if tomlLines >= 2 {
		return "toml"
	}

	return ""
}

// splitLines splits s by newline, preserving line order.
func splitLines(s string) []string {
	// Handle both \r\n and \n line endings.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}
