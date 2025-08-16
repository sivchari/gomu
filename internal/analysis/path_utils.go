package analysis

import (
	"path/filepath"
	"strings"
)

// GetRelativePath returns the relative path from base to target.
// If an error occurs, it returns the target path as-is.
func GetRelativePath(base, target string) string {
	relPath, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}

	return relPath
}

// IsGoSourceFile checks if a file is a Go source file (not a test file).
func IsGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// IsGoTestFile checks if a file is a Go test file.
func IsGoTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}
