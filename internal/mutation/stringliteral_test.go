package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseStringLit(t *testing.T, code string) ast.Node {
	t.Helper()

	expr, err := parser.ParseExpr(code)
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	var lit ast.Node

	ast.Inspect(expr, func(n ast.Node) bool {
		if bl, ok := n.(*ast.BasicLit); ok {
			lit = bl

			return false
		}

		return true
	})

	if lit == nil {
		return expr
	}

	return lit
}

func TestStringLiteralMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &StringLiteralMutator{}
	if mutator.Name() != stringLiteralMutatorName {
		t.Errorf("Expected name %q, got %s", stringLiteralMutatorName, mutator.Name())
	}
}

func TestStringLiteralMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &StringLiteralMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{name: "interpreted string", code: `"hello"`, expected: true},
		{name: "empty string", code: `""`, expected: true},
		{name: "raw string", code: "`raw`", expected: true},
		{name: "int literal", code: "10", expected: false},
		{name: "char literal", code: "'a'", expected: false},
		{name: "identifier", code: "x", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := parseStringLit(t, tt.code)

			if canMutate := mutator.CanMutate(node); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestStringLiteralMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &StringLiteralMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name        string
		code        string
		wantMutated string
	}{
		{name: "non-empty to empty", code: `"hello"`, wantMutated: `""`},
		{name: "empty to non-empty", code: `""`, wantMutated: `"mutated"`},
		{name: "raw to empty", code: "`raw`", wantMutated: `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := parseStringLit(t, tt.code)

			mutants := mutator.Mutate(node, fset)

			if len(mutants) != 1 {
				t.Fatalf("Expected 1 mutant, got %d", len(mutants))
			}

			m := mutants[0]

			if m.Type != stringLiteralType {
				t.Errorf("Type = %q, want %q", m.Type, stringLiteralType)
			}

			if m.Original != tt.code {
				t.Errorf("Original = %q, want %q", m.Original, tt.code)
			}

			if m.Mutated != tt.wantMutated {
				t.Errorf("Mutated = %q, want %q", m.Mutated, tt.wantMutated)
			}

			if m.Description == "" {
				t.Error("Expected non-empty description")
			}
		})
	}
}

func TestStringLiteralMutator_Mutate_NonString(t *testing.T) {
	t.Parallel()

	mutator := &StringLiteralMutator{}
	fset := token.NewFileSet()

	node := parseStringLit(t, "10")

	if mutants := mutator.Mutate(node, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for non-string literal", mutants)
	}
}

func TestStringLiteralMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		mutantType string
		original   string
		mutated    string
		expected   bool
		wantValue  string
	}{
		{
			name:       "non-empty to empty",
			code:       `"hello"`,
			mutantType: stringLiteralType,
			original:   `"hello"`,
			mutated:    `""`,
			expected:   true,
			wantValue:  `""`,
		},
		{
			name:       "empty to non-empty",
			code:       `""`,
			mutantType: stringLiteralType,
			original:   `""`,
			mutated:    `"mutated"`,
			expected:   true,
			wantValue:  `"mutated"`,
		},
		{
			name:       "wrong mutant type",
			code:       `"hello"`,
			mutantType: "unknown",
			original:   `"hello"`,
			mutated:    `""`,
			expected:   false,
			wantValue:  `"hello"`,
		},
		{
			name:       "wrong original",
			code:       `"hello"`,
			mutantType: stringLiteralType,
			original:   `"world"`,
			mutated:    `""`,
			expected:   false,
			wantValue:  `"hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &StringLiteralMutator{}
			node := parseStringLit(t, tt.code)

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: tt.original,
				Mutated:  tt.mutated,
			}

			if result := mutator.Apply(node, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			lit, ok := node.(*ast.BasicLit)
			if !ok {
				t.Fatalf("expected *ast.BasicLit, got %T", node)
			}

			if lit.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", lit.Value, tt.wantValue)
			}
		})
	}
}

func TestStringLiteralMutator_Apply_NonLit(t *testing.T) {
	t.Parallel()

	mutator := &StringLiteralMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{Type: stringLiteralType, Original: `"x"`, Mutated: `""`}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-BasicLit node")
	}
}
