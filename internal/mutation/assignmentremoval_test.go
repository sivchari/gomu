package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseAssignStmtForRemoval(t *testing.T, fset *token.FileSet, code string) ast.Node {
	t.Helper()

	src := "package main\nfunc test() {\n\t" + code + "\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var stmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok {
			stmt = as

			return false
		}

		return true
	})

	if stmt == nil {
		t.Fatalf("AssignStmt not found in: %s", code)
	}

	return stmt
}

func TestAssignmentRemovalMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}
	if mutator.Name() != assignmentRemovalMutatorName {
		t.Errorf("Expected name %q, got %s", assignmentRemovalMutatorName, mutator.Name())
	}
}

func TestAssignmentRemovalMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{name: "simple assign", code: "a = b", expected: true},
		{name: "compound assign", code: "a += b", expected: true},
		{name: "multi assign", code: "a, b = b, a", expected: true},
		{name: "short var decl", code: "a := b", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			stmt := parseAssignStmtForRemoval(t, fset, tt.code)

			if canMutate := mutator.CanMutate(stmt); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestAssignmentRemovalMutator_CanMutate_NonAssign(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-AssignStmt node")
	}
}

func TestAssignmentRemovalMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}
	fset := token.NewFileSet()
	stmt := parseAssignStmtForRemoval(t, fset, "total = total + b")

	mutants := mutator.Mutate(stmt, fset)

	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant, got %d", len(mutants))
	}

	m := mutants[0]

	if m.Type != assignmentRemovalType {
		t.Errorf("Type = %q, want %q", m.Type, assignmentRemovalType)
	}

	if m.Original != "total" {
		t.Errorf("Original = %q, want %q", m.Original, "total")
	}

	if m.Mutated != assignmentRemovalMutated {
		t.Errorf("Mutated = %q, want %q", m.Mutated, assignmentRemovalMutated)
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestAssignmentRemovalMutator_Mutate_Define(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}
	fset := token.NewFileSet()
	stmt := parseAssignStmtForRemoval(t, fset, "a := b")

	if mutants := mutator.Mutate(stmt, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for short var decl", mutants)
	}
}

func TestAssignmentRemovalMutator_Apply_AlwaysFalse(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}
	fset := token.NewFileSet()
	stmt := parseAssignStmtForRemoval(t, fset, "a = b")

	mutant := Mutant{Type: assignmentRemovalType, Original: "a", Mutated: assignmentRemovalMutated}

	if mutator.Apply(stmt, mutant) {
		t.Error("Apply() = true, want false (removal requires cursor)")
	}
}

func TestAssignmentRemovalMutator_ApplyWithCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        string
		mutantType  string
		expected    bool
		wantReplace bool
	}{
		{
			name:        "removes simple assignment",
			code:        "a = b",
			mutantType:  assignmentRemovalType,
			expected:    true,
			wantReplace: true,
		},
		{
			name:        "wrong mutant type",
			code:        "a = b",
			mutantType:  "unknown",
			expected:    false,
			wantReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &AssignmentRemovalMutator{}
			fset := token.NewFileSet()
			stmt := parseAssignStmtForRemoval(t, fset, tt.code)

			replaced := false

			var replacement ast.Node

			replaceFunc := func(n ast.Node) {
				replaced = true
				replacement = n
			}

			mutant := Mutant{Type: tt.mutantType, Original: "a", Mutated: assignmentRemovalMutated}

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

func TestAssignmentRemovalMutator_ApplyWithCursor_NonAssign(t *testing.T) {
	t.Parallel()

	mutator := &AssignmentRemovalMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	called := false
	mutant := Mutant{Type: assignmentRemovalType, Original: "a", Mutated: assignmentRemovalMutated}

	if mutator.ApplyWithCursor(expr, func(ast.Node) { called = true }, mutant) {
		t.Error("ApplyWithCursor() = true, want false for non-AssignStmt node")
	}

	if called {
		t.Error("replaceFunc should not be called for non-AssignStmt node")
	}
}
