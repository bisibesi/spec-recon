package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigWithDefaults(t *testing.T) {
	// Load config without a file (should use defaults)
	cfg, err := Load("nonexistent.yaml")
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	// Verify defaults
	if cfg.Project.RootDir == "" {
		t.Error("Expected RootDir to be set")
	}

	if cfg.Output.Dir == "" {
		t.Error("Expected Output.Dir to be set")
	}

	if cfg.Output.FileName == "" {
		t.Error("Expected Output.FileName to be set")
	}

	if len(cfg.Project.Encoding) == 0 {
		t.Error("Expected at least one encoding hint")
	}

	if len(cfg.Analysis.UtilPatterns) == 0 {
		t.Error("Expected at least one util pattern")
	}

	t.Logf("Config loaded successfully with defaults")
	cfg.Print()
}

func TestIsUtil(t *testing.T) {
	cfg := &Config{
		Analysis: AnalysisConfig{
			UtilPatterns: []string{
				"*Util",
				"*Utils",
				"*Helper",
				"*DTO",
				"*VO",
			},
		},
	}

	tests := []struct {
		className string
		expected  bool
	}{
		{"StringUtil", true},
		{"DateUtils", true},
		{"ValidationHelper", true},
		{"UserDTO", true},
		{"ProductVO", true},
		{"UserController", false},
		{"UserService", false},
		{"UserMapper", false},
		{"User", false},
	}

	for _, tt := range tests {
		result := cfg.IsUtil(tt.className)
		if result != tt.expected {
			t.Errorf("IsUtil(%s) = %v, expected %v", tt.className, result, tt.expected)
		}
	}
}

func TestShouldExclude(t *testing.T) {
	cfg := &Config{
		Analysis: AnalysisConfig{
			ExcludeDirs: []string{
				"**/test/**",
				"**/target/**",
				"**/.git/**",
			},
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"src/test/java/UserTest.java", true},
		{"src/main/java/User.java", false},
		{"project/target/classes/User.class", true},
		{"project/.git/config", true},
		{"src/main/java/service/UserService.java", false},
		{"build/target/output.jar", true}, // Contains "/target/"
		{"myproject/.git/HEAD", true},     // Contains "/.git/"
	}

	for _, tt := range tests {
		result := cfg.ShouldExclude(tt.path)
		if result != tt.expected {
			t.Errorf("ShouldExclude(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestGetOutputPath(t *testing.T) {
	cfg := &Config{
		Output: OutputConfig{
			Dir:      "/tmp/output",
			FileName: "test-report",
		},
	}

	expected := filepath.Join("/tmp/output", "test-report.xlsx")
	result := cfg.GetOutputPath()

	if result != expected {
		t.Errorf("GetOutputPath() = %s, expected %s", result, expected)
	}
}

func TestValidate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "spec-recon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		cfg       *Config
		shouldErr bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Project: ProjectConfig{
					RootDir:  tmpDir,
					Encoding: []string{"utf-8"},
				},
				Output: OutputConfig{
					FileName: "report",
				},
			},
			shouldErr: false,
		},
		{
			name: "Nonexistent root directory",
			cfg: &Config{
				Project: ProjectConfig{
					RootDir:  "/nonexistent/directory",
					Encoding: []string{"utf-8"},
				},
				Output: OutputConfig{
					FileName: "report",
				},
			},
			shouldErr: true,
		},
		{
			name: "Empty encoding list",
			cfg: &Config{
				Project: ProjectConfig{
					RootDir:  tmpDir,
					Encoding: []string{},
				},
				Output: OutputConfig{
					FileName: "report",
				},
			},
			shouldErr: true,
		},
		{
			name: "Empty output filename",
			cfg: &Config{
				Project: ProjectConfig{
					RootDir:  tmpDir,
					Encoding: []string{"utf-8"},
				},
				Output: OutputConfig{
					FileName: "",
				},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		str      string
		pattern  string
		expected bool
	}{
		{"StringUtil", "*Util", true},
		{"UserDTO", "*DTO", true},
		{"DateUtils", "*Utils", true},
		{"UserController", "*Util", false},
		{"Helper", "Helper", true},
		{"TestHelper", "*Helper", true},
		{"HelperTest", "Helper*", true},
		{"MyHelper", "*Helper*", true},
		{"User", "*", true},
	}

	for _, tt := range tests {
		result := matchPattern(tt.str, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchPattern(%s, %s) = %v, expected %v", tt.str, tt.pattern, result, tt.expected)
		}
	}
}
