package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestLogicalMutator_Name(t *testing.T) {
	mutator := &LogicalMutator{}
	if mutator.Name() != "logical" {
		t.Errorf("Expected name 'logical', got %s", mutator.Name())
	}
}

func TestLogicalMutator_CanMutate(t *testing.T) {
	mutator := &LogicalMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "logical and",
			code:     "a && b",
			expected: true,
		},
		{
			name:     "logical or",
			code:     "a || b",
			expected: true,
		},
		{
			name:     "logical not",
			code:     "!a",
			expected: true,
		},
		{
			name:     "arithmetic addition",
			code:     "a + b",
			expected: false,
		},
		{
			name:     "equality",
			code:     "a == b",
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

func TestLogicalMutator_Mutate_BinaryExpr(t *testing.T) {
	mutator := &LogicalMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "logical and",
			code:     "a && b",
			expected: []string{"||"},
		},
		{
			name:     "logical or",
			code:     "a || b",
			expected: []string{"&&"},
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
				if mutant.Type != "logical_binary" {
					t.Errorf("Expected type 'logical_binary', got %s", mutant.Type)
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

func TestLogicalMutator_Mutate_UnaryExpr(t *testing.T) {
	mutator := &LogicalMutator{}
	fset := token.NewFileSet()

	// Parse unary NOT expression
	src := `package main
func test() bool {
	return !condition
}`

	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Find the unary expression
	var unaryExpr *ast.UnaryExpr
	ast.Inspect(file, func(node ast.Node) bool {
		if expr, ok := node.(*ast.UnaryExpr); ok && expr.Op == token.NOT {
			unaryExpr = expr
			return false
		}
		return true
	})

	if unaryExpr == nil {
		t.Fatal("Unary NOT expression not found")
	}

	mutants := mutator.Mutate(unaryExpr, fset)

	// Logical NOT removal should generate 1 mutant
	expectedMutants := 1
	if len(mutants) != expectedMutants {
		t.Errorf("Expected %d mutants, got %d", expectedMutants, len(mutants))
	}

	if len(mutants) > 0 {
		mutant := mutants[0]
		if mutant.Type != "logical_not_removal" {
			t.Errorf("Expected type 'logical_not_removal', got %s", mutant.Type)
		}
		if mutant.Original != "!" {
			t.Errorf("Expected original '!', got %s", mutant.Original)
		}
		if mutant.Mutated != "" {
			t.Errorf("Expected mutated '', got %s", mutant.Mutated)
		}
	}
}

func TestLogicalMutator_IsLogicalOp(t *testing.T) {
	mutator := &LogicalMutator{}

	logicalOps := []token.Token{token.LAND, token.LOR}
	nonLogicalOps := []token.Token{
		token.ADD, token.SUB, token.MUL, token.QUO,
		token.EQL, token.NEQ, token.LSS, token.GTR,
	}

	for _, op := range logicalOps {
		if !mutator.isLogicalOp(op) {
			t.Errorf("Expected %s to be a logical operator", op)
		}
	}

	for _, op := range nonLogicalOps {
		if mutator.isLogicalOp(op) {
			t.Errorf("Expected %s to not be a logical operator", op)
		}
	}
}

func TestLogicalMutator_GetLogicalMutations(t *testing.T) {
	mutator := &LogicalMutator{}

	tests := []struct {
		op       token.Token
		expected []token.Token
	}{
		{token.LAND, []token.Token{token.LOR}},
		{token.LOR, []token.Token{token.LAND}},
		{token.ADD, nil}, // Not a logical operator
	}

	for _, tt := range tests {
		result := mutator.getLogicalMutations(tt.op)
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