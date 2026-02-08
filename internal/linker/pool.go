package linker

import (
	"fmt"
	"regexp"
	"strings"

	"spec-recon/internal/javaparser"
	"spec-recon/internal/model"
	"spec-recon/internal/xmlparser"
)

// ComponentPool stores all parsed components for quick lookup
type ComponentPool struct {
	// ClassMap: FullClassName -> Node
	ClassMap map[string]*model.Node

	// MethodMap: FullClassName.MethodName -> Node
	MethodMap map[string]*model.Node

	// MethodBodyMap: FullClassName.MethodName -> Body content
	MethodBodyMap map[string]string

	// SQLMap: Namespace.ID -> Node
	SQLMap map[string]*model.Node

	// FieldTypeMap: FullClassName -> (FieldName -> FieldType)
	FieldTypeMap map[string]map[string]string

	// Source content for call tracing
	SourceMap map[string]string // FullClassName -> file content
}

// NewComponentPool creates a new empty component pool
func NewComponentPool() *ComponentPool {
	return &ComponentPool{
		ClassMap:      make(map[string]*model.Node),
		MethodMap:     make(map[string]*model.Node),
		MethodBodyMap: make(map[string]string),
		SQLMap:        make(map[string]*model.Node),
		FieldTypeMap:  make(map[string]map[string]string),
		SourceMap:     make(map[string]string),
	}
}

// AddJavaClass adds a parsed Java class to the pool
func (pool *ComponentPool) AddJavaClass(javaClass *javaparser.JavaClass, sourceContent string) error {
	fullClassName := javaClass.Package + "." + javaClass.Name

	// Create class node
	classNode := &model.Node{
		ID:       fullClassName,
		Type:     determineNodeType(javaClass),
		Package:  javaClass.Package,
		Method:   "", // Class node usually has empty method name or class name
		Children: []*model.Node{},
	}

	pool.ClassMap[fullClassName] = classNode
	pool.SourceMap[fullClassName] = sourceContent

	// Build field type map
	fieldTypes := make(map[string]string)
	for _, field := range javaClass.Fields {
		// Extract simple type name from full type
		simpleType := extractSimpleTypeName(field.Type)
		fieldTypes[field.Name] = simpleType
	}
	pool.FieldTypeMap[fullClassName] = fieldTypes

	// Add methods
	for _, method := range javaClass.Methods {
		methodKey := fullClassName + "." + method.Name

		methodNode := &model.Node{
			ID:           methodKey,
			Type:         classNode.Type, // Inherit from class
			Package:      javaClass.Package,
			Method:       method.Name,
			Params:       method.Params,
			ReturnDetail: method.ReturnType,
			Body:         method.Body, // Store body for return type inference
			URL:          extractURL(javaClass, &method),
			Annotation:   method.GetHTTPMethod(), // Store HTTP Method (GET, POST) here
			Children:     []*model.Node{},
		}

		pool.MethodMap[methodKey] = methodNode
		pool.MethodBodyMap[methodKey] = method.Body

		// Add method as child of class
		methodNode.Parent = classNode
		classNode.Children = append(classNode.Children, methodNode)
	}

	return nil
}

// AddMapperXML adds a parsed MyBatis XML to the pool
func (pool *ComponentPool) AddMapperXML(mapperXML *xmlparser.MapperXML) error {
	for _, sql := range mapperXML.SQLs {
		sqlKey := mapperXML.Namespace + "." + sql.ID

		sqlNode := &model.Node{
			ID:      sqlKey,
			Type:    model.NodeTypeSQL,
			Package: mapperXML.Namespace,
			Method:  sql.ID,
			// Note: Node struct doesn't have SQLQuery field, putting it in Comment or similar if needed.
			// Ideally Node definition should have query info, or we use Comment field for now.
			Comment:  sql.Content,
			Children: []*model.Node{},
		}

		pool.SQLMap[sqlKey] = sqlNode
	}

	return nil
}

// GetClass retrieves a class node by full class name
func (pool *ComponentPool) GetClass(fullClassName string) *model.Node {
	return pool.ClassMap[fullClassName]
}

// GetMethod retrieves a method node by full method key
func (pool *ComponentPool) GetMethod(fullMethodKey string) *model.Node {
	return pool.MethodMap[fullMethodKey]
}

// GetSQL retrieves a SQL node by namespace and ID
func (pool *ComponentPool) GetSQL(namespace, id string) *model.Node {
	key := namespace + "." + id
	return pool.SQLMap[key]
}

// GetFieldType returns the type of a field in a class
func (pool *ComponentPool) GetFieldType(fullClassName, fieldName string) string {
	if fieldTypes, ok := pool.FieldTypeMap[fullClassName]; ok {
		return fieldTypes[fieldName]
	}
	return ""
}

// GetSourceContent retrieves the source code for a class
func (pool *ComponentPool) GetSourceContent(fullClassName string) string {
	return pool.SourceMap[fullClassName]
}

// Helper functions

func determineNodeType(javaClass *javaparser.JavaClass) model.NodeType {
	for _, ann := range javaClass.Annotations {
		switch ann.Name {
		case "Controller", "RestController":
			return model.NodeTypeController
		case "Service":
			return model.NodeTypeService
		case "Repository", "Mapper":
			return model.NodeTypeMapper
		}
	}

	// Check class name patterns
	if strings.HasSuffix(javaClass.Name, "Controller") {
		return model.NodeTypeController
	}
	if strings.HasSuffix(javaClass.Name, "Service") {
		return model.NodeTypeService
	}
	if strings.HasSuffix(javaClass.Name, "Mapper") || strings.HasSuffix(javaClass.Name, "Repository") {
		return model.NodeTypeMapper
	}

	// Default to utility
	return model.NodeTypeUtil
}

func extractSimpleTypeName(fullType string) string {
	// Get last part after dot: com.company.UserService -> UserService
	// Note: We deliberately PRESERVE generics (e.g. List<String>) so they can be stored in FieldTypeMap
	parts := strings.Split(fullType, ".")
	return strings.TrimSpace(parts[len(parts)-1])
}

func extractURL(javaClass *javaparser.JavaClass, method *javaparser.Method) string {
	classURL := javaClass.GetClassLevelURL()
	methodURL := method.GetMethodURL(classURL)
	return methodURL
}

// FindMethodByName finds methods with a specific name in a class
func (pool *ComponentPool) FindMethodByName(fullClassName, methodName string) []*model.Node {
	var results []*model.Node

	// Look for exact match first
	exactKey := fullClassName + "." + methodName
	if node := pool.MethodMap[exactKey]; node != nil {
		results = append(results, node)
		return results
	}

	// Look for partial matches (in case of overloading)
	prefix := fullClassName + "."
	for key, node := range pool.MethodMap {
		if strings.HasPrefix(key, prefix) && strings.Contains(key, methodName) {
			results = append(results, node)
		}
	}

	return results
}

// ResolveFieldType resolves a field name to its full class name
func (pool *ComponentPool) ResolveFieldType(fullClassName, fieldName string) string {
	rawType := pool.GetFieldType(fullClassName, fieldName)
	if rawType == "" {
		return ""
	}

	// Strip generics for linking purposes (e.g. List<User> -> List)
	searchType := rawType
	if idx := strings.Index(searchType, "<"); idx != -1 {
		searchType = searchType[:idx]
	}

	// Try to find full class name by matching simple type
	pkg := extractPackage(fullClassName)

	// Try same package first
	candidate := pkg + "." + searchType
	if pool.GetClass(candidate) != nil {
		return candidate
	}

	// Search all classes for matching simple name
	for fullName := range pool.ClassMap {
		// Note: extractSimpleTypeName here is extracting Class Name from full package path
		// Since class definitions don't have generics in map keys, this compares "UserService" == searchType
		if extractSimpleTypeName(fullName) == searchType {
			return fullName
		}
	}

	return ""
}

func extractPackage(fullClassName string) string {
	lastDot := strings.LastIndex(fullClassName, ".")
	if lastDot == -1 {
		return ""
	}
	return fullClassName[:lastDot]
}

// FindMethodCalls finds method calls in a source string
func FindMethodCalls(source string) []MethodCall {
	var calls []MethodCall

	// Pattern 1: instance.methodName(
	instancePattern := regexp.MustCompile(`(\w+)\.(\w+)\s*\(`)
	matches := instancePattern.FindAllStringSubmatch(source, -1)
	for _, match := range matches {
		calls = append(calls, MethodCall{
			Variable:   match[1],
			MethodName: match[2],
			IsStatic:   false,
		})
	}

	// Pattern 2: ClassName.methodName( (static calls)
	staticPattern := regexp.MustCompile(`([A-Z]\w+)\.(\w+)\s*\(`)
	matches = staticPattern.FindAllStringSubmatch(source, -1)
	for _, match := range matches {
		calls = append(calls, MethodCall{
			Variable:   match[1], // This is the class name
			MethodName: match[2],
			IsStatic:   true,
		})
	}

	return calls
}

// MethodCall represents a method invocation found in source code
type MethodCall struct {
	Variable   string // Variable name or class name
	MethodName string
	IsStatic   bool
}

func (mc MethodCall) String() string {
	if mc.IsStatic {
		return fmt.Sprintf("%s.%s() [static]", mc.Variable, mc.MethodName)
	}
	return fmt.Sprintf("%s.%s()", mc.Variable, mc.MethodName)
}
