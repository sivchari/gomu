package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseStmt(t *testing.T, fset *token.FileSet, code string) ast.Node {
	t.Helper()

	src := "package main\nfunc test() {\n\t" + code + "\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var stmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IncDecStmt, *ast.DeferStmt, *ast.GoStmt, *ast.SendStmt:
			stmt = n

			return false
		}

		return true
	})

	if stmt == nil {
		t.Fatalf("removable statement not found in: %s", code)
	}

	return stmt
}

func TestStatementRemovalMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}
	if mutator.Name() != statementRemovalMutatorName {
		t.Errorf("Expected name %q, got %s", statementRemovalMutatorName, mutator.Name())
	}
}

func TestStatementRemovalMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}

	tests := []struct {
		name string
		code string
	}{
		{name: "increment", code: "i++"},
		{name: "decrement", code: "i--"},
		{name: "defer", code: "defer f()"},
		{name: "go", code: "go f()"},
		{name: "send", code: "ch <- 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			stmt := parseStmt(t, fset, tt.code)

			if !mutator.CanMutate(stmt) {
				t.Errorf("CanMutate() = false, want true for %s", tt.code)
			}
		})
	}
}

func TestStatementRemovalMutator_CanMutate_Excluded(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}
	fset := token.NewFileSet()

	// Assignment and expression statements are handled by other mutators.
	src := "package main\nfunc test() {\n\ta := 0\n\ta = 1\n\tf()\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.AssignStmt, *ast.ExprStmt:
			if mutator.CanMutate(n) {
				t.Errorf("CanMutate() = true, want false for %T", n)
			}
		}

		return true
	})
}

func TestStatementRemovalMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}

	tests := []struct {
		name         string
		code         string
		wantOriginal string
	}{
		{name: "increment", code: "i++", wantOriginal: "i++"},
		{name: "defer", code: "defer f()", wantOriginal: "defer f()"},
		{name: "go", code: "go f()", wantOriginal: "go f()"},
		{name: "send", code: "ch <- 1", wantOriginal: "ch <- 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			stmt := parseStmt(t, fset, tt.code)

			mutants := mutator.Mutate(stmt, fset)

			if len(mutants) != 1 {
				t.Fatalf("Expected 1 mutant, got %d", len(mutants))
			}

			m := mutants[0]

			if m.Type != statementRemovalType {
				t.Errorf("Type = %q, want %q", m.Type, statementRemovalType)
			}

			if m.Original != tt.wantOriginal {
				t.Errorf("Original = %q, want %q", m.Original, tt.wantOriginal)
			}

			if m.Mutated != statementRemovalMutated {
				t.Errorf("Mutated = %q, want %q", m.Mutated, statementRemovalMutated)
			}

			if m.Line <= 0 {
				t.Errorf("Expected positive line number, got %d", m.Line)
			}

			if m.Description == "" {
				t.Error("Expected non-empty description")
			}
		})
	}
}

func TestStatementRemovalMutator_Mutate_NonRemovable(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}
	fset := token.NewFileSet()

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutants := mutator.Mutate(expr, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for non-removable node", mutants)
	}
}

func TestStatementRemovalMutator_Apply_AlwaysFalse(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}
	fset := token.NewFileSet()
	stmt := parseStmt(t, fset, "i++")

	mutant := Mutant{Type: statementRemovalType, Original: "i++", Mutated: statementRemovalMutated}

	if mutator.Apply(stmt, mutant) {
		t.Error("Apply() = true, want false (removal requires cursor)")
	}
}

func TestStatementRemovalMutator_ApplyWithCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        string
		mutantType  string
		expected    bool
		wantReplace bool
	}{
		{
			name:        "removes increment statement",
			code:        "i++",
			mutantType:  statementRemovalType,
			expected:    true,
			wantReplace: true,
		},
		{
			name:        "removes defer statement",
			code:        "defer f()",
			mutantType:  statementRemovalType,
			expected:    true,
			wantReplace: true,
		},
		{
			name:        "wrong mutant type",
			code:        "i++",
			mutantType:  "unknown",
			expected:    false,
			wantReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &StatementRemovalMutator{}
			fset := token.NewFileSet()
			stmt := parseStmt(t, fset, tt.code)

			replaced := false

			var replacement ast.Node

			replaceFunc := func(n ast.Node) {
				replaced = true
				replacement = n
			}

			mutant := Mutant{Type: tt.mutantType, Original: tt.code, Mutated: statementRemovalMutated}

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

func TestStatementRemovalMutator_ApplyWithCursor_NonRemovable(t *testing.T) {
	t.Parallel()

	mutator := &StatementRemovalMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	called := false
	mutant := Mutant{Type: statementRemovalType, Original: "i++", Mutated: statementRemovalMutated}

	if mutator.ApplyWithCursor(expr, func(ast.Node) { called = true }, mutant) {
		t.Error("ApplyWithCursor() = true, want false for non-removable node")
	}

	if called {
		t.Error("replaceFunc should not be called for non-removable node")
	}
}
