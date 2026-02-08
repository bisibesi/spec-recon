package linker

import (
	"fmt"
	"strings"

	"spec-recon/internal/javaparser"
	"spec-recon/internal/model"
	"spec-recon/internal/xmlparser"
)

// IgnoredTokens is a blocklist of Java keywords and common constructs that should not be traced as method calls
var IgnoredTokens = map[string]bool{
	// Java keywords
	"if":           true,
	"else":         true,
	"for":          true,
	"while":        true,
	"switch":       true,
	"case":         true,
	"try":          true,
	"catch":        true,
	"finally":      true,
	"synchronized": true,
	"return":       true,
	"throw":        true,
	"throws":       true,
	"assert":       true,
	"break":        true,
	"continue":     true,
	"do":           true,

	// Constructors and object creation
	"new":   true,
	"this":  true,
	"super": true,

	// Common library methods (logging, I/O)
	"println": true,
	"print":   true,
	"printf":  true,
	"info":    true,
	"debug":   true,
	"warn":    true,
	"error":   true,
	"trace":   true,
	"log":     true,
	"out":     true,
	"err":     true,

	// Common Java classes that are usually noise
	"System":    true,
	"String":    true,
	"Integer":   true,
	"Long":      true,
	"Double":    true,
	"Boolean":   true,
	"Object":    true,
	"Class":     true,
	"Exception": true,

	// Common Spring/View classes (usually not business logic)
	"ModelAndView": true,
	"Model":        true,
	"View":         true,
	"RedirectView": true,
}

// Linker orchestrates the creation of the call graph
type Linker struct {
	Pool *ComponentPool
}

// NewLinker creates a new Linker
func NewLinker(pool *ComponentPool) *Linker {
	return &Linker{
		Pool: pool,
	}
}

// BuildCallGraph orchestrates linking and returns the node tree/graph
func (l *Linker) BuildCallGraph() []*model.Node {
	l.Link()
	return l.GetAllNodes()
}

// LoadJavaClasses loads parsed Java classes into the pool
func (l *Linker) LoadJavaClasses(classes []*javaparser.JavaClass, sourceContents map[string]string) error {
	for _, cls := range classes {
		fullClassName := cls.Package + "." + cls.Name
		content := sourceContents[fullClassName]
		if err := l.Pool.AddJavaClass(cls, content); err != nil {
			return err
		}
	}
	return nil
}

// LoadMapperXMLs loads parsed XML mappers into the pool
func (l *Linker) LoadMapperXMLs(mappers []*xmlparser.MapperXML) error {
	for _, mapper := range mappers {
		if err := l.Pool.AddMapperXML(mapper); err != nil {
			return err
		}
	}
	return nil
}

// Link performs the linking process to build the call graph
func (l *Linker) Link() error {
	// 1. Link Java Methods (heuristic call tracing)
	if err := l.linkJavaMethods(); err != nil {
		return err
	}

	// 2. Link Mappers to XML
	if err := l.linkMappersToXML(); err != nil {
		return err
	}

	return nil
}

// GetAllNodes returns all nodes in the graph
func (l *Linker) GetAllNodes() []*model.Node {
	nodes := []*model.Node{}

	// Add classes
	for _, node := range l.Pool.ClassMap {
		nodes = append(nodes, node)
	}

	return nodes
}

// linkJavaMethods trace calls within method bodies
func (l *Linker) linkJavaMethods() error {
	for methodKey, methodNode := range l.Pool.MethodMap {
		// Get method body
		body := l.Pool.MethodBodyMap[methodKey]
		if body == "" {
			continue // No body (interface or abstract), nothing to trace
		}

		// Parse calls in the body
		calls := FindMethodCalls(body)

		// Reconstruct FullClassName from methodKey (package.Class.method)
		// Assuming methodKey is created as FullClassName + "." + MethodName
		lastDot := strings.LastIndex(methodKey, ".")
		if lastDot == -1 {
			continue
		}
		fullClassName := methodKey[:lastDot]

		for _, call := range calls {
			// JAVA IDENTIFIER VALIDATION: Reject invalid identifiers
			// This catches "if (...)", "switch (...)", "return ...", etc.
			if !isValidJavaIdentifier(call.Variable) || !isValidJavaIdentifier(call.MethodName) {
				continue
			}

			// INVALID TOKEN CHECK: Prefix-based keyword detection
			// This catches "if", "throw", "new", etc. even in context
			if IsInvalidToken(call.Variable) || IsInvalidToken(call.MethodName) {
				continue
			}

			// NOISE FILTER: Skip Java keywords and common constructs
			if IgnoredTokens[call.Variable] || IgnoredTokens[call.MethodName] {
				continue
			}

			// STRICT METHOD CALL VALIDATION: Block exceptions, keywords, invalid constructors
			if !IsValidMethodCall(call.Variable) || !IsValidMethodCall(call.MethodName) {
				continue
			}

			// CONSTRUCTOR FILTER: Skip "new ClassName()" patterns
			// If this is a static call and the variable starts with uppercase,
			// it might be a constructor. We skip these unless they're known business components.
			if call.IsStatic && isConstructorCall(call.Variable) {
				continue
			}

			// RETURN TYPE FILTER: Skip constructor calls matching the method's return type
			// E.g., if method returns ModelAndView, skip "ModelAndView" as a child call
			if call.IsStatic && methodNode.ReturnDetail != "" {
				// Extract simple type name from return type (remove generics, packages)
				returnType := methodNode.ReturnDetail
				if idx := strings.Index(returnType, "<"); idx > 0 {
					returnType = returnType[:idx]
				}
				if idx := strings.LastIndex(returnType, "."); idx >= 0 {
					returnType = returnType[idx+1:]
				}

				// If the call variable matches the return type, it's likely a constructor
				if call.Variable == returnType {
					fmt.Printf("[LINKER DROP] Ignored constructor matching return type: %s\n", call.Variable)
					continue
				}
			}

			var targetNodes []*model.Node

			if call.IsStatic {
				// Static call: Variable is ClassName
				targetClass := l.findClassBySimpleName(call.Variable)
				if targetClass != "" {
					// DATA CLASS FILTER: Skip data structures (DTO, VO, Model, Entity, etc.)
					if IsDataClass(targetClass) {
						fmt.Printf("[LINKER SKIP] Data Class ignored: %s\n", targetClass)
						continue
					}
					targetNodes = l.Pool.FindMethodByName(targetClass, call.MethodName)
				}

			} else {
				// Instance call: Variable is variable name
				// 1. Resolve variable type
				variableType := l.Pool.ResolveFieldType(fullClassName, call.Variable)

				if variableType != "" {
					// DATA CLASS FILTER: Skip data structures (DTO, VO, Model, Entity, etc.)
					if IsDataClass(variableType) {
						fmt.Printf("[LINKER SKIP] Data Class ignored: %s\n", variableType)
						continue
					}
					// 2. Find method in target type
					targetNodes = l.Pool.FindMethodByName(variableType, call.MethodName)
				}
			}

			// Link found targets
			for _, target := range targetNodes {
				methodNode.AddChild(target)
			}
		}
	}
	return nil
}

// linkMappersToXML links Mapper interface methods to XML SQL nodes
func (l *Linker) linkMappersToXML() error {
	for methodKey, methodNode := range l.Pool.MethodMap {
		// Check if this method belongs to a Mapper interface

		lastDot := strings.LastIndex(methodKey, ".")
		if lastDot == -1 {
			continue
		}
		fullClassName := methodKey[:lastDot]

		classNode := l.Pool.ClassMap[fullClassName]

		if classNode != nil && classNode.Type == model.NodeTypeMapper {
			// This is a mapper method. Look for matching XML.
			// Try exact match first
			sqlNode := l.Pool.GetSQL(fullClassName, methodNode.Method)
			if sqlNode != nil {
				methodNode.AddChild(sqlNode)
			}
		}
	}
	return nil
}

// Helper to find class by simple name (e.g., "StringUtil")
func (l *Linker) findClassBySimpleName(simpleName string) string {
	// 1. Check exact match in map (unlikely unless full name used)
	if _, ok := l.Pool.ClassMap[simpleName]; ok {
		return simpleName
	}

	// 2. Scan all classes
	for fullName := range l.Pool.ClassMap {
		// Use helper from pool.go (it IS available in same package)
		if extractSimpleTypeName(fullName) == simpleName {
			return fullName
		}
	}
	return ""
}

// isConstructorCall checks if a variable name looks like a constructor call
// (e.g., ModelAndView, ResponseEntity, etc.)
func isConstructorCall(varName string) bool {
	// If it's in the ignored tokens list, it's likely a constructor we want to skip
	if IgnoredTokens[varName] {
		return true
	}

	// Additional heuristic: Common Spring/Java classes that are typically constructors
	// These are classes we create but don't want to trace as business logic
	constructorPatterns := []string{
		"ResponseEntity",
		"HttpEntity",
		"ArrayList",
		"HashMap",
		"HashSet",
		"LinkedList",
		"TreeMap",
		"TreeSet",
		"Date",
		"SimpleDateFormat",
		"StringBuilder",
		"StringBuffer",
	}

	for _, pattern := range constructorPatterns {
		if varName == pattern {
			return true
		}
	}

	return false
}

// isValidJavaIdentifier checks if a string is a valid Java identifier
// Rejects strings with spaces, parentheses, operators, etc.
func isValidJavaIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Java identifier must match: ^[a-zA-Z_$][a-zA-Z0-9_$]*$
	// This rejects "if (...)", "switch (...)", "return value", etc.
	for i, ch := range name {
		if i == 0 {
			// First character: must be letter, underscore, or dollar sign
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '$') {
				return false
			}
		} else {
			// Subsequent characters: letter, digit, underscore, or dollar sign
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$') {
				return false
			}
		}
	}

	return true
}

// IsInvalidToken performs prefix-based validation to catch keywords in context
// This catches cases like "if (...)", "throw new", "switch (...)" that might slip through regex
func IsInvalidToken(name string) bool {
	if name == "" {
		return true // Invalid: empty
	}

	// Normalize to lowercase for checking
	lower := strings.ToLower(name)

	// STRICT BLOCKLIST: Java keywords and control flow statements
	// These should NEVER appear as method calls or variables in the call graph
	strictBlocklist := map[string]bool{
		"if":           true,
		"else":         true,
		"for":          true,
		"while":        true,
		"do":           true,
		"switch":       true,
		"case":         true,
		"default":      true,
		"return":       true,
		"throw":        true,
		"throws":       true,
		"new":          true,
		"try":          true,
		"catch":        true,
		"finally":      true,
		"synchronized": true,
		"assert":       true,
		"break":        true,
		"continue":     true,
		"goto":         true,
		"instanceof":   true,
		"this":         true,
		"super":        true,
		"null":         true,
		"true":         true,
		"false":        true,
	}

	// Exact match check
	if strictBlocklist[lower] {
		fmt.Printf("[LINKER DROP] Ignored keyword: %s\n", name)
		return true
	}

	// PREFIX CHECK: Block if token STARTS with these keywords
	// This catches "if", "ifSomething", etc.
	keywordPrefixes := []string{
		"if", "for", "while", "switch", "catch",
		"synchronized", "return", "throw", "assert",
		"new", "instanceof",
	}

	for _, prefix := range keywordPrefixes {
		if strings.HasPrefix(lower, prefix) {
			// Additional check: if it's exactly the keyword or followed by non-letter
			if lower == prefix || (len(lower) > len(prefix) && !isLetter(rune(lower[len(prefix)]))) {
				fmt.Printf("[LINKER DROP] Ignored keyword/prefix: %s (prefix: %s)\n", name, prefix)
				return true // Invalid
			}
		}
	}

	// Check for Exception/Error constructors
	if strings.HasSuffix(name, "Exception") || strings.HasSuffix(name, "Error") {
		fmt.Printf("[LINKER DROP] Ignored Exception/Error: %s\n", name)
		return true // Invalid
	}

	return false // Valid
}

// Helper to check if a rune is a letter
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// IsValidMethodCall performs strict validation on method calls to filter out noise
// Blocks: Java keywords, Exception constructors, and invalid class instantiations
func IsValidMethodCall(name string) bool {
	if name == "" {
		return false
	}

	// Block Java keywords that might slip through
	javaKeywords := map[string]bool{
		"if":           true,
		"else":         true,
		"for":          true,
		"while":        true,
		"switch":       true,
		"case":         true,
		"default":      true,
		"try":          true,
		"catch":        true,
		"finally":      true,
		"synchronized": true,
		"return":       true,
		"throw":        true,
		"throws":       true,
		"new":          true,
		"super":        true,
		"this":         true,
		"break":        true,
		"continue":     true,
		"do":           true,
		"instanceof":   true,
		"assert":       true,
	}

	if javaKeywords[name] {
		return false
	}

	// Block Exception constructors (anything ending with "Exception")
	if strings.HasSuffix(name, "Exception") {
		return false
	}

	// Block Error constructors (anything ending with "Error")
	if strings.HasSuffix(name, "Error") {
		return false
	}

	return true
}

// IsDataClass checks if a package name indicates a data structure class
// Data classes (DTO, VO, Model, Entity, etc.) should be excluded from call graphs
// as they represent data structures, not business logic flows
func IsDataClass(packageName string) bool {
	if packageName == "" {
		return false
	}

	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(packageName)

	// Check for common data class package patterns
	dataPackages := []string{
		".model",
		".vo",
		".dto",
		".domain",
		".entity",
		".bean",
	}

	for _, pattern := range dataPackages {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}
