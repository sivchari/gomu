package execution

import (
	"fmt"
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
			name: "mutates addition to subtraction",
			fileContent: `package main

func Add(x, y int) int {
	return x + y
}`,
			mutant: mutation.Mutant{
				ID:       "arith-1",
				Type:     "arithmetic_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "+",
				Mutated:  "-",
			},
			expectError: false,
			expected:    "x - y",
		},
		{
			name: "mutates multiplication to division",
			fileContent: `package main

func Multiply(a, b int) int {
	return a * b
}`,
			mutant: mutation.Mutant{
				ID:       "arith-2",
				Type:     "arithmetic_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "*",
				Mutated:  "/",
			},
			expectError: false,
			expected:    "a / b",
		},
		{
			name: "mutates modulo to multiplication",
			fileContent: `package main

func Modulo(x, y int) int {
	return x % y
}`,
			mutant: mutation.Mutant{
				ID:       "arith-3",
				Type:     "arithmetic_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "%",
				Mutated:  "*",
			},
			expectError: false,
			expected:    "x * y",
		},
		{
			name: "fails with invalid token",
			fileContent: `package main

func Something(x, y int) int {
	return x + y
}`,
			mutant: mutation.Mutant{
				ID:       "arith-4",
				Type:     "arithmetic_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
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
	tests := []struct {
		name        string
		fileContent string
		mutant      mutation.Mutant
		expected    string
	}{
		{
			name: "mutates += to -=",
			fileContent: `package main

func Accumulate(x int) {
	sum := 0
	sum += x
}`,
			mutant: mutation.Mutant{
				ID:       "assign-1",
				Type:     "arithmetic_assign",
				Line:     5,
				Column:   2, // Adjust column to match actual position
				Original: "+=",
				Mutated:  "-=",
			},
			expected: "sum -= x",
		},
		{
			name: "mutates *= to /=",
			fileContent: `package main

func Scale(factor int) {
	value := 10
	value *= factor
}`,
			mutant: mutation.Mutant{
				ID:       "assign-2",
				Type:     "arithmetic_assign",
				Line:     5,
				Column:   2, // Adjust column to match actual position
				Original: "*=",
				Mutated:  "/=",
			},
			expected: "value /= factor",
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
			testFile := filepath.Join(tempDir, "assign.go")

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

func TestMutateIncDec(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		mutant      mutation.Mutant
		expected    string
	}{
		{
			name: "mutates ++ to --",
			fileContent: `package main

func Increment(x int) int {
	x++
	return x
}`,
			mutant: mutation.Mutant{
				ID:       "incdec-1",
				Type:     "arithmetic_incdec",
				Line:     4,
				Column:   2,
				Original: "++",
				Mutated:  "--",
			},
			expected: "x--",
		},
		{
			name: "mutates -- to ++",
			fileContent: `package main

func Decrement(x int) int {
	x--
	return x
}`,
			mutant: mutation.Mutant{
				ID:       "incdec-2",
				Type:     "arithmetic_incdec",
				Line:     4,
				Column:   2,
				Original: "--",
				Mutated:  "++",
			},
			expected: "x++",
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
			testFile := filepath.Join(tempDir, "incdec.go")

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
	tests := []struct {
		name        string
		fileContent string
		mutant      mutation.Mutant
		expected    string
	}{
		{
			name: "mutates && to ||",
			fileContent: `package main

func BothTrue(a, b bool) bool {
	return a && b
}`,
			mutant: mutation.Mutant{
				ID:       "logic-1",
				Type:     "logical_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "&&",
				Mutated:  "||",
			},
			expected: "a || b",
		},
		{
			name: "mutates || to &&",
			fileContent: `package main

func EitherTrue(x, y bool) bool {
	return x || y
}`,
			mutant: mutation.Mutant{
				ID:       "logic-2",
				Type:     "logical_binary",
				Line:     4,
				Column:   9, // Column of the start of expression
				Original: "||",
				Mutated:  "&&",
			},
			expected: "x && y",
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
			testFile := filepath.Join(tempDir, "logic.go")

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

func TestWriteModifiedAST(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, func())
		expectError bool
	}{
		{
			name: "successfully writes modified AST",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "write.go")

				content := `package main

func Test() {
	x := 1 + 2
}`
				os.WriteFile(testFile, []byte(content), 0644)

				return testFile, func() {}
			},
			expectError: false,
		},
		{
			name: "handles write permission error",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "readonly.go")

				content := `package main

func Test() {}`
				os.WriteFile(testFile, []byte(content), 0644)

				// Make directory read-only
				os.Chmod(tempDir, 0555)

				return testFile, func() {
					os.Chmod(tempDir, 0755)
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile, cleanup := tt.setupFunc(t)
			defer cleanup()

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
				Column:   7, // Column of the start of expression (1 in "1 + 2")
				Original: "+",
				Mutated:  "-",
			}

			err = mutator.ApplyMutation(mutant)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBackupFileEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (*SourceMutator, string)
		mutantID    string
		expectError bool
	}{
		{
			name: "backup file with special characters in name",
			setupFunc: func(t *testing.T) (*SourceMutator, string) {
				mutator, _ := NewSourceMutator()
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "file-with-dashes_and_underscores.go")
				os.WriteFile(testFile, []byte("package main"), 0644)

				return mutator, testFile
			},
			mutantID:    "special-123",
			expectError: false,
		},
		{
			name: "backup very large file",
			setupFunc: func(t *testing.T) (*SourceMutator, string) {
				mutator, _ := NewSourceMutator()
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "large.go")

				// Create a large file (1MB)
				largeContent := make([]byte, 1024*1024)
				for i := range largeContent {
					largeContent[i] = byte('a' + (i % 26))
				}
				os.WriteFile(testFile, largeContent, 0644)

				return mutator, testFile
			},
			mutantID:    "large-1",
			expectError: false,
		},
		{
			name: "backup file that gets deleted during operation",
			setupFunc: func(t *testing.T) (*SourceMutator, string) {
				mutator, _ := NewSourceMutator()
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "vanish.go")
				os.WriteFile(testFile, []byte("package main"), 0644)

				// Delete file after creation
				os.Remove(testFile)

				return mutator, testFile
			},
			mutantID:    "vanish-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, testFile := tt.setupFunc(t)
			defer mutator.Cleanup()

			err := mutator.backupFile(testFile, tt.mutantID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify backup exists
				backupPath := mutator.getBackupPath(testFile, tt.mutantID)
				if _, err := os.Stat(backupPath); err != nil {
					t.Errorf("backup file should exist: %v", err)
				}
			}
		})
	}
}

func TestRestoreOriginalEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (*SourceMutator, string, string)
		expectError bool
	}{
		{
			name: "restore when original file is deleted",
			setupFunc: func(t *testing.T) (*SourceMutator, string, string) {
				mutator, _ := NewSourceMutator()
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "delete.go")

				// Create and backup file
				os.WriteFile(testFile, []byte("original"), 0644)
				mutator.backupFile(testFile, "del-1")

				// Delete original
				os.Remove(testFile)

				return mutator, testFile, "del-1"
			},
			expectError: false, // Should still restore
		},
		{
			name: "restore when backup is corrupted",
			setupFunc: func(t *testing.T) (*SourceMutator, string, string) {
				mutator, _ := NewSourceMutator()
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "corrupt.go")

				os.WriteFile(testFile, []byte("original"), 0644)
				mutator.backupFile(testFile, "cor-1")

				// Corrupt the backup
				backupPath := mutator.getBackupPath(testFile, "cor-1")
				os.Remove(backupPath)

				return mutator, testFile, "cor-1"
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, testFile, mutantID := tt.setupFunc(t)
			defer mutator.Cleanup()

			err := mutator.RestoreOriginal(testFile, mutantID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCleanupEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) *SourceMutator
		expectError bool
	}{
		{
			name: "cleanup with nested directories",
			setupFunc: func(_ *testing.T) *SourceMutator {
				mutator, _ := NewSourceMutator()

				// Create nested structure
				nestedDir := filepath.Join(mutator.backupDir, "nested", "deep", "dir")
				os.MkdirAll(nestedDir, 0750)

				// Add some files
				os.WriteFile(filepath.Join(nestedDir, "file1.bak"), []byte("data"), 0600)
				os.WriteFile(filepath.Join(mutator.backupDir, "file2.bak"), []byte("data"), 0600)

				return mutator
			},
			expectError: false,
		},
		{
			name: "cleanup already cleaned directory",
			setupFunc: func(_ *testing.T) *SourceMutator {
				mutator, _ := NewSourceMutator()

				// Pre-cleanup
				os.RemoveAll(mutator.backupDir)

				return mutator
			},
			expectError: false, // RemoveAll doesn't error on non-existent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator := tt.setupFunc(t)

			err := mutator.Cleanup()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify directory is gone
				if _, err := os.Stat(mutator.backupDir); !os.IsNotExist(err) {
					t.Error("backup directory should not exist after cleanup")
				}
			}
		})
	}
}

func TestConcurrentBackupOperations(t *testing.T) {
	mutator, err := NewSourceMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	tempDir := t.TempDir()
	numFiles := 5
	content := "package main\n\nfunc main() {}\n"

	// Create multiple test files
	testFiles := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))

		err = os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file %d: %v", i, err)
		}

		testFiles[i] = testFile
	}

	// Test concurrent backup operations
	done := make(chan error, numFiles)

	for i, testFile := range testFiles {
		go func(id int, filePath string) {
			err := mutator.backupFile(filePath, fmt.Sprintf("test-%d", id))
			done <- err
		}(i, testFile)
	}

	// Wait for all backups to complete
	for i := 0; i < numFiles; i++ {
		err := <-done
		if err != nil {
			t.Errorf("concurrent backup %d failed: %v", i, err)
		}
	}

	// Verify all backups exist and have correct content
	for i, testFile := range testFiles {
		backupPath := mutator.getBackupPath(testFile, fmt.Sprintf("test-%d", i))

		backupContent, err := os.ReadFile(backupPath)
		if err != nil {
			t.Errorf("failed to read backup %d: %v", i, err)
		} else if string(backupContent) != content {
			t.Errorf("backup %d content mismatch", i)
		}
	}
}

func TestUniqueBackupPaths(t *testing.T) {
	mutator, err := NewSourceMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	tests := []struct {
		name      string
		filePaths []string
	}{
		{
			name: "different file paths generate different backup paths",
			filePaths: []string{
				"/path/to/file1.go",
				"/path/to/file2.go",
				"/different/path/file1.go",
			},
		},
		{
			name: "same file path generates same backup path",
			filePaths: []string{
				"/path/to/same.go",
				"/path/to/same.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupPaths := make([]string, len(tt.filePaths))

			for i, filePath := range tt.filePaths {
				// Use same mutant ID for testing same file path consistency
				mutantID := "test-1"
				if tt.name == "different file paths generate different backup paths" {
					mutantID = fmt.Sprintf("test-%d", i)
				}

				backupPaths[i] = mutator.getBackupPath(filePath, mutantID)
			}

			// Check for uniqueness when paths are different
			if tt.name == "different file paths generate different backup paths" {
				for i := 0; i < len(backupPaths); i++ {
					for j := i + 1; j < len(backupPaths); j++ {
						if backupPaths[i] == backupPaths[j] {
							t.Errorf("backup paths should be different: %s == %s", backupPaths[i], backupPaths[j])
						}
					}
				}
			}

			// Check for consistency when paths are same
			if tt.name == "same file path generates same backup path" {
				for i := 1; i < len(backupPaths); i++ {
					if backupPaths[i] != backupPaths[0] {
						t.Errorf("backup paths should be same: %s != %s", backupPaths[i], backupPaths[0])
					}
				}
			}
		})
	}
}
