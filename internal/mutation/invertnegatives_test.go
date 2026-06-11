package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestInvertNegativesMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &InvertNegativesMutator{}
	if mutator.Name() != invertNegativesMutatorName {
		t.Errorf("Expected name %q, got %s", invertNegativesMutatorName, mutator.Name())
	}
}

func TestInvertNegativesMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &InvertNegativesMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "unary minus on identifier",
			code:     "-a",
			expected: true,
		},
		{
			name:     "unary minus on literal",
			code:     "-5",
			expected: true,
		},
		{
			name:     "unary plus",
			code:     "+a",
			expected: false,
		},
		{
			name:     "logical not",
			code:     "!a",
			expected: false,
		},
		{
			name:     "binary subtraction",
			code:     "a - b",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expr, err := parser.ParseExpr(tt.code)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			if canMutate := mutator.CanMutate(expr); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestInvertNegativesMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &InvertNegativesMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc test() { _ = -a }"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var expr ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if ue, ok := n.(*ast.UnaryExpr); ok {
			expr = ue

			return false
		}

		return true
	})

	if expr == nil {
		t.Fatal("UnaryExpr not found")
	}

	mutants := mutator.Mutate(expr, fset)

	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant, got %d", len(mutants))
	}

	m := mutants[0]

	if m.Type != invertNegativesType {
		t.Errorf("Type = %q, want %q", m.Type, invertNegativesType)
	}

	if m.Original != "-" {
		t.Errorf("Original = %q, want %q", m.Original, "-")
	}

	if m.Mutated != "+" {
		t.Errorf("Mutated = %q, want %q", m.Mutated, "+")
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestInvertNegativesMutator_Mutate_NonUnaryMinus(t *testing.T) {
	t.Parallel()

	mutator := &InvertNegativesMutator{}
	fset := token.NewFileSet()

	expr, err := parser.ParseExpr("!a")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutants := mutator.Mutate(expr, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for non unary-minus node", mutants)
	}
}

func TestInvertNegativesMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		mutantType string
		original   string
		mutated    string
		expected   bool
		wantOp     token.Token
	}{
		{
			name:       "apply unary minus to plus",
			code:       "-a",
			mutantType: invertNegativesType,
			original:   "-",
			mutated:    "+",
			expected:   true,
			wantOp:     token.ADD,
		},
		{
			name:       "wrong mutant type",
			code:       "-a",
			mutantType: "unknown",
			original:   "-",
			mutated:    "+",
			expected:   false,
			wantOp:     token.SUB,
		},
		{
			name:       "wrong original operator",
			code:       "-a",
			mutantType: invertNegativesType,
			original:   "+",
			mutated:    "+",
			expected:   false,
			wantOp:     token.SUB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &InvertNegativesMutator{}

			expr, err := parser.ParseExpr(tt.code)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: tt.original,
				Mutated:  tt.mutated,
			}

			if result := mutator.Apply(expr, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			ue, ok := expr.(*ast.UnaryExpr)
			if !ok {
				t.Fatalf("expected *ast.UnaryExpr, got %T", expr)
			}

			if ue.Op != tt.wantOp {
				t.Errorf("Op = %v, want %v", ue.Op, tt.wantOp)
			}
		})
	}
}

func TestInvertNegativesMutator_Apply_NonUnaryNode(t *testing.T) {
	t.Parallel()

	mutator := &InvertNegativesMutator{}

	expr, err := parser.ParseExpr("a - b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:     invertNegativesType,
		Original: "-",
		Mutated:  "+",
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-UnaryExpr node")
	}
}
