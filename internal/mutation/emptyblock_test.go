package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseBlockStmt(t *testing.T, fset *token.FileSet, body string) ast.Node {
	t.Helper()

	src := "package main\nfunc test() {\n" + body + "\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var block ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if bs, ok := n.(*ast.BlockStmt); ok {
			block = bs

			return false
		}

		return true
	})

	if block == nil {
		t.Fatalf("BlockStmt not found in: %s", body)
	}

	return block
}

func TestEmptyBlockMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &EmptyBlockMutator{}
	if mutator.Name() != emptyBlockMutatorName {
		t.Errorf("Expected name %q, got %s", emptyBlockMutatorName, mutator.Name())
	}
}

func TestEmptyBlockMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &EmptyBlockMutator{}

	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{name: "non-empty block", body: "\tx := 1\n\t_ = x", expected: true},
		{name: "empty block", body: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			block := parseBlockStmt(t, fset, tt.body)

			if canMutate := mutator.CanMutate(block); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestEmptyBlockMutator_CanMutate_NonBlock(t *testing.T) {
	t.Parallel()

	mutator := &EmptyBlockMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-BlockStmt node")
	}
}

func TestEmptyBlockMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &EmptyBlockMutator{}
	fset := token.NewFileSet()
	block := parseBlockStmt(t, fset, "\tx := 1\n\t_ = x")

	mutants := mutator.Mutate(block, fset)

	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant, got %d", len(mutants))
	}

	m := mutants[0]

	if m.Type != emptyBlockType {
		t.Errorf("Type = %q, want %q", m.Type, emptyBlockType)
	}

	if m.Mutated != emptyBlockMutated {
		t.Errorf("Mutated = %q, want %q", m.Mutated, emptyBlockMutated)
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestEmptyBlockMutator_Mutate_Empty(t *testing.T) {
	t.Parallel()

	mutator := &EmptyBlockMutator{}
	fset := token.NewFileSet()
	block := parseBlockStmt(t, fset, "")

	if mutants := mutator.Mutate(block, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for empty block", mutants)
	}
}

func TestEmptyBlockMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		mutantType string
		expected   bool
		wantLen    int
	}{
		{
			name:       "empties non-empty block",
			body:       "\tx := 1\n\t_ = x",
			mutantType: emptyBlockType,
			expected:   true,
			wantLen:    0,
		},
		{
			name:       "wrong mutant type",
			body:       "\tx := 1\n\t_ = x",
			mutantType: "unknown",
			expected:   false,
			wantLen:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &EmptyBlockMutator{}
			fset := token.NewFileSet()
			block := parseBlockStmt(t, fset, tt.body)

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: emptyBlockOriginal,
				Mutated:  emptyBlockMutated,
			}

			if result := mutator.Apply(block, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			bs, ok := block.(*ast.BlockStmt)
			if !ok {
				t.Fatalf("expected *ast.BlockStmt, got %T", block)
			}

			if len(bs.List) != tt.wantLen {
				t.Errorf("len(List) = %d, want %d", len(bs.List), tt.wantLen)
			}
		})
	}
}

func TestEmptyBlockMutator_Apply_NonBlockNode(t *testing.T) {
	t.Parallel()

	mutator := &EmptyBlockMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{Type: emptyBlockType, Original: emptyBlockOriginal, Mutated: emptyBlockMutated}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-BlockStmt node")
	}
}
