package mutation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNew verifies the engine registers exactly the mutators the registry
// exposes, without a hardcoded count or name list (which used to omit
// "bitwise").
func TestNew(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	want := SupportedMutators()
	if len(engine.GetMutators()) != len(want) {
		t.Errorf("Expected %d mutators, got %d", len(want), len(engine.GetMutators()))
	}

	registered := make(map[string]bool, len(want))
	for _, m := range want {
		registered[m.Name()] = true
	}

	for _, m := range engine.GetMutators() {
		if !registered[m.Name()] {
			t.Errorf("engine mutator %q not found in SupportedMutators()", m.Name())
		}
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

	engine, err := New()
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

	expectedTypes := []string{arithmeticBinaryType, conditionalBinaryType, logicalBinaryType}
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

	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(testFile)
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	// Mutation limits are no longer supported - just check that we got some mutants
	if len(mutants) == 0 {
		t.Error("Expected to generate some mutants")
	}
}

func TestGenerateMutants_InvalidFile(t *testing.T) {
	engine, err := New()
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

	engine, err := New()
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

func NoMutations() {
}
`

	err := os.WriteFile(testFile, []byte(noMutationCode), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	engine, err := New()
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

func TestGetFileSet(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	fset := engine.GetFileSet()
	if fset == nil {
		t.Error("Expected FileSet to be non-nil")
	}
}

func TestMutantIDGeneration(t *testing.T) {
	// Create temporary Go file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package main

func Add(a, b int) int {
	return a + b
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(testFile)
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	// Check that all IDs are unique and follow expected format
	seenIDs := make(map[string]bool)
	for _, mutant := range mutants {
		if seenIDs[mutant.ID] {
			t.Errorf("Duplicate mutant ID found: %s", mutant.ID)
		}

		seenIDs[mutant.ID] = true

		// ID should start with file path
		if !strings.HasPrefix(mutant.ID, testFile) {
			t.Errorf("Mutant ID should start with file path, got: %s", mutant.ID)
		}
	}
}
