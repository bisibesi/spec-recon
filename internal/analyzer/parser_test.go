package analyzer

import (
	"path/filepath"
	"strings"
	"testing"

	"spec-recon/internal/javaparser"
	"spec-recon/internal/xmlparser"
)

// Test data paths
const (
	testDataDir           = "../../testdata/hybrid_sample"
	userControllerPath    = "com/company/legacy/UserController.java"
	userServicePath       = "com/company/legacy/UserService.java"
	userMapperPath        = "com/company/legacy/UserMapper.java"
	productControllerPath = "com/company/modern/ProductApiController.java"
	productServicePath    = "com/company/modern/ProductService.java"
	productMapperPath     = "com/company/modern/ProductMapper.java"
	userMapperXMLPath     = "resources/sqlmap/UserMapper.xml"
	productMapperXMLPath  = "resources/sqlmap/ProductMapper.xml"
)

// TestReadFileEncoding tests the encoding detection and file reading
func TestReadFileEncoding(t *testing.T) {
	// Test UTF-8 file
	userControllerFull := filepath.Join(testDataDir, userControllerPath)
	content, err := ReadFile(userControllerFull)
	if err != nil {
		t.Fatalf("Failed to read UserController: %v", err)
	}

	if !IsValidContent(content) {
		t.Error("File content is empty after reading")
	}

	t.Logf("✅ File reading works, content length: %d", len(content))
}

// TestJavaParserLegacyController tests parsing of legacy Spring MVC controller
func TestJavaParserLegacyController(t *testing.T) {
	userControllerFull := filepath.Join(testDataDir, userControllerPath)
	content, err := ReadFile(userControllerFull)
	if err != nil {
		t.Fatalf("Failed to read UserController: %v", err)
	}

	javaClass, err := javaparser.ParseJavaFile(content)
	if err != nil {
		t.Fatalf("Failed to parse UserController: %v", err)
	}

	// Verify package
	if !strings.Contains(javaClass.Package, "legacy") {
		t.Errorf("Expected package to contain 'legacy', got: %s", javaClass.Package)
	}

	// Verify class name
	if javaClass.Name != "UserController" {
		t.Errorf("Expected class name 'UserController', got: %s", javaClass.Name)
	}

	// Verify it's a controller
	if !javaClass.IsController() {
		t.Error("UserController should be detected as a controller")
	}

	// Verify methods exist
	if len(javaClass.Methods) == 0 {
		t.Error("Expected to find methods in UserController")
	}

	// Find the 'login' method
	var loginMethod *javaparser.Method
	for i := range javaClass.Methods {
		if javaClass.Methods[i].Name == "login" {
			loginMethod = &javaClass.Methods[i]
			break
		}
	}

	if loginMethod == nil {
		t.Fatal("Expected to find 'login' method")
	}

	// Verify login method return type contains "ModelAndView"
	if !strings.Contains(loginMethod.ReturnType, "ModelAndView") {
		t.Errorf("Expected login method to return ModelAndView, got: %s", loginMethod.ReturnType)
	}

	t.Logf("✅ UserController: Found %d methods, %d fields", len(javaClass.Methods), len(javaClass.Fields))
}

// TestJavaParserModernController tests parsing of modern REST API controller
func TestJavaParserModernController(t *testing.T) {
	productControllerFull := filepath.Join(testDataDir, productControllerPath)
	content, err := ReadFile(productControllerFull)
	if err != nil {
		t.Fatalf("Failed to read ProductApiController: %v", err)
	}

	javaClass, err := javaparser.ParseJavaFile(content)
	if err != nil {
		t.Fatalf("Failed to parse ProductApiController: %v", err)
	}

	// Verify class name
	if javaClass.Name != "ProductApiController" {
		t.Errorf("Expected class name 'ProductApiController', got: %s", javaClass.Name)
	}

	// Verify it's a REST controller
	if !javaClass.IsController() {
		t.Error("ProductApiController should be detected as a controller")
	}

	// Verify class-level @RequestMapping
	classURL := javaClass.GetClassLevelURL()
	if !strings.Contains(classURL, "product") {
		t.Errorf("Expected class URL to contain 'product', got: %s", classURL)
	}

	// Verify methods exist
	if len(javaClass.Methods) == 0 {
		t.Error("Expected to find methods in ProductApiController")
	}

	// Check for modern patterns
	foundResponseEntity := false
	foundProductDTO := false

	for _, method := range javaClass.Methods {
		// Check return type for ResponseEntity
		if strings.Contains(method.ReturnType, "ResponseEntity") {
			foundResponseEntity = true
		}

		// Check parameters for ProductDTO
		if strings.Contains(method.Params, "ProductDTO") {
			foundProductDTO = true
		}

		// Verify method has HTTP mapping
		if method.IsEndpoint() {
			httpMethod := method.GetHTTPMethod()
			methodURL := method.GetMethodURL(classURL)
			t.Logf("  Endpoint: %s %s (returns %s)", httpMethod, methodURL, method.ReturnType)
		}
	}

	if !foundResponseEntity {
		t.Error("Expected to find ResponseEntity return type in ProductApiController")
	}

	if !foundProductDTO {
		t.Error("Expected to find ProductDTO parameter in ProductApiController")
	}

	t.Logf("✅ ProductApiController: Found %d methods", len(javaClass.Methods))
}

// TestJavaParserService tests parsing of service classes
func TestJavaParserService(t *testing.T) {
	userServiceFull := filepath.Join(testDataDir, userServicePath)
	content, err := ReadFile(userServiceFull)
	if err != nil {
		t.Fatalf("Failed to read UserService: %v", err)
	}

	javaClass, err := javaparser.ParseJavaFile(content)
	if err != nil {
		t.Fatalf("Failed to parse UserService: %v", err)
	}

	// Verify class name
	if javaClass.Name != "UserService" {
		t.Errorf("Expected class name 'UserService', got: %s", javaClass.Name)
	}

	// Verify it's a service
	if !javaClass.IsService() {
		t.Error("UserService should be detected as a service")
	}

	// Verify it has methods
	if len(javaClass.Methods) == 0 {
		t.Error("Expected to find methods in UserService")
	}

	t.Logf("✅ UserService: Found %d methods", len(javaClass.Methods))
}

// TestXMLParserUserMapper tests parsing of MyBatis XML mapper
func TestXMLParserUserMapper(t *testing.T) {
	userMapperXMLFull := filepath.Join(testDataDir, userMapperXMLPath)
	content, err := ReadFile(userMapperXMLFull)
	if err != nil {
		t.Fatalf("Failed to read UserMapper.xml: %v", err)
	}

	mapper, err := xmlparser.ParseXMLFile(content)
	if err != nil {
		t.Fatalf("Failed to parse UserMapper.xml: %v", err)
	}

	// Verify namespace
	if !strings.Contains(mapper.Namespace, "UserMapper") {
		t.Errorf("Expected namespace to contain 'UserMapper', got: %s", mapper.Namespace)
	}

	// Verify SQL statements exist
	if len(mapper.SQLs) == 0 {
		t.Fatal("Expected to find SQL statements in UserMapper.xml")
	}

	// Look for specific SQL ID: selectUserByCredentials
	foundSelectUser := false
	for _, sql := range mapper.SQLs {
		if sql.ID == "selectUserByCredentials" {
			foundSelectUser = true

			// Verify SQL content
			if !strings.Contains(strings.ToUpper(sql.Content), "SELECT") {
				t.Error("Expected selectUserByCredentials to contain SELECT statement")
			}

			t.Logf("  SQL: %s [%s]", sql.ID, sql.Type)
			t.Logf("  Content: %s", sql.Content)
		}
	}

	if !foundSelectUser {
		t.Error("Expected to find SQL with ID 'selectUserByCredentials'")
	}

	// Verify counts
	counts := mapper.CountByType()
	t.Logf("✅ UserMapper.xml: %d total statements (select:%d, insert:%d, update:%d, delete:%d)",
		mapper.CountStatements(), counts["select"], counts["insert"], counts["update"], counts["delete"])
}

// TestXMLParserProductMapper tests parsing of Product mapper
func TestXMLParserProductMapper(t *testing.T) {
	productMapperXMLFull := filepath.Join(testDataDir, productMapperXMLPath)
	content, err := ReadFile(productMapperXMLFull)
	if err != nil {
		t.Fatalf("Failed to read ProductMapper.xml: %v", err)
	}

	mapper, err := xmlparser.ParseXMLFile(content)
	if err != nil {
		t.Fatalf("Failed to parse ProductMapper.xml: %v", err)
	}

	// Verify namespace
	if !strings.Contains(mapper.Namespace, "ProductMapper") {
		t.Errorf("Expected namespace to contain 'ProductMapper', got: %s", mapper.Namespace)
	}

	// Verify SQL statements
	if len(mapper.SQLs) == 0 {
		t.Error("Expected to find SQL statements in ProductMapper.xml")
	}

	t.Logf("✅ ProductMapper.xml: Found %d SQL statements", len(mapper.SQLs))
}

// TestFileLinkingLogic tests the linking between Controller -> Service -> Mapper
func TestFileLinkingLogic(t *testing.T) {
	// Parse UserController
	userControllerFull := filepath.Join(testDataDir, userControllerPath)
	controllerContent, _ := ReadFile(userControllerFull)
	controller, err := javaparser.ParseJavaFile(controllerContent)
	if err != nil {
		t.Fatalf("Failed to parse UserController: %v", err)
	}

	// Get injected services
	services := controller.GetInjectedServices()
	if len(services) == 0 {
		t.Error("Expected UserController to have @Autowired services")
	}

	t.Logf("Controller injected services: %v", services)

	// Parse UserService
	userServiceFull := filepath.Join(testDataDir, userServicePath)
	serviceContent, _ := ReadFile(userServiceFull)
	service, err := javaparser.ParseJavaFile(serviceContent)
	if err != nil {
		t.Fatalf("Failed to parse UserService: %v", err)
	}

	// Get service's injected mappers
	mappers := service.GetInjectedServices()
	t.Logf("Service injected mappers: %v", mappers)

	// Parse UserMapper.xml
	userMapperXMLFull := filepath.Join(testDataDir, userMapperXMLPath)
	mapperContent, _ := ReadFile(userMapperXMLFull)
	mapper, err := xmlparser.ParseXMLFile(mapperContent)
	if err != nil {
		t.Fatalf("Failed to parse UserMapper.xml: %v", err)
	}

	// Verify mapper matches the interface name
	if !mapper.MatchesJavaInterface("UserMapper") {
		t.Errorf("Mapper namespace doesn't match UserMapper interface")
	}

	t.Log("✅ Linking logic validated: Controller -> Service -> Mapper -> SQL")
}

// TestUtilityFunctions tests the utility helper functions
func TestUtilityFunctions(t *testing.T) {
	// Test CombineURLPaths
	combined := CombineURLPaths("/api/v1/users", "/login")
	if combined != "/api/v1/users/login" {
		t.Errorf("CombineURLPaths failed: got %s", combined)
	}

	// Test ExtractAnnotationValue
	value := ExtractAnnotationValue(`@RequestMapping("/users")`)
	if value != "/users" {
		t.Errorf("ExtractAnnotationValue failed: got %s", value)
	}

	// Test ParseMethodParams
	params := ParseMethodParams("String username, int age, List<String> roles")
	if len(params) != 3 {
		t.Errorf("ParseMethodParams failed: expected 3 params, got %d", len(params))
	}

	// Test ExtractGenericType
	genericType := ExtractGenericType("ResponseEntity<ProductDTO>")
	if genericType != "ProductDTO" {
		t.Errorf("ExtractGenericType failed: got %s", genericType)
	}

	t.Log("✅ Utility functions working correctly")
}

// TestEncodingDetection tests the encoding detection logic
func TestEncodingDetection(t *testing.T) {
	// Test UTF-8 detection
	utf8Data := []byte("Hello, World! 안녕하세요")
	encoding := DetectEncoding(utf8Data)
	if encoding != "UTF-8" {
		t.Errorf("Expected UTF-8, got: %s", encoding)
	}

	t.Log("✅ Encoding detection working")
}

// TestHybridSampleCompleteness tests that all expected files can be parsed
func TestHybridSampleCompleteness(t *testing.T) {
	testFiles := []struct {
		path     string
		fileType string
	}{
		{userControllerPath, "java"},
		{userServicePath, "java"},
		{userMapperPath, "java"},
		{productControllerPath, "java"},
		{productServicePath, "java"},
		{productMapperPath, "java"},
		{userMapperXMLPath, "xml"},
		{productMapperXMLPath, "xml"},
	}

	successCount := 0
	for _, tf := range testFiles {
		fullPath := filepath.Join(testDataDir, tf.path)
		content, err := ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", tf.path, err)
			continue
		}

		if tf.fileType == "java" {
			_, err := javaparser.ParseJavaFile(content)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", tf.path, err)
			} else {
				successCount++
			}
		} else if tf.fileType == "xml" {
			_, err := xmlparser.ParseXMLFile(content)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", tf.path, err)
			} else {
				successCount++
			}
		}
	}

	t.Logf("✅ Parsed %d/%d files successfully", successCount, len(testFiles))

	if successCount != len(testFiles) {
		t.Errorf("Not all files parsed successfully")
	}
}
