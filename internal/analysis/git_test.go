package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitIntegration_IsGitRepository(t *testing.T) {
	// Test with non-git directory
	tmpDir := t.TempDir()
	git := NewGitIntegration(tmpDir)

	if git.IsGitRepository() {
		t.Error("Expected false for non-git directory")
	}

	// Test with git directory (create .git directory)
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	if !git.IsGitRepository() {
		t.Error("Expected true for git directory")
	}
}

func TestGitIntegration_GetAllGoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	git := NewGitIntegration(tmpDir)

	// Create test files
	files := []string{
		"main.go",
		"utils.go",
		"main_test.go",  // Should be excluded
		"utils_test.go", // Should be excluded
		"subdir/module.go",
		"subdir/module_test.go", // Should be excluded
		"vendor/external.go",    // Should be excluded
		".hidden/file.go",       // Should be excluded
		"README.md",             // Should be excluded
	}

	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(filePath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(filePath, []byte("package main"), 0600); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	goFiles, err := git.GetAllGoFiles()
	if err != nil {
		t.Fatalf("Failed to get Go files: %v", err)
	}

	// Should only include non-test Go files
	expected := []string{
		filepath.Join(tmpDir, "main.go"),
		filepath.Join(tmpDir, "utils.go"),
		filepath.Join(tmpDir, "subdir", "module.go"),
	}

	if len(goFiles) != len(expected) {
		t.Errorf("Expected %d Go files, got %d", len(expected), len(goFiles))
	}

	// Check that all expected files are present
	fileSet := make(map[string]bool)
	for _, file := range goFiles {
		fileSet[file] = true
	}

	for _, expectedFile := range expected {
		if !fileSet[expectedFile] {
			t.Errorf("Expected file %s not found in results", expectedFile)
		}
	}
}

func TestGitIntegration_GetChangedFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	git := NewGitIntegration(tmpDir)

	_, err := git.GetChangedFiles("main")
	if err == nil {
		t.Error("Expected error for non-git repository")
	}
}

func TestGitIntegration_GetCurrentBranch_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	git := NewGitIntegration(tmpDir)

	_, err := git.GetCurrentBranch()
	if err == nil {
		t.Error("Expected error for non-git repository")
	}
}

func TestGitIntegration_HasUncommittedChanges_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	git := NewGitIntegration(tmpDir)

	_, err := git.HasUncommittedChanges()
	if err == nil {
		t.Error("Expected error for non-git repository")
	}
}

// Note: Tests for actual git operations would require setting up a real git repository
// and are more complex to implement in unit tests. These would be better suited for
// integration tests.
