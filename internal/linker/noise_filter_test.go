package linker

import (
	"testing"

	"spec-recon/internal/javaparser"
	"spec-recon/internal/model"
)

// TestNoiseFilter verifies that Java keywords and constructors are filtered out
func TestNoiseFilter(t *testing.T) {
	// Create a test Java class with noise in the method body
	testClass := &javaparser.JavaClass{
		Package: "com.test",
		Name:    "TestController",
		Annotations: []javaparser.Annotation{
			{Name: "Controller"},
		},
		Fields: []javaparser.Field{
			{Name: "userService", Type: "UserService"},
		},
		Methods: []javaparser.Method{
			{
				Name:       "testMethod",
				ReturnType: "String",
				Params:     "User user",
				Body: `
					if (user != null) {
						ModelAndView mav = new ModelAndView("home");
						String result = userService.processUser(user);
						System.out.println("Processing user");
						return result;
					}
					return null;
				`,
			},
		},
	}

	// Create a UserService class
	userServiceClass := &javaparser.JavaClass{
		Package: "com.test",
		Name:    "UserService",
		Annotations: []javaparser.Annotation{
			{Name: "Service"},
		},
		Methods: []javaparser.Method{
			{
				Name:       "processUser",
				ReturnType: "String",
				Params:     "User user",
				Body:       "return user.getName();",
			},
		},
	}

	// Initialize pool and linker
	pool := NewComponentPool()
	linker := NewLinker(pool)

	// Add classes
	pool.AddJavaClass(testClass, "")
	pool.AddJavaClass(userServiceClass, "")

	// Run linking
	if err := linker.Link(); err != nil {
		t.Fatalf("Linking failed: %v", err)
	}

	// Verify results
	testMethod := pool.GetMethod("com.test.TestController.testMethod")
	if testMethod == nil {
		t.Fatal("TestController.testMethod not found")
	}

	t.Logf("TestMethod has %d children", len(testMethod.Children))

	// Check that noise is filtered out
	for _, child := range testMethod.Children {
		t.Logf("Child found: %s (Type: %s)", child.ID, child.Type)

		// These should NOT be in the children
		if child.Method == "if" {
			t.Error("❌ FAIL: 'if' keyword was not filtered out")
		}
		if child.Method == "new" {
			t.Error("❌ FAIL: 'new' keyword was not filtered out")
		}
		if child.Method == "return" {
			t.Error("❌ FAIL: 'return' keyword was not filtered out")
		}
		if child.ID == "ModelAndView" || child.Method == "ModelAndView" {
			t.Error("❌ FAIL: 'ModelAndView' constructor was not filtered out")
		}
		if child.ID == "System.out.println" || child.Method == "println" {
			t.Error("❌ FAIL: 'System.out.println' was not filtered out")
		}
	}

	// Verify that valid business logic call IS present
	foundValidCall := false
	for _, child := range testMethod.Children {
		if child.Method == "processUser" && child.Type == model.NodeTypeService {
			foundValidCall = true
			t.Log("✅ PASS: Valid business logic call 'processUser' was preserved")
		}
	}

	if !foundValidCall {
		t.Error("❌ FAIL: Valid business logic call 'processUser' was not found")
	}

	// Additional verification: Check that the number of children is reasonable
	// Should only have 1 child (userService.processUser), not 5+ with noise
	if len(testMethod.Children) > 2 {
		t.Errorf("❌ FAIL: Too many children (%d). Expected 1-2 (only valid business calls)", len(testMethod.Children))
	}

	if len(testMethod.Children) == 0 {
		t.Error("❌ FAIL: No children found. Expected at least 1 (userService.processUser)")
	}

	t.Log("✅ Noise filter test completed")
}

// TestIgnoredTokensMap verifies the IgnoredTokens map contains expected keywords
func TestIgnoredTokensMap(t *testing.T) {
	expectedKeywords := []string{
		"if", "else", "for", "while", "return", "new", "this", "super",
		"System", "ModelAndView", "println", "log",
	}

	for _, keyword := range expectedKeywords {
		if !IgnoredTokens[keyword] {
			t.Errorf("Expected keyword '%s' to be in IgnoredTokens map", keyword)
		}
	}

	t.Logf("✅ IgnoredTokens map contains %d entries", len(IgnoredTokens))
}

// TestConstructorFilter verifies that constructor calls are properly filtered
func TestConstructorFilter(t *testing.T) {
	testCases := []struct {
		name     string
		varName  string
		expected bool
	}{
		{"ModelAndView constructor", "ModelAndView", true},
		{"ResponseEntity constructor", "ResponseEntity", true},
		{"ArrayList constructor", "ArrayList", true},
		{"HashMap constructor", "HashMap", true},
		{"System class", "System", true},
		{"Valid UserService", "UserService", false},
		{"Valid ProductMapper", "ProductMapper", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isConstructorCall(tc.varName)
			if result != tc.expected {
				t.Errorf("isConstructorCall(%s) = %v, expected %v", tc.varName, result, tc.expected)
			}
		})
	}
}
