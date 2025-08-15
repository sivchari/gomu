package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

const (
	testPackageContent = "package main\n\nfunc main() {}\n"
	nonExistentFile    = "/nonexistent/file.go"
)

func verifyMutation(t *testing.T, testFile, expectedChange string) {
	t.Helper()

	mutatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read mutated file: %v", err)
	}

	if !strings.Contains(string(mutatedContent), expectedChange) {
		t.Errorf("expected mutated content to contain %q, got: %s", expectedChange, string(mutatedContent))
	}
}

func verifyRestoration(t *testing.T, mutator *SourceMutator, testFile, originalContent string) {
	t.Helper()

	err := mutator.RestoreOriginal(testFile, "1")
	if err != nil {
		t.Fatalf("failed to restore file: %v", err)
	}

	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(restoredContent) != originalContent {
		t.Errorf("restored content doesn't match original")
	}
}

func TestNewSourceMutator(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "creates mutator successfully",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if tt.expectError && err == nil {
				t.Error("expected error but got none")

				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if mutator == nil {
				t.Error("mutator should not be nil")

				return
			}

			if mutator.backupDir == "" {
				t.Error("backup directory should not be empty")
			}

			// Verify backup directory exists
			_, err = os.Stat(mutator.backupDir)
			if err != nil {
				t.Errorf("backup directory should exist: %v", err)
			}

			// Cleanup
			err = mutator.Cleanup()
			if err != nil {
				t.Errorf("cleanup failed: %v", err)
			}
		})
	}
}

func TestSourceMutatorCleanup(t *testing.T) {
	tests := []struct {
		name             string
		setupMutator     func(t *testing.T) (*SourceMutator, string)
		expectError      bool
		shouldExistAfter bool
	}{
		{
			name: "cleanup removes backup directory",
			setupMutator: func(t *testing.T) (*SourceMutator, string) {
				mutator, err := NewSourceMutator()
				if err != nil {
					t.Fatalf("failed to create mutator: %v", err)
				}

				return mutator, mutator.backupDir
			},
			expectError:      false,
			shouldExistAfter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, backupDir := tt.setupMutator(t)

			// Verify directory exists before cleanup
			_, err := os.Stat(backupDir)
			if err != nil {
				t.Errorf("backup directory should exist before cleanup: %v", err)

				return
			}

			// Cleanup
			err = mutator.Cleanup()
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify directory status after cleanup
			_, err = os.Stat(backupDir)
			exists := !os.IsNotExist(err)

			if tt.shouldExistAfter && !exists {
				t.Error("directory should exist after cleanup")
			}

			if !tt.shouldExistAfter && exists {
				t.Error("directory should not exist after cleanup")
			}
		})
	}
}

func TestBackupFile(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(t *testing.T) (string, string) // returns filepath and content
		expectError    bool
		validateBackup bool
	}{
		{
			name: "backup valid file successfully",
			setupFile: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "test.go")
				content := testPackageContent
				err := os.WriteFile(testFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}

				return testFile, content
			},
			expectError:    false,
			validateBackup: true,
		},
		{
			name: "backup non-existent file fails",
			setupFile: func(_ *testing.T) (string, string) {
				return nonExistentFile, ""
			},
			expectError:    true,
			validateBackup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			testFile, originalContent := tt.setupFile(t)

			err = mutator.backupFile(testFile, "test-mutant-1")
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.validateBackup && err == nil {
				backupPath := mutator.getBackupPath(testFile, "test-mutant-1")

				backupContent, err := os.ReadFile(backupPath)
				if err != nil {
					t.Errorf("failed to read backup file: %v", err)
				} else if string(backupContent) != originalContent {
					t.Errorf("backup content mismatch: expected %q, got %q", originalContent, string(backupContent))
				}
			}
		})
	}
}

func TestRestoreOriginal(t *testing.T) {
	tests := []struct {
		name            string
		setupScenario   func(t *testing.T) (*SourceMutator, string, string, string) // mutator, filepath, original, modified
		expectError     bool
		validateRestore bool
	}{
		{
			name: "restore file successfully",
			setupScenario: func(t *testing.T) (*SourceMutator, string, string, string) {
				mutator, err := NewSourceMutator()
				if err != nil {
					t.Fatalf("failed to create mutator: %v", err)
				}

				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "test.go")
				originalContent := "package main\n\nfunc main() {}\n"
				modifiedContent := "package main\n\nfunc main() { println(\"modified\") }\n"

				// Create original file and backup
				err = os.WriteFile(testFile, []byte(originalContent), 0644)
				if err != nil {
					t.Fatalf("failed to create original file: %v", err)
				}

				err = mutator.backupFile(testFile, "test-mutant-1")
				if err != nil {
					t.Fatalf("failed to backup file: %v", err)
				}

				// Modify the file
				err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
				if err != nil {
					t.Fatalf("failed to modify file: %v", err)
				}

				return mutator, testFile, originalContent, modifiedContent
			},
			expectError:     false,
			validateRestore: true,
		},
		{
			name: "restore non-existent backup fails",
			setupScenario: func(t *testing.T) (*SourceMutator, string, string, string) {
				mutator, err := NewSourceMutator()
				if err != nil {
					t.Fatalf("failed to create mutator: %v", err)
				}

				return mutator, "/nonexistent/file.go", "", ""
			},
			expectError:     true,
			validateRestore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, testFile, originalContent, _ := tt.setupScenario(t)
			defer mutator.Cleanup()

			err := mutator.RestoreOriginal(testFile, "test-mutant-1")
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.validateRestore && err == nil {
				restoredContent, err := os.ReadFile(testFile)
				if err != nil {
					t.Errorf("failed to read restored file: %v", err)
				} else if string(restoredContent) != originalContent {
					t.Errorf("restored content mismatch: expected %q, got %q", originalContent, string(restoredContent))
				}
			}
		})
	}
}

func TestGetBackupPath(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		expectContains []string
	}{
		{
			name:     "generates valid backup path",
			filePath: "/path/to/test.go",
			expectContains: []string{
				"test.go",
				"_original",
			},
		},
		{
			name:     "handles different file extensions",
			filePath: "/src/calculator.go",
			expectContains: []string{
				"calculator.go",
				"_original",
			},
		},
		{
			name:     "handles files without extension",
			filePath: "/bin/executable",
			expectContains: []string{
				"executable",
				"_original",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			backupPath := mutator.getBackupPath(tt.filePath, "test-mutant-1")

			if backupPath == "" {
				t.Error("backup path should not be empty")

				return
			}

			if !strings.Contains(backupPath, mutator.backupDir) {
				t.Errorf("backup path should contain backup directory: %s", backupPath)
			}

			for _, expectedSubstring := range tt.expectContains {
				if !strings.Contains(backupPath, expectedSubstring) {
					t.Errorf("backup path should contain %q, got: %s", expectedSubstring, backupPath)
				}
			}
		})
	}
}

func TestApplyMutation(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(t *testing.T) (string, string)
		mutant         func(filePath string) mutation.Mutant
		expectError    bool
		expectedChange string
	}{
		{
			name: "apply arithmetic binary mutation",
			setupFile: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "test.go")
				content := `package main

func Add(a, b int) int {
	return a + b
}
`
				err := os.WriteFile(testFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}

				return testFile, content
			},
			mutant: func(filePath string) mutation.Mutant {
				return mutation.Mutant{
					ID:       "1",
					Type:     "arithmetic_binary",
					FilePath: filePath,
					Line:     4,
					Column:   9,
					Original: "+",
					Mutated:  "-",
				}
			},
			expectError:    false,
			expectedChange: "a - b",
		},
		{
			name: "apply mutation to non-existent file",
			setupFile: func(_ *testing.T) (string, string) {
				return nonExistentFile, ""
			},
			mutant: func(filePath string) mutation.Mutant {
				return mutation.Mutant{
					ID:       "1",
					Type:     "arithmetic_binary",
					FilePath: filePath,
					Line:     1,
					Column:   1,
					Original: "+",
					Mutated:  "-",
				}
			},
			expectError:    true,
			expectedChange: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			testFile, originalContent := tt.setupFile(t)
			mutant := tt.mutant(testFile)

			err = mutator.ApplyMutation(mutant)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectedChange != "" && err == nil {
				verifyMutation(t, testFile, tt.expectedChange)
				verifyRestoration(t, mutator, testFile, originalContent)
			}
		})
	}
}

func TestMutateFileInvalidSyntax(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		expectedErrorMsg string
	}{
		{
			name: "invalid syntax causes parse error",
			fileContent: `package main

func main() {
	invalid syntax here!!!
}
`,
			expectedErrorMsg: "failed to parse file",
		},
		{
			name:             "empty file causes parse error",
			fileContent:      "",
			expectedErrorMsg: "failed to parse file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "invalid.go")

			err = os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			mutant := mutation.Mutant{
				ID:       "1",
				Type:     "arithmetic_binary",
				FilePath: testFile,
				Line:     1,
				Column:   1,
				Original: "+",
				Mutated:  "-",
			}

			err = mutator.mutateFile(mutant)
			if err == nil {
				t.Error("expected error but got none")
			} else if !strings.Contains(err.Error(), tt.expectedErrorMsg) {
				t.Errorf("expected error to contain %q, got: %v", tt.expectedErrorMsg, err)
			}
		})
	}
}

func TestMutateFileTargetNotFound(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		mutant      mutation.Mutant
	}{
		{
			name: "mutation target at non-existent line",
			fileContent: `package main

func main() {
	println("hello")
}
`,
			mutant: mutation.Mutant{
				ID:       "1",
				Type:     "arithmetic_binary",
				FilePath: "",  // will be set in test
				Line:     999, // Line that doesn't exist
				Column:   1,
				Original: "+",
				Mutated:  "-",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")

			err = os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			mutant := tt.mutant
			mutant.FilePath = testFile

			err = mutator.mutateFile(mutant)
			if err == nil {
				t.Error("expected error but got none")
			} else if !strings.Contains(err.Error(), "failed to find mutation target") {
				t.Errorf("expected error about mutation target, got: %v", err)
			}
		})
	}
}

func TestApplyMutationToNode(t *testing.T) {
	tests := []struct {
		name         string
		mutationType string
		expectResult bool
	}{
		{"arithmetic_binary", "arithmetic_binary", false}, // Will fail because node type doesn't match
		{"arithmetic_assign", "arithmetic_assign", false},
		{"arithmetic_incdec", "arithmetic_incdec", false},
		{"conditional_binary", "conditional_binary", false},
		{"logical_binary", "logical_binary", false},
		{"logical_not_removal", "logical_not_removal", false},
		{"unknown_type", "unknown_type", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			// Use nil since the method checks type anyway
			result := mutator.applyMutationToNode(nil, mutation.Mutant{
				Type:    tt.mutationType,
				Mutated: "+",
			})

			if result != tt.expectResult {
				t.Errorf("expected %v, got %v", tt.expectResult, result)
			}
		})
	}
}

func TestComplexMutationScenario(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		mutants         []mutation.Mutant
		expectSuccess   bool
		expectedChanges []string
	}{
		{
			name: "multiple mutations on same file",
			fileContent: `package main

func Calculate(a, b int) int {
	if a > b {
		return a + b
	}
	return a - b
}
`,
			mutants: []mutation.Mutant{
				{
					ID:       "1",
					Type:     "arithmetic_binary",
					FilePath: "", // will be set in test
					Line:     5,
					Column:   10,
					Original: "+",
					Mutated:  "-",
				},
				{
					ID:       "2",
					Type:     "conditional_binary",
					FilePath: "", // will be set in test
					Line:     4,
					Column:   5,
					Original: ">",
					Mutated:  "<",
				},
			},
			expectSuccess:   true,
			expectedChanges: []string{"a - b", "a < b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "complex.go")

			err = os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			originalContent := tt.fileContent

			// Test each mutation separately
			for i, mutant := range tt.mutants {
				mutant.FilePath = testFile

				err = mutator.ApplyMutation(mutant)
				if tt.expectSuccess && err != nil {
					t.Errorf("mutation %d failed: %v", i, err)

					continue
				}

				if !tt.expectSuccess && err == nil {
					t.Errorf("mutation %d should have failed", i)

					continue
				}

				if tt.expectSuccess && i < len(tt.expectedChanges) {
					mutatedContent, readErr := os.ReadFile(testFile)
					if readErr != nil {
						t.Errorf("failed to read mutated file for mutation %d: %v", i, readErr)
					} else if !strings.Contains(string(mutatedContent), tt.expectedChanges[i]) {
						t.Errorf("mutation %d: expected content to contain %q", i, tt.expectedChanges[i])
					}
				}

				// Restore for next mutation
				err = mutator.RestoreOriginal(testFile, mutant.ID)
				if err != nil {
					t.Errorf("failed to restore original after mutation %d: %v", i, err)
				}

				// Verify restoration
				restoredContent, readErr := os.ReadFile(testFile)
				if readErr != nil {
					t.Errorf("failed to read restored file after mutation %d: %v", i, readErr)
				} else if string(restoredContent) != originalContent {
					t.Errorf("content not properly restored after mutation %d", i)
				}
			}
		})
	}
}

func TestMutateArithmeticBinary(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		mutant      mutation.Mutant
		expectError bool
		expected    string
	}{
		{
			name: "mutates arithmetic operators",
			fileContent: `package main

func Add(x, y int) int {
	return x + y
}`,
			mutant: mutation.Mutant{
				ID:       "arith-1",
				Type:     "arithmetic_binary",
				Line:     4,
				Column:   9,
				Original: "+",
				Mutated:  "-",
			},
			expectError: false,
			expected:    "x - y",
		},
		{
			name: "fails with invalid token",
			fileContent: `package main

func Something(x, y int) int {
	return x + y
}`,
			mutant: mutation.Mutant{
				ID:       "arith-2",
				Type:     "arithmetic_binary",
				Line:     4,
				Column:   9,
				Original: "+",
				Mutated:  "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "arith.go")

			err = os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			tt.mutant.FilePath = testFile

			err = mutator.ApplyMutation(tt.mutant)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				content, _ := os.ReadFile(testFile)
				if !strings.Contains(string(content), tt.expected) {
					t.Errorf("expected content to contain %q, got: %s", tt.expected, string(content))
				}
			}
		})
	}
}

func TestMutateArithmeticAssign(t *testing.T) {
	fileContent := `package main

func Accumulate(x int) {
	sum := 0
	sum += x
}`
	mutant := mutation.Mutant{
		ID:       "assign-1",
		Type:     "arithmetic_assign",
		Line:     5,
		Column:   2,
		Original: "+=",
		Mutated:  "-=",
	}

	mutator, err := NewSourceMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "assign.go")

	err = os.WriteFile(testFile, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mutant.FilePath = testFile

	err = mutator.ApplyMutation(mutant)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "sum -= x") {
		t.Errorf("expected content to contain 'sum -= x', got: %s", string(content))
	}
}

func TestMutateIncDec(t *testing.T) {
	fileContent := `package main

func Increment(x int) int {
	x++
	return x
}`
	mutant := mutation.Mutant{
		ID:       "incdec-1",
		Type:     "arithmetic_incdec",
		Line:     4,
		Column:   2,
		Original: "++",
		Mutated:  "--",
	}

	mutator, err := NewSourceMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "incdec.go")

	err = os.WriteFile(testFile, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mutant.FilePath = testFile

	err = mutator.ApplyMutation(mutant)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "x--") {
		t.Errorf("expected content to contain 'x--', got: %s", string(content))
	}
}

func TestMutateConditional(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		mutant      mutation.Mutant
		expected    string
	}{
		{
			name: "mutates > to <",
			fileContent: `package main

func IsGreater(a, b int) bool {
	return a > b
}`,
			mutant: mutation.Mutant{
				ID:       "cond-1",
				Type:     "conditional_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: ">",
				Mutated:  "<",
			},
			expected: "a < b",
		},
		{
			name: "mutates == to !=",
			fileContent: `package main

func IsEqual(x, y int) bool {
	return x == y
}`,
			mutant: mutation.Mutant{
				ID:       "cond-2",
				Type:     "conditional_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "==",
				Mutated:  "!=",
			},
			expected: "x != y",
		},
		{
			name: "mutates <= to >=",
			fileContent: `package main

func IsLessOrEqual(m, n int) bool {
	return m <= n
}`,
			mutant: mutation.Mutant{
				ID:       "cond-3",
				Type:     "conditional_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "<=",
				Mutated:  ">=",
			},
			expected: "m >= n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewSourceMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "cond.go")

			err = os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			tt.mutant.FilePath = testFile

			err = mutator.ApplyMutation(tt.mutant)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			content, _ := os.ReadFile(testFile)
			if !strings.Contains(string(content), tt.expected) {
				t.Errorf("expected content to contain %q, got: %s", tt.expected, string(content))
			}
		})
	}
}

func TestMutateLogicalBinary(t *testing.T) {
	fileContent := `package main

func BothTrue(a, b bool) bool {
	return a && b
}`
	mutant := mutation.Mutant{
		ID:       "logic-1",
		Type:     "logical_binary",
		Line:     4,
		Column:   9,
		Original: "&&",
		Mutated:  "||",
	}

	mutator, err := NewSourceMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "logic.go")

	err = os.WriteFile(testFile, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mutant.FilePath = testFile

	err = mutator.ApplyMutation(mutant)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "a || b") {
		t.Errorf("expected content to contain 'a || b', got: %s", string(content))
	}
}

func TestWriteModifiedAST(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "write.go")

	content := `package main

func Test() {
	x := 1 + 2
}`
	os.WriteFile(testFile, []byte(content), 0644)

	mutator, err := NewSourceMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	// Apply a simple mutation to test writeModifiedAST
	mutant := mutation.Mutant{
		ID:       "test-1",
		Type:     "arithmetic_binary",
		FilePath: testFile,
		Line:     4,
		Column:   7,
		Original: "+",
		Mutated:  "-",
	}

	err = mutator.ApplyMutation(mutant)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the file was modified
	resultContent, _ := os.ReadFile(testFile)
	if !strings.Contains(string(resultContent), "1 - 2") {
		t.Error("expected file to be modified with mutation")
	}
}
