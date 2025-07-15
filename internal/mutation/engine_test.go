package mutation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.Default()

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}

	if engine.config != cfg {
		t.Error("Engine config does not match provided config")
	}

	if len(engine.mutators) != 3 {
		t.Errorf("Expected 3 mutators, got %d", len(engine.mutators))
	}

	// Check mutator types
	mutatorNames := make(map[string]bool)
	for _, mutator := range engine.mutators {
		mutatorNames[mutator.Name()] = true
	}

	expectedMutators := []string{"arithmetic", "conditional", "logical"}
	for _, expected := range expectedMutators {
		if !mutatorNames[expected] {
			t.Errorf("Expected mutator %s not found", expected)
		}
	}
}

func TestNew_CustomMutators(t *testing.T) {
	cfg := config.Default()
	cfg.Mutation.Types = []string{"arithmetic", "conditional"}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	if len(engine.mutators) != 2 {
		t.Errorf("Expected 2 mutators, got %d", len(engine.mutators))
	}
}

func TestNew_InvalidMutator(t *testing.T) {
	cfg := config.Default()
	cfg.Mutation.Types = []string{"arithmetic", "invalid", "conditional"}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	// Should ignore invalid mutator
	if len(engine.mutators) != 2 {
		t.Errorf("Expected 2 mutators (invalid should be ignored), got %d", len(engine.mutators))
	}
}

func TestGenerateMutants(t *testing.T) {
	// Create temporary Go file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package main

func Add(a, b int) int {
	return a + b
}

func IsPositive(n int) bool {
	return n > 0
}

func LogicalTest(a, b bool) bool {
	return a && b
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := config.Default()

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(testFile)
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	if len(mutants) == 0 {
		t.Error("Expected mutants to be generated, got 0")
	}

	// Check that all mutants have required fields
	for i, mutant := range mutants {
		if mutant.ID == "" {
			t.Errorf("Mutant %d has empty ID", i)
		}

		if mutant.FilePath != testFile {
			t.Errorf("Mutant %d has wrong file path: %s", i, mutant.FilePath)
		}

		if mutant.Line <= 0 {
			t.Errorf("Mutant %d has invalid line number: %d", i, mutant.Line)
		}

		if mutant.Column <= 0 {
			t.Errorf("Mutant %d has invalid column number: %d", i, mutant.Column)
		}

		if mutant.Type == "" {
			t.Errorf("Mutant %d has empty type", i)
		}

		if mutant.Original == "" {
			t.Errorf("Mutant %d has empty original", i)
		}

		if mutant.Mutated == "" {
			t.Errorf("Mutant %d has empty mutated", i)
		}

		if mutant.Description == "" {
			t.Errorf("Mutant %d has empty description", i)
		}
	}

	// Check that we have different types of mutations
	mutationTypes := make(map[string]bool)
	for _, mutant := range mutants {
		mutationTypes[mutant.Type] = true
	}

	expectedTypes := []string{"arithmetic_binary", "conditional_binary", "logical_binary"}
	for _, expectedType := range expectedTypes {
		if !mutationTypes[expectedType] {
			t.Errorf("Expected mutation type %s not found", expectedType)
		}
	}
}

func TestGenerateMutants_MutationLimit(t *testing.T) {
	// Create temporary Go file with many operations
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	var codeBuilder strings.Builder

	codeBuilder.WriteString("package main\n\n")
	codeBuilder.WriteString("func ManyOperations() int {\n")
	codeBuilder.WriteString("    result := 0\n")

	// Add many arithmetic operations to exceed mutation limit
	for i := 0; i < 20; i++ {
		codeBuilder.WriteString("    result = result + 1\n")
		codeBuilder.WriteString("    result = result - 1\n")
		codeBuilder.WriteString("    result = result * 2\n")
		codeBuilder.WriteString("    result = result / 1\n")
	}

	codeBuilder.WriteString("    return result\n")
	codeBuilder.WriteString("}\n")

	err := os.WriteFile(testFile, []byte(codeBuilder.String()), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := config.Default()
	cfg.Mutation.Limit = 10 // Set low limit

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(testFile)
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	if len(mutants) > cfg.Mutation.Limit {
		t.Errorf("Expected mutants to be limited to %d, got %d", cfg.Mutation.Limit, len(mutants))
	}
}

func TestGenerateMutants_InvalidFile(t *testing.T) {
	cfg := config.Default()

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	// Test with nonexistent file
	_, err = engine.GenerateMutants("/nonexistent/file.go")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestGenerateMutants_InvalidSyntax(t *testing.T) {
	// Create temporary Go file with invalid syntax
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.go")

	invalidCode := `package main

func Invalid() {
    return +
}
`

	err := os.WriteFile(testFile, []byte(invalidCode), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := config.Default()

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	_, err = engine.GenerateMutants(testFile)
	if err == nil {
		t.Error("Expected error for invalid syntax, got nil")
	}
}

func TestGenerateMutants_NoMutations(t *testing.T) {
	// Create temporary Go file with no mutatable code
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nomutations.go")

	noMutationCode := `package main

import "fmt"

func NoMutations() {
    fmt.Println("hello")
}
`

	err := os.WriteFile(testFile, []byte(noMutationCode), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := config.Default()

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(testFile)
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	// Should return empty slice, not error
	if len(mutants) != 0 {
		t.Errorf("Expected 0 mutants for file with no mutations, got %d", len(mutants))
	}
}

func TestCreateMutator(t *testing.T) {
	cfg := config.Default()

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	tests := []struct {
		name     string
		expected string
	}{
		{"arithmetic", "arithmetic"},
		{"conditional", "conditional"},
		{"logical", "logical"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator := engine.createMutator(tt.name)
			if tt.expected == "" {
				if mutator != nil {
					t.Errorf("Expected nil mutator for %s, got %T", tt.name, mutator)
				}
			} else {
				if mutator == nil {
					t.Errorf("Expected mutator for %s, got nil", tt.name)
				} else if mutator.Name() != tt.expected {
					t.Errorf("Expected mutator name %s, got %s", tt.expected, mutator.Name())
				}
			}
		})
	}
}
