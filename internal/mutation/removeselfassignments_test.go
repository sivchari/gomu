package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestRemoveSelfAssignmentsMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &RemoveSelfAssignmentsMutator{}
	if mutator.Name() != removeSelfAssignmentsMutatorName {
		t.Errorf("Expected name %q, got %s", removeSelfAssignmentsMutatorName, mutator.Name())
	}
}

func TestRemoveSelfAssignmentsMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &RemoveSelfAssignmentsMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{name: "add assign", code: "a += b", expected: true},
		{name: "sub assign", code: "a -= b", expected: true},
		{name: "mul assign", code: "a *= b", expected: true},
		{name: "quo assign", code: "a /= b", expected: true},
		{name: "rem assign", code: "a %= b", expected: true},
		{name: "and assign", code: "a &= b", expected: true},
		{name: "or assign", code: "a |= b", expected: true},
		{name: "xor assign", code: "a ^= b", expected: true},
		{name: "shl assign", code: "a <<= b", expected: true},
		{name: "shr assign", code: "a >>= b", expected: true},
		{name: "and not assign", code: "a &^= b", expected: true},
		{name: "simple assign", code: "a = b", expected: false},
		{name: "define", code: "a := b", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stmt := parseAssignStmt(t, tt.code)

			if canMutate := mutator.CanMutate(stmt); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestRemoveSelfAssignmentsMutator_CanMutate_NonAssign(t *testing.T) {
	t.Parallel()

	mutator := &RemoveSelfAssignmentsMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-AssignStmt node")
	}
}

func TestRemoveSelfAssignmentsMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &RemoveSelfAssignmentsMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name string
		code string
		op   string
	}{
		{name: "add assign", code: "a += b", op: "+="},
		{name: "and not assign", code: "a &^= b", op: "&^="},
		{name: "shl assign", code: "a <<= b", op: "<<="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stmt := parseAssignStmtWithFset(t, fset, tt.code)

			mutants := mutator.Mutate(stmt, fset)

			if len(mutants) != 1 {
				t.Fatalf("Expected 1 mutant, got %d", len(mutants))
			}

			m := mutants[0]

			if m.Type != removeSelfAssignmentsType {
				t.Errorf("Type = %q, want %q", m.Type, removeSelfAssignmentsType)
			}

			if m.Original != tt.op {
				t.Errorf("Original = %q, want %q", m.Original, tt.op)
			}

			if m.Mutated != "=" {
				t.Errorf("Mutated = %q, want %q", m.Mutated, "=")
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

func TestRemoveSelfAssignmentsMutator_Mutate_NonCompound(t *testing.T) {
	t.Parallel()

	mutator := &RemoveSelfAssignmentsMutator{}
	fset := token.NewFileSet()

	stmt := parseAssignStmtWithFset(t, fset, "a = b")

	if mutants := mutator.Mutate(stmt, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for simple assignment", mutants)
	}
}

func TestRemoveSelfAssignmentsMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		mutantType string
		original   string
		mutated    string
		expected   bool
		wantTok    token.Token
	}{
		{
			name:       "apply add assign to simple assign",
			code:       "a += b",
			mutantType: removeSelfAssignmentsType,
			original:   "+=",
			mutated:    "=",
			expected:   true,
			wantTok:    token.ASSIGN,
		},
		{
			name:       "apply and not assign to simple assign",
			code:       "a &^= b",
			mutantType: removeSelfAssignmentsType,
			original:   "&^=",
			mutated:    "=",
			expected:   true,
			wantTok:    token.ASSIGN,
		},
		{
			name:       "wrong mutant type",
			code:       "a += b",
			mutantType: "unknown",
			original:   "+=",
			mutated:    "=",
			expected:   false,
			wantTok:    token.ADD_ASSIGN,
		},
		{
			name:       "wrong original operator",
			code:       "a += b",
			mutantType: removeSelfAssignmentsType,
			original:   "-=",
			mutated:    "=",
			expected:   false,
			wantTok:    token.ADD_ASSIGN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &RemoveSelfAssignmentsMutator{}
			fset := token.NewFileSet()
			stmt := parseAssignStmtWithFset(t, fset, tt.code)

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: tt.original,
				Mutated:  tt.mutated,
			}

			if result := mutator.Apply(stmt, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			as, ok := stmt.(*ast.AssignStmt)
			if !ok {
				t.Fatalf("expected *ast.AssignStmt, got %T", stmt)
			}

			if as.Tok != tt.wantTok {
				t.Errorf("Tok = %v, want %v", as.Tok, tt.wantTok)
			}
		})
	}
}

func TestRemoveSelfAssignmentsMutator_Apply_NonAssignNode(t *testing.T) {
	t.Parallel()

	mutator := &RemoveSelfAssignmentsMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:     removeSelfAssignmentsType,
		Original: "+=",
		Mutated:  "=",
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-AssignStmt node")
	}
}

// parseAssignStmt parses a single assignment statement and returns its AST node.
func parseAssignStmt(t *testing.T, code string) ast.Node {
	t.Helper()

	return parseAssignStmtWithFset(t, token.NewFileSet(), code)
}

// parseAssignStmtWithFset parses a single assignment statement using the given file set.
// Only syntax is checked, so undeclared identifiers in code are fine.
func parseAssignStmtWithFset(t *testing.T, fset *token.FileSet, code string) ast.Node {
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
