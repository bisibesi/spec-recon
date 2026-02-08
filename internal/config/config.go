package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Project  ProjectConfig  `mapstructure:"project"`
	Analysis AnalysisConfig `mapstructure:"analysis"`
	Output   OutputConfig   `mapstructure:"output"`
}

// ProjectConfig holds project-specific settings
type ProjectConfig struct {
	RootDir     string   `mapstructure:"root_dir"`     // Root directory to analyze
	BasePackage string   `mapstructure:"base_package"` // Base Java package (e.g., "com.company")
	Encoding    []string `mapstructure:"encoding"`     // Encoding hints (e.g., ["utf-8", "euc-kr", "ms949"])
}

// AnalysisConfig holds analysis behavior settings
type AnalysisConfig struct {
	ExcludeDirs  []string `mapstructure:"exclude_dirs"`  // Directories to exclude
	UtilPatterns []string `mapstructure:"util_patterns"` // Patterns for utility classes to exclude
	IncludeUtils bool     `mapstructure:"include_utils"` // Whether to include utility classes in output
}

// OutputConfig holds output settings
type OutputConfig struct {
	Dir      string `mapstructure:"dir"`       // Output directory
	FileName string `mapstructure:"file_name"` // Output file name (without extension)
}

// Load reads the configuration from a file or uses defaults
// If configPath is empty, it looks for "config.yaml" in the current directory
// If the file doesn't exist, it uses sensible defaults
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set sensible defaults
	setDefaults(v)

	// Determine config file to use
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Set config file
	v.SetConfigFile(configPath)

	// Read config file (ignore error if file doesn't exist)
	if err := v.ReadInConfig(); err != nil {
		// Check if it's just a file not found error
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") ||
			strings.Contains(err.Error(), "cannot find") {
			// Config file not found - use defaults
			fmt.Println("==========================================")
			fmt.Println("Config file not found. Using defaults:")
			fmt.Println("  Source: ./src")
			fmt.Println("  Output: ./output")
			fmt.Println("==========================================")
		} else {
			// Config file found but has some other error
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		fmt.Printf("Loaded config from: %s\n", v.ConfigFileUsed())
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Normalize paths
	if err := cfg.normalizePaths(); err != nil {
		return nil, err
	}

	// Create output directory if it doesn't exist
	if err := cfg.EnsureOutputDir(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults configures sensible default values
func setDefaults(v *viper.Viper) {
	// Project defaults - use ./src for double-click usability
	v.SetDefault("project.root_dir", "./src")
	v.SetDefault("project.base_package", "")
	v.SetDefault("project.encoding", []string{"utf-8", "euc-kr", "ms949"})

	// Analysis defaults
	v.SetDefault("analysis.exclude_dirs", []string{
		"**/test/**",
		"**/tests/**",
		"**/target/**",
		"**/build/**",
		"**/out/**",
		"**/.git/**",
		"**/.svn/**",
		"**/node_modules/**",
	})
	v.SetDefault("analysis.util_patterns", []string{
		"*Util",
		"*Utils",
		"*Helper",
		"*Helpers",
		"*DTO",
		"*VO",
		"*Entity",
		"*Constant",
		"*Constants",
		"*Config",
		"*Configuration",
	})
	v.SetDefault("analysis.include_utils", false)

	// Output defaults
	v.SetDefault("output.dir", "./output")
	v.SetDefault("output.file_name", "spec-recon-report")
}

// normalizePaths converts relative paths to absolute paths
func (c *Config) normalizePaths() error {
	// Normalize root directory
	absRoot, err := filepath.Abs(c.Project.RootDir)
	if err != nil {
		return fmt.Errorf("failed to resolve root_dir: %w", err)
	}
	c.Project.RootDir = absRoot

	// Normalize output directory
	absOutput, err := filepath.Abs(c.Output.Dir)
	if err != nil {
		return fmt.Errorf("failed to resolve output.dir: %w", err)
	}
	c.Output.Dir = absOutput

	return nil
}

// EnsureOutputDir creates the output directory if it doesn't exist
func (c *Config) EnsureOutputDir() error {
	if err := os.MkdirAll(c.Output.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// IsUtil checks if a class name matches any utility pattern
func (c *Config) IsUtil(className string) bool {
	for _, pattern := range c.Analysis.UtilPatterns {
		if matchPattern(className, pattern) {
			return true
		}
	}
	return false
}

// ShouldExclude checks if a file path should be excluded based on exclude_dirs
func (c *Config) ShouldExclude(filePath string) bool {
	// Convert to forward slashes for consistent matching
	normalizedPath := filepath.ToSlash(filePath)

	for _, pattern := range c.Analysis.ExcludeDirs {
		if matchPathPattern(normalizedPath, pattern) {
			return true
		}
	}
	return false
}

// GetOutputPath returns the full path for the output Excel file
func (c *Config) GetOutputPath() string {
	return filepath.Join(c.Output.Dir, c.Output.FileName+".xlsx")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check if root directory exists
	if _, err := os.Stat(c.Project.RootDir); os.IsNotExist(err) {
		return fmt.Errorf("root_dir does not exist: %s", c.Project.RootDir)
	}

	// Check if encoding list is not empty
	if len(c.Project.Encoding) == 0 {
		return fmt.Errorf("project.encoding must contain at least one encoding")
	}

	// Check if output filename is not empty
	if c.Output.FileName == "" {
		return fmt.Errorf("output.file_name cannot be empty")
	}

	return nil
}

// matchPattern checks if a string matches a simple glob pattern
// Supports only '*' wildcard at the beginning or end
func matchPattern(str, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// *foo* - contains
		middle := pattern[1 : len(pattern)-1]
		return strings.Contains(str, middle)
	} else if strings.HasPrefix(pattern, "*") {
		// *foo - ends with
		suffix := pattern[1:]
		return strings.HasSuffix(str, suffix)
	} else if strings.HasSuffix(pattern, "*") {
		// foo* - starts with
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(str, prefix)
	}

	// Exact match
	return str == pattern
}

// matchPathPattern checks if a path matches a glob pattern
// Supports ** for recursive directory matching
func matchPathPattern(path, pattern string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	if strings.Contains(pattern, "**") {
		// **/ pattern - match anywhere in path
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.Trim(parts[0], "/")
			suffix := strings.Trim(parts[1], "/")

			// Handle prefix matching
			hasPrefix := true
			if prefix != "" {
				hasPrefix = strings.HasPrefix(path, prefix+"/") || strings.Contains(path, "/"+prefix+"/")
			}

			// Handle suffix matching (must match directory or file name)
			hasSuffix := true
			if suffix != "" {
				// Check if suffix is a directory that contains the file
				hasSuffix = strings.Contains(path, "/"+suffix+"/") ||
					strings.HasSuffix(path, "/"+suffix) ||
					strings.HasPrefix(path, suffix+"/")
			}

			return hasPrefix && hasSuffix
		}
	}

	// Simple contains matching for patterns with *
	cleanPattern := strings.Trim(pattern, "*")
	return strings.Contains(path, cleanPattern)
}

// Print displays the current configuration
func (c *Config) Print() {
	fmt.Println("=== Spec Recon Configuration ===")
	fmt.Printf("Project Root:     %s\n", c.Project.RootDir)
	fmt.Printf("Base Package:     %s\n", c.Project.BasePackage)
	fmt.Printf("Encoding Hints:   %v\n", c.Project.Encoding)
	fmt.Printf("Exclude Dirs:     %v\n", c.Analysis.ExcludeDirs)
	fmt.Printf("Util Patterns:    %v\n", c.Analysis.UtilPatterns)
	fmt.Printf("Include Utils:    %v\n", c.Analysis.IncludeUtils)
	fmt.Printf("Output Directory: %s\n", c.Output.Dir)
	fmt.Printf("Output File:      %s\n", c.GetOutputPath())
	fmt.Println("================================")
}
