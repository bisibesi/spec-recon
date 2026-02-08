package analyzer

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

// ReadFile reads a file with automatic encoding detection
// Supports UTF-8 and EUC-KR/CP949 encoding
// For Java files, comments are removed to prevent regex false positives
// For XML files, comments are preserved
func ReadFile(path string) (string, error) {
	// Read raw bytes
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Try UTF-8 first
	content := string(rawBytes)
	if utf8.Valid(rawBytes) {
		// Valid UTF-8, check if we should remove comments
		if IsJavaFile(path) {
			return removeComments(content), nil
		}
		return content, nil
	}

	// Not valid UTF-8, try EUC-KR decoding
	decoder := korean.EUCKR.NewDecoder()
	decodedBytes, _, err := transform.Bytes(decoder, rawBytes)
	if err != nil {
		// If EUC-KR fails, fall back to original (might be corrupted)
		if IsJavaFile(path) {
			return removeComments(content), nil
		}
		return content, nil
	}

	content = string(decodedBytes)
	if IsJavaFile(path) {
		return removeComments(content), nil
	}
	return content, nil
}

// removeComments removes Java comments to prevent regex false positives
func removeComments(content string) string {
	// Remove multi-line comments: /* ... */ and /** ... */
	multiLineCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	content = multiLineCommentRegex.ReplaceAllString(content, "")

	// Remove single-line comments: //
	singleLineCommentRegex := regexp.MustCompile(`//.*`)
	content = singleLineCommentRegex.ReplaceAllString(content, "")

	return content
}

// NormalizeWhitespace reduces multiple consecutive whitespace to single space
func NormalizeWhitespace(s string) string {
	// Replace multiple spaces/tabs/newlines with single space
	wsRegex := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(wsRegex.ReplaceAllString(s, " "))
}

// ExtractAnnotationValue extracts the value from an annotation
// Examples:
//   - @RequestMapping("/users") -> "/users"
//   - @RequestMapping(value = "/users") -> "/users"
//   - @PostMapping(value = "/save", method = RequestMethod.POST) -> "/save"
func ExtractAnnotationValue(annotation string) string {
	// Try simple pattern: @Annotation("value")
	simpleRegex := regexp.MustCompile(`@\w+\s*\(\s*"([^"]+)"\s*\)`)
	if matches := simpleRegex.FindStringSubmatch(annotation); len(matches) > 1 {
		return matches[1]
	}

	// Try value= pattern: @Annotation(value = "value")
	valueRegex := regexp.MustCompile(`value\s*=\s*"([^"]+)"`)
	if matches := valueRegex.FindStringSubmatch(annotation); len(matches) > 1 {
		return matches[1]
	}

	// Try path= pattern: @Annotation(path = "value")
	pathRegex := regexp.MustCompile(`path\s*=\s*"([^"]+)"`)
	if matches := pathRegex.FindStringSubmatch(annotation); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// CombineURLPaths combines class-level and method-level URL paths
func CombineURLPaths(classPath, methodPath string) string {
	classPath = strings.TrimSpace(classPath)
	methodPath = strings.TrimSpace(methodPath)

	if classPath == "" {
		return methodPath
	}
	if methodPath == "" {
		return classPath
	}

	// Ensure no double slashes
	classPath = strings.TrimSuffix(classPath, "/")
	methodPath = strings.TrimPrefix(methodPath, "/")

	return classPath + "/" + methodPath
}

// IsJavaFile checks if a file is a Java source file
func IsJavaFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".java")
}

// IsXMLFile checks if a file is an XML file
func IsXMLFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".xml")
}

// TrimQuotes removes surrounding quotes from a string
func TrimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// CleanSQLContent cleans up SQL content by normalizing whitespace
func CleanSQLContent(sql string) string {
	// Remove excessive whitespace
	sql = NormalizeWhitespace(sql)

	// Remove leading/trailing whitespace from each line
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, " ")
}

// ExtractGenericType extracts the generic type from a type string
// Example: "ResponseEntity<String>" -> "String"
func ExtractGenericType(typeStr string) string {
	genericRegex := regexp.MustCompile(`<([^>]+)>`)
	if matches := genericRegex.FindStringSubmatch(typeStr); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// SplitPackageAndClass splits a fully qualified class name
// Example: "com.company.UserController" -> ("com.company", "UserController")
func SplitPackageAndClass(fqcn string) (pkg, class string) {
	lastDot := strings.LastIndex(fqcn, ".")
	if lastDot == -1 {
		return "", fqcn
	}
	return fqcn[:lastDot], fqcn[lastDot+1:]
}

// RemoveMultipleSpaces removes multiple consecutive spaces
func RemoveMultipleSpaces(s string) string {
	spaceRegex := regexp.MustCompile(` +`)
	return spaceRegex.ReplaceAllString(s, " ")
}

// StripHTMLTags removes HTML tags from a string (for JavaDoc)
func StripHTMLTags(s string) string {
	htmlRegex := regexp.MustCompile(`<[^>]+>`)
	return htmlRegex.ReplaceAllString(s, "")
}

// ParseMethodParams parses method parameters into a slice
// Example: "String name, int age" -> ["String name", "int age"]
func ParseMethodParams(params string) []string {
	params = strings.TrimSpace(params)
	if params == "" {
		return []string{}
	}

	// Split by comma, but respect generics
	result := []string{}
	current := ""
	depth := 0

	for _, char := range params {
		if char == '<' {
			depth++
		} else if char == '>' {
			depth--
		} else if char == ',' && depth == 0 {
			result = append(result, strings.TrimSpace(current))
			current = ""
			continue
		}
		current += string(char)
	}

	if current != "" {
		result = append(result, strings.TrimSpace(current))
	}

	return result
}

// IsValidContent checks if the content seems to be valid (not empty or just whitespace)
func IsValidContent(content string) bool {
	return len(strings.TrimSpace(content)) > 0
}

// DetectEncoding attempts to detect if content is UTF-8 or EUC-KR
func DetectEncoding(data []byte) string {
	if utf8.Valid(data) {
		return "UTF-8"
	}
	return "EUC-KR"
}

// BytesToString safely converts bytes to string with encoding detection
func BytesToString(data []byte) (string, error) {
	if utf8.Valid(data) {
		return string(data), nil
	}

	// Try EUC-KR
	decoder := korean.EUCKR.NewDecoder()
	decoded, _, err := transform.Bytes(decoder, data)
	if err != nil {
		return string(data), fmt.Errorf("encoding detection failed: %w", err)
	}

	return string(decoded), nil
}
