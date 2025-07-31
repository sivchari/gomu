package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestConditionalMutator_Name(t *testing.T) {
	mutator := &ConditionalMutator{}
	if mutator.Name() != conditionalMutatorName {
		t.Errorf("Expected name 'conditional', got %s", mutator.Name())
	}
}

func TestConditionalMutator_CanMutate(t *testing.T) {
	mutator := &ConditionalMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "equality",
			code:     "a == b",
			expected: true,
		},
		{
			name:     "inequality",
			code:     "a != b",
			expected: true,
		},
		{
			name:     "less than",
			code:     "a < b",
			expected: true,
		},
		{
			name:     "less than or equal",
			code:     "a <= b",
			expected: true,
		},
		{
			name:     "greater than",
			code:     "a > b",
			expected: true,
		},
		{
			name:     "greater than or equal",
			code:     "a >= b",
			expected: true,
		},
		{
			name:     "arithmetic addition",
			code:     "a + b",
			expected: false,
		},
		{
			name:     "logical and",
			code:     "a && b",
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

func TestConditionalMutator_Mutate(t *testing.T) {
	mutator := &ConditionalMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "equality",
			code:     "a == b",
			expected: []string{"!=", "<", "<=", ">", ">="},
		},
		{
			name:     "inequality",
			code:     "a != b",
			expected: []string{"==", "<", "<=", ">", ">="},
		},
		{
			name:     "less than",
			code:     "a < b",
			expected: []string{"<=", ">", ">=", "==", "!="},
		},
		{
			name:     "less than or equal",
			code:     "a <= b",
			expected: []string{"<", ">", ">=", "==", "!="},
		},
		{
			name:     "greater than",
			code:     "a > b",
			expected: []string{">=", "<", "<=", "==", "!="},
		},
		{
			name:     "greater than or equal",
			code:     "a >= b",
			expected: []string{">", "<", "<=", "==", "!="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package main\nfunc test() { _ = " + tt.code + " }"

			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Find the binary expression
			var expr ast.Expr

			ast.Inspect(file, func(node ast.Node) bool {
				if be, ok := node.(*ast.BinaryExpr); ok {
					expr = be

					return false
				}

				return true
			})

			if expr == nil {
				t.Fatalf("Binary expression not found in: %s", tt.code)
			}

			mutants := mutator.Mutate(expr, fset)

			if len(mutants) != len(tt.expected) {
				t.Errorf("Expected %d mutants, got %d", len(tt.expected), len(mutants))
			}

			// Check that all expected mutations are present
			mutatedOps := make(map[string]bool)
			for _, mutant := range mutants {
				mutatedOps[mutant.Mutated] = true
			}

			for _, expectedOp := range tt.expected {
				if !mutatedOps[expectedOp] {
					t.Errorf("Expected mutation to %s not found", expectedOp)
				}
			}

			// Check mutant properties
			for _, mutant := range mutants {
				if mutant.Type != "conditional_binary" {
					t.Errorf("Expected mutant type 'conditional_binary', got %s", mutant.Type)
				}

				if mutant.Line <= 0 {
					t.Errorf("Expected positive line number, got %d", mutant.Line)
				}

				if mutant.Description == "" {
					t.Error("Expected non-empty description")
				}
			}
		})
	}
}

func TestConditionalMutator_IsConditionalOp(t *testing.T) {
	mutator := &ConditionalMutator{}

	conditionalOps := []token.Token{
		token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ,
	}

	nonConditionalOps := []token.Token{
		token.ADD, token.SUB, token.MUL, token.QUO, token.LAND, token.LOR,
	}

	for _, op := range conditionalOps {
		if !mutator.isConditionalOp(op) {
			t.Errorf("Expected %s to be a conditional operator", op)
		}
	}

	for _, op := range nonConditionalOps {
		if mutator.isConditionalOp(op) {
			t.Errorf("Expected %s to not be a conditional operator", op)
		}
	}
}

func TestConditionalMutator_GetConditionalMutations(t *testing.T) {
	mutator := &ConditionalMutator{}

	tests := []struct {
		op       token.Token
		expected []token.Token
	}{
		{
			token.EQL,
			[]token.Token{token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ},
		},
		{
			token.NEQ,
			[]token.Token{token.EQL, token.LSS, token.LEQ, token.GTR, token.GEQ},
		},
		{
			token.LSS,
			[]token.Token{token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ},
		},
		{
			token.LEQ,
			[]token.Token{token.LSS, token.GTR, token.GEQ, token.EQL, token.NEQ},
		},
		{
			token.GTR,
			[]token.Token{token.GEQ, token.LSS, token.LEQ, token.EQL, token.NEQ},
		},
		{
			token.GEQ,
			[]token.Token{token.GTR, token.LSS, token.LEQ, token.EQL, token.NEQ},
		},
		{
			token.ADD, // Not a conditional operator
			nil,
		},
	}

	for _, tt := range tests {
		result := mutator.getConditionalMutations(tt.op)
		if len(result) != len(tt.expected) {
			t.Errorf("For %s: expected %d mutations, got %d", tt.op, len(tt.expected), len(result))

			continue
		}

		for i, expected := range tt.expected {
			if result[i] != expected {
				t.Errorf("For %s: expected mutation %d to be %s, got %s", tt.op, i, expected, result[i])
			}
		}
	}
}

func TestConditionalMutator_AllMutationsGenerated(t *testing.T) {
	mutator := &ConditionalMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected []string // All mutations should be generated (including invalid ones)
	}{
		{
			name:     "err != nil - generates all mutations including invalid ones",
			code:     "err != nil",
			expected: []string{"==", "<", "<=", ">", ">="}, // All mutations, even invalid ordering operators
		},
		{
			name:     "x == nil - generates all mutations including invalid ones",
			code:     "x == nil",
			expected: []string{"!=", "<", "<=", ">", ">="}, // All mutations, even invalid ordering operators
		},
		{
			name:     "a != b - generates all mutations",
			code:     "a != b",
			expected: []string{"==", "<", "<=", ">", ">="}, // All mutations
		},
		{
			name:     "a < b - generates all mutations",
			code:     "a < b",
			expected: []string{"<=", ">", ">=", "==", "!="}, // All mutations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package main\nfunc test() { _ = " + tt.code + " }"

			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Find the binary expression
			var expr ast.Expr

			ast.Inspect(file, func(node ast.Node) bool {
				if be, ok := node.(*ast.BinaryExpr); ok {
					expr = be

					return false
				}

				return true
			})

			if expr == nil {
				t.Fatalf("Binary expression not found in: %s", tt.code)
			}

			mutants := mutator.Mutate(expr, fset)

			if len(mutants) != len(tt.expected) {
				t.Errorf("Expected %d mutants, got %d", len(tt.expected), len(mutants))
			}

			// Check that all expected mutations are present
			mutatedOps := make(map[string]bool)
			for _, mutant := range mutants {
				mutatedOps[mutant.Mutated] = true
			}

			for _, expectedOp := range tt.expected {
				if !mutatedOps[expectedOp] {
					t.Errorf("Expected mutation to %s not found", expectedOp)
				}
			}

			// Verify all mutations are generated (standard mutation testing behavior)
			for _, mutant := range mutants {
				if mutant.Type != "conditional_binary" {
					t.Errorf("Expected mutant type 'conditional_binary', got %s", mutant.Type)
				}
			}
		})
	}
}

func TestConditionalMutator_Apply(t *testing.T) {
	mutator := &ConditionalMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name        string
		code        string
		mutantType  string
		mutantValue string
		expected    bool
	}{
		{
			name:        "apply conditional mutation EQL to NEQ",
			code:        "a == b",
			mutantType:  "conditional_binary",
			mutantValue: "!=",
			expected:    true,
		},
		{
			name:        "apply conditional mutation NEQ to EQL",
			code:        "a != b",
			mutantType:  "conditional_binary",
			mutantValue: "==",
			expected:    true,
		},
		{
			name:        "apply conditional mutation LSS to GTR",
			code:        "a < b",
			mutantType:  "conditional_binary",
			mutantValue: ">",
			expected:    true,
		},
		{
			name:        "apply conditional mutation LEQ to GEQ",
			code:        "a <= b",
			mutantType:  "conditional_binary",
			mutantValue: ">=",
			expected:    true,
		},
		{
			name:        "apply conditional mutation GTR to LSS",
			code:        "a > b",
			mutantType:  "conditional_binary",
			mutantValue: "<",
			expected:    true,
		},
		{
			name:        "apply conditional mutation GEQ to LEQ",
			code:        "a >= b",
			mutantType:  "conditional_binary",
			mutantValue: "<=",
			expected:    true,
		},
		{
			name:        "unknown mutation type",
			code:        "a == b",
			mutantType:  "unknown",
			mutantValue: "!=",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package main\nfunc test() { _ = " + tt.code + " }"

			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			var node ast.Node
			ast.Inspect(file, func(n ast.Node) bool {
				if be, ok := n.(*ast.BinaryExpr); ok {
					node = be
					return false
				}
				return true
			})

			if node == nil {
				t.Fatalf("Target node not found for: %s", tt.code)
			}

			mutant := Mutant{
				Type:    tt.mutantType,
				Mutated: tt.mutantValue,
			}

			result := mutator.Apply(node, mutant)
			if result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
