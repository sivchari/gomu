package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseBranchStmt(t *testing.T, fset *token.FileSet, body string) ast.Node {
	t.Helper()

	src := "package main\nfunc test() {\n\tfor {\n\t\t" + body + "\n\t}\n}"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var stmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if bs, ok := n.(*ast.BranchStmt); ok {
			stmt = bs

			return false
		}

		return true
	})

	if stmt == nil {
		t.Fatalf("BranchStmt not found in: %s", body)
	}

	return stmt
}

func TestBreakContinueMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &BreakContinueMutator{}
	if mutator.Name() != breakContinueMutatorName {
		t.Errorf("Expected name %q, got %s", breakContinueMutatorName, mutator.Name())
	}
}

func TestBreakContinueMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &BreakContinueMutator{}

	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{name: "break", body: "break", expected: true},
		{name: "continue", body: "continue", expected: true},
		{name: "labeled break", body: "break Loop", expected: false},
		{name: "labeled continue", body: "continue Loop", expected: false},
		{name: "goto", body: "goto Loop", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			stmt := parseBranchStmt(t, fset, tt.body)

			if canMutate := mutator.CanMutate(stmt); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestBreakContinueMutator_CanMutate_NonBranch(t *testing.T) {
	t.Parallel()

	mutator := &BreakContinueMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-BranchStmt node")
	}
}

func TestBreakContinueMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &BreakContinueMutator{}

	tests := []struct {
		name         string
		body         string
		wantOriginal string
		wantMutated  string
	}{
		{name: "break to continue", body: "break", wantOriginal: "break", wantMutated: "continue"},
		{name: "continue to break", body: "continue", wantOriginal: "continue", wantMutated: "break"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			stmt := parseBranchStmt(t, fset, tt.body)

			mutants := mutator.Mutate(stmt, fset)

			if len(mutants) != 1 {
				t.Fatalf("Expected 1 mutant, got %d", len(mutants))
			}

			m := mutants[0]

			if m.Type != breakContinueType {
				t.Errorf("Type = %q, want %q", m.Type, breakContinueType)
			}

			if m.Original != tt.wantOriginal {
				t.Errorf("Original = %q, want %q", m.Original, tt.wantOriginal)
			}

			if m.Mutated != tt.wantMutated {
				t.Errorf("Mutated = %q, want %q", m.Mutated, tt.wantMutated)
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

func TestBreakContinueMutator_Mutate_Labeled(t *testing.T) {
	t.Parallel()

	mutator := &BreakContinueMutator{}
	fset := token.NewFileSet()
	stmt := parseBranchStmt(t, fset, "break Loop")

	if mutants := mutator.Mutate(stmt, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for labeled branch statement", mutants)
	}
}

func TestBreakContinueMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		mutantType string
		original   string
		expected   bool
		wantTok    token.Token
	}{
		{
			name:       "apply break to continue",
			body:       "break",
			mutantType: breakContinueType,
			original:   "break",
			expected:   true,
			wantTok:    token.CONTINUE,
		},
		{
			name:       "apply continue to break",
			body:       "continue",
			mutantType: breakContinueType,
			original:   "continue",
			expected:   true,
			wantTok:    token.BREAK,
		},
		{
			name:       "wrong mutant type",
			body:       "break",
			mutantType: "unknown",
			original:   "break",
			expected:   false,
			wantTok:    token.BREAK,
		},
		{
			name:       "wrong original",
			body:       "break",
			mutantType: breakContinueType,
			original:   "continue",
			expected:   false,
			wantTok:    token.BREAK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &BreakContinueMutator{}
			fset := token.NewFileSet()
			stmt := parseBranchStmt(t, fset, tt.body)

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: tt.original,
			}

			if result := mutator.Apply(stmt, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			bs, ok := stmt.(*ast.BranchStmt)
			if !ok {
				t.Fatalf("expected *ast.BranchStmt, got %T", stmt)
			}

			if bs.Tok != tt.wantTok {
				t.Errorf("Tok = %v, want %v", bs.Tok, tt.wantTok)
			}
		})
	}
}

func TestBreakContinueMutator_Apply_NonBranchNode(t *testing.T) {
	t.Parallel()

	mutator := &BreakContinueMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{Type: breakContinueType, Original: "break"}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-BranchStmt node")
	}
}
