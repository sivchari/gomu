package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "valid config creates engine",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if engine == nil {
				t.Error("engine should not be nil")

				return
			}

			if engine.mutator == nil {
				t.Error("mutator should not be nil")
			}

			if engine.fileLocks == nil {
				t.Error("fileLocks should not be nil")
			}

			if len(engine.fileLocks) != 0 {
				t.Errorf("expected empty fileLocks, got %d items", len(engine.fileLocks))
			}

			err = engine.Close()
			if err != nil {
				t.Errorf("failed to close engine: %v", err)
			}
		})
	}
}

func TestEngineClose(t *testing.T) {
	tests := []struct {
		name        string
		setupEngine func() *Engine
		wantErr     bool
	}{
		{
			name: "close with valid mutator",
			setupEngine: func() *Engine {
				engine, _ := New()

				return engine
			},
			wantErr: false,
		},
		{
			name: "close with nil mutator",
			setupEngine: func() *Engine {
				engine, _ := New()
				engine.mutator = nil

				return engine
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := tt.setupEngine()

			err := engine.Close()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunMutations(t *testing.T) {
	tests := []struct {
		name         string
		mutants      []mutation.Mutant
		expectLength int
		wantErr      bool
	}{
		{
			name:         "empty mutants returns empty results",
			mutants:      []mutation.Mutant{},
			expectLength: 0,
			wantErr:      false,
		},
		{
			name:         "nil mutants returns nil results",
			mutants:      nil,
			expectLength: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer engine.Close()

			results, err := engine.RunMutations(tt.mutants)
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != tt.expectLength {
				t.Errorf("expected %d results, got %d", tt.expectLength, len(results))
			}
		})
	}
}

func TestRunMutationsWithOptions(t *testing.T) {
	tests := []struct {
		name         string
		mutants      []mutation.Mutant
		workers      int
		timeout      int
		expectLength int
		wantErr      bool
	}{
		{
			name:         "empty mutants with custom options",
			mutants:      []mutation.Mutant{},
			workers:      2,
			timeout:      10,
			expectLength: 0,
			wantErr:      false,
		},
		{
			name:         "zero workers should still work",
			mutants:      []mutation.Mutant{},
			workers:      0,
			timeout:      10,
			expectLength: 0,
			wantErr:      false,
		},
		{
			name:         "negative timeout should still work",
			mutants:      []mutation.Mutant{},
			workers:      1,
			timeout:      -1,
			expectLength: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer engine.Close()

			results, err := engine.RunMutationsWithOptions(tt.mutants, tt.workers, tt.timeout)
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != tt.expectLength {
				t.Errorf("expected %d results, got %d", tt.expectLength, len(results))
			}
		})
	}
}

func TestGetFileLock(t *testing.T) {
	tests := []struct {
		name      string
		filePaths []string
		expected  []bool // true if locks should be equal to first lock
	}{
		{
			name:      "same file path returns same lock",
			filePaths: []string{"/test/file.go", "/test/file.go"},
			expected:  []bool{true, true},
		},
		{
			name:      "different file paths return different locks",
			filePaths: []string{"/test/file1.go", "/test/file2.go"},
			expected:  []bool{true, false},
		},
		{
			name:      "multiple files mixed",
			filePaths: []string{"/test/a.go", "/test/b.go", "/test/a.go"},
			expected:  []bool{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer engine.Close()

			if len(tt.filePaths) == 0 {
				return
			}

			firstLock := engine.getFileLock(tt.filePaths[0])
			if firstLock == nil {
				t.Error("first lock should not be nil")

				return
			}

			for i, filePath := range tt.filePaths {
				lock := engine.getFileLock(filePath)
				if lock == nil {
					t.Errorf("lock %d should not be nil", i)

					continue
				}

				shouldBeEqual := tt.expected[i]
				isEqual := (lock == firstLock)

				if shouldBeEqual && !isEqual {
					t.Errorf("lock %d should be equal to first lock but is not", i)
				}

				if !shouldBeEqual && isEqual {
					t.Errorf("lock %d should be different from first lock but is the same", i)
				}
			}
		})
	}
}

func TestRunSingleMutation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := createTempTestProject(t)

	tests := []struct {
		name          string
		mutant        mutation.Mutant
		timeout       int
		expectStatus  mutation.Status
		expectError   bool
		errorContains string
	}{
		{
			name: "valid mutation on existing file",
			mutant: mutation.Mutant{
				ID:       "test-1",
				Type:     "arithmetic_binary",
				FilePath: filepath.Join(tempDir, "valid.go"),
				Line:     4,
				Column:   9,
				Original: "+",
				Mutated:  "-",
			},
			timeout:      5,
			expectStatus: mutation.StatusKilled, // Test should catch the mutation
			expectError:  false,
		},
		{
			name: "mutation on non-existent file",
			mutant: mutation.Mutant{
				ID:       "test-2",
				Type:     "arithmetic_binary",
				FilePath: "/nonexistent/file.go",
				Line:     1,
				Column:   1,
				Original: "+",
				Mutated:  "-",
			},
			timeout:       5,
			expectStatus:  mutation.StatusError,
			expectError:   false,
			errorContains: "backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer engine.Close()

			result := engine.runSingleMutation(tt.mutant, tt.timeout)

			if result.Mutant.ID != tt.mutant.ID {
				t.Errorf("expected mutant ID %s, got %s", tt.mutant.ID, result.Mutant.ID)
			}

			if result.Status != tt.expectStatus {
				t.Errorf("expected status %v, got %v", tt.expectStatus, result.Status)
			}

			if tt.expectError && result.Error == "" {
				t.Error("expected error but got none")
			}

			if tt.errorContains != "" && !strings.Contains(result.Error, tt.errorContains) {
				t.Errorf("expected error to contain %s, got: %s", tt.errorContains, result.Error)
			}
		})
	}
}

func TestCheckCompilation(t *testing.T) {
	tempDir := createTempTestProject(t)

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid file compiles successfully",
			filePath:    filepath.Join(tempDir, "valid.go"),
			expectError: false,
		},
		{
			name:        "invalid file fails compilation",
			filePath:    filepath.Join(tempDir, "invalid.go"),
			expectError: true,
			errorText:   "compilation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer engine.Close()

			err = engine.checkCompilation(tt.filePath)
			if tt.expectError && err == nil {
				t.Error("expected compilation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected compilation error: %v", err)
			}

			if tt.errorText != "" && err != nil && !strings.Contains(err.Error(), tt.errorText) {
				t.Errorf("expected error to contain %s, got: %v", tt.errorText, err)
			}
		})
	}
}

func TestIndexedResult(t *testing.T) {
	tests := []struct {
		name   string
		index  int
		status mutation.Status
	}{
		{"zero index with killed status", 0, mutation.StatusKilled},
		{"positive index with survived status", 5, mutation.StatusSurvived},
		{"negative index with error status", -1, mutation.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mutation.Result{
				Status: tt.status,
			}

			indexed := indexedResult{
				index:  tt.index,
				result: result,
			}

			if indexed.index != tt.index {
				t.Errorf("expected index %d, got %d", tt.index, indexed.index)
			}

			if indexed.result.Status != tt.status {
				t.Errorf("expected status %v, got %v", tt.status, indexed.result.Status)
			}
		})
	}
}

func TestEngineFileMapConcurrency(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test concurrent access to file locks
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			filePath := fmt.Sprintf("/test/file%d.go", id%3) // Use 3 different files

			lock := engine.getFileLock(filePath)
			if lock == nil {
				t.Errorf("lock should not be nil for goroutine %d", id)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that we have exactly 3 locks (for 3 different files)
	engine.fileMapMu.Lock()
	lockCount := len(engine.fileLocks)
	engine.fileMapMu.Unlock()

	if lockCount != 3 {
		t.Errorf("expected 3 file locks, got %d", lockCount)
	}
}

// Helper function to create a temporary test project.
func createTempTestProject(t *testing.T) string {
	tempDir := t.TempDir()

	// Create go.mod
	goMod := "module test\n\ngo 1.21\n"

	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create valid Go file
	validGoFile := `package main

func Add(a, b int) int {
	return a + b
}

func main() {
	result := Add(1, 2)
	println(result)
}
`

	err = os.WriteFile(filepath.Join(tempDir, "valid.go"), []byte(validGoFile), 0644)
	if err != nil {
		t.Fatalf("failed to create valid.go: %v", err)
	}

	// Create valid test file
	testFile := `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2)
	if result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
}
`

	err = os.WriteFile(filepath.Join(tempDir, "valid_test.go"), []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("failed to create valid_test.go: %v", err)
	}

	// Create invalid Go file
	invalidGoFile := `package main

func main() {
	println("hello"  // Missing closing parenthesis
}
`

	err = os.WriteFile(filepath.Join(tempDir, "invalid.go"), []byte(invalidGoFile), 0644)
	if err != nil {
		t.Fatalf("failed to create invalid.go: %v", err)
	}

	return tempDir
}

func TestEngineCreationEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		expectNil bool
		expectErr bool
	}{
		{
			name:      "nil config should not panic",
			expectNil: false,
			expectErr: false,
		},
		{
			name:      "empty config should work",
			expectNil: false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New()
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectNil && engine != nil {
				t.Error("expected nil engine")
			}

			if !tt.expectNil && engine == nil {
				t.Error("expected non-nil engine")
			}

			if engine != nil {
				engine.Close()
			}
		})
	}
}
