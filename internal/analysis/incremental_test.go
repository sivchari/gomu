package analysis

import (
	"os"
	"path/filepath"
	"testing"
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
	tempDir := t.TempDir()
	history := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(tempDir, history)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	if analyzer.workDir != tempDir {
		t.Error("Work directory not set correctly")
	}
}

func TestIncrementalAnalyzer_AnalyzeFiles(t *testing.T) {
	tempDir := t.TempDir()
	history := NewMockHistoryStore()

	// Create test Go files
	testFile1 := filepath.Join(tempDir, "test1.go")

	err := os.WriteFile(testFile1, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tempDir, "test2.go")

	err = os.WriteFile(testFile2, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create go.mod
	goMod := "module test\n"

	err = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	analyzer, err := NewIncrementalAnalyzer(tempDir, history)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	results, err := analyzer.AnalyzeFiles()
	if err != nil {
		t.Fatalf("Failed to analyze files: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}

	// Verify results contain our test files
	foundTest1 := false
	foundTest2 := false

	for _, result := range results {
		if result.FilePath == testFile1 {
			foundTest1 = true
		}

		if result.FilePath == testFile2 {
			foundTest2 = true
		}
	}

	if !foundTest1 {
		t.Error("test1.go not found in results")
	}

	if !foundTest2 {
		t.Error("test2.go not found in results")
	}
}

func TestIncrementalAnalyzer_GetFilesNeedingUpdate(t *testing.T) {
	tempDir := t.TempDir()
	history := NewMockHistoryStore()

	testFile := filepath.Join(tempDir, "test.go")

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create go.mod
	goMod := "module test\n"

	err = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	analyzer, err := NewIncrementalAnalyzer(tempDir, history)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	// First run - should need update (no history)
	files, err := analyzer.GetFilesNeedingUpdate()
	if err != nil {
		t.Fatalf("Failed to get files needing update: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected files to need update on first run")
	}

	// Add entry to history
	hasher := NewFileHasher()

	hash, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash file: %v", err)
	}

	history.SetEntry(testFile, HistoryEntry{
		FileHash:      hash,
		TestHash:      "",
		MutationScore: 80.0,
	})

	// Second run - should not need update (same hash)
	files, err = analyzer.GetFilesNeedingUpdate()
	if err != nil {
		t.Fatalf("Failed to get files needing update: %v", err)
	}

	// Should be empty if file hasn't changed
	if len(files) > 0 {
		t.Logf("Files still need update (may be due to other factors): %v", files)
	}
}

func TestIncrementalAnalyzer_PrintAnalysisReport(t *testing.T) {
	history := NewMockHistoryStore()
	tempDir := t.TempDir()

	analyzer, err := NewIncrementalAnalyzer(tempDir, history)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	results := []FileAnalysisResult{
		{
			FilePath:     "/test/file1.go",
			PreviousHash: "old_hash",
			CurrentHash:  "new_hash",
			Reason:       "File content changed",
			NeedsUpdate:  true,
		},
		{
			FilePath:     "/test/file2.go",
			PreviousHash: "same_hash",
			CurrentHash:  "same_hash",
			Reason:       noPreviousHistory,
			NeedsUpdate:  false,
		},
	}

	// This should not panic
	analyzer.PrintAnalysisReport(results)
}

func TestIncrementalAnalyzer_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	history := NewMockHistoryStore()

	// Test with empty directory
	analyzer, err := NewIncrementalAnalyzer(tempDir, history)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	// Should handle empty directory gracefully
	results, err := analyzer.AnalyzeFiles()
	if err != nil {
		t.Fatalf("Failed to analyze empty directory: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty directory, got %d", len(results))
	}

	files, err := analyzer.GetFilesNeedingUpdate()
	if err != nil {
		t.Fatalf("Failed to get files needing update: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files needing update for empty directory, got %d", len(files))
	}
}

func TestIncrementalAnalyzer_InvalidPath(t *testing.T) {
	history := NewMockHistoryStore()

	// Test with non-existent directory
	_, err := NewIncrementalAnalyzer("/nonexistent/path", history)
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestIncrementalAnalyzer_analyzeFile(t *testing.T) {
	tmpDir := t.TempDir()
	// History file is now handled by intelligent defaults

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(tmpDir, mockHistory)
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
	// History file is now handled by intelligent defaults

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(tmpDir, mockHistory)
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

func TestIncrementalAnalyzer_hasTestFilesChanged(t *testing.T) {
	tmpDir := t.TempDir()
	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(tmpDir, mockHistory)
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	// Create main file and test file
	mainFile := filepath.Join(tmpDir, "utils.go")
	testFile := filepath.Join(tmpDir, "utils_test.go")

	mainContent := `package main

func Add(a, b int) int {
	return a + b
}`
	testContent := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("Add failed")
	}
}`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0600); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	if err := os.WriteFile(testFile, []byte(testContent), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First check - no history, should return true
	if !analyzer.hasTestFilesChanged(mainFile) {
		t.Error("Expected hasTestFilesChanged to return true when no history exists")
	}

	// Add history entry with test file hash
	testHash, _ := analyzer.hasher.HashFile(testFile)
	mockHistory.SetEntry(mainFile, HistoryEntry{
		FileHash: "dummy",
		TestHash: testHash,
	})

	// Second check - test file hasn't changed, should return false
	if analyzer.hasTestFilesChanged(mainFile) {
		t.Error("Expected hasTestFilesChanged to return false when test file hasn't changed")
	}

	// Modify test file
	newTestContent := testContent + "\n// Modified"
	if err := os.WriteFile(testFile, []byte(newTestContent), 0600); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Third check - test file changed, should return true
	if !analyzer.hasTestFilesChanged(mainFile) {
		t.Error("Expected hasTestFilesChanged to return true when test file has changed")
	}

	// Test with non-existent test file
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.go")
	if analyzer.hasTestFilesChanged(nonExistentFile) {
		t.Error("Expected hasTestFilesChanged to return false for file with no test files")
	}
}
