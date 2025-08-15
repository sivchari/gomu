package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sivchari/gomu/internal/ignore"
)

func TestFindTargetFiles_WithGomuignore(t *testing.T) {
	// Create test directory structure
	tempDir := t.TempDir()

	structure := map[string]string{
		"main.go":                   "package main",
		"config.go":                 "package main",
		"internal/app/server.go":    "package app",
		"internal/db/connection.go": "package db",
		"vendor/package/lib.go":     "package lib",
		"testdata/sample.go":        "package testdata",
		"generated/proto/api.pb.go": "package proto",
		"docs/example.go":           "package docs",
		"app.log":                   "log content",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	// Create .gomuignore file
	gomuignore := `# Generated code
generated/

# Documentation
docs/

# Logs
*.log`

	ignoreFile := filepath.Join(tempDir, ".gomuignore")
	if err := os.WriteFile(ignoreFile, []byte(gomuignore), 0644); err != nil {
		t.Fatalf("failed to create .gomuignore file: %v", err)
	}

	// Test with .gomuignore
	t.Run("with .gomuignore", func(t *testing.T) {
		ignoreParser := ignore.New()
		if err := ignoreParser.LoadFromFile(ignoreFile); err != nil {
			t.Fatalf("failed to load .gomuignore: %v", err)
		}

		analyzer, err := New(WithIgnoreParser(ignoreParser))
		if err != nil {
			t.Fatalf("failed to create analyzer: %v", err)
		}

		files, err := analyzer.FindTargetFiles(tempDir)
		if err != nil {
			t.Fatalf("FindTargetFiles error: %v", err)
		}

		got := getRelativePaths(tempDir, files)
		want := []string{
			"config.go",
			"internal/app/server.go",
			"internal/db/connection.go",
			"main.go",
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("FindTargetFiles() mismatch (-want +got):\n%s", diff)
		}
	})

	// Test without .gomuignore (vendor/ and testdata/ are excluded by default)
	t.Run("without .gomuignore", func(t *testing.T) {
		analyzer, err := New()
		if err != nil {
			t.Fatalf("failed to create analyzer: %v", err)
		}

		files, err := analyzer.FindTargetFiles(tempDir)
		if err != nil {
			t.Fatalf("FindTargetFiles error: %v", err)
		}

		got := getRelativePaths(tempDir, files)
		want := []string{
			"config.go",
			"docs/example.go",
			"generated/proto/api.pb.go",
			"internal/app/server.go",
			"internal/db/connection.go",
			"main.go",
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("FindTargetFiles() mismatch (-want +got):\n%s", diff)
		}
	})
}

// getRelativePaths converts absolute paths to relative paths and sorts them.
func getRelativePaths(rootPath string, files []string) []string {
	relPaths := make([]string, 0, len(files))

	for _, file := range files {
		relPath, err := filepath.Rel(rootPath, file)
		if err != nil {
			relPath = file
		}

		relPaths = append(relPaths, relPath)
	}

	sort.Strings(relPaths)

	return relPaths
}
