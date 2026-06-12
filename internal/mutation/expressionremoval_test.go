package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseExprStmt(t *testing.T, fset *token.FileSet, code string) ast.Node {
	t.Helper()

	src := "package main\nfunc test() {\n\t" + code + "\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var stmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if es, ok := n.(*ast.ExprStmt); ok {
			stmt = es

			return false
		}

		return true
	})

	if stmt == nil {
		t.Fatalf("ExprStmt not found in: %s", code)
	}

	return stmt
}

func TestExpressionRemovalMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}
	if mutator.Name() != expressionRemovalMutatorName {
		t.Errorf("Expected name %q, got %s", expressionRemovalMutatorName, mutator.Name())
	}
}

func TestExpressionRemovalMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}

	fset := token.NewFileSet()
	stmt := parseExprStmt(t, fset, "doWork()")

	if !mutator.CanMutate(stmt) {
		t.Error("CanMutate() = false, want true for ExprStmt")
	}
}

func TestExpressionRemovalMutator_CanMutate_NonExprStmt(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-ExprStmt node")
	}
}

func TestExpressionRemovalMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}
	fset := token.NewFileSet()
	stmt := parseExprStmt(t, fset, "doWork(x)")

	mutants := mutator.Mutate(stmt, fset)

	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant, got %d", len(mutants))
	}

	m := mutants[0]

	if m.Type != expressionRemovalType {
		t.Errorf("Type = %q, want %q", m.Type, expressionRemovalType)
	}

	if m.Original != "doWork(x)" {
		t.Errorf("Original = %q, want %q", m.Original, "doWork(x)")
	}

	if m.Mutated != expressionRemovalMutated {
		t.Errorf("Mutated = %q, want %q", m.Mutated, expressionRemovalMutated)
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestExpressionRemovalMutator_Mutate_NonExprStmt(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}
	fset := token.NewFileSet()

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutants := mutator.Mutate(expr, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for non-ExprStmt node", mutants)
	}
}

func TestExpressionRemovalMutator_Apply_AlwaysFalse(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}
	fset := token.NewFileSet()
	stmt := parseExprStmt(t, fset, "doWork()")

	mutant := Mutant{Type: expressionRemovalType, Original: "doWork()", Mutated: expressionRemovalMutated}

	if mutator.Apply(stmt, mutant) {
		t.Error("Apply() = true, want false (removal requires cursor)")
	}
}

func TestExpressionRemovalMutator_ApplyWithCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        string
		mutantType  string
		expected    bool
		wantReplace bool
	}{
		{
			name:        "removes expression statement",
			code:        "doWork()",
			mutantType:  expressionRemovalType,
			expected:    true,
			wantReplace: true,
		},
		{
			name:        "wrong mutant type",
			code:        "doWork()",
			mutantType:  "unknown",
			expected:    false,
			wantReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &ExpressionRemovalMutator{}
			fset := token.NewFileSet()
			stmt := parseExprStmt(t, fset, tt.code)

			replaced := false

			var replacement ast.Node

			replaceFunc := func(n ast.Node) {
				replaced = true
				replacement = n
			}

			mutant := Mutant{Type: tt.mutantType, Original: "doWork()", Mutated: expressionRemovalMutated}

			if result := mutator.ApplyWithCursor(stmt, replaceFunc, mutant); result != tt.expected {
				t.Errorf("ApplyWithCursor() = %v, expected %v", result, tt.expected)
			}

			if replaced != tt.wantReplace {
				t.Errorf("replaceFunc called = %v, want %v", replaced, tt.wantReplace)
			}

			if tt.wantReplace {
				if _, ok := replacement.(*ast.EmptyStmt); !ok {
					t.Errorf("replacement = %T, want *ast.EmptyStmt", replacement)
				}
			}
		})
	}
}

func TestExpressionRemovalMutator_ApplyWithCursor_NonExprStmt(t *testing.T) {
	t.Parallel()

	mutator := &ExpressionRemovalMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	called := false
	mutant := Mutant{Type: expressionRemovalType, Original: "doWork()", Mutated: expressionRemovalMutated}

	if mutator.ApplyWithCursor(expr, func(ast.Node) { called = true }, mutant) {
		t.Error("ApplyWithCursor() = true, want false for non-ExprStmt node")
	}

	if called {
		t.Error("replaceFunc should not be called for non-ExprStmt node")
	}
}
