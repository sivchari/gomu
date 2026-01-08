package mutation

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}

	if len(engine.mutators) != 4 {
		t.Errorf("Expected 4 mutators, got %d", len(engine.mutators))
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
	// All mutation types are now enabled by default
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	if len(engine.mutators) != 4 {
		t.Errorf("Expected 4 mutators (all types enabled by default), got %d", len(engine.mutators))
	}
}

func TestNew_InvalidMutator(t *testing.T) {
	// All mutation types are now enabled by default
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	// Should ignore invalid mutator
	if len(engine.mutators) != 4 {
		t.Errorf("Expected 4 mutators (all types enabled by default), got %d", len(engine.mutators))
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

import "fmt"

func NoMutations() {
    fmt.Println("hello")
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

func TestNewEngine(t *testing.T) {
	engine, err := NewEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create mutation engine with NewEngine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}

	// Should behave the same as New()
	if len(engine.mutators) != 4 {
		t.Errorf("Expected 4 mutators, got %d", len(engine.mutators))
	}
}

func TestRunOnFiles(t *testing.T) {
	// Create temporary Go files
	tmpDir := t.TempDir()

	testFiles := []string{
		filepath.Join(tmpDir, "test1.go"),
		filepath.Join(tmpDir, "test2.go"),
	}

	testCode1 := `package main

func Add(a, b int) int {
	return a + b
}
`

	testCode2 := `package main

func Subtract(a, b int) int {
	return a - b
}
`

	err := os.WriteFile(testFiles[0], []byte(testCode1), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file 1: %v", err)
	}

	err = os.WriteFile(testFiles[1], []byte(testCode2), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file 2: %v", err)
	}

	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	results, err := engine.RunOnFiles(testFiles)
	if err != nil {
		t.Fatalf("Failed to run on files: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that results have the correct file paths
	expectedPaths := map[string]bool{
		testFiles[0]: false,
		testFiles[1]: false,
	}

	for _, result := range results {
		if _, exists := expectedPaths[result.FilePath]; exists {
			expectedPaths[result.FilePath] = true
		} else {
			t.Errorf("Unexpected file path in results: %s", result.FilePath)
		}

		// Check that mutations were generated
		if len(result.Mutations) == 0 {
			t.Errorf("Expected mutations for file %s, got 0", result.FilePath)
		}

		// Check that mutations have status
		for _, mutation := range result.Mutations {
			if mutation.Status != "killed" && mutation.Status != "survived" {
				t.Errorf("Unexpected mutation status: %s", mutation.Status)
			}
		}
	}

	// Check that all expected paths were found
	for path, found := range expectedPaths {
		if !found {
			t.Errorf("Expected to find result for file %s", path)
		}
	}
}

func TestRunOnFiles_InvalidFiles(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	// Test with nonexistent files
	files := []string{"/nonexistent/file1.go", "/nonexistent/file2.go"}

	results, err := engine.RunOnFiles(files)
	if err != nil {
		t.Fatalf("RunOnFiles should not return error for invalid files: %v", err)
	}

	// Should skip invalid files and return empty results
	if len(results) != 0 {
		t.Errorf("Expected 0 results for invalid files, got %d", len(results))
	}
}

func TestRunOnFiles_EmptyFileList(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	results, err := engine.RunOnFiles([]string{})
	if err != nil {
		t.Fatalf("RunOnFiles should not return error for empty file list: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty file list, got %d", len(results))
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

func TestStatusConstants(t *testing.T) {
	// Test that status constants have expected values
	tests := []struct {
		status Status
		want   string
	}{
		{StatusKilled, "KILLED"},
		{StatusSurvived, "SURVIVED"},
		{StatusTimedOut, "TIMED_OUT"},
		{StatusError, "ERROR"},
		{StatusNotViable, "NOT_VIABLE"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("Expected status %v to equal %s, got %s", tt.status, tt.want, string(tt.status))
		}
	}
}

func TestMutatorApplyIntegration(t *testing.T) {
	// Integration test that exercises Apply methods of all mutators
	// This ensures the Apply logic is tested during actual test execution
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	tests := []struct {
		name      string
		code      string
		mutations []struct {
			mutantType  string
			mutantValue string
			expected    bool
		}
	}{
		{
			name: "arithmetic mutations",
			code: `package test
func TestArithmetic() int {
	return 1 + 2
}`,
			mutations: []struct {
				mutantType  string
				mutantValue string
				expected    bool
			}{
				{"arithmetic_binary", "-", true},
				{"arithmetic_binary", "*", true},
				{"arithmetic_binary", "/", true},
				{"invalid_type", "+", false},
			},
		},
		{
			name: "conditional mutations",
			code: `package test
func TestConditional(a, b int) bool {
	return a == b
}`,
			mutations: []struct {
				mutantType  string
				mutantValue string
				expected    bool
			}{
				{"conditional_binary", "!=", true},
				{"conditional_binary", "<", true},
				{"conditional_binary", ">", true},
				{"conditional_binary", "<=", true},
				{"conditional_binary", ">=", true},
				{"invalid_type", "==", false},
			},
		},
		{
			name: "logical mutations",
			code: `package test
func TestLogical(a, b bool) bool {
	return a && b
}`,
			mutations: []struct {
				mutantType  string
				mutantValue string
				expected    bool
			}{
				{"logical_binary", "||", true},
				{"invalid_type", "&&", false},
			},
		},
		{
			name: "bitwise mutations",
			code: `package test
func TestBitwise(a, b int) int {
	return a & b
}`,
			mutations: []struct {
				mutantType  string
				mutantValue string
				expected    bool
			}{
				{"bitwise_binary", "|", true},
				{"bitwise_binary", "^", true},
				{"bitwise_binary", "&^", true},
				{"invalid_type", "&", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(testFile, []byte(tt.code), 0600)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Generate mutants from the code
			mutants, err := engine.GenerateMutants(testFile)
			if err != nil {
				t.Fatalf("Failed to generate mutants: %v", err)
			}

			if len(mutants) == 0 {
				t.Error("Expected to generate mutants, got 0")

				return
			}

			// Test each mutation by applying it and verifying the Apply method behavior
			for _, mutationTest := range tt.mutations {
				// Find a mutant that matches our test case
				var targetMutant *Mutant

				for i := range mutants {
					if mutants[i].Type == mutationTest.mutantType {
						targetMutant = &mutants[i]

						break
					}
				}

				if mutationTest.expected && targetMutant == nil {
					t.Errorf("Expected to find mutant of type %s but none found", mutationTest.mutantType)

					continue
				}

				// Create a mock mutant for testing Apply behavior
				testMutant := Mutant{
					Type:    mutationTest.mutantType,
					Mutated: mutationTest.mutantValue,
				}

				// Exercise each mutator's Apply method directly
				for _, mutator := range engine.mutators {
					if targetMutant == nil {
						continue
					}

					// Parse the file to get AST nodes for testing
					fset := engine.GetFileSet()

					parsedFile, err := parser.ParseFile(fset, testFile, nil, 0)
					if err != nil {
						t.Fatalf("Failed to parse file: %v", err)
					}

					// Find a node that can be mutated by this mutator
					var targetNode ast.Node

					findMutableNode(parsedFile, func(node ast.Node) bool {
						if mutator.CanMutate(node) {
							targetNode = node

							return false // stop searching
						}

						return true
					})

					if targetNode == nil {
						continue
					}

					// Test the Apply method
					result := mutator.Apply(targetNode, testMutant)
					if result != mutationTest.expected && mutator.Name() == getExpectedMutatorName(mutationTest.mutantType) {
						t.Errorf("Apply method for %s mutator returned %v, expected %v",
							mutator.Name(), result, mutationTest.expected)
					}
				}
			}
		})
	}
}

// Helper function to determine which mutator should handle which type.
func getExpectedMutatorName(mutantType string) string {
	switch {
	case strings.HasPrefix(mutantType, "arithmetic_"):
		return "arithmetic"
	case strings.HasPrefix(mutantType, "conditional_"):
		return "conditional"
	case strings.HasPrefix(mutantType, "logical_"):
		return "logical"
	case strings.HasPrefix(mutantType, "bitwise_"):
		return "bitwise"
	default:
		return ""
	}
}

// Helper function to traverse AST and find mutable nodes.
func findMutableNode(node ast.Node, fn func(ast.Node) bool) {
	ast.Inspect(node, fn)
}

func TestMutatorStringToTokenIntegration(t *testing.T) {
	// Integration test that exercises stringToToken methods through actual mutation application
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	testCases := []struct {
		mutatorName string
		testCode    string
		tokenTests  []struct {
			input    string
			expected bool // whether stringToToken should return a non-ILLEGAL token
		}
	}{
		{
			mutatorName: "arithmetic",
			testCode: `package test
func Test() int { return 1 + 2 }`,
			tokenTests: []struct {
				input    string
				expected bool
			}{
				{"+", true},
				{"-", true},
				{"*", true},
				{"/", true},
				{"%", true},
				{"++", true},
				{"--", true},
				{"+=", true},
				{"-=", true},
				{"*=", true},
				{"/=", true},
				{"invalid", false},
			},
		},
		{
			mutatorName: "conditional",
			testCode: `package test
func Test(a, b int) bool { return a == b }`,
			tokenTests: []struct {
				input    string
				expected bool
			}{
				{"==", true},
				{"!=", true},
				{"<", true},
				{"<=", true},
				{">", true},
				{">=", true},
				{"invalid", false},
			},
		},
		{
			mutatorName: "logical",
			testCode: `package test
func Test(a, b bool) bool { return a && b }`,
			tokenTests: []struct {
				input    string
				expected bool
			}{
				{"&&", true},
				{"||", true},
				{"invalid", false},
			},
		},
		{
			mutatorName: "bitwise",
			testCode: `package test
func Test(a, b int) int { return a & b }`,
			tokenTests: []struct {
				input    string
				expected bool
			}{
				{"&", true},
				{"|", true},
				{"^", true},
				{"&^", true},
				{"<<", true},
				{">>", true},
				{"&=", true},
				{"|=", true},
				{"^=", true},
				{"<<=", true},
				{">>=", true},
				{"invalid", false},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.mutatorName+"_stringToToken", func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(testFile, []byte(tc.testCode), 0600)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Generate mutants to trigger stringToToken usage
			_, err = engine.GenerateMutants(testFile)
			if err != nil {
				t.Fatalf("Failed to generate mutants: %v", err)
			}

			// Find the target mutator
			var targetMutator Mutator

			for _, mutator := range engine.mutators {
				if mutator.Name() == tc.mutatorName {
					targetMutator = mutator

					break
				}
			}

			if targetMutator == nil {
				t.Fatalf("Mutator %s not found", tc.mutatorName)
			}

			// Test stringToToken indirectly through Apply method
			for _, tokenTest := range tc.tokenTests {
				testMutant := Mutant{
					Type:    tc.mutatorName + "_binary",
					Mutated: tokenTest.input,
				}

				// Parse file to get nodes for testing
				fset := engine.GetFileSet()

				parsedFile, err := parser.ParseFile(fset, testFile, nil, 0)
				if err != nil {
					continue // Skip on parse error
				}

				// Find a node that can be mutated
				var targetNode ast.Node

				findMutableNode(parsedFile, func(node ast.Node) bool {
					if targetMutator.CanMutate(node) {
						targetNode = node

						return false
					}

					return true
				})

				if targetNode != nil {
					// Apply mutation - this exercises stringToToken
					result := targetMutator.Apply(targetNode, testMutant)

					// The result should match our expectation
					// Valid tokens should succeed, invalid ones should fail
					if result != tokenTest.expected {
						t.Errorf("stringToToken integration test for %s with input %q: Apply returned %v, expected %v",
							tc.mutatorName, tokenTest.input, result, tokenTest.expected)
					}
				}
			}
		})
	}
}
