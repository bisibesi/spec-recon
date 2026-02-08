package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/exporter"
	"spec-recon/internal/javaparser"
	"spec-recon/internal/linker"
	"spec-recon/internal/model"
	"spec-recon/internal/xmlparser"
)

func TestEndToEndFlow(t *testing.T) {
	// Setup Paths
	rootDir, _ := filepath.Abs("../../testdata/hybrid_sample")
	outputDir, _ := filepath.Abs("../../output/e2e_test")

	// Cleanup output
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)

	// 1. Configure
	cfg := &config.Config{
		Project: config.ProjectConfig{
			RootDir: rootDir,
		},
		Analysis: config.AnalysisConfig{
			ExcludeDirs: []string{"**/test/**"},
		},
		Output: config.OutputConfig{
			Dir:      outputDir,
			FileName: "e2e_report",
		},
	}
	cfg.EnsureOutputDir()

	// 2. Scan & Parse
	files, err := analyzer.ScanDirectory(cfg.Project.RootDir, cfg.Analysis.ExcludeDirs)
	if err != nil {
		t.Fatalf("Scanning failed: %v", err)
	}

	pool := linker.NewComponentPool()
	for _, path := range files {
		content, _ := analyzer.ReadFile(path)
		if strings.HasSuffix(path, ".java") {
			cls, _ := javaparser.ParseJavaFile(content)
			pool.AddJavaClass(cls, content)
		} else if strings.HasSuffix(path, ".xml") {
			mapper, _ := xmlparser.ParseXMLFile(content)
			pool.AddMapperXML(mapper)
		}
	}

	// 3. Link
	mainLinker := linker.NewLinker(pool)
	tree := mainLinker.BuildCallGraph()

	// 4. Summary
	summary := &model.Summary{
		TotalControllers: 1, // Mock counts or calculate
		AnalysisDate:     "2023-10-27",
		ClassMap:         pool.ClassMap,
		FieldTypeMap:     pool.FieldTypeMap,
	}

	// 5. Export (Excel, HTML, OpenAPI)
	// Skip Word because we don't have a template.docx
	exporters := exporter.GetExporters([]string{"excel", "html", "json"})

	for _, exp := range exporters {
		if err := exp.Export(summary, tree, cfg); err != nil {
			t.Errorf("Export failed: %v", err)
		}
	}

	// 6. Verify Outputs
	expectedFiles := []string{
		"e2e_report.xlsx",
		"e2e_report.html",
		"openapi.json",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected output file missing: %s", path)
		} else {
			t.Logf("✅ Verified output: %s", f)
		}
	}

	// 7. STRICT VALIDATION: Check for banned keywords in Excel
	t.Log("Running strict keyword validation on Excel output...")
	excelPath := filepath.Join(outputDir, "e2e_report.xlsx")
	if err := validateExcelNoBannedKeywords(t, excelPath); err != nil {
		t.Fatalf("❌ BANNED KEYWORD FOUND: %v", err)
	}
	t.Log("✅ No banned keywords found in Excel")

	// 8. STRICT VALIDATION: Check for DTO fields in HTML/JSON
	t.Log("Running DTO field validation...")
	htmlPath := filepath.Join(outputDir, "e2e_report.html")
	if err := validateDTOFields(t, htmlPath); err != nil {
		t.Errorf("⚠️  DTO field validation: %v", err)
	} else {
		t.Log("✅ DTO fields found in documentation")
	}
}

// validateExcelNoBannedKeywords checks that Excel output doesn't contain banned keywords
func validateExcelNoBannedKeywords(t *testing.T, excelPath string) error {
	// Import excelize for reading Excel
	// Note: This requires github.com/xuri/excelize/v2
	// For now, we'll read the file as text and do basic checks
	// In production, use excelize to read cells properly

	// Banned keywords that should NEVER appear in output
	bannedKeywords := []string{
		"if",
		"throw",
		"new",
		"IllegalArgumentException",
		"NullPointerException",
		"switch",
		"catch",
		"synchronized",
	}

	// Read file content (simplified check)
	// In production, use excelize.OpenFile and iterate cells
	content, err := os.ReadFile(excelPath)
	if err != nil {
		return err
	}

	contentStr := string(content)
	for _, keyword := range bannedKeywords {
		// Check if keyword appears (case-sensitive exact match would be better with excelize)
		if strings.Contains(contentStr, keyword) {
			return fmt.Errorf("found banned keyword '%s' in Excel output", keyword)
		}
	}

	return nil
}

// validateDTOFields checks that documentation contains expected DTO fields
func validateDTOFields(t *testing.T, htmlPath string) error {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return err
	}

	contentStr := strings.ToLower(string(content))

	// Expected fields from ProductDTO (based on testdata)
	expectedFields := []string{
		"productid",
		"productname",
		"price",
		"name", // Could be from various DTOs
	}

	foundAny := false
	for _, field := range expectedFields {
		if strings.Contains(contentStr, field) {
			t.Logf("  Found DTO field: %s", field)
			foundAny = true
		}
	}

	if !foundAny {
		return fmt.Errorf("no DTO fields found in documentation (expected: %v)", expectedFields)
	}

	return nil
}
