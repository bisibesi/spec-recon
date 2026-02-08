package analyzer

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ScanDirectory walks the root directory and finds .java and .xml files
// It excludes directories matching excludePatterns
func ScanDirectory(root string, excludePatterns []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check exclusion for directories
		if d.IsDir() {
			// Skip .git always
			if d.Name() == ".git" || d.Name() == ".svn" {
				return filepath.SkipDir
			}

			// Normalize path for matching (forward slashes)
			relPath, _ := filepath.Rel(root, path)
			relPath = filepath.ToSlash(relPath)

			for _, pat := range excludePatterns {
				if matchGlob(relPath, pat) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Filter files
		if strings.HasSuffix(path, ".java") || strings.HasSuffix(path, ".xml") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return files, nil
}

// matchGlob is a simple wrapper around doublestar logic or just simple matching
// For now, we reuse the logic from config (conceptually), but here providing a simple implementation
func matchGlob(path, pattern string) bool {
	// Simple star matching
	if strings.Contains(pattern, "**") {
		// Just check contains for now as simplified logic
		clean := strings.ReplaceAll(pattern, "**", "")
		clean = strings.Trim(clean, "/")
		if clean != "" && strings.Contains(path, clean) {
			return true
		}
	}
	return false
}
