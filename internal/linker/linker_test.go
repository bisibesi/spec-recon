package linker

import (
	"path/filepath"
	"strings"
	"testing"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/javaparser"
	"spec-recon/internal/model"
	"spec-recon/internal/xmlparser"
)

const (
	testDataDir = "../../testdata/hybrid_sample"
)

func TestLinkerIntegration(t *testing.T) {
	// 1. Parse all files
	javaFiles := []string{
		"com/company/legacy/UserController.java",
		"com/company/legacy/UserService.java",
		"com/company/legacy/UserMapper.java",
		"com/company/modern/ProductApiController.java",
		"com/company/modern/ProductService.java",
		"com/company/modern/ProductMapper.java",
		"com/company/common/StringUtil.java",
		"com/company/common/ProductDTO.java",
	}

	xmlFiles := []string{
		"resources/sqlmap/UserMapper.xml",
		"resources/sqlmap/ProductMapper.xml",
	}

	classes := []*javaparser.JavaClass{}
	sourceContents := make(map[string]string)
	mappers := []*xmlparser.MapperXML{}

	// Parse Java
	for _, f := range javaFiles {
		fullPath := filepath.Join(testDataDir, f)
		content, err := analyzer.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", f, err)
		}

		cls, err := javaparser.ParseJavaFile(content)
		if err != nil {
			t.Fatalf("Failed to parse %s: %v", f, err)
		}

		classes = append(classes, cls)
		fullClassName := cls.Package + "." + cls.Name
		sourceContents[fullClassName] = content
	}

	// Parse XML
	for _, f := range xmlFiles {
		fullPath := filepath.Join(testDataDir, f)
		content, err := analyzer.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", f, err)
		}

		mapper, err := xmlparser.ParseXMLFile(content)
		if err != nil {
			t.Fatalf("Failed to parse %s: %v", f, err)
		}
		mappers = append(mappers, mapper)
	}

	// 2. Initialize Linker
	pool := NewComponentPool()
	linker := NewLinker(pool)

	// 3. Load Data
	if err := linker.LoadJavaClasses(classes, sourceContents); err != nil {
		t.Fatalf("Failed to load Java classes: %v", err)
	}
	if err := linker.LoadMapperXMLs(mappers); err != nil {
		t.Fatalf("Failed to load XML mappers: %v", err)
	}

	// 4. Run Linking
	if err := linker.Link(); err != nil {
		t.Fatalf("Linking failed: %v", err)
	}

	// 5. Verification
	verifyLegacyFlow(t, linker)
	verifyModernFlow(t, linker)
}

func verifyLegacyFlow(t *testing.T, l *Linker) {
	t.Log("--- Verifying Legacy Flow (Usage of ModelAndView) ---")

	// 1. Controller: UserController.login -> UserService.authenticateUser
	pool := l.Pool
	userControllerLogin := pool.GetMethod("com.company.legacy.UserController.login")
	if userControllerLogin == nil {
		t.Fatal("Legacy: UserController.login not found")
	}

	t.Logf("UserController.login found. Children: %d", len(userControllerLogin.Children))

	foundServiceCall := false
	for _, child := range userControllerLogin.Children {
		// Child ID should be com.company.legacy.UserService.authenticateUser
		if strings.Contains(child.ID, "UserService.authenticateUser") {
			foundServiceCall = true
			t.Log("✅ Link found: UserController.login -> UserService.authenticateUser")

			// 2. Service: UserService.authenticateUser -> UserMapper.selectUserByCredentials
			verifyServiceToMapper(t, child)
		}
	}

	if !foundServiceCall {
		t.Error("❌ Link MISSING: UserController.login -> UserService.authenticateUser")
		// Debug
		body := pool.MethodBodyMap["com.company.legacy.UserController.login"]
		t.Logf("Method Body:\n%s", body)
		fieldType := pool.ResolveFieldType("com.company.legacy.UserController", "userService")
		t.Logf("Resolved userService type: %s", fieldType)
	}
}

func verifyServiceToMapper(t *testing.T, serviceMethod *model.Node) {
	foundMapperCall := false
	for _, child := range serviceMethod.Children {
		if strings.Contains(child.ID, "UserMapper.selectUserByCredentials") {
			foundMapperCall = true
			t.Log("✅ Link found: UserService.authenticateUser -> UserMapper.selectUserByCredentials")

			// 3. Mapper: UserMapper.selectUserByCredentials -> XML SQL
			verifyMapperToSQL(t, child)
		}
	}

	if !foundMapperCall {
		t.Error("❌ Link MISSING: UserService.authenticateUser -> UserMapper.selectUserByCredentials")
	}
}

func verifyMapperToSQL(t *testing.T, mapperMethod *model.Node) {
	foundSQL := false
	for _, child := range mapperMethod.Children {
		if child.Type == model.NodeTypeSQL && strings.Contains(child.ID, "selectUserByCredentials") {
			foundSQL = true
			t.Log("✅ Link found: UserMapper.selectUserByCredentials -> XML SQL")
			t.Logf("   Query: %s", child.Comment)
		}
	}

	if !foundSQL {
		t.Error("❌ Link MISSING: UserMapper.selectUserByCredentials -> XML SQL")
	}
}

func verifyModernFlow(t *testing.T, l *Linker) {
	t.Log("--- Verifying Modern Flow (REST API) ---")

	// ProductApiController.getProductList -> ProductService.getProductList
	pool := l.Pool
	methodNode := pool.GetMethod("com.company.modern.ProductApiController.getProductList")
	if methodNode == nil {
		t.Fatal("Modern: ProductApiController.getProductList not found")
	}

	foundService := false
	for _, child := range methodNode.Children {
		if strings.Contains(child.ID, "ProductService.getProductList") {
			foundService = true
			t.Log("✅ Link found: ProductApiController.getProductList -> ProductService.getProductList")

			// Verify next hop: ProductService.getProductList -> ProductMapper.selectAllProducts
			foundMapper := false
			for _, subChild := range child.Children {
				if strings.Contains(subChild.ID, "ProductMapper.selectAllProducts") {
					foundMapper = true
					t.Log("✅ Link found: ProductService.getProductList -> ProductMapper.selectAllProducts")
				}
			}
			if !foundMapper {
				t.Error("❌ Link MISSING: ProductService.getProductList -> ProductMapper.selectAllProducts")
			}
		}
	}

	if !foundService {
		t.Error("❌ Link MISSING: ProductApiController.getProductList -> ProductService.getProductList")
		// Debug
		body := pool.MethodBodyMap["com.company.modern.ProductApiController.getProductList"]
		t.Logf("Controller Body:\n%s", body)
	}
}
