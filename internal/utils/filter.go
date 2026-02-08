package utils

import "strings"

// IsNoise determines if a name should be filtered out from reports
// This implements the STRICT filtering logic shared across all exporters
func IsNoise(name string) bool {
	// RULE 1: Empty or whitespace-only names
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return true
	}

	// Normalize to lowercase for case-insensitive matching
	lower := strings.ToLower(trimmed)

	// RULE 2: Java Keywords & Control Flow
	keywords := map[string]bool{
		"if":       true,
		"else":     true,
		"switch":   true,
		"case":     true,
		"for":      true,
		"while":    true,
		"do":       true,
		"return":   true,
		"new":      true,
		"throw":    true,
		"throws":   true,
		"try":      true,
		"catch":    true,
		"finally":  true,
		"break":    true,
		"continue": true,
	}

	if keywords[lower] {
		return true
	}

	// RULE 3: Exception Types (ends with "Exception")
	if strings.HasSuffix(trimmed, "Exception") {
		return true
	}

	// RULE 4: System/View Types (when used as method names)
	systemTypes := map[string]bool{
		"modelandview": true,
		"model":        true,
		"void":         true,
		"string":       true, // When used as view return
	}

	if systemTypes[lower] {
		return true
	}

	return false
}
