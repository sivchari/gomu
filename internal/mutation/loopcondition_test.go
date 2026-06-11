package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseForStmt(t *testing.T, fset *token.FileSet, code string) ast.Node {
	t.Helper()

	src := "package main\nfunc test() {\n\t" + code + "\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var stmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if fs, ok := n.(*ast.ForStmt); ok {
			stmt = fs

			return false
		}

		return true
	})

	if stmt == nil {
		t.Fatalf("ForStmt not found in: %s", code)
	}

	return stmt
}

func TestLoopConditionMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &LoopConditionMutator{}
	if mutator.Name() != loopConditionMutatorName {
		t.Errorf("Expected name %q, got %s", loopConditionMutatorName, mutator.Name())
	}
}

func TestLoopConditionMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &LoopConditionMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{name: "for with condition", code: "for i := 0; i < n; i++ {\n}", expected: true},
		{name: "for with only condition", code: "for i < n {\n}", expected: true},
		{name: "infinite for", code: "for {\n}", expected: false},
		{name: "for with true literal", code: "for true {\n}", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			stmt := parseForStmt(t, fset, tt.code)

			if canMutate := mutator.CanMutate(stmt); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestLoopConditionMutator_CanMutate_NonFor(t *testing.T) {
	t.Parallel()

	mutator := &LoopConditionMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-ForStmt node")
	}
}

func TestLoopConditionMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &LoopConditionMutator{}
	fset := token.NewFileSet()
	stmt := parseForStmt(t, fset, "for i := 0; i < n; i++ {\n}")

	mutants := mutator.Mutate(stmt, fset)

	if len(mutants) != 2 {
		t.Fatalf("Expected 2 mutants, got %d", len(mutants))
	}

	mutatedValues := make(map[string]bool)

	for _, m := range mutants {
		mutatedValues[m.Mutated] = true

		if m.Type != loopConditionType {
			t.Errorf("Type = %q, want %q", m.Type, loopConditionType)
		}

		if m.Original != "i < n" {
			t.Errorf("Original = %q, want %q", m.Original, "i < n")
		}

		if m.Line <= 0 {
			t.Errorf("Expected positive line number, got %d", m.Line)
		}

		if m.Description == "" {
			t.Error("Expected non-empty description")
		}
	}

	for _, want := range []string{"true", "false"} {
		if !mutatedValues[want] {
			t.Errorf("Expected mutation to %q not found", want)
		}
	}
}

func TestLoopConditionMutator_Mutate_Infinite(t *testing.T) {
	t.Parallel()

	mutator := &LoopConditionMutator{}
	fset := token.NewFileSet()
	stmt := parseForStmt(t, fset, "for {\n}")

	if mutants := mutator.Mutate(stmt, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for infinite loop", mutants)
	}
}

func TestLoopConditionMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		mutantType string
		original   string
		mutated    string
		expected   bool
		wantCond   string
	}{
		{
			name:       "replace condition with false",
			code:       "for i := 0; i < n; i++ {\n}",
			mutantType: loopConditionType,
			original:   "i < n",
			mutated:    "false",
			expected:   true,
			wantCond:   "false",
		},
		{
			name:       "replace condition with true",
			code:       "for i < n {\n}",
			mutantType: loopConditionType,
			original:   "i < n",
			mutated:    "true",
			expected:   true,
			wantCond:   "true",
		},
		{
			name:       "wrong mutant type",
			code:       "for i < n {\n}",
			mutantType: "unknown",
			original:   "i < n",
			mutated:    "false",
			expected:   false,
			wantCond:   "i < n",
		},
		{
			name:       "wrong original",
			code:       "for i < n {\n}",
			mutantType: loopConditionType,
			original:   "i > n",
			mutated:    "false",
			expected:   false,
			wantCond:   "i < n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &LoopConditionMutator{}
			fset := token.NewFileSet()
			stmt := parseForStmt(t, fset, tt.code)

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: tt.original,
				Mutated:  tt.mutated,
			}

			if result := mutator.Apply(stmt, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			fs, ok := stmt.(*ast.ForStmt)
			if !ok {
				t.Fatalf("expected *ast.ForStmt, got %T", stmt)
			}

			if got := exprToString(fs.Cond); got != tt.wantCond {
				t.Errorf("Cond = %q, want %q", got, tt.wantCond)
			}
		})
	}
}

func TestLoopConditionMutator_Apply_NonForNode(t *testing.T) {
	t.Parallel()

	mutator := &LoopConditionMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{Type: loopConditionType, Original: "i < n", Mutated: "false"}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-ForStmt node")
	}
}
