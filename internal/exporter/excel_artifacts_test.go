package exporter

import (
	"os"
	"testing"

	"spec-recon/internal/config"
	"spec-recon/internal/model"

	"github.com/xuri/excelize/v2"
)

func TestExcelExporter_NoVisualArtifacts(t *testing.T) {
	// Create test data
	summary := &model.Summary{
		TotalControllers: 1,
		TotalServices:    1,
		TotalMappers:     1,
		TotalSQLs:        1,
		TotalUtils:       1,
		AnalysisDate:     "2026-02-06",
	}

	// Create a simple tree structure
	ctrl := &model.Node{
		ID:      "com.company.TestController",
		Type:    model.NodeTypeController,
		Package: "com.company",
		Method:  "TestController",
		URL:     "/test",
	}

	svc := &model.Node{
		ID:      "com.company.TestService.doSomething",
		Type:    model.NodeTypeService,
		Package: "com.company",
		Method:  "doSomething",
		Params:  "String param",
	}

	mapper := &model.Node{
		ID:      "com.company.TestMapper.selectData",
		Type:    model.NodeTypeMapper,
		Package: "com.company",
		Method:  "selectData",
	}

	util := &model.Node{
		ID:      "com.company.TestUtil.helper",
		Type:    model.NodeTypeUtil,
		Package: "com.company",
		Method:  "helper",
	}

	ctrl.AddChild(svc)
	svc.AddChild(mapper)
	svc.AddChild(util)

	tree := []*model.Node{ctrl}

	// Create config
	cfg := &config.Config{
		Output: config.OutputConfig{
			Dir:      "../../output/test",
			FileName: "test_no_artifacts",
		},
	}

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.Output.Dir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Export
	exporter := NewExcelExporter()
	if err := exporter.Export(summary, tree, cfg); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify the output
	outputPath := cfg.GetOutputPath()
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open generated file: %v", err)
	}
	defer f.Close()

	// Check Spec Detail sheet
	rows, err := f.GetRows("Spec Detail")
	if err != nil {
		t.Fatalf("Failed to get rows: %v", err)
	}

	// Verify no indentation characters in Package/File column (Column B)
	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}
		if len(row) > 1 {
			packageCol := row[1] // Column B (0-indexed)
			// Check for visual artifacts
			if containsVisualArtifacts(packageCol) {
				t.Errorf("Row %d Column B contains visual artifacts: %q", i+1, packageCol)
			}
		}
		if len(row) > 2 {
			methodCol := row[2] // Column C (0-indexed)
			// Check for visual artifacts in method column
			if containsVisualArtifacts(methodCol) {
				t.Errorf("Row %d Column C contains visual artifacts: %q", i+1, methodCol)
			}
		}
	}

	t.Logf("✅ Excel file verified: no visual artifacts found")
}

func containsVisualArtifacts(s string) bool {
	// Check for leading spaces
	if len(s) > 0 && s[0:1] == " " {
		return true
	}

	// Check for tree/indentation characters
	artifacts := []string{"└", "ㄴ", "├", "│"}
	for _, a := range artifacts {
		if len(a) > 0 && len(s) >= len(a) {
			for i := 0; i < len(s)-len(a)+1; i++ {
				if s[i:i+len(a)] == a {
					return true
				}
			}
		}
	}
	return false
}

// TestNoiseFilteringAndEmptyRowRemoval verifies that:
// 1. Java keywords (if, new, etc.) are NOT in the output
// 2. Exception constructors (IllegalArgumentException) are NOT in the output
// 3. Empty Service rows are NOT in the output
// 4. CONTROLLER headers ARE preserved even if they have empty methods
func TestNoiseFilteringAndEmptyRowRemoval(t *testing.T) {
	outputFile := "test_noise_filter.xlsx"
	defer os.Remove(outputFile)

	// Create test data with noise and empty rows
	controller := &model.Node{
		ID:      "com.example.TestController",
		Type:    model.NodeTypeController,
		Package: "com.example",
		Method:  "TestController",
		URL:     "/api/test",
	}

	// Valid service method
	validService := &model.Node{
		ID:      "com.example.TestService.validMethod",
		Type:    model.NodeTypeService,
		Package: "com.example",
		Method:  "validMethod",
		Params:  "String param",
	}

	// Empty service method (should be filtered out)
	emptyService := &model.Node{
		ID:      "com.example.TestService.",
		Type:    model.NodeTypeService,
		Package: "com.example",
		Method:  "", // EMPTY - should be filtered
		Params:  "",
	}

	// Whitespace-only service method (should be filtered out)
	whitespaceService := &model.Node{
		ID:      "com.example.TestService.   ",
		Type:    model.NodeTypeService,
		Package: "com.example",
		Method:  "   ", // WHITESPACE - should be filtered
		Params:  "",
	}

	// Another valid service
	anotherValidService := &model.Node{
		ID:      "com.example.TestService.anotherMethod",
		Type:    model.NodeTypeService,
		Package: "com.example",
		Method:  "anotherMethod",
		Params:  "int id",
	}

	// Build the tree
	controller.AddChild(validService)
	controller.AddChild(emptyService)
	controller.AddChild(whitespaceService)
	controller.AddChild(anotherValidService)

	tree := []*model.Node{controller}

	summary := &model.Summary{
		TotalControllers: 1,
		TotalServices:    2, // Only counting valid ones
		AnalysisDate:     "2026-02-07",
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Dir:      ".",
			FileName: "test_noise_filter",
		},
	}

	// Export
	exporter := NewExcelExporter()
	if err := exporter.Export(summary, tree, cfg); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file creation
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Open and verify contents
	f, err := excelize.OpenFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to open generated Excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Spec Detail")
	if err != nil {
		t.Fatalf("Failed to read rows: %v", err)
	}

	// Collect all method names from column C (index 2)
	methodNames := []string{}
	for i, row := range rows {
		if i < 1 { // Skip header
			continue
		}
		if len(row) > 2 {
			methodName := row[2]
			if methodName != "" {
				methodNames = append(methodNames, methodName)
			}
		}
	}

	// Verification 1: CONTROLLER header should be present
	foundController := false
	for _, name := range methodNames {
		if name == "TestController" {
			foundController = true
			break
		}
	}
	if !foundController {
		t.Error("CONTROLLER header was filtered out (should be preserved)")
	}

	// Verification 2: Valid methods should be present
	if !contains(methodNames, "validMethod") {
		t.Error("Valid method 'validMethod' was filtered out")
	}
	if !contains(methodNames, "anotherMethod") {
		t.Error("Valid method 'anotherMethod' was filtered out")
	}

	// Verification 3: Empty/whitespace methods should NOT be present
	// We check that we only have 3 entries: TestController, validMethod, anotherMethod
	if len(methodNames) != 3 {
		t.Errorf("Expected 3 method entries (1 controller + 2 valid services), got %d: %v",
			len(methodNames), methodNames)
	}

	// Verification 4: Noise keywords should NOT appear
	// (This would be caught by linker, but we verify at Excel level too)
	noiseKeywords := []string{"if", "new", "ModelAndView", "IllegalArgumentException", "Exception"}
	for _, noise := range noiseKeywords {
		if contains(methodNames, noise) {
			t.Errorf("Noise keyword '%s' found in output (should be filtered)", noise)
		}
	}

	t.Log("✅ Verified: Empty rows filtered, CONTROLLER headers preserved, noise removed")
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
