package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSystemIntegration(t *testing.T) {
	// 1. Setup Environment
	rootDir, _ := filepath.Abs("..")
	cmdDir := filepath.Join(rootDir, "cmd", "spec-recon")
	// configPath := filepath.Join(rootDir, "config.example.yaml") // Unused
	outputDir := filepath.Join(rootDir, "output", "system_test")

	binaryName := "spec-recon-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(rootDir, binaryName)

	// Clean up previous runs
	os.Remove(binaryPath)
	os.RemoveAll(outputDir)

	// 2. Build the Application
	t.Logf("Building application from %s...", cmdDir)
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = cmdDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build application: %v", err)
	}
	defer os.Remove(binaryPath) // Cleanup binary

	// 3. Create a Custom Config for the Test
	testConfigContent := `
project:
  root_dir: "./testdata/hybrid_sample"
  base_package: "com.company"
  encoding: ["utf-8"]

analysis:
  exclude_dirs: ["**/test/**"]
  util_patterns: ["*Util"]

output:
  dir: "./output"
  file_name: "e2e_report"
`
	testConfigPath := filepath.Join(rootDir, "config_test.yaml")
	if err := os.WriteFile(testConfigPath, []byte(testConfigContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(testConfigPath)

	// 4. Run the Binary
	t.Log("Running application binary...")
	runCmd := exec.Command(binaryPath, "-config", testConfigPath, "-format", "excel,html,word,json")
	runCmd.Dir = rootDir
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		t.Fatalf("Application run failed: %v", err)
	}

	// 5. Verify Outputs in Unified Directory
	unifiedOutputDir := filepath.Join(rootDir, "output")
	expectedFiles := []string{
		"e2e_report.xlsx",
		"e2e_report.html",
		"e2e_report.docx",
		"openapi.json",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(unifiedOutputDir, f)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			t.Errorf("Expected output file missing: %s", f)
		} else if info.Size() == 0 {
			t.Errorf("Output file is empty: %s", f)
		} else {
			t.Logf("✅ Verified output: %s (%d bytes)", f, info.Size())
		}
	}

	// 6. ZERO TOLERANCE CHECK: Verify no empty Method cells in Excel
	t.Log("Running zero-tolerance check for empty Method cells...")
	verifyNoEmptyMethodCells(t, filepath.Join(unifiedOutputDir, "e2e_report.xlsx"))
}

// verifyNoEmptyMethodCells checks that no rows have empty Method/ID cells
func verifyNoEmptyMethodCells(t *testing.T, excelPath string) {
	// This requires excelize package
	// We'll use a simple approach: run the verification script
	rootDir, _ := filepath.Abs("..")
	scriptPath := filepath.Join(rootDir, "scripts", "verify_empty_cells.go")
	cmd := exec.Command("go", "run", scriptPath, excelPath)
	cmd.Dir = rootDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Empty cell verification failed: %v\nOutput: %s", err, string(output))
	} else {
		t.Logf("✅ Zero-tolerance check passed: %s", string(output))
	}
}
