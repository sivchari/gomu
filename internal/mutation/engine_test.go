package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/analysis"
)

// logTypeFromUsesMap logs type info from Uses map for a binary expression's left operand.
func logTypeFromUsesMap(t *testing.T, fileInfo *analysis.FileInfo, be *ast.BinaryExpr, pos token.Position) {
	t.Helper()

	if ident, ok := be.X.(*ast.Ident); ok {
		logTypeFromIdent(t, fileInfo, be, pos, ident)

		return
	}

	if sel, ok := be.X.(*ast.SelectorExpr); ok {
		logTypeFromSelector(t, fileInfo, be, pos, sel)

		return
	}

	t.Logf("Line %d: %T NO type info", pos.Line, be.X)
}

// logTypeFromIdent logs type info for an identifier from Uses map.
func logTypeFromIdent(t *testing.T, fileInfo *analysis.FileInfo, be *ast.BinaryExpr, pos token.Position, ident *ast.Ident) {
	t.Helper()

	obj := fileInfo.TypeInfo.Uses[ident]
	if obj == nil {
		t.Logf("Line %d: %T NO type info in Types or Uses", pos.Line, be.X)

		return
	}

	underlying := obj.Type().Underlying()
	_, isInterface := underlying.(*types.Interface)
	t.Logf("Line %d: %T left type (Uses): %v (interface=%v)", pos.Line, be.X, obj.Type(), isInterface)
}

// logTypeFromSelector logs type info for a selector expression from Uses map.
func logTypeFromSelector(t *testing.T, fileInfo *analysis.FileInfo, be *ast.BinaryExpr, pos token.Position, sel *ast.SelectorExpr) {
	t.Helper()

	obj := fileInfo.TypeInfo.Uses[sel.Sel]
	if obj == nil {
		t.Logf("Line %d: %T NO type info in Types or Uses", pos.Line, be.X)

		return
	}

	underlying := obj.Type().Underlying()
	_, isInterface := underlying.(*types.Interface)
	t.Logf("Line %d: %T left type (Uses.Sel): %v (interface=%v)", pos.Line, be.X, obj.Type(), isInterface)
}

func TestNew(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}

	if len(engine.mutators) != 7 {
		t.Errorf("Expected 7 mutators, got %d", len(engine.mutators))
	}

	// Check mutator types
	mutatorNames := make(map[string]bool)

	for _, mutator := range engine.mutators {
		mutatorNames[mutator.Name()] = true
	}

	expectedMutators := []string{"arithmetic", "branch", "conditional", "error_handling", "logical", "return"}

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

	if len(engine.mutators) != 7 {
		t.Errorf("Expected 7 mutators (all types enabled by default), got %d", len(engine.mutators))
	}
}

func TestNew_InvalidMutator(t *testing.T) {
	// All mutation types are now enabled by default
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	// Should ignore invalid mutator
	if len(engine.mutators) != 7 {
		t.Errorf("Expected 7 mutators (all types enabled by default), got %d", len(engine.mutators))
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

func TestGenerateMutants_InterfaceNilFiltering(t *testing.T) {
	// Test that interface != nil comparisons don't generate <, <=, >, >= mutations
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testCode := `package test

var errGlobal error

func Check() bool {
	return errGlobal != nil
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

	// Log all mutants for debugging
	for _, m := range mutants {
		t.Logf("Mutant: %s -> %s (type: %s)", m.Original, m.Mutated, m.Type)
	}

	// Check that no ordered comparisons were generated for interface type
	for _, m := range mutants {
		if m.Type == conditionalBinaryType {
			if m.Mutated == "<" || m.Mutated == "<=" || m.Mutated == ">" || m.Mutated == ">=" {
				t.Errorf("Should not generate ordered comparison mutation %s for interface type", m.Mutated)
			}
		}
	}

	// Should only have == mutation (from != original)
	conditionalCount := 0

	for _, m := range mutants {
		if m.Type == conditionalBinaryType {
			conditionalCount++
		}
	}

	if conditionalCount != 1 {
		t.Errorf("Expected 1 conditional mutation (== only), got %d", conditionalCount)
	}
}

func TestGenerateMutants_RealEngineFile(t *testing.T) {
	// Test on the real engine.go file to verify type filtering works
	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	// Use the actual engine.go file
	mutants, err := engine.GenerateMutants("engine.go")
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	t.Logf("Total mutants: %d", len(mutants))

	// Check for ordered comparisons on interface types (err != nil patterns)
	// These should be filtered out
	orderedOnInterface := 0

	for _, m := range mutants {
		if m.Type == "conditionalBinaryType" && m.Original == "!=" {
			if m.Mutated == "<" || m.Mutated == "<=" || m.Mutated == ">" || m.Mutated == ">=" {
				t.Logf("Found ordered comparison at line %d: %s -> %s", m.Line, m.Original, m.Mutated)

				orderedOnInterface++
			}
		}
	}

	if orderedOnInterface > 0 {
		t.Errorf("Found %d ordered comparison mutations that should have been filtered", orderedOnInterface)
	}
}

func TestTypeInfoDebug(t *testing.T) {
	// Debug test to investigate type info contents
	a, err := analysis.New()
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	fileInfo, err := a.ParseFile("engine.go")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if fileInfo.TypeInfo == nil {
		t.Fatal("TypeInfo is nil - this is why type filtering fails")
	}

	t.Logf("TypeInfo.Types has %d entries", len(fileInfo.TypeInfo.Types))
	t.Logf("TypeInfo.Uses has %d entries", len(fileInfo.TypeInfo.Uses))
	t.Logf("TypeInfo.Defs has %d entries", len(fileInfo.TypeInfo.Defs))

	// Find binary expressions and check their type info
	ast.Inspect(fileInfo.FileAST, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		// Check if left operand has type info
		tv, hasType := fileInfo.TypeInfo.Types[be.X]
		pos := a.GetFileSet().Position(be.Pos())

		if hasType {
			underlying := tv.Type.Underlying()
			_, isInterface := underlying.(*types.Interface)
			t.Logf("Line %d: %T left type (Types): %v (interface=%v)", pos.Line, be.X, tv.Type, isInterface)

			return true
		}

		// Try Uses map for Ident
		logTypeFromUsesMap(t, fileInfo, be, pos)

		return true
	})
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
				{arithmeticBinaryType, "-", true},
				{arithmeticBinaryType, "*", true},
				{arithmeticBinaryType, "/", true},
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
				{conditionalBinaryType, "!=", true},
				{conditionalBinaryType, "<", true},
				{conditionalBinaryType, ">", true},
				{conditionalBinaryType, "<=", true},
				{conditionalBinaryType, ">=", true},
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
				{logicalBinaryType, "||", true},
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

					// Find a node that can be mutated by this mutator and get its original operator
					var (
						targetNode ast.Node
						originalOp string
					)

					findMutableNode(parsedFile, func(node ast.Node) bool {
						if mutator.CanMutate(node) {
							targetNode = node
							// Get the original operator
							if expr, ok := node.(*ast.BinaryExpr); ok {
								originalOp = expr.Op.String()
							} else if stmt, ok := node.(*ast.AssignStmt); ok {
								originalOp = stmt.Tok.String()
							} else if stmt, ok := node.(*ast.IncDecStmt); ok {
								originalOp = stmt.Tok.String()
							}

							return false // stop searching
						}

						return true
					})

					if targetNode == nil {
						continue
					}

					// Create a mock mutant for testing Apply behavior
					testMutant := Mutant{
						Type:     mutationTest.mutantType,
						Original: originalOp,
						Mutated:  mutationTest.mutantValue,
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
	case strings.HasPrefix(mutantType, "branch_"):
		return "branch"
	case strings.HasPrefix(mutantType, "conditional_"):
		return "conditional"
	case strings.HasPrefix(mutantType, "error_"):
		return "error_handling"
	case strings.HasPrefix(mutantType, "logical_"):
		return "logical"
	case strings.HasPrefix(mutantType, "bitwise_"):
		return "bitwise"
	case strings.HasPrefix(mutantType, "return_"):
		return "return"
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
				// Parse file to get nodes for testing
				fset := engine.GetFileSet()

				parsedFile, err := parser.ParseFile(fset, testFile, nil, 0)
				if err != nil {
					continue // Skip on parse error
				}

				// Find a node that can be mutated and get its original operator
				var (
					targetNode ast.Node
					originalOp string
				)

				findMutableNode(parsedFile, func(node ast.Node) bool {
					if targetMutator.CanMutate(node) {
						targetNode = node
						// Get the original operator
						if expr, ok := node.(*ast.BinaryExpr); ok {
							originalOp = expr.Op.String()
						} else if stmt, ok := node.(*ast.AssignStmt); ok {
							originalOp = stmt.Tok.String()
						} else if stmt, ok := node.(*ast.IncDecStmt); ok {
							originalOp = stmt.Tok.String()
						}

						return false
					}

					return true
				})

				if targetNode != nil {
					testMutant := Mutant{
						Type:     tc.mutatorName + "_binary",
						Original: originalOp,
						Mutated:  tokenTest.input,
					}

					// Apply mutation - this exercises stringToToken
					result := targetMutator.Apply(targetNode, testMutant)

					// The result should match our expectation:
					// - Valid tokens should succeed (Apply returns true)
					// - Invalid tokens should fail (Apply returns false)
					// Note: Apply returns true even if mutating to the same operator
					if result != tokenTest.expected {
						t.Errorf("stringToToken integration test for %s with input %q (original=%q): Apply returned %v, expected %v",
							tc.mutatorName, tokenTest.input, originalOp, result, tokenTest.expected)
					}
				}
			}
		})
	}
}
