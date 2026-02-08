package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/javaparser"
	"spec-recon/internal/linker"
	"spec-recon/internal/model"
)

const (
	testDataDir    = "../../../testdata/hybrid_sample"
	baseOutputFile = "test_output_base" // Will produce test_output_base/openapi.json ?? No, dir(file) + openapi.json
)

func TestOpenAPIExport(t *testing.T) {
	// 1. Setup Data
	javaFiles := []string{
		"com/company/legacy/UserController.java",
		"com/company/modern/ProductApiController.java",
	}

	pool := linker.NewComponentPool()

	for _, f := range javaFiles {
		content, err := analyzer.ReadFile(filepath.Join(testDataDir, f))
		if err != nil {
			t.Fatal(err)
		}
		cls, err := javaparser.ParseJavaFile(content)
		if err != nil {
			t.Fatal(err)
		}
		pool.AddJavaClass(cls, content)
	}

	tree := []*model.Node{}
	for _, node := range pool.ClassMap {
		if node.Type == model.NodeTypeController {
			tree = append(tree, node)
		}
	}

	summary := &model.Summary{} // Empty is fine for OpenAPI

	// Config should point to a file, and OpenAPI uses the DIR of that file.
	// We'll use current dir.
	cfg := &config.Config{
		Output: config.OutputConfig{
			Dir:      ".",
			FileName: "dummy",
		},
	}
	targetFile := "openapi.json"

	// 2. Build
	exporter := NewOpenAPIExporter()
	if err := exporter.Export(summary, tree, cfg); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer os.Remove(targetFile)

	// 3. Verify Content
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}

	var result OpenAPI
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatal(err)
	}

	// Verify that view controllers (ModelAndView) are filtered out
	if _, ok := result.Paths["/user/login"]; ok {
		t.Error("View controller endpoint /user/login should be filtered out (returns ModelAndView)")
	}

	if _, ok := result.Paths["/user/list"]; ok {
		t.Error("View controller endpoint /user/list should be filtered out (returns ModelAndView)")
	}

	// Check Modern REST API Path (/api/v1/product/register)
	if path, ok := result.Paths["/api/v1/product/register"]; !ok {
		t.Error("Missing REST API path /api/v1/product/register")
	} else {
		if _, ok := path["post"]; !ok {
			t.Error("Missing POST method for /api/v1/product/register")
		}
	}

	// Verify we have at least some endpoints (not empty)
	if len(result.Paths) == 0 {
		t.Error("No paths in OpenAPI spec - filtering may be too aggressive")
	}
}
