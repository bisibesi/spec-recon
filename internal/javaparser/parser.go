package javaparser

import (
	"regexp"
	"strings"

	"spec-recon/internal/logger"
)

// Annotation represents a Java annotation with its attributes
type Annotation struct {
	Name       string            // e.g., "RequestMapping", "Autowired"
	Attributes map[string]string // e.g., {"value": "/users", "method": "GET"}
	Raw        string            // Original annotation text
}

// Field represents a class field (for dependency injection detection)
type Field struct {
	Name        string       // e.g., "userService"
	Type        string       // e.g., "UserService"
	Annotations []Annotation // e.g., @Autowired
}

// Method represents a Java method
type Method struct {
	Name        string       // e.g., "login"
	Params      string       // e.g., "String username, String password"
	ParamsList  []string     // e.g., ["String username", "String password"]
	ReturnType  string       // e.g., "ModelAndView", "ResponseEntity<String>"
	Annotations []Annotation // e.g., @PostMapping("/login")
	JavaDoc     string       // Method documentation
	Body        string       // Method body (for call tracing)
}

// JavaClass represents a parsed Java class
type JavaClass struct {
	Package     string       // e.g., "com.company.legacy"
	Name        string       // e.g., "UserController"
	Imports     []string     // Import statements
	Annotations []Annotation // Class-level annotations
	Fields      []Field      // Class fields (for @Autowired detection)
	Methods     []Method     // Class methods
}

// ParseJavaFile parses a Java source file and extracts metadata
func ParseJavaFile(content string) (*JavaClass, error) {
	javaClass := &JavaClass{
		Imports:     []string{},
		Annotations: []Annotation{},
		Fields:      []Field{},
		Methods:     []Method{},
	}

	// Extract package
	javaClass.Package = extractPackage(content)

	// Extract class name
	javaClass.Name = extractClassName(content)

	// Extract imports
	javaClass.Imports = extractImports(content)

	// Extract class-level annotations
	javaClass.Annotations = extractClassAnnotations(content)

	// Extract fields (for @Autowired detection)
	javaClass.Fields = extractFields(content)

	// Extract methods
	javaClass.Methods = extractMethods(content)

	return javaClass, nil
}

// extractPackage extracts the package declaration
func extractPackage(content string) string {
	packageRegex := regexp.MustCompile(`package\s+([\w.]+)\s*;`)
	matches := packageRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractClassName extracts the class name from class declaration
func extractClassName(content string) string {
	// Match: public class ClassName or @Controller public class ClassName, or public interface
	classRegex := regexp.MustCompile(`(?:public\s+)?(?:class|interface)\s+(\w+)`)
	matches := classRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractImports extracts all import statements
func extractImports(content string) []string {
	imports := []string{}
	importRegex := regexp.MustCompile(`import\s+([\w.]+)\s*;`)
	matches := importRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}
	return imports
}

// extractClassAnnotations extracts class-level annotations
func extractClassAnnotations(content string) []Annotation {
	annotations := []Annotation{}

	// Find class declaration position
	classRegex := regexp.MustCompile(`(?:public\s+)?(?:class|interface)\s+\w+`)
	classMatch := classRegex.FindStringIndex(content)
	if classMatch == nil {
		return annotations
	}

	// Look for annotations before class declaration
	beforeClass := content[:classMatch[0]]

	// Extract annotations
	annotationRegex := regexp.MustCompile(`@(\w+)(?:\s*\(([^)]*)\))?`)
	matches := annotationRegex.FindAllStringSubmatch(beforeClass, -1)

	for _, match := range matches {
		annotation := Annotation{
			Name:       match[1],
			Attributes: make(map[string]string),
			Raw:        match[0],
		}

		// Parse attributes if present
		if len(match) > 2 && match[2] != "" {
			parseAnnotationAttributes(&annotation, match[2])
		}

		annotations = append(annotations, annotation)
	}

	return annotations
}

// extractFields extracts class fields (for @Autowired detection)
func extractFields(content string) []Field {
	fields := []Field{}

	// Pattern: @Annotation? modifiers Type Name (= ...)?;
	// Supports:
	// - @Autowired private List<String> name;
	// - private static final Map<String, Object> map = new HashMap<>();
	// - protected String[] array;
	fieldRegex := regexp.MustCompile(`(?s)(@\w+(?:\([^)]*\))?\s+)?(?:private|public|protected)(?:\s+(?:static|final|transient|volatile))*\s+([\w<>,\s?\[\]]+)\s+(\w+)\s*(?:=.*?)?;`)
	matches := fieldRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		rawType := strings.TrimSpace(match[2])
		fieldName := match[3]

		field := Field{
			Type:        rawType,
			Name:        fieldName,
			Annotations: []Annotation{},
		}

		// Check if field has @Autowired annotation
		if match[1] != "" {
			annotationText := strings.TrimSpace(match[1])
			if strings.HasPrefix(annotationText, "@") {
				// Simple parsing for annotation name (removes attributes)
				parts := strings.Split(annotationText, "(")
				nameRaw := parts[0]
				annotationName := strings.TrimPrefix(nameRaw, "@")

				field.Annotations = append(field.Annotations, Annotation{
					Name: annotationName,
					Raw:  annotationText,
				})
			}
		}

		fields = append(fields, field)
	}

	return fields
}

// extractMethods extracts all methods from the class with their bodies
func extractMethods(content string) []Method {
	methods := []Method{}

	// Find method signatures and match with their bodies or terminator (;)
	// Pattern supports both { (body) and ; (abstract/interface)
	// Groups:
	// 1: Annotations
	// 2: Access Modifier (optional for interfaces)
	// 3: Return Type
	// 4: Method Name
	// 5: Params
	// 6: Body Start ({) OR Terminator (;)

	// Complex regex explanation:
	// (?s) : dot matches newline
	// ((?:@\w+(?:\([^)]*\))?\s+)*) : Group 1 - Annotations
	// (public|private|protected)?\s* : Group 2 - Access modifier (can be empty/implicit in interface)
	// ([\w<>,\[\]\s]+)\s+ : Group 3 - Return Type
	// (\w+)\s* : Group 4 - Method Name
	// \(([^)]*)\)\s* : Group 5 - Params
	// (?:throws\s+[\w,\s]+)?\s* : throws declaration
	// (\{|;) : Group 6 - Body start brace OR semi-colon

	methodRegex := regexp.MustCompile(`(?s)((?:@\w+(?:\([^)]*\))?\s+)*)(public|private|protected)?\s*([\w<>,\[\]\s]+)\s+(\w+)\s*\(([^)]*)\)\s*(?:throws\s+[\w,\s]+)?\s*(\{|;)`)

	matches := methodRegex.FindAllStringSubmatchIndex(content, -1)

	for _, matchIdx := range matches {
		// Extract matched groups using indices
		annotationsText := content[matchIdx[2]:matchIdx[3]] // Group 1
		// Group 2 is access modifier, we ignore it for now
		returnType := strings.TrimSpace(content[matchIdx[6]:matchIdx[7]]) // Group 3
		methodName := content[matchIdx[8]:matchIdx[9]]                    // Group 4
		params := strings.TrimSpace(content[matchIdx[10]:matchIdx[11]])   // Group 5

		terminatorOrBrace := content[matchIdx[12]:matchIdx[13]] // Group 6

		methodBody := ""
		if terminatorOrBrace == "{" {
			// Find closing brace
			bodyStart := matchIdx[13] // Position right after the opening {
			bodyEnd := findClosingBrace(content, bodyStart)
			if bodyEnd > bodyStart && bodyEnd <= len(content) {
				methodBody = content[bodyStart : bodyEnd-1]
			}
			logger.Debug("[PARSER] Captured Body for %s: %d chars", methodName, len(methodBody))
		}

		method := Method{
			ReturnType:  returnType,
			Name:        methodName,
			Params:      params,
			Body:        methodBody, // Check later if we need to store it
			Annotations: []Annotation{},
		}

		// Parse parameters
		if method.Params != "" {
			method.ParamsList = parseMethodParams(method.Params)
		}

		// Parse annotations
		if annotationsText != "" {
			method.Annotations = parseMethodAnnotations(annotationsText)
		}

		methods = append(methods, method)
	}

	return methods
}

// findClosingBrace finds the matching closing brace for an opening brace
// Handles nested braces, strings, chars, and comments
func findClosingBrace(content string, start int) int {
	depth := 1
	inString := false
	inChar := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i := start; i < len(content); i++ {
		char := content[i]

		if inLineComment {
			if char == '\n' {
				inLineComment = false
			}
			continue
		}

		if inBlockComment {
			if char == '*' && i+1 < len(content) && content[i+1] == '/' {
				inBlockComment = false
				i++ // Skip /
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		if inChar {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '\'' {
				inChar = false
			}
			continue
		}

		// Check for comments start
		if char == '/' && i+1 < len(content) {
			if content[i+1] == '/' {
				inLineComment = true
				i++
				continue
			}
			if content[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		if char == '"' {
			inString = true
			escaped = false
			continue
		}

		if char == '\'' {
			inChar = true
			escaped = false
			continue
		}

		if char == '{' {
			depth++
		} else if char == '}' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(content)
}

// parseMethodAnnotations parses method-level annotations
func parseMethodAnnotations(annotationsText string) []Annotation {
	annotations := []Annotation{}

	// Match individual annotations
	annotationRegex := regexp.MustCompile(`@(\w+)(?:\s*\(([^)]*)\))?`)
	matches := annotationRegex.FindAllStringSubmatch(annotationsText, -1)

	for _, match := range matches {
		annotation := Annotation{
			Name:       match[1],
			Attributes: make(map[string]string),
			Raw:        match[0],
		}

		// Parse attributes if present
		if len(match) > 2 && match[2] != "" {
			parseAnnotationAttributes(&annotation, match[2])
		}

		annotations = append(annotations, annotation)
	}

	return annotations
}

// parseAnnotationAttributes parses annotation attributes into a map
func parseAnnotationAttributes(annotation *Annotation, attributesText string) {
	attributesText = strings.TrimSpace(attributesText)

	// Check for simple value: @Annotation("value")
	if strings.HasPrefix(attributesText, "\"") || strings.HasPrefix(attributesText, "'") {
		annotation.Attributes["value"] = trimQuotes(attributesText)
		return
	}

	// Parse key-value pairs: value = "/users", method = RequestMethod.POST
	attrRegex := regexp.MustCompile(`(\w+)\s*=\s*([^,]+)`)
	matches := attrRegex.FindAllStringSubmatch(attributesText, -1)

	for _, match := range matches {
		key := match[1]
		value := strings.TrimSpace(match[2])
		value = trimQuotes(value)
		annotation.Attributes[key] = value
	}
}

// Helper functions

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func parseMethodParams(params string) []string {
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

func extractAnnotationValue(annotation string) string {
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

func combineURLPaths(classPath, methodPath string) string {
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

// GetClassLevelURL returns the class-level URL from @RequestMapping
func (jc *JavaClass) GetClassLevelURL() string {
	for _, ann := range jc.Annotations {
		if ann.Name == "RequestMapping" {
			if value, ok := ann.Attributes["value"]; ok {
				return value
			}
			// Try to extract from raw annotation
			return extractAnnotationValue(ann.Raw)
		}
	}
	return ""
}

// IsController checks if the class is a Spring Controller
func (jc *JavaClass) IsController() bool {
	for _, ann := range jc.Annotations {
		if ann.Name == "Controller" || ann.Name == "RestController" {
			return true
		}
	}
	return false
}

// IsService checks if the class is a Spring Service
func (jc *JavaClass) IsService() bool {
	for _, ann := range jc.Annotations {
		if ann.Name == "Service" {
			return true
		}
	}
	return false
}

// GetInjectedServices returns the names of @Autowired service fields
func (jc *JavaClass) GetInjectedServices() []string {
	services := []string{}
	for _, field := range jc.Fields {
		for _, ann := range field.Annotations {
			if ann.Name == "Autowired" {
				services = append(services, field.Name)
			}
		}
	}
	return services
}

// GetMethodURL returns the full URL for a method (class path + method path)
func (m *Method) GetMethodURL(classPath string) string {
	methodPath := ""

	for _, ann := range m.Annotations {
		// Check for mapping annotations
		if strings.HasSuffix(ann.Name, "Mapping") {
			if value, ok := ann.Attributes["value"]; ok {
				methodPath = value
				break
			}
			// Try to extract from raw
			methodPath = extractAnnotationValue(ann.Raw)
			if methodPath != "" {
				break
			}
		}
	}

	return combineURLPaths(classPath, methodPath)
}

// GetHTTPMethod returns the HTTP method for this method
func (m *Method) GetHTTPMethod() string {
	for _, ann := range m.Annotations {
		switch ann.Name {
		case "GetMapping":
			return "GET"
		case "PostMapping":
			return "POST"
		case "PutMapping":
			return "PUT"
		case "DeleteMapping":
			return "DELETE"
		case "PatchMapping":
			return "PATCH"
		case "RequestMapping":
			// Check method attribute
			if method, ok := ann.Attributes["method"]; ok {
				// Handle "RequestMethod.POST"
				if strings.Contains(method, ".") {
					parts := strings.Split(method, ".")
					method = parts[len(parts)-1]
				}
				return strings.ToUpper(method)
			}
			return "GET" // Default
		}
	}
	return ""
}

// IsEndpoint checks if this method is an HTTP endpoint
func (m *Method) IsEndpoint() bool {
	for _, ann := range m.Annotations {
		if strings.HasSuffix(ann.Name, "Mapping") {
			return true
		}
	}
	return false
}
