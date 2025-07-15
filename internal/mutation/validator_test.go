package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

const declaredAndNotUsedError = "declared and not used: x"

func TestTypeValidator_IsValidArithmeticMutation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		op       token.Token
		newOp    token.Token
		expected bool
	}{
		{
			name:     "int + to - (valid)",
			code:     "package main\nfunc main() { _ = 5 + 3 }",
			op:       token.ADD,
			newOp:    token.SUB,
			expected: true,
		},
		{
			name:     "float64 + to % (invalid)",
			code:     "package main\nfunc main() { _ = 5.0 + 3.0 }",
			op:       token.ADD,
			newOp:    token.REM,
			expected: false,
		},
		{
			name: "string + to - (invalid)",
			code: `package main
func main() { _ = "hello" + "world" }`,
			op:       token.ADD,
			newOp:    token.SUB,
			expected: false,
		},
		{
			name:     "int * to / (valid)",
			code:     "package main\nfunc main() { _ = 5 * 3 }",
			op:       token.MUL,
			newOp:    token.QUO,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the code
			fset := token.NewFileSet()

			node, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Create type info
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
			}

			// Type check with error suppression
			config := &types.Config{
				Error: func(err error) {
					// Suppress "declared and not used" and similar errors
					if err.Error() != declaredAndNotUsedError {
						t.Logf("Type checking warning: %v", err)
					}
				},
			}

			_, err = config.Check("main", fset, []*ast.File{node}, info)
			if err != nil {
				t.Fatalf("Type checking failed: %v", err)
			}

			// Create validator
			validator := NewTypeValidator(info)

			// Find the binary expression
			var binExpr *ast.BinaryExpr
			ast.Inspect(node, func(n ast.Node) bool {
				if be, ok := n.(*ast.BinaryExpr); ok && be.Op == tt.op {
					binExpr = be

					return false
				}

				return true
			})

			if binExpr == nil {
				t.Fatalf("Could not find binary expression with operator %s", tt.op.String())
			}

			// Test the validation
			result := validator.IsValidArithmeticMutation(binExpr, tt.newOp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for mutation %s -> %s", tt.expected, result, tt.op.String(), tt.newOp.String())
			}
		})
	}
}

func TestTypeValidator_IsValidConditionalMutation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		op       token.Token
		newOp    token.Token
		expected bool
	}{
		{
			name:     "int == to != (valid)",
			code:     "package main\nfunc main() { _ = 5 == 3 }",
			op:       token.EQL,
			newOp:    token.NEQ,
			expected: true,
		},
		{
			name:     "int == to < (valid)",
			code:     "package main\nfunc main() { _ = 5 == 3 }",
			op:       token.EQL,
			newOp:    token.LSS,
			expected: true,
		},
		{
			name: "string == to < (valid for comparable types)",
			code: `package main
func main() { _ = "a" == "b" }`,
			op:       token.EQL,
			newOp:    token.LSS,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the code
			fset := token.NewFileSet()

			node, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Create type info
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
			}

			// Type check with error suppression
			config := &types.Config{
				Error: func(err error) {
					// Suppress "declared and not used" and similar errors
					if err.Error() != declaredAndNotUsedError {
						t.Logf("Type checking warning: %v", err)
					}
				},
			}

			_, err = config.Check("main", fset, []*ast.File{node}, info)
			if err != nil {
				t.Fatalf("Type checking failed: %v", err)
			}

			// Create validator
			validator := NewTypeValidator(info)

			// Find the binary expression
			var binExpr *ast.BinaryExpr
			ast.Inspect(node, func(n ast.Node) bool {
				if be, ok := n.(*ast.BinaryExpr); ok && be.Op == tt.op {
					binExpr = be

					return false
				}

				return true
			})

			if binExpr == nil {
				t.Fatalf("Could not find binary expression with operator %s", tt.op.String())
			}

			// Test the validation
			result := validator.IsValidConditionalMutation(binExpr, tt.newOp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for mutation %s -> %s", tt.expected, result, tt.op.String(), tt.newOp.String())
			}
		})
	}
}

func TestTypeValidator_IsValidLogicalMutation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		op       token.Token
		newOp    token.Token
		expected bool
	}{
		{
			name:     "bool && to || (valid)",
			code:     "package main\nfunc main() { _ = true && false }",
			op:       token.LAND,
			newOp:    token.LOR,
			expected: true,
		},
		{
			name:     "bool || to && (valid)",
			code:     "package main\nfunc main() { _ = true || false }",
			op:       token.LOR,
			newOp:    token.LAND,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the code
			fset := token.NewFileSet()

			node, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Create type info
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
			}

			// Type check with error suppression
			config := &types.Config{
				Error: func(err error) {
					// Suppress "declared and not used" and similar errors
					if err.Error() != declaredAndNotUsedError {
						t.Logf("Type checking warning: %v", err)
					}
				},
			}

			_, err = config.Check("main", fset, []*ast.File{node}, info)
			if err != nil {
				t.Fatalf("Type checking failed: %v", err)
			}

			// Create validator
			validator := NewTypeValidator(info)

			// Find the binary expression
			var binExpr *ast.BinaryExpr
			ast.Inspect(node, func(n ast.Node) bool {
				if be, ok := n.(*ast.BinaryExpr); ok && be.Op == tt.op {
					binExpr = be

					return false
				}

				return true
			})

			if binExpr == nil {
				t.Fatalf("Could not find binary expression with operator %s", tt.op.String())
			}

			// Test the validation
			result := validator.IsValidLogicalMutation(binExpr, tt.newOp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for mutation %s -> %s", tt.expected, result, tt.op.String(), tt.newOp.String())
			}
		})
	}
}

func TestTypeValidator_WithoutTypeInfo(t *testing.T) {
	// Test that validator gracefully handles nil type info
	validator := NewTypeValidator(nil)

	// Create a dummy binary expression
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, "", "package main\nfunc main() { _ = 5 + 3 }", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var binExpr *ast.BinaryExpr
	ast.Inspect(node, func(n ast.Node) bool {
		if be, ok := n.(*ast.BinaryExpr); ok {
			binExpr = be

			return false
		}

		return true
	})

	// Should return true (allow all) when no type info available
	result := validator.IsValidArithmeticMutation(binExpr, token.REM)
	if !result {
		t.Error("Expected true when no type info available")
	}
}
