package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sivchari/gomu/internal/config"
)

const testContent = `package main

func test() {
	return
}
`

const noPreviousHistory = "No previous history"

// MockHistoryStore is a mock implementation for testing.
type MockHistoryStore struct {
	entries map[string]HistoryEntry
}

func NewMockHistoryStore() *MockHistoryStore {
	return &MockHistoryStore{
		entries: make(map[string]HistoryEntry),
	}
}

func (m *MockHistoryStore) GetEntry(filePath string) (HistoryEntry, bool) {
	entry, exists := m.entries[filePath]

	return entry, exists
}

func (m *MockHistoryStore) HasChanged(filePath, currentHash string) bool {
	entry, exists := m.entries[filePath]
	if !exists {
		return true
	}

	return entry.FileHash != currentHash
}

func (m *MockHistoryStore) SetEntry(filePath string, entry HistoryEntry) {
	m.entries[filePath] = entry
}

func TestIncrementalAnalyzer_NewIncrementalAnalyzer(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Incremental.HistoryFile = filepath.Join(tmpDir, "history.json")

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(cfg, tmpDir, mockHistory)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	if analyzer == nil {
		t.Fatal("Expected analyzer to be non-nil")
	}

	if analyzer.config != cfg {
		t.Error("Config should match")
	}

	if analyzer.workDir != tmpDir {
		t.Errorf("Expected workDir %s, got %s", tmpDir, analyzer.workDir)
	}
}

func TestIncrementalAnalyzer_AnalyzeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Incremental.HistoryFile = filepath.Join(tmpDir, "history.json")
	cfg.Incremental.UseGitDiff = false // Disable git diff for testing

	// Create test files
	testFile := filepath.Join(tmpDir, "test.go")
	content := testContent

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(cfg, tmpDir, mockHistory)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	results, err := analyzer.AnalyzeFiles()
	if err != nil {
		t.Fatalf("Failed to analyze files: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one analysis result")
	}

	// First run should mark all files as needing update
	found := false

	for _, result := range results {
		if result.FilePath == testFile {
			found = true

			if !result.NeedsUpdate {
				t.Error("First run should mark file as needing update")
			}

			if result.Reason != noPreviousHistory {
				t.Errorf("Expected reason '%s', got '%s'", noPreviousHistory, result.Reason)
			}
		}
	}

	if !found {
		t.Error("Test file not found in analysis results")
	}
}

func TestIncrementalAnalyzer_GetFilesNeedingUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Incremental.HistoryFile = filepath.Join(tmpDir, "history.json")
	cfg.Incremental.UseGitDiff = false

	// Create test files
	testFile := filepath.Join(tmpDir, "test.go")
	content := testContent

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(cfg, tmpDir, mockHistory)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	files, err := analyzer.GetFilesNeedingUpdate()
	if err != nil {
		t.Fatalf("Failed to get files needing update: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected at least one file needing update")
	}

	// Check that our test file is in the results
	found := false

	for _, file := range files {
		if file == testFile {
			found = true

			break
		}
	}

	if !found {
		t.Error("Test file should be in files needing update")
	}
}

func TestIncrementalAnalyzer_analyzeFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Incremental.HistoryFile = filepath.Join(tmpDir, "history.json")

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(cfg, tmpDir, mockHistory)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	// Test with non-existent file
	result, err := analyzer.analyzeFile("/non/existent/file.go")
	if err != nil {
		t.Fatalf("Failed to analyze non-existent file: %v", err)
	}

	if result.NeedsUpdate {
		t.Error("Non-existent file should not need update")
	}

	if result.Reason != "File does not exist" {
		t.Errorf("Expected reason 'File does not exist', got '%s'", result.Reason)
	}

	// Test with existing file
	testFile := filepath.Join(tmpDir, "test.go")
	content := testContent

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err = analyzer.analyzeFile(testFile)
	if err != nil {
		t.Fatalf("Failed to analyze existing file: %v", err)
	}

	if !result.NeedsUpdate {
		t.Error("File with no history should need update")
	}

	if result.Reason != "No previous history" {
		t.Errorf("Expected reason 'No previous history', got '%s'", result.Reason)
	}

	if result.CurrentHash == "" {
		t.Error("Current hash should not be empty")
	}
}

func TestIncrementalAnalyzer_findRelatedTestFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Incremental.HistoryFile = filepath.Join(tmpDir, "history.json")

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(cfg, tmpDir, mockHistory)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	// Create main file and related test files
	mainFile := filepath.Join(tmpDir, "utils.go")
	testFile1 := filepath.Join(tmpDir, "utils_test.go")
	testFile2 := filepath.Join(tmpDir, "test_utils.go")

	files := []string{mainFile, testFile1, testFile2}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("package main"), 0600); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	relatedFiles := analyzer.findRelatedTestFiles(mainFile)

	if len(relatedFiles) != 2 {
		t.Errorf("Expected 2 related test files, got %d", len(relatedFiles))
	}

	// Check that both test files are found
	foundFiles := make(map[string]bool)
	for _, file := range relatedFiles {
		foundFiles[file] = true
	}

	if !foundFiles[testFile1] {
		t.Error("Expected to find utils_test.go")
	}

	if !foundFiles[testFile2] {
		t.Error("Expected to find test_utils.go")
	}
}
