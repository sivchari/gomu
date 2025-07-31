package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestBitwiseMutator_Name(t *testing.T) {
	mutator := &BitwiseMutator{}
	if mutator.Name() != bitwiseMutatorName {
		t.Errorf("Expected name '%s', got %s", bitwiseMutatorName, mutator.Name())
	}
}

func TestBitwiseMutator_CanMutate(t *testing.T) {
	mutator := &BitwiseMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "bitwise AND",
			code:     "x & y",
			expected: true,
		},
		{
			name:     "bitwise OR",
			code:     "x | y",
			expected: true,
		},
		{
			name:     "bitwise XOR",
			code:     "x ^ y",
			expected: true,
		},
		{
			name:     "bitwise AND NOT",
			code:     "x &^ y",
			expected: true,
		},
		{
			name:     "left shift",
			code:     "x << 2",
			expected: true,
		},
		{
			name:     "right shift",
			code:     "x >> 2",
			expected: true,
		},
		{
			name:     "arithmetic addition",
			code:     "x + y",
			expected: false,
		},
		{
			name:     "logical AND",
			code:     "x && y",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestBitwiseMutator_Mutate(t *testing.T) {
	mutator := &BitwiseMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected int // expected number of mutants
	}{
		{
			name:     "bitwise AND",
			code:     "x & y",
			expected: 3, // &, |, ^, &^
		},
		{
			name:     "bitwise OR",
			code:     "x | y",
			expected: 3, // &, ^, &^
		},
		{
			name:     "bitwise XOR",
			code:     "x ^ y",
			expected: 3, // &, |, &^
		},
		{
			name:     "left shift",
			code:     "x << 2",
			expected: 1, // >>
		},
		{
			name:     "right shift",
			code:     "x >> 2",
			expected: 1, // <<
		},
		{
			name:     "arithmetic addition",
			code:     "x + y",
			expected: 0, // not a bitwise operation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package main\nfunc test() { _ = " + tt.code + " }"

			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Find expression to mutate
			var expr ast.Expr
			ast.Inspect(file, func(node ast.Node) bool {
				if e, ok := node.(ast.Expr); ok && mutator.CanMutate(e) {
					expr = e

					return false
				}

				return true
			})

			if expr == nil && tt.expected > 0 {
				t.Fatalf("Expected expression not found in: %s", tt.code)
			}

			var mutants []Mutant
			if expr != nil {
				mutants = mutator.Mutate(expr, fset)
			}

			if len(mutants) != tt.expected {
				t.Errorf("Expected %d mutants, got %d", tt.expected, len(mutants))
			}

			// Check mutant properties
			for _, mutant := range mutants {
				if mutant.Line <= 0 {
					t.Errorf("Expected positive line number, got %d", mutant.Line)
				}

				if mutant.Description == "" {
					t.Error("Expected non-empty description")
				}

				if mutant.Type == "" {
					t.Error("Expected non-empty type")
				}
			}
		})
	}
}
