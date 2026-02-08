package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"spec-recon/internal/logger"
	"spec-recon/internal/model"
)

// ExtractEndpoints extracts API endpoint definitions from controller nodes
// This focuses ONLY on the public API interface, not internal call chains
// classMap is used to resolve nested schemas for DTOs
// fieldTypeMap contains the field definitions for each class
func ExtractEndpoints(nodes []*model.Node, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) []model.EndpointDef {
	var endpoints []model.EndpointDef

	// DEBUG: Show pool statistics
	fmt.Printf("[DEBUG] Total Classes in Pool: %d\n", len(classMap))
	fmt.Printf("[DEBUG] Total FieldType Mappings: %d\n", len(fieldTypeMap))

	// DEBUG: List all class keys for visibility
	if len(classMap) > 0 {
		fmt.Printf("[DEBUG] Sample classes in pool:\n")
		count := 0
		for key := range classMap {
			if count < 10 { // Show first 10
				fmt.Printf("  - %s\n", key)
				count++
			}
		}
	}

	for _, node := range nodes {
		// Only process Controller nodes
		if node.Type != model.NodeTypeController {
			continue
		}

		// Process each method in the controller
		for _, method := range node.Children {
			// Skip non-controller methods (shouldn't happen, but be safe)
			if method.Type != model.NodeTypeController {
				continue
			}

			// KEYWORD FILTER: Block Java keywords and control flow statements
			// These should never appear as API endpoints
			if isJavaKeyword(method.Method) {
				fmt.Printf("[API SKIP] Blocked keyword as endpoint: %s\n", method.Method)
				continue
			}

			// CONSTRUCTOR FILTER: Block "new" and constructor patterns
			if strings.HasPrefix(method.Method, "new ") || method.Method == "new" {
				fmt.Printf("[API SKIP] Blocked constructor as endpoint: %s\n", method.Method)
				continue
			}

			// Extract endpoint definition from this method
			endpoint := extractEndpointFromMethod(node, method, classMap, fieldTypeMap)
			if endpoint != nil {
				// Filter out View Controllers (web pages, not REST APIs)
				if isViewEndpoint(endpoint, method) {
					fmt.Printf("[API SKIP] Excluded View Endpoint: %s (Type: %s)\n", endpoint.Path, endpoint.Response.Type)
					continue
				}

				endpoints = append(endpoints, *endpoint)
			}
		}
	}

	return endpoints
}

// extractEndpointFromMethod creates an EndpointDef from a controller method node
func extractEndpointFromMethod(controller *model.Node, method *model.Node, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) *model.EndpointDef {
	endpoint := model.NewEndpointDef()

	// Extract HTTP method from annotation
	endpoint.Method = extractHTTPMethod(method)
	if endpoint.Method == "" {
		endpoint.Method = "GET" // Default
	}

	// Extract path from URL
	endpoint.Path = method.URL
	if endpoint.Path == "" {
		endpoint.Path = "/" + method.Method
	}

	// Controller and method names
	endpoint.ControllerName = extractSimpleName(controller.ID)
	endpoint.MethodName = method.Method

	// Extract summary and description from comments
	endpoint.Summary = extractSummary(method.Comment)
	endpoint.Description = method.Comment

	// Extract parameters with schema resolution
	endpoint.Params = extractParameters(method, classMap, fieldTypeMap)

	// Extract response with schema resolution
	endpoint.Response = extractResponse(method, classMap, fieldTypeMap)

	return endpoint
}

// extractHTTPMethod extracts the HTTP method from annotations
func extractHTTPMethod(method *model.Node) string {
	// Check the Annotation field which stores HTTP method
	if method.Annotation != "" {
		return strings.ToUpper(method.Annotation)
	}

	// Try to infer from method name
	methodName := strings.ToLower(method.Method)
	if strings.HasPrefix(methodName, "get") || strings.HasPrefix(methodName, "list") || strings.HasPrefix(methodName, "find") {
		return "GET"
	}
	if strings.HasPrefix(methodName, "create") || strings.HasPrefix(methodName, "add") || strings.HasPrefix(methodName, "insert") {
		return "POST"
	}
	if strings.HasPrefix(methodName, "update") || strings.HasPrefix(methodName, "modify") {
		return "PUT"
	}
	if strings.HasPrefix(methodName, "delete") || strings.HasPrefix(methodName, "remove") {
		return "DELETE"
	}

	return "GET" // Default
}

// extractParameters extracts parameter definitions from method signature
func extractParameters(method *model.Node, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) []model.ParamDef {
	var params []model.ParamDef

	if method.Params == "" {
		return params
	}

	// Parse parameter string: "String userId, User user"
	paramParts := strings.Split(method.Params, ",")
	for _, part := range paramParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		param := parseParameter(part, classMap, fieldTypeMap)
		if param != nil {
			params = append(params, *param)
		}
	}

	return params
}

// parseParameter parses a single parameter string
func parseParameter(paramStr string, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) *model.ParamDef {
	// Pattern: "Type name" or "@Annotation Type name"
	parts := strings.Fields(paramStr)
	if len(parts) < 2 {
		return nil
	}

	param := &model.ParamDef{
		Required: true, // Default to required
	}

	// Check for annotations
	startIdx := 0
	for i, part := range parts {
		if strings.HasPrefix(part, "@") {
			// Determine parameter location from annotation
			annotation := strings.ToLower(part)
			if strings.Contains(annotation, "requestbody") {
				param.In = "Body"
				param.Description = "Request body"
			} else if strings.Contains(annotation, "pathvariable") {
				param.In = "Path"
				param.Description = "Path variable"
			} else if strings.Contains(annotation, "requestparam") {
				param.In = "Query"
				param.Description = "Query parameter"
			} else if strings.Contains(annotation, "requestheader") {
				param.In = "Header"
				param.Description = "Header parameter"
			}
			startIdx = i + 1
		}
	}

	// Extract type and name
	remainingParts := parts[startIdx:]
	if len(remainingParts) >= 2 {
		param.Type = remainingParts[0]
		param.Name = remainingParts[1]
	} else if len(remainingParts) == 1 {
		// Just a type, no name
		param.Type = remainingParts[0]
		param.Name = "param"
	}

	// Default to Query if not specified
	if param.In == "" {
		// Heuristic: complex types are usually Body, primitives are Query
		if isComplexType(param.Type) {
			param.In = "Body"
			// Check if it's a DTO
			if strings.HasSuffix(param.Type, "DTO") || strings.HasSuffix(param.Type, "Dto") {
				param.Description = fmt.Sprintf("%s (Data Transfer Object)", param.Type)
			} else {
				param.Description = fmt.Sprintf("%s (Object)", param.Type)
			}
		} else {
			param.In = "Query"
			param.Description = fmt.Sprintf("Query parameter (%s)", param.Type)
		}
	}

	// If description is still generic, enhance it
	if param.Description == "Request body" && isComplexType(param.Type) {
		if strings.HasSuffix(param.Type, "DTO") || strings.HasSuffix(param.Type, "Dto") {
			param.Description = fmt.Sprintf("%s (Data Transfer Object)", param.Type)
		} else {
			param.Description = fmt.Sprintf("%s (Object)", param.Type)
		}
	}

	// Resolve nested schema for complex types
	if isComplexType(param.Type) {
		param.Fields = resolveSchema(param.Type, classMap, fieldTypeMap)
	}

	return param
}

// extractResponse extracts response definition from method
func extractResponse(method *model.Node, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) model.ResponseDef {
	response := model.ResponseDef{
		Type:        method.ReturnDetail,
		Description: "Successful response",
		StatusCode:  200,
	}

	// Clean up response type
	if response.Type == "" {
		response.Type = "void"
	}

	// INFERENCE: If type is generic/wrapper (Object, ?, ResponseEntity), try to infer from body
	if isDynamicType(response.Type) || strings.Contains(response.Type, "Response") || strings.Contains(response.Type, "?") || strings.Contains(response.Type, "Map") {
		inferred := inferReturnType(method, classMap)
		if inferred != "" {
			fmt.Printf("[INFER] Replaced '%s' with '%s' for method %s\n", response.Type, inferred, method.Method)
			response.Type = inferred
		}

		// MAP INFERENCE: If type is (still) Map, Dynamic, or Response wrapper, try to infer fields from map.put() calls
		if isDynamicType(response.Type) || strings.Contains(response.Type, "Map") || strings.Contains(response.Type, "Response") {
			virtualFields := inferMapSchema(method, classMap, fieldTypeMap)
			if len(virtualFields) > 0 {
				fmt.Printf("[INFER] Constructed virtual schema for Map in method %s\n", method.Method)
				response.Fields = virtualFields
				// We don't return early here, as we might want standard resolution?
				// No, resolveSchema would fail/return empty for Map. So we can just set fields.
			}
		}
	}

	// Determine response description based on type
	returnType := strings.ToLower(response.Type)
	if strings.Contains(returnType, "modelandview") {
		response.Description = "Returns a view"
	} else if strings.Contains(returnType, "responseentity") {
		response.Description = "Returns response entity"
	} else if strings.Contains(returnType, "list") {
		response.Description = "Returns a list of items"
	} else if strings.Contains(returnType, "void") {
		response.Description = "No content"
		response.StatusCode = 204
	}

	// Resolve nested schema for complex response types
	// ONLY if fields haven't already been inferred (e.g. by Map inference or Service Hop)
	if isComplexType(response.Type) && len(response.Fields) == 0 {
		response.Fields = resolveSchema(response.Type, classMap, fieldTypeMap)
	}

	return response
}

// extractSummary extracts a short summary from comment
func extractSummary(comment string) string {
	if comment == "" {
		return ""
	}

	// Take first sentence or first line
	lines := strings.Split(comment, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		// Remove common JavaDoc markers
		firstLine = strings.TrimPrefix(firstLine, "/**")
		firstLine = strings.TrimPrefix(firstLine, "/*")
		firstLine = strings.TrimPrefix(firstLine, "*")
		firstLine = strings.TrimSpace(firstLine)

		// Limit to first sentence
		if idx := strings.Index(firstLine, "."); idx > 0 {
			return firstLine[:idx+1]
		}
		return firstLine
	}

	return comment
}

// extractSimpleName extracts simple class name from full name
func extractSimpleName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// isComplexType checks if a type is complex (not primitive)
func isComplexType(typeName string) bool {
	primitives := map[string]bool{
		"String":    true,
		"int":       true,
		"Integer":   true,
		"long":      true,
		"Long":      true,
		"double":    true,
		"Double":    true,
		"float":     true,
		"Float":     true,
		"boolean":   true,
		"Boolean":   true,
		"char":      true,
		"Character": true,
	}

	// Remove generics
	if idx := strings.Index(typeName, "<"); idx > 0 {
		typeName = typeName[:idx]
	}

	return !primitives[typeName]
}

// Helper function to clean parameter annotations
func cleanAnnotation(annotation string) string {
	// Remove @ and extract annotation name
	annotation = strings.TrimPrefix(annotation, "@")

	// Remove parentheses and their content
	re := regexp.MustCompile(`\([^)]*\)`)
	annotation = re.ReplaceAllString(annotation, "")

	return strings.TrimSpace(annotation)
}

// isViewEndpoint checks if an endpoint is a view controller (web page) rather than a REST API
// View controllers return HTML pages and should not be included in API documentation
func isViewEndpoint(endpoint *model.EndpointDef, method *model.Node) bool {
	returnType := endpoint.Response.Type

	// Check for explicit view return types
	viewTypes := []string{
		"ModelAndView",
		"Model",
		"View",
		"RedirectView",
		"ModelMap",
	}

	for _, viewType := range viewTypes {
		if strings.Contains(returnType, viewType) {
			return true
		}
	}

	// Check for String return type without @ResponseBody
	// String with @ResponseBody is a REST API (returns JSON/text)
	// String without @ResponseBody is a view name (returns HTML page)
	if returnType == "String" {
		// Check if method has @ResponseBody annotation
		// The Annotation field contains the primary annotation
		hasResponseBody := strings.Contains(method.Annotation, "ResponseBody")

		// If String return type without @ResponseBody, it's a view
		if !hasResponseBody {
			return true
		}
	}

	return false
}

// isJavaKeyword checks if a name is a Java keyword or control flow statement
// These should never appear as API endpoint method names
func isJavaKeyword(name string) bool {
	if name == "" {
		return false
	}

	// Normalize to lowercase
	lower := strings.ToLower(name)

	// Comprehensive blocklist of Java keywords
	keywords := map[string]bool{
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
		"void":         true,
		"class":        true,
		"interface":    true,
		"enum":         true,
		"extends":      true,
		"implements":   true,
		"import":       true,
		"package":      true,
		"private":      true,
		"protected":    true,
		"public":       true,
		"static":       true,
		"final":        true,
		"abstract":     true,
		"native":       true,
		"strictfp":     true,
		"transient":    true,
		"volatile":     true,

		// Spring/View types that should not be method names
		"modelandview": true, // lowercase for case-insensitive match
		"model":        true,
		"view":         true,
		"redirectview": true,
	}

	return keywords[lower]
}

// cleanTypeName removes generic wrappers and array notation to get the core type
// Examples:
//
//	List<ProductDTO> -> ProductDTO
//	Set<UserDTO> -> UserDTO
//	Map<String, ProductDTO> -> ProductDTO (takes the last type)
//	ProductDTO[] -> ProductDTO
//	Optional<ProductDTO> -> ProductDTO
//	ResponseEntity<List<ProductDTO>> -> ProductDTO (handles nested generics)
func cleanTypeName(raw string) string {
	cleaned := strings.TrimSpace(raw)

	// Remove array notation
	cleaned = strings.ReplaceAll(cleaned, "[]", "")

	// Handle generics: recursively extract inner types
	// Keep extracting until no more generics
	for strings.Contains(cleaned, "<") && strings.Contains(cleaned, ">") {
		startIdx := strings.Index(cleaned, "<")
		endIdx := strings.LastIndex(cleaned, ">")
		if endIdx > startIdx {
			// Extract content between < and >
			genericContent := cleaned[startIdx+1 : endIdx]

			// For Map<K,V>, take the last type (V)
			// For List<T>, take T
			// For ResponseEntity<List<ProductDTO>>, take List<ProductDTO> and loop again
			parts := strings.Split(genericContent, ",")
			if len(parts) > 0 {
				cleaned = strings.TrimSpace(parts[len(parts)-1])
			} else {
				break // Safety: avoid infinite loop
			}
		} else {
			break
		}
	}

	return cleaned
}

// isSystemType checks if a type is a system/framework type that should be skipped
// These types don't have meaningful schemas to extract
func isSystemType(name string) bool {
	systemTypes := map[string]bool{
		// Primitives
		"void":    true,
		"int":     true,
		"long":    true,
		"double":  true,
		"float":   true,
		"boolean": true,
		"char":    true,

		// Wrapper classes
		"String":    true,
		"Integer":   true,
		"Long":      true,
		"Double":    true,
		"Float":     true,
		"Boolean":   true,
		"Character": true,

		// Collections (after cleaning, these shouldn't appear, but just in case)
		"List":     true,
		"Set":      true,
		"Map":      true,
		"Optional": true,

		// Servlet types
		"HttpServletRequest":  true,
		"HttpServletResponse": true,
		"ServletRequest":      true,
		"ServletResponse":     true,

		// Spring types
		"Model":          true,
		"ModelAndView":   true,
		"ResponseEntity": true,
		"HttpEntity":     true,

		// Noise
		"new":    true,
		"Object": true,
	}

	return systemTypes[name]
}

// resolveSchema resolves the schema (fields) of a complex type RECURSIVELY
// It returns a flattened list of all fields with proper depth tracking for nested structures
func resolveSchema(typeName string, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) []model.ParamDef {
	fields := resolveSchemaRecursive(typeName, classMap, fieldTypeMap, 1, make(map[string]bool))
	return deduplicateFields(fields)
}

// resolveSchemaRecursive is the recursive implementation of schema resolution
// depth: current nesting level (0=root, 1=child, 2=grandchild, etc.)
// visited: tracks visited types to prevent infinite recursion
func resolveSchemaRecursive(typeName string, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string, depth int, visited map[string]bool) []model.ParamDef {
	var results []model.ParamDef

	// BASE CASE 1: Max depth reached (prevent infinite loops)
	if depth > 5 {
		fmt.Printf("[RECURSIVE] Max depth (5) reached for '%s', stopping recursion\n", typeName)
		return results
	}

	// STEP 0: Check for COLLECTION types (List<T>, Set<T>) and unwrap them
	// This must happen BEFORE cleaning/stripping the generics
	if isCollectionType(typeName) {
		inner := getInnerType(typeName)
		if inner != "" {
			fmt.Printf("[RECURSIVE] Unwrapping Collection: '%s' -> '%s' at depth %d\n", typeName, inner, depth)
			// Recurse on the inner type using the SAME depth
			// (because the List wrapper itself is not a separate level in terms of data fields)
			return resolveSchemaRecursive(inner, classMap, fieldTypeMap, depth, visited)
		}
	}

	// STEP 1: Clean the type name (remove generics, arrays)
	cleanType := cleanTypeName(typeName)

	// BASE CASE 2: Circular reference detection
	if visited[cleanType] {
		fmt.Printf("[RECURSIVE] Circular reference detected for '%s' at depth %d, stopping\n", cleanType, depth)
		return results
	}

	// Mark as visited
	visited[cleanType] = true
	defer func() { delete(visited, cleanType) }() // Unmark after processing

	// STEP 2: Check for DYNAMIC TYPES (Object, Map, etc.)
	if isDynamicType(cleanType) {
		fmt.Printf("[RECURSIVE] Dynamic type detected: '%s' at depth %d\n", cleanType, depth)
		return []model.ParamDef{
			{
				Name:        "(Dynamic)",
				Type:        "Map / JSON Object",
				Depth:       depth,
				Description: "Structure varies dynamically (Key-Value pairs).",
			},
		}
	}

	// STEP 3: Skip system types early
	if isSystemType(cleanType) {
		fmt.Printf("[RECURSIVE] Skipping system type: '%s' at depth %d\n", cleanType, depth)
		return results
	}

	// STEP 4: Try exact match in classMap
	node := classMap[cleanType]
	var matchedKey string

	if node != nil {
		matchedKey = cleanType
		fmt.Printf("[RECURSIVE] Exact match FOUND: '%s' at depth %d\n", cleanType, depth)
	} else {
		// STEP 5: Fuzzy match using suffix check
		for key, n := range classMap {
			if strings.HasSuffix(key, "."+cleanType) {
				node = n
				matchedKey = key
				fmt.Printf("[RECURSIVE] Fuzzy match: '%s' -> '%s' at depth %d\n", cleanType, key, depth)
				break
			}
		}
	}

	// STEP 6: If still not found, return empty
	if node == nil {
		fmt.Printf("[RECURSIVE] FAILED to resolve: '%s' at depth %d\n", cleanType, depth)
		return results
	}

	// STEP 7: Extract fields from FieldTypeMap
	var fieldTypes map[string]string
	var ok bool

	if fieldTypes, ok = fieldTypeMap[matchedKey]; ok {
		fmt.Printf("[RECURSIVE] Found %d fields for '%s' at depth %d\n", len(fieldTypes), cleanType, depth)
	} else {
		// Try fuzzy match on fieldTypeMap
		for key, ft := range fieldTypeMap {
			if strings.HasSuffix(key, "."+cleanType) {
				fieldTypes = ft
				fmt.Printf("[RECURSIVE] Found %d fields for '%s' (fuzzy) at depth %d\n", len(fieldTypes), cleanType, depth)
				break
			}
		}
	}

	// STEP 8: Build ParamDef list from fields WITH RECURSION
	if fieldTypes != nil {
		for fieldName, fieldType := range fieldTypes {
			// Create the parent field
			paramDef := model.ParamDef{
				Name:        fieldName,
				Type:        fieldType,
				Depth:       depth,
				Description: fmt.Sprintf("Field of %s", cleanType),
			}

			// Add parent field to results
			results = append(results, paramDef)

			// RECURSION: If the field type is complex, resolve its children
			if isComplexType(fieldType) {
				childFields := resolveSchemaRecursive(fieldType, classMap, fieldTypeMap, depth+1, visited)
				// Append child fields immediately after parent
				results = append(results, childFields...)
			}
		}
		fmt.Printf("[RECURSIVE] Extracted %d total fields (including nested) for '%s' at depth %d\n", len(results), cleanType, depth)
	}

	return results
}

// isDynamicType checks if a type represents a dynamic/generic structure
// These types have no fixed schema and should be documented as dynamic
func isDynamicType(typeName string) bool {
	if typeName == "" {
		return false
	}

	// Normalize for comparison
	lower := strings.ToLower(typeName)

	// Check for generic type parameters
	if typeName == "?" || typeName == "T" || typeName == "E" || typeName == "K" || typeName == "V" {
		return true
	}

	// Check for Object types
	if typeName == "Object" || typeName == "java.lang.Object" {
		return true
	}

	// Check for Map types (Map, HashMap, LinkedHashMap, TreeMap, etc.)
	mapPrefixes := []string{"map", "hashmap", "linkedhashmap", "treemap", "concurrenthashmap"}
	for _, prefix := range mapPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}

// getInnerType extracts the text inside generic brackets <...>
// Example: List<MemberDto> -> MemberDto
func getInnerType(typeStr string) string {
	start := strings.Index(typeStr, "<")
	end := strings.LastIndex(typeStr, ">")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(typeStr[start+1 : end])
	}
	return ""
}

// isCollectionType checks if a type is a Java Collection
// This helps identify containers that should be unwrapped to find their content schema
func isCollectionType(typeName string) bool {
	if typeName == "" {
		return false
	}
	lower := strings.ToLower(typeName)

	// Direct collection types
	if lower == "list" || lower == "set" || lower == "collection" || lower == "iterable" || lower == "page" {
		return true
	}

	// Generic collections (List<...>)
	prefixes := []string{
		"list<",
		"arraylist<",
		"linkedlist<",
		"set<",
		"hashset<",
		"treeset<",
		"collection<",
		"iterable<",
		"page<",  // Spring Data Page
		"slice<", // Spring Data Slice
	}

	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}

	// Fully qualified names (java.util.List)
	if strings.Contains(lower, ".list") || strings.Contains(lower, ".set") {
		return true
	}

	return false
}

// inferReturnType attempts to infer the concrete return type from the method body
func inferReturnType(node *model.Node, classMap map[string]*model.Node) string {
	if node.Body == "" {
		return ""
	}

	// 1. Explicit New: return new UserDTO(...)
	reNew := regexp.MustCompile(`return\s+new\s+([a-zA-Z0-9_<>,\s.]+)\s*\(`)
	if matches := reNew.FindStringSubmatch(node.Body); len(matches) > 1 {
		return matches[1]
	}

	// 2. Wrap New: return new ResponseDto(new UserDTO(...))
	reWrapper := regexp.MustCompile(`new\s+[a-zA-Z0-9_<>]+(?:\(.*\))?\(\s*new\s+([a-zA-Z0-9_<>,\s.]+)\s*\(`)
	if matches := reWrapper.FindStringSubmatch(node.Body); len(matches) > 1 {
		return matches[1]
	}

	// 3. Builder Pattern: return UserDTO.builder()
	reBuilder := regexp.MustCompile(`return\s+([a-zA-Z0-9_<>,\s.]+)\.builder\s*\(`)
	if matches := reBuilder.FindStringSubmatch(node.Body); len(matches) > 1 {
		return matches[1]
	}

	// 4. Variable Back-tracing (Strategy 3)
	// Match: return new ResponseDto(variableName);
	// We allow generics in the wrapper e.g. new ResponseEntity<Object>(data)
	reReturnVar := regexp.MustCompile(`return\s+new\s+[a-zA-Z0-9_<>]+(?:\(.*\))?\s*\(\s*([a-zA-Z0-9_]+)\s*(?:,|\))`)
	if matches := reReturnVar.FindStringSubmatch(node.Body); len(matches) > 1 {
		varName := matches[1]

		// Only proceed if varName is not "null" or "true/false"
		if varName != "null" && varName != "true" && varName != "false" {
			// Search for declaration: Type varName = ...
			// Regex: (Start of line or separator) Type varName =
			// Note: We escape varName safely
			declPattern := fmt.Sprintf(`(?:^|[;{}])\s*([A-Z][a-zA-Z0-9_<>]*)\s+%s\s*=`, regexp.QuoteMeta(varName))
			reDecl := regexp.MustCompile(declPattern)
			if declMatches := reDecl.FindStringSubmatch(node.Body); len(declMatches) > 1 {
				return declMatches[1]
			}
		}
	}

	// 5. Naming Convention Heuristic (Strategy 4 - Fallback)
	// getSchoolList -> SchoolListResponseDto
	if strings.HasPrefix(node.Method, "get") {
		baseName := strings.TrimPrefix(node.Method, "get")
		if len(baseName) > 0 {
			// Check common response suffixes
			candidates := []string{
				baseName + "ResponseDto",
				baseName + "Response",
				baseName + "Dto",
			}

			for _, candidate := range candidates {
				// Check ClassMap for fuzzy match
				for key := range classMap {
					if strings.HasSuffix(key, "."+candidate) || key == candidate {
						return candidate
					}
				}
			}
		}
	}

	return ""
}

// inferMapSchema attempts to reconstruct the schema of a Map return type by analyzing .put() calls
// inferMapSchema attempts to reconstruct the schema of a Map return type by analyzing .put() calls
// or by hopping to the service method being called
func inferMapSchema(node *model.Node, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) []model.ParamDef {
	var results []model.ParamDef
	if node.Body == "" {
		return results
	}

	// Strategy 1: Local Map Inference
	// Search for Map creation and put calls in the current method body
	targetVar := ""

	// A. Direct Return
	reDirect := regexp.MustCompile(`return\s+(?P<Var>\w+)\s*;`)
	if matches := reDirect.FindStringSubmatch(node.Body); len(matches) > 1 {
		targetVar = matches[1]
	}

	// B. Wrapped (Constructor)
	if targetVar == "" {
		reWrapped := regexp.MustCompile(`return\s+new\s+[a-zA-Z0-9_<>]+(?:\(.*\))?\s*\(\s*(?P<Var>\w+)\s*(?:,|\))`)
		if matches := reWrapped.FindStringSubmatch(node.Body); len(matches) > 1 {
			targetVar = matches[1]
		}
	}

	// C. Static Factory
	if targetVar == "" {
		reFactory := regexp.MustCompile(`return\s+[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+\s*\(\s*(?P<Var>\w+)\s*(?:,|\))`)
		if matches := reFactory.FindStringSubmatch(node.Body); len(matches) > 1 {
			targetVar = matches[1]
		}
	}

	// Try extracting from local map if target variable found
	if targetVar != "" && targetVar != "null" && targetVar != "true" && targetVar != "false" {
		// Verify targetVar is a Map declaration
		reMapDecl := regexp.MustCompile(fmt.Sprintf(`\b(?:Map|HashMap|LinkedHashMap|ModelMap)(?:<.*?>)?\s+\b%s\b\s*=`, regexp.QuoteMeta(targetVar)))
		if reMapDecl.MatchString(node.Body) {
			// Scan 'put' Calls
			rePut := regexp.MustCompile(fmt.Sprintf(`\b%s\.put\(\s*"([^"]+)"\s*,\s*(.*?)\s*\);`, regexp.QuoteMeta(targetVar)))
			putMatches := rePut.FindAllStringSubmatch(node.Body, -1)

			for _, match := range putMatches {
				if len(match) < 3 {
					continue
				}
				key := match[1]
				valueStr := strings.TrimSpace(match[2])

				// Resolve Value Type
				valueType := "Object"

				// Case A: Constructor
				reConstructor := regexp.MustCompile(`new\s+([a-zA-Z0-9_<>,\s.]+)\s*\(`)
				if cMatches := reConstructor.FindStringSubmatch(valueStr); len(cMatches) > 1 {
					valueType = cMatches[1]
				} else if strings.HasPrefix(valueStr, "\"") {
					valueType = "String"
				} else if valueStr == "true" || valueStr == "false" {
					valueType = "boolean"
				} else if regexp.MustCompile(`^\d+$`).MatchString(valueStr) {
					valueType = "int"
				} else {
					// Case B: Variable Back-tracing
					declPattern := fmt.Sprintf(`(?:^|[;{}])\s*([A-Z][a-zA-Z0-9_<>]*)\s+%s\s*=`, regexp.QuoteMeta(valueStr))
					reDecl := regexp.MustCompile(declPattern)
					if dMatches := reDecl.FindStringSubmatch(node.Body); len(dMatches) > 1 {
						valueType = dMatches[1]
					}
				}

				// Heuristic: Force primitive types for common pagination keys
				valueType = applyTypeHeuristics(key, valueType)

				param := model.ParamDef{
					Name:        key,
					Type:        valueType,
					Depth:       1,
					Description: fmt.Sprintf("inferred map key"),
				}

				results = append(results, param)

				if isComplexType(valueType) {
					childFields := resolveSchemaRecursive(valueType, classMap, fieldTypeMap, 2, make(map[string]bool))
					results = append(results, childFields...)
				}
			}
		}
	}

	if len(results) > 0 {
		fmt.Printf("[INFER] Successfully inferred Map schema for %s (var: %s)\n", node.Method, targetVar)
		return deduplicateFields(results)
	}

	// Strategy 2: Service Hop
	// 1. Extract Return Statement for context
	// Regex for return statement: "return <content> ;"
	// Use (?s) to allow matching across newlines
	reReturn := regexp.MustCompile(`(?s)return\s+(.*?);`)
	returnMatch := reReturn.FindStringSubmatch(node.Body)

	if len(returnMatch) > 1 {
		returnStmt := returnMatch[1] // The content between return and ;
		// Clean up newlines for cleaner logging
		cleanReturnStmt := strings.Join(strings.Fields(returnStmt), " ")
		logger.Debug("[HOP-TRACE] Method %s Return Stmt: %s", node.Method, cleanReturnStmt)

		// 2. Scan for Candidates (Relaxed Regex)
		// We look for any method call: varName.methodName(...)
		reCall := regexp.MustCompile(`(\w+)\.(\w+)\(`)
		candidates := reCall.FindAllStringSubmatch(returnStmt, -1)

		for _, candidate := range candidates {
			varName := candidate[1]
			methodName := candidate[2]
			logger.Debug("[HOP-TRACE] Candidate: %s.%s", varName, methodName)

			// 3. Resolve Field Type
			if node.Parent != nil {
				if classFields, ok := fieldTypeMap[node.Parent.ID]; ok {
					if serviceType, found := classFields[varName]; found {
						serviceType = cleanTypeName(serviceType)
						logger.Debug("[HOP-TRACE] Resolved Field '%s' to Type '%s'", varName, serviceType)

						// 4. Execute Hop
						serviceNode := resolveImplementationClass(classMap, serviceType)
						if serviceNode != nil {
							logger.Info("[HOP] Resolved '%s' -> Implementation '%s'", serviceType, serviceNode.ID)

							for _, child := range serviceNode.Children {
								if child.Method == methodName {
									logger.Info("[HOP] Jumping to Service: %s -> %s.%s", node.Method, serviceNode.ID, methodName)

									// PHASE 12: Generic Filter - Ambiguity Check
									// If return type is vague (Object, <Object>, <?>), we must scan the body!
									if child.ReturnDetail != "" && !isAmbiguousType(child.ReturnDetail) {
										logger.Info("[HOP-TYPE] Service method '%s' returns concrete type '%s'. Using it.", child.Method, child.ReturnDetail)
										return resolveSchema(child.ReturnDetail, classMap, fieldTypeMap)
									} else {
										logger.Debug("[HOP-SKIP] Service return type '%s' is ambiguous. Falling back to body scan.", child.ReturnDetail)
									}

									return inferMapSchema(child, classMap, fieldTypeMap)
								}
							}
						} else {
							logger.Debug("[HOP-FAIL] Could not resolve class for type '%s' (Tried Impl/Fuzzy)", serviceType)
						}
					}
				}
			}
		}
	}

	// Strategy 3: Blind Map Scan (Fallback)
	// If all else failed, assume any Map created in this method could be the return value.
	if len(results) == 0 {
		logger.Debug("[BLIND-TRACE] Starting blind scan on method '%s' (%d chars)", node.Method, len(node.Body))

		// 1. Broad Regex for Map Assignment
		// Matches: "result = new HashMap" or "map = new java.util.LinkedHashMap"
		reMapBlind := regexp.MustCompile(`(?P<Var>\w+)\s*=\s*new\s+(?:[\w\.]+\.)?(?:Hash|LinkedHash|Tree|Model|ConcurrentHash)Map`)
		blindMatches := reMapBlind.FindAllStringSubmatch(node.Body, -1)

		candidateVars := make([]string, 0)
		seenVars := make(map[string]bool)

		for _, match := range blindMatches {
			v := match[1]
			if !seenVars[v] {
				candidateVars = append(candidateVars, v)
				seenVars[v] = true
			}
		}

		// 2. Fallback: Common Variable Names
		// If regex missed (e.g., declared elsewhere or obscure type), try common names
		if len(candidateVars) == 0 {
			commonNames := []string{"map", "result", "data", "res", "response"}
			for _, name := range commonNames {
				// Crude check: does the body contain "name.put"?
				if strings.Contains(node.Body, name+".put") {
					if !seenVars[name] {
						candidateVars = append(candidateVars, name)
						seenVars[name] = true
					}
				}
			}
		}

		for _, blindVar := range candidateVars {
			logger.Debug("[BLIND-TRACE] Found candidate Map variable: '%s'", blindVar)
			logger.Info("[INFER-BLIND] Found orphaned Map variable '%s' in method '%s'. Scavenging fields...", blindVar, node.Method)

			// Scan 'put' Calls on blindVar (Relaxed multiline)
			rePutBlind := regexp.MustCompile(fmt.Sprintf(`(?s)\b%s\.put\(\s*"([^"]+)"\s*,\s*(.*?)\s*\);`, regexp.QuoteMeta(blindVar)))
			putMatches := rePutBlind.FindAllStringSubmatch(node.Body, -1)

			for _, match := range putMatches {
				if len(match) < 3 {
					continue
				}
				key := match[1]
				valueStr := strings.TrimSpace(match[2])

				// Resolve Value Type
				valueType := "Object"

				// Case A: Constructor
				reConstructor := regexp.MustCompile(`new\s+([a-zA-Z0-9_<>,\s.]+)\s*\(`)
				if cMatches := reConstructor.FindStringSubmatch(valueStr); len(cMatches) > 1 {
					valueType = cMatches[1]
				} else if strings.HasPrefix(valueStr, "\"") {
					valueType = "String"
				} else if valueStr == "true" || valueStr == "false" {
					valueType = "boolean"
				} else if regexp.MustCompile(`^\d+$`).MatchString(valueStr) {
					valueType = "int"
				} else {
					// Case B: Variable Back-tracing
					declPattern := fmt.Sprintf(`(?:^|[;{}])\s*([A-Z][a-zA-Z0-9_<>]*)\s+%s\s*=`, regexp.QuoteMeta(valueStr))
					reDecl := regexp.MustCompile(declPattern)
					if dMatches := reDecl.FindStringSubmatch(node.Body); len(dMatches) > 1 {
						valueType = dMatches[1]
					}
				}

				// Heuristic: Force primitive types for common pagination keys
				valueType = applyTypeHeuristics(key, valueType)

				param := model.ParamDef{
					Name:        key,
					Type:        valueType,
					Depth:       1,
					Description: fmt.Sprintf("scavenged map key"),
				}

				results = append(results, param)

				if isComplexType(valueType) {
					childFields := resolveSchemaRecursive(valueType, classMap, fieldTypeMap, 2, make(map[string]bool))
					results = append(results, childFields...)
				}
			}
		}
	}

	// Strategy 4: Return Variable Trace
	// If Blind Scan failed, check the return statement for a variable and trace its type.
	if len(results) == 0 {
		var returnVar string

		// Check "return var;" (Simple)
		reSimple := regexp.MustCompile(`return\s+([a-zA-Z0-9_]+)\s*;`)
		if m := reSimple.FindStringSubmatch(node.Body); len(m) > 1 {
			returnVar = m[1]
		}

		// Check "return func(var)" or "return new Obj(var)" (Wrapped)
		if returnVar == "" {
			reComplex := regexp.MustCompile(`return\s+[^;]*\(\s*([a-zA-Z0-9_]+)\s*\)`)
			if m := reComplex.FindStringSubmatch(node.Body); len(m) > 1 {
				returnVar = m[1]
			}
		}

		if returnVar != "" && returnVar != "null" && returnVar != "true" && returnVar != "false" {
			// 2. Find Declaration
			// Matches: "Type varName =" or "Type varName;"
			reDecl := regexp.MustCompile(fmt.Sprintf(`(?P<Type>[a-zA-Z0-9_<>\[\]]+)\s+\b%s\b\s*[:=;]`, regexp.QuoteMeta(returnVar)))
			if match := reDecl.FindStringSubmatch(node.Body); len(match) > 1 {
				declType := match[1]
				logger.Info("[VAR-TRACE] Found return variable '%s' with type '%s'", returnVar, declType)

				// 3. Resolve Type
				var resolvedFields []model.ParamDef

				if isSystemType(declType) {
					// Case A: Primitive / System Type (e.g., int, String)
					resolvedFields = append(resolvedFields, model.ParamDef{
						Name:        "result",
						Type:        declType,
						Description: fmt.Sprintf("Return value (%s)", returnVar),
						Depth:       0,
					})
				} else {
					// Case B: DTO / Complex Type
					resolvedFields = resolveSchema(declType, classMap, fieldTypeMap)
				}

				// 4. Return Immediately
				if len(resolvedFields) > 0 {
					return resolvedFields
				}
			}
		}
	}

	return deduplicateFields(results)
}

// resolveImplementationClass finds the concrete implementation class for a given interface or class name
// It uses suffix matching (Impl) and fuzzy package matching to locate the correct node.
func resolveImplementationClass(classMap map[string]*model.Node, targetType string) *model.Node {
	// Strategy A: Exact Match
	if node, ok := classMap[targetType]; ok {
		// Even if exact match found, check if it's likely an interface (empty body or name doesn't end in Impl)
		// and if a better Impl exists.
		if strings.HasSuffix(targetType, "Impl") {
			return node
		}
		// If exact match is NOT Impl, check strategy B just in case
		if implNode, ok := classMap[targetType+"Impl"]; ok {
			return implNode
		}
		return node
	}

	// Strategy B: Impl Suffix
	if node, ok := classMap[targetType+"Impl"]; ok {
		return node
	}

	// Strategy C: Fuzzy Suffix Match (Package Handling)
	var candidate *model.Node

	suffix := "." + targetType
	implSuffix := "." + targetType + "Impl"

	for key, node := range classMap {
		if strings.HasSuffix(key, implSuffix) {
			return node // Strongest match found, return immediately
		}
		if strings.HasSuffix(key, suffix) {
			candidate = node
		}
	}

	return candidate // Return whatever we found (or nil)
}

// isAmbiguousType checks if a type is too vague to be useful without body scanning
// Returns true for: void, Object, Map, <Object>, <?>
func isAmbiguousType(typeName string) bool {
	if typeName == "" || typeName == "void" {
		return true
	}

	// Check basic dynamic types (Map, Object, T, ?)
	if isDynamicType(typeName) {
		return true
	}

	// Check Generics with Object or Wildcard
	// e.g. ResponseDto<Object>, List<?>
	if strings.Contains(typeName, "<Object>") || strings.Contains(typeName, "<?>") {
		return true
	}

	// Check Raw Collections (List, Set, Collection without <...>)
	// If it matches a collection type but has NO generic brackets, it's a raw collection of Objects.
	if isCollectionType(typeName) && !strings.Contains(typeName, "<") {
		return true
	}

	return false
}

// applyTypeHeuristics forces primitive types for known keys (e.g. pagination)
func applyTypeHeuristics(key, currentType string) string {
	lowerKey := strings.ToLower(key)
	if lowerKey == "totalelements" || lowerKey == "totalpages" || lowerKey == "totalcount" {
		return "long"
	}
	if lowerKey == "size" || lowerKey == "page" || lowerKey == "number" || lowerKey == "numberofelements" {
		return "int"
	}
	return currentType
}

// deduplicateFields ensures field uniqueness, prioritizing concrete types over Object
func deduplicateFields(fields []model.ParamDef) []model.ParamDef {
	if len(fields) == 0 {
		return fields
	}
	var unique []model.ParamDef
	seen := make(map[string]int) // Name -> Index in unique

	for _, f := range fields {
		if idx, exists := seen[f.Name]; exists {
			// Determine if we should overwrite
			existing := unique[idx]
			// If existing is "Object" or vague but new is Concrete -> Overwrite
			isExistingVague := existing.Type == "Object" || existing.Type == "java.lang.Object" || existing.Type == "Map"
			isNewConcrete := f.Type != "Object" && f.Type != "java.lang.Object" && f.Type != "Map"

			if isExistingVague && isNewConcrete {
				unique[idx] = f
			}
			// Else ignore (keep existing) - First come, first served (usually Direct Map Scan comes before Blind Scan)
		} else {
			seen[f.Name] = len(unique)
			unique = append(unique, f)
		}
	}
	return unique
}

// inferMapSchemaLegacy is the previous implementation preserved for safety during refactor
func inferMapSchemaLegacy(node *model.Node, classMap map[string]*model.Node, fieldTypeMap map[string]map[string]string) []model.ParamDef {
	var results []model.ParamDef
	if node.Body == "" {
		return results
	}

	// Step 1: Find Return Variable Name
	// Strategies:
	// A. Direct return: return map;
	// B. Wrapped: return new ResponseDto(map);
	// C. Static Factory: return ResponseDto.ok(map);

	var targetVar string

	// A. Direct Return
	reDirect := regexp.MustCompile(`return\s+(?P<Var>\w+)\s*;`)
	if matches := reDirect.FindStringSubmatch(node.Body); len(matches) > 1 {
		targetVar = matches[1]
	}

	// B. Wrapped (Constructor) - Only if not found yet
	if targetVar == "" {
		reWrapped := regexp.MustCompile(`return\s+new\s+[a-zA-Z0-9_<>]+(?:\(.*\))?\s*\(\s*(?P<Var>\w+)\s*(?:,|\))`)
		if matches := reWrapped.FindStringSubmatch(node.Body); len(matches) > 1 {
			targetVar = matches[1]
		}
	}

	// C. Static Factory - Only if not found yet
	if targetVar == "" {
		reFactory := regexp.MustCompile(`return\s+[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+\s*\(\s*(?P<Var>\w+)\s*(?:,|\))`)
		if matches := reFactory.FindStringSubmatch(node.Body); len(matches) > 1 {
			targetVar = matches[1]
		}
	}

	if targetVar == "" {
		return results
	}

	// Ignore if targetVar is "null" or "true" etc
	if targetVar == "null" || targetVar == "true" || targetVar == "false" {
		return results
	}

	// Step 2: Verify targetVar is a Map
	// Check declaration: Map<...> targetVar = ... or HashMap targetVar ...
	// we match word boundary
	reMapDecl := regexp.MustCompile(fmt.Sprintf(`\b(?:Map|HashMap|LinkedHashMap|ModelMap)(?:<.*?>)?\s+\b%s\b\s*=`, regexp.QuoteMeta(targetVar)))
	if !reMapDecl.MatchString(node.Body) {
		return results
	}

	// Step 3: Scan 'put' Calls on targetVar
	// targetVar.put("key", value);
	rePut := regexp.MustCompile(fmt.Sprintf(`\b%s\.put\(\s*"([^"]+)"\s*,\s*(.*?)\s*\);`, regexp.QuoteMeta(targetVar)))
	putMatches := rePut.FindAllStringSubmatch(node.Body, -1)

	for _, match := range putMatches {
		if len(match) < 3 {
			continue
		}
		key := match[1]
		valueStr := strings.TrimSpace(match[2])

		// Step 4: Resolve Value Type
		valueType := "Object" // Default

		// Case A: Constructor - new UserDto()
		reConstructor := regexp.MustCompile(`new\s+([a-zA-Z0-9_<>,\s.]+)\s*\(`)
		if cMatches := reConstructor.FindStringSubmatch(valueStr); len(cMatches) > 1 {
			valueType = cMatches[1]
		} else if strings.HasPrefix(valueStr, "\"") {
			valueType = "String"
		} else if valueStr == "true" || valueStr == "false" {
			valueType = "boolean"
		} else if regexp.MustCompile(`^\d+$`).MatchString(valueStr) {
			valueType = "int"
		}

		// Heuristic Typing
		valueType = applyTypeHeuristics(key, valueType)

		// Step 5: Create ParamDef (Flattened Parent)
		param := model.ParamDef{
			Name:        key,
			Type:        valueType,
			Depth:       1, // Virtual depth 1 (children of the Map)
			Description: fmt.Sprintf("inferred map key"),
		}

		results = append(results, param)

		// Recursion: If it's a complex type, resolve its schema
		if isComplexType(valueType) {
			childFields := resolveSchemaRecursive(valueType, classMap, fieldTypeMap, 2, make(map[string]bool))
			results = append(results, childFields...)
		}
	}

	if len(results) > 0 {
		logger.Debug("[INFER] Successfully inferred Map schema for %s (var: %s)", node.Method, targetVar)
	}

	return results
}
