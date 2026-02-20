package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestTypeChecker_IsValidMutation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		mutant   Mutant
		expected bool
	}{
		{
			name: "arithmetic on int should be valid",
			code: `package test
func foo() {
	x := 1 + 2
	_ = x
}`,
			mutant: Mutant{
				Type:    "arithmetic_binary",
				Mutated: "-",
			},
			expected: true,
		},
		{
			name: "arithmetic on string should only allow +",
			code: `package test
func foo() {
	x := "a" + "b"
	_ = x
}`,
			mutant: Mutant{
				Type:    "arithmetic_binary",
				Mutated: "-",
			},
			expected: false,
		},
		{
			name: "string concatenation should be valid",
			code: `package test
func foo() {
	x := "a" + "b"
	_ = x
}`,
			mutant: Mutant{
				Type:    "arithmetic_binary",
				Mutated: "+",
			},
			expected: true,
		},
		{
			name: "comparison on int should be valid",
			code: `package test
func foo() bool {
	return 1 < 2
}`,
			mutant: Mutant{
				Type:    "conditional_binary",
				Mutated: ">",
			},
			expected: true,
		},
		{
			name: "ordered comparison on string should be valid",
			code: `package test
func foo() bool {
	return "a" < "b"
}`,
			mutant: Mutant{
				Type:    "conditional_binary",
				Mutated: ">=",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			// Type check the code
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
			}

			config := &types.Config{
				Error: func(_ error) {}, // Ignore errors
			}

			_, err = config.Check("test", fset, []*ast.File{f}, info)
			if err != nil {
				t.Fatalf("failed to type check: %v", err)
			}

			tc := NewTypeChecker(info)

			// Find the binary expression in the AST
			var binaryExpr *ast.BinaryExpr
			ast.Inspect(f, func(n ast.Node) bool {
				if be, ok := n.(*ast.BinaryExpr); ok {
					binaryExpr = be
					return false
				}
				return true
			})

			if binaryExpr == nil {
				t.Fatal("no binary expression found")
			}

			result := tc.IsValidMutation(binaryExpr, tt.mutant)
			if result != tt.expected {
				t.Errorf("IsValidMutation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeChecker_NilTypeInfo(t *testing.T) {
	tc := NewTypeChecker(nil)

	// With nil type info, all mutations should be valid
	mutant := Mutant{
		Type:    "arithmetic_binary",
		Mutated: "-",
	}

	if !tc.IsValidMutation(nil, mutant) {
		t.Error("expected mutation to be valid when type info is nil")
	}
}

func TestTypeChecker_AssignToBinaryOp(t *testing.T) {
	tc := NewTypeChecker(nil)

	tests := []struct {
		op       string
		expected string
	}{
		{"+=", "+"},
		{"-=", "-"},
		{"*=", "*"},
		{"/=", "/"},
		{"%=", "%"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			result := tc.assignToBinaryOp(tt.op)
			if result != tt.expected {
				t.Errorf("assignToBinaryOp(%s) = %s, want %s", tt.op, result, tt.expected)
			}
		})
	}
}

func TestFilterMutants(t *testing.T) {
	code := `package test
func foo() {
	x := "a" + "b"
	_ = x
}`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("failed to parse code: %v", err)
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	config := &types.Config{
		Error: func(_ error) {},
	}

	_, err = config.Check("test", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("failed to type check: %v", err)
	}

	// Find the binary expression
	var binaryExpr *ast.BinaryExpr
	ast.Inspect(f, func(n ast.Node) bool {
		if be, ok := n.(*ast.BinaryExpr); ok {
			binaryExpr = be
			return false
		}
		return true
	})

	mutants := []Mutant{
		{Type: "arithmetic_binary", Mutated: "+"}, // Valid for string
		{Type: "arithmetic_binary", Mutated: "-"}, // Invalid for string
		{Type: "arithmetic_binary", Mutated: "*"}, // Invalid for string
		{Type: "arithmetic_binary", Mutated: "/"}, // Invalid for string
	}

	filtered := FilterMutants(mutants, binaryExpr, info)

	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered mutant, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].Mutated != "+" {
		t.Errorf("expected mutated to be '+', got '%s'", filtered[0].Mutated)
	}
}
