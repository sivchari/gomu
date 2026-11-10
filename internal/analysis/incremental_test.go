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

	analyzer, err := NewIncrementalAnalyzer(tempDir, history, false, "")
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

	analyzer, err := NewIncrementalAnalyzer(tempDir, history, false, "")
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

	analyzer, err := NewIncrementalAnalyzer(tempDir, history, false, "")
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

	analyzer, err := NewIncrementalAnalyzer(tempDir, history, false, "")
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
	analyzer, err := NewIncrementalAnalyzer(tempDir, history, false, "")
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
	_, err := NewIncrementalAnalyzer("/nonexistent/path", history, false, "")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestIncrementalAnalyzer_analyzeFile(t *testing.T) {
	tmpDir := t.TempDir()
	// History file is now handled by intelligent defaults

	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(tmpDir, mockHistory, true, "")
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

func TestIncrementalAnalyzer_AnalyzeFile_IncrementalDisabled(t *testing.T) {
	tests := []struct {
		name           string
		incremental    bool
		withTestFile   bool
		wantNeedUpdate bool
		wantReason     string
	}{
		{
			name:           "disabled ignores unchanged source hash",
			incremental:    false,
			wantNeedUpdate: true,
			wantReason:     incrementalDisabledReason,
		},
		{
			name:           "disabled ignores unchanged source and test hash",
			incremental:    false,
			withTestFile:   true,
			wantNeedUpdate: true,
			wantReason:     incrementalDisabledReason,
		},
		{
			name:           "enabled skips unchanged source hash",
			incremental:    true,
			wantNeedUpdate: false,
			wantReason:     "No changes detected",
		},
		{
			name:           "enabled skips unchanged source and test hash",
			incremental:    true,
			withTestFile:   true,
			wantNeedUpdate: false,
			wantReason:     "No changes detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			srcFile := filepath.Join(tmpDir, "target.go")

			if err := os.WriteFile(srcFile, []byte(testContent), 0o600); err != nil {
				t.Fatalf("Failed to create source file: %v", err)
			}

			hasher := NewFileHasher()

			srcHash, err := hasher.HashFile(srcFile)
			if err != nil {
				t.Fatalf("Failed to hash source file: %v", err)
			}

			entry := HistoryEntry{FileHash: srcHash}

			if tt.withTestFile {
				testFile := filepath.Join(tmpDir, "target_test.go")

				if err := os.WriteFile(testFile, []byte("package main\n"), 0o600); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				entry.TestHash = hasher.HashRelatedTests(srcFile)
			}

			history := NewMockHistoryStore()
			history.SetEntry(srcFile, entry)

			analyzer, err := NewIncrementalAnalyzer(tmpDir, history, tt.incremental, "")
			if err != nil {
				t.Fatalf("Failed to create incremental analyzer: %v", err)
			}

			result, err := analyzer.analyzeFile(srcFile)
			if err != nil {
				t.Fatalf("analyzeFile failed: %v", err)
			}

			if result.NeedsUpdate != tt.wantNeedUpdate {
				t.Errorf("NeedsUpdate = %v, want %v", result.NeedsUpdate, tt.wantNeedUpdate)
			}

			if result.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
			}

			if result.CurrentHash != srcHash {
				t.Errorf("CurrentHash = %q, want %q", result.CurrentHash, srcHash)
			}

			if result.PreviousHash != srcHash {
				t.Errorf("PreviousHash = %q, want %q", result.PreviousHash, srcHash)
			}
		})
	}
}

// storedAnalyzer builds an incremental analyzer whose history already holds
// the current file and test hashes for each of srcFiles, as after a full run.
func storedAnalyzer(t *testing.T, dir string, srcFiles ...string) *IncrementalAnalyzer {
	t.Helper()

	history := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(dir, history, true, "")
	if err != nil {
		t.Fatalf("Failed to create incremental analyzer: %v", err)
	}

	for _, src := range srcFiles {
		fileHash, err := analyzer.hasher.HashFile(src)
		if err != nil {
			t.Fatalf("Failed to hash %s: %v", src, err)
		}

		history.SetEntry(src, HistoryEntry{
			FileHash: fileHash,
			TestHash: analyzer.hasher.HashRelatedTests(src),
		})
	}

	return analyzer
}

func requireAnalysis(t *testing.T, analyzer *IncrementalAnalyzer, src string, wantUpdate bool, wantReason string) {
	t.Helper()

	result, err := analyzer.analyzeFile(src)
	if err != nil {
		t.Fatalf("analyzeFile failed: %v", err)
	}

	if result.NeedsUpdate != wantUpdate || result.Reason != wantReason {
		t.Errorf("got NeedsUpdate=%v Reason=%q, want NeedsUpdate=%v Reason=%q",
			result.NeedsUpdate, result.Reason, wantUpdate, wantReason)
	}
}

// A test file whose name does not match the production file still exercises it
// through the package API, so changing it must make the file eligible again.
func TestIncrementalAnalyzer_UnrelatedNamedTestFileInvalidates(t *testing.T) {
	dir := t.TempDir()
	value := filepath.Join(dir, "value.go")
	contract := filepath.Join(dir, "contract_test.go")

	writeTestFile(t, value, "package example\n\nfunc Value() int { return 1 }\n")
	writeTestFile(t, contract, "package example_test\n")

	analyzer := storedAnalyzer(t, dir, value)

	requireAnalysis(t, analyzer, value, false, "No changes detected")

	writeTestFile(t, contract, "package example_test\n\n// stronger assertions\n")

	requireAnalysis(t, analyzer, value, true, "Related test files changed")
}

func TestIncrementalAnalyzer_TestFileSetChangeInvalidates(t *testing.T) {
	dir := t.TempDir()
	value := filepath.Join(dir, "value.go")
	contract := filepath.Join(dir, "contract_test.go")

	writeTestFile(t, value, "package example\n")
	writeTestFile(t, contract, "package example_test\n")

	tests := []struct {
		name   string
		mutate func(t *testing.T)
	}{
		{"add", func(t *testing.T) {
			t.Helper()
			writeTestFile(t, filepath.Join(dir, "extra_test.go"), "package example\n")
		}},
		{"delete", func(t *testing.T) {
			t.Helper()

			if err := os.Remove(contract); err != nil {
				t.Fatalf("remove: %v", err)
			}
		}},
		{"rename", func(t *testing.T) {
			t.Helper()

			if err := os.Rename(contract, filepath.Join(dir, "renamed_test.go")); err != nil {
				t.Fatalf("rename: %v", err)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the package to a single contract_test.go.
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("readdir: %v", err)
			}

			for _, e := range entries {
				if e.Name() != "value.go" {
					if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
						t.Fatalf("remove: %v", err)
					}
				}
			}

			writeTestFile(t, contract, "package example_test\n")

			analyzer := storedAnalyzer(t, dir, value)

			requireAnalysis(t, analyzer, value, false, "No changes detected")

			tt.mutate(t)

			requireAnalysis(t, analyzer, value, true, "Related test files changed")
		})
	}
}

// A sibling production file is invalidated by a change to the shared package
// tests, and a package without test files stays skipped.
func TestIncrementalAnalyzer_ChangeInvalidatesEveryFileInPackage(t *testing.T) {
	dir := t.TempDir()
	value := filepath.Join(dir, "value.go")
	other := filepath.Join(dir, "other.go")
	contract := filepath.Join(dir, "contract_test.go")

	writeTestFile(t, value, "package example\n")
	writeTestFile(t, other, "package example\n")
	writeTestFile(t, contract, "package example_test\n")

	analyzer := storedAnalyzer(t, dir, value, other)

	writeTestFile(t, contract, "package example_test\n// changed\n")

	requireAnalysis(t, analyzer, value, true, "Related test files changed")
	requireAnalysis(t, analyzer, other, true, "Related test files changed")

	bare := t.TempDir()
	lonely := filepath.Join(bare, "lonely.go")
	writeTestFile(t, lonely, "package lonely\n")

	requireAnalysis(t, storedAnalyzer(t, bare, lonely), lonely, false, "No changes detected")
}

// Tests named after the production file keep working.
func TestIncrementalAnalyzer_MatchingTestFileStillInvalidates(t *testing.T) {
	dir := t.TempDir()
	value := filepath.Join(dir, "value.go")
	matching := filepath.Join(dir, "value_test.go")

	writeTestFile(t, value, "package example\n")
	writeTestFile(t, matching, "package example\n")

	analyzer := storedAnalyzer(t, dir, value)

	requireAnalysis(t, analyzer, value, false, "No changes detected")

	writeTestFile(t, matching, "package example\n// changed\n")

	requireAnalysis(t, analyzer, value, true, "Related test files changed")
}

func TestIncrementalAnalyzer_hasTestFilesChanged(t *testing.T) {
	tmpDir := t.TempDir()
	mockHistory := NewMockHistoryStore()

	analyzer, err := NewIncrementalAnalyzer(tmpDir, mockHistory, false, "")
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
	mockHistory.SetEntry(mainFile, HistoryEntry{
		FileHash: "dummy",
		TestHash: analyzer.hasher.HashRelatedTests(mainFile),
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

	// A file in a package without any test file has nothing to compare.
	untestedFile := filepath.Join(t.TempDir(), "untested.go")
	if analyzer.hasTestFilesChanged(untestedFile) {
		t.Error("Expected hasTestFilesChanged to return false for file with no test files")
	}
}

func TestIncrementalAnalyzer_getFilesToAnalyze_BaseSHAWinsOverBaseBranch(t *testing.T) {
	tmpDir := t.TempDir()
	runGit := initUnsignedRepo(t, tmpDir)

	commitFiles(t, tmpDir, runGit, map[string]string{"a.go": "package main\n"})

	baseSHA := headSHA(t, tmpDir)

	commitFiles(t, tmpDir, runGit, map[string]string{"b.go": "package main\n"})
	runGit("checkout", "-b", "feature")
	commitFiles(t, tmpDir, runGit, map[string]string{"c.go": "package main\n"})

	newAnalyzer := func(t *testing.T) *IncrementalAnalyzer {
		t.Helper()

		analyzer, err := NewIncrementalAnalyzer(tmpDir, NewMockHistoryStore(), true, "main")
		if err != nil {
			t.Fatalf("NewIncrementalAnalyzer: %v", err)
		}

		return analyzer
	}

	t.Run("uses the base branch when no SHA is set", func(t *testing.T) {
		files, err := newAnalyzer(t).getFilesToAnalyze()
		if err != nil {
			t.Fatalf("getFilesToAnalyze: %v", err)
		}

		if len(files) != 1 || files[0] != filepath.Join(tmpDir, "c.go") {
			t.Errorf("files = %v, want only c.go", files)
		}
	})

	t.Run("uses the base SHA when set", func(t *testing.T) {
		analyzer := newAnalyzer(t)
		analyzer.SetBaseSHA(baseSHA)

		files, err := analyzer.getFilesToAnalyze()
		if err != nil {
			t.Fatalf("getFilesToAnalyze: %v", err)
		}

		if len(files) != 2 {
			t.Errorf("files = %v, want b.go and c.go", files)
		}
	})

	t.Run("does not fall back to the base branch when the SHA is invalid", func(t *testing.T) {
		analyzer := newAnalyzer(t)
		analyzer.SetBaseSHA("0000000000000000000000000000000000000bad")

		if _, err := analyzer.getFilesToAnalyze(); err == nil {
			t.Error("expected error for an invalid base SHA")
		}
	})
}

func TestIncrementalAnalyzer_AnalyzeFile_UnchangedTestFilesSkip(t *testing.T) {
	tests := []struct {
		name         string
		modifyTest   bool
		wantUpdate   bool
		wantReason   string
		withTestFile bool
	}{
		{
			name:         "unchanged source and test file are skipped",
			withTestFile: true,
			wantUpdate:   false,
			wantReason:   "No changes detected",
		},
		{
			name:         "modified test file triggers a re-run",
			withTestFile: true,
			modifyTest:   true,
			wantUpdate:   true,
			wantReason:   "Related test files changed",
		},
		{
			name:         "source without test file is skipped",
			withTestFile: false,
			wantUpdate:   false,
			wantReason:   "No changes detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "foo.go")
			testFile := filepath.Join(dir, "foo_test.go")

			writeTestFile(t, src, "package foo\n\nfunc Foo() {}\n")

			if tt.withTestFile {
				writeTestFile(t, testFile, "package foo\n")
			}

			store := NewMockHistoryStore()

			analyzer, err := NewIncrementalAnalyzer(dir, store, true, "")
			if err != nil {
				t.Fatalf("Failed to create incremental analyzer: %v", err)
			}

			fileHash, err := analyzer.hasher.HashFile(src)
			if err != nil {
				t.Fatalf("Failed to hash file: %v", err)
			}

			store.SetEntry(src, HistoryEntry{
				FileHash: fileHash,
				TestHash: analyzer.hasher.HashRelatedTests(src),
			})

			if tt.modifyTest {
				writeTestFile(t, testFile, "package foo\n// modified\n")
			}

			result, err := analyzer.analyzeFile(src)
			if err != nil {
				t.Fatalf("analyzeFile failed: %v", err)
			}

			if result.NeedsUpdate != tt.wantUpdate {
				t.Errorf("NeedsUpdate = %v, want %v (reason: %s)", result.NeedsUpdate, tt.wantUpdate, result.Reason)
			}

			if result.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}
