package linker

import (
	"testing"
)

// TestJavaIdentifierValidation tests the isValidJavaIdentifier function
func TestJavaIdentifierValidation(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid identifiers
		{"Valid simple name", "userName", true},
		{"Valid with underscore", "user_name", true},
		{"Valid with dollar", "$variable", true},
		{"Valid starting with underscore", "_private", true},
		{"Valid camelCase", "getUserName", true},
		{"Valid PascalCase", "UserService", true},

		// Invalid identifiers (noise that should be filtered)
		{"Invalid with space", "if (user", false},
		{"Invalid with parentheses", "switch (value)", false},
		{"Invalid with operator", "return value", false},
		{"Invalid starting with digit", "123invalid", false},
		{"Invalid with dot", "user.name", false},
		{"Invalid with equals", "x = 5", false},
		{"Invalid with comparison", "x == null", false},
		{"Invalid empty string", "", false},
		{"Invalid with brackets", "array[0]", false},
		{"Invalid with semicolon", "statement;", false},

		// Edge cases
		{"Single letter", "x", true},
		{"Single underscore", "_", true},
		{"Single dollar", "$", true},
		{"Mixed valid chars", "_$validName123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidJavaIdentifier(tc.input)
			if result != tc.expected {
				t.Errorf("isValidJavaIdentifier(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestNoiseFilterWithInvalidIdentifiers tests that invalid identifiers are filtered out
func TestNoiseFilterWithInvalidIdentifiers(t *testing.T) {
	// This test verifies that the noise filter catches identifiers with
	// spaces, parentheses, and operators

	invalidIdentifiers := []string{
		"if (condition)",
		"switch (value)",
		"return result",
		"x == null",
		"user.getName()",
		"array[index]",
	}

	for _, invalid := range invalidIdentifiers {
		if isValidJavaIdentifier(invalid) {
			t.Errorf("Expected %q to be invalid, but it was accepted", invalid)
		}
	}
}
