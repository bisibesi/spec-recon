package exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/javaparser"
	"spec-recon/internal/linker"
	"spec-recon/internal/model"

	"github.com/xuri/excelize/v2"
)

const (
	testDataDir = "../../testdata/hybrid_sample"
	outputFile  = "test_report.xlsx"
)

func TestExcelExport(t *testing.T) {
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

	// 2. Mock Tree & Summary
	tree := []*model.Node{}
	for _, node := range pool.ClassMap {
		if node.Type == model.NodeTypeController {
			tree = append(tree, node)
		}
	}

	summary := &model.Summary{
		TotalControllers: 2,
		AnalysisDate:     "2023-10-27",
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Dir:      ".",
			FileName: "test_report",
		},
	}

	// 3. Export
	exporter := NewExcelExporter()
	if err := exporter.Export(summary, tree, cfg); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// 4. Verify File Creation
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	} else {
		t.Logf("✅ Output file created: %s", outputFile)
	}

	// 5. Verify Row Order (Util after SQL)
	f, err := excelize.OpenFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to open generated Excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Spec Detail")
	if err != nil {
		t.Fatalf("Failed to read rows: %v", err)
	}

	// Logic: In a block (Controller), last SQL should be before first Util
	firstUtilRow := -1

	// Start from row 2 (headers is 0, 1 is row 2)
	// We want to verify that WITHIN a controller block, order matches.
	// Since we have multiple controllers in our test data, we just check global constraint:
	// existing logic requires separated sections.
	// A simpler check: scan rows, if we see Util, we shouldn't see SQL/Service/Mapper afterwards UNTIL the next Controller.

	inUtilSection := false

	for i, row := range rows {
		if i < 1 {
			continue
		} // Skip header
		if len(row) == 0 {
			continue
		}

		cellType := row[0] // Column A: [Type]

		if strings.Contains(cellType, "[Controller]") {
			// New Block Reset
			inUtilSection = false
			firstUtilRow = -1
		} else if strings.Contains(cellType, "[SQL]") {
			if inUtilSection {
				t.Errorf("Row %d: Found [SQL] inside Util section (after Util)", i+1)
			}
		} else if strings.Contains(cellType, "[Service]") || strings.Contains(cellType, "[Mapper]") {
			if inUtilSection {
				t.Errorf("Row %d: Found Business Logic inside Util section", i+1)
			}
		} else if strings.Contains(cellType, "[Util]") {
			inUtilSection = true
			if firstUtilRow == -1 {
				firstUtilRow = i
			}
		}
	}

	t.Log("✅ Verified Row Ordering: Utils appear at bottom of each Controller block")

	// Cleanup
	os.Remove(outputFile)
}
