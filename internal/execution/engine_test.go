package execution

import (
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

			if engine.overlay == nil {
				t.Error("overlay should not be nil")
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
			name: "close with valid overlay",
			setupEngine: func() *Engine {
				engine, _ := New()

				return engine
			},
			wantErr: false,
		},
		{
			name: "close with nil overlay",
			setupEngine: func() *Engine {
				engine, _ := New()
				engine.overlay = nil

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
	tempDir := createTempTestProject(t)

	tests := []struct {
		name         string
		mutants      []mutation.Mutant
		workers      int
		timeout      int
		expectLength int
		wantErr      bool
		checkResults bool
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
		{
			name: "single mutant execution",
			mutants: []mutation.Mutant{
				{
					ID:       "test-single",
					Type:     "arithmetic_binary",
					FilePath: filepath.Join(tempDir, "valid.go"),
					Line:     4,
					Column:   9,
					Original: "+",
					Mutated:  "-",
				},
			},
			workers:      1,
			timeout:      10,
			expectLength: 1,
			wantErr:      false,
			checkResults: true,
		},
		{
			name: "multiple mutants execution with overlay - same file parallel",
			mutants: []mutation.Mutant{
				{
					ID:       "test-1",
					Type:     "arithmetic_binary",
					FilePath: filepath.Join(tempDir, "valid.go"),
					Line:     4,
					Column:   9,
					Original: "+",
					Mutated:  "-",
				},
				{
					ID:       "test-2",
					Type:     "arithmetic_binary",
					FilePath: filepath.Join(tempDir, "valid.go"),
					Line:     4,
					Column:   9,
					Original: "+",
					Mutated:  "*",
				},
			},
			workers:      2,
			timeout:      10,
			expectLength: 2,
			wantErr:      false,
			checkResults: true,
		},
		{
			name: "mutants with invalid file",
			mutants: []mutation.Mutant{
				{
					ID:       "test-invalid",
					Type:     "arithmetic_binary",
					FilePath: "/nonexistent/file.go",
					Line:     1,
					Column:   1,
					Original: "+",
					Mutated:  "-",
				},
			},
			workers:      1,
			timeout:      10,
			expectLength: 1,
			wantErr:      false,
			checkResults: true,
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

			if tt.checkResults {
				for i, result := range results {
					if i >= len(tt.mutants) {
						continue
					}

					if result.Mutant.ID != tt.mutants[i].ID {
						t.Errorf("result %d: expected mutant ID %s, got %s", i, tt.mutants[i].ID, result.Mutant.ID)
					}

					validStatuses := []mutation.Status{
						mutation.StatusKilled,
						mutation.StatusSurvived,
						mutation.StatusError,
						mutation.StatusNotViable,
						mutation.StatusTimedOut,
					}
					found := false

					for _, status := range validStatuses {
						if result.Status == status {
							found = true

							break
						}
					}

					if !found {
						t.Errorf("result %d: unexpected status %v", i, result.Status)
					}
				}
			}
		})
	}
}

func TestRunSingleMutation(t *testing.T) {
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
			timeout:      30,
			expectStatus: mutation.StatusKilled,
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
			errorContains: "prepare mutation",
		},
		{
			name: "test timeout scenario",
			mutant: mutation.Mutant{
				ID:       "test-4",
				Type:     "arithmetic_binary",
				FilePath: filepath.Join(tempDir, "valid.go"),
				Line:     4,
				Column:   9,
				Original: "+",
				Mutated:  "-",
			},
			timeout:       0,
			expectStatus:  mutation.StatusTimedOut,
			expectError:   false,
			errorContains: "timed out",
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
				t.Errorf("expected status %v, got %v\nError: %s\nOutput: %s",
					tt.expectStatus, result.Status, result.Error, result.Output)
			}

			if tt.expectError && result.Error == "" {
				t.Error("expected error but got none")
			}

			if tt.errorContains != "" && !strings.Contains(strings.ToLower(result.Error), strings.ToLower(tt.errorContains)) {
				t.Errorf("expected error to contain %s, got: %s", tt.errorContains, result.Error)
			}
		})
	}
}

func TestCheckCompilationWithOverlay(t *testing.T) {
	tempDir := createTempTestProject(t)

	t.Run("valid overlay compiles successfully", func(t *testing.T) {
		engine, err := New()
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		defer engine.Close()

		mutant := mutation.Mutant{
			ID:       "test-compile",
			Type:     "arithmetic_binary",
			FilePath: filepath.Join(tempDir, "valid.go"),
			Line:     4,
			Column:   9,
			Original: "+",
			Mutated:  "-",
		}

		ctx, err := engine.overlay.PrepareMutation(mutant)
		if err != nil {
			t.Fatalf("failed to prepare mutation: %v", err)
		}
		defer engine.overlay.CleanupMutation(ctx)

		err = engine.checkCompilationWithOverlay(ctx)
		if err != nil {
			t.Errorf("unexpected compilation error: %v", err)
		}
	})
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

func TestOverlayParallelExecution(t *testing.T) {
	tempDir := createTempTestProject(t)

	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Create multiple mutants for the same file - overlay allows true parallel execution
	mutants := make([]mutation.Mutant, 5)
	for i := range mutants {
		mutants[i] = mutation.Mutant{
			ID:       string(rune('a' + i)),
			Type:     "arithmetic_binary",
			FilePath: filepath.Join(tempDir, "valid.go"),
			Line:     4,
			Column:   9,
			Original: "+",
			Mutated:  "-",
		}
	}

	results, err := engine.RunMutationsWithOptions(mutants, 5, 30)
	if err != nil {
		t.Fatalf("failed to run mutations: %v", err)
	}

	if len(results) != len(mutants) {
		t.Errorf("expected %d results, got %d", len(mutants), len(results))
	}

	// All should complete without error status (killed or survived)
	for i, result := range results {
		if result.Status == mutation.StatusError {
			t.Errorf("mutant %d failed with error: %s", i, result.Error)
		}
	}
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

// Helper function to create a temporary test project.
func createTempTestProject(t *testing.T) string {
	tempDir := t.TempDir()

	goMod := "module test\n\ngo 1.21\n"

	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

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

	return tempDir
}
