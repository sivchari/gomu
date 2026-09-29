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
				Type:    arithmeticBinaryType,
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
				Type:    arithmeticBinaryType,
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
				Type:    arithmeticBinaryType,
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
				be, ok := n.(*ast.BinaryExpr)
				if !ok {
					return true
				}

				binaryExpr = be

				return false
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
		Type:    arithmeticBinaryType,
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

func TestTypeChecker_InterfaceNilComparison(t *testing.T) {
	// Test that interface != nil cannot be mutated to <, <=, >, >=
	code := `package test

var errGlobal error

func foo() bool {
	return errGlobal != nil
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

	t.Logf("Types map has %d entries", len(info.Types))

	tc := NewTypeChecker(info)

	// Find the binary expression
	var binaryExpr *ast.BinaryExpr

	ast.Inspect(f, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		binaryExpr = be

		return false
	})

	if binaryExpr == nil {
		t.Fatal("no binary expression found")
	}

	// Check left operand type
	leftType := tc.getExprType(binaryExpr.X)
	if leftType == nil {
		t.Log("Left operand type is nil - type info not available for this expression")
	} else {
		t.Logf("Left operand type: %s (underlying: %T)", leftType, leftType.Underlying())
	}

	// Test mutations
	tests := []struct {
		mutated  string
		expected bool
	}{
		{"==", true},  // Valid: interface can be compared with ==
		{"!=", true},  // Valid: interface can be compared with !=
		{"<", false},  // Invalid: interface cannot use ordered comparison
		{"<=", false}, // Invalid
		{">", false},  // Invalid
		{">=", false}, // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.mutated, func(t *testing.T) {
			mutant := Mutant{
				Type:    "conditional_binary",
				Mutated: tt.mutated,
			}

			result := tc.IsValidMutation(binaryExpr, mutant)
			if result != tt.expected {
				t.Errorf("IsValidMutation for %s = %v, want %v", tt.mutated, result, tt.expected)
			}
		})
	}
}

func TestTypeChecker_IsValidErrorNilifyMutation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		original string
		expected bool
	}{
		{
			name: "error typed err",
			code: `package test
func foo() error {
	err := error(nil)
	return err
}`,
			original: "err",
			expected: true,
		},
		{
			name: "error typed renamed identifier",
			code: `package test
func foo() error {
	acquireErr := error(nil)
	return acquireErr
}`,
			original: "acquireErr",
			expected: true,
		},
		{
			name: "error typed identifier named failure",
			code: `package test
func foo() error {
	failure := error(nil)
	return failure
}`,
			original: "failure",
			expected: true,
		},
		{
			name: "named error result",
			code: `package test
func foo() (err error) {
	return err
}`,
			original: "err",
			expected: true,
		},
		{
			name: "non-error variable named err",
			code: `package test
func foo() int {
	err := 42
	return err
}`,
			original: "err",
			expected: false,
		},
		{
			name: "multi-result return, error operand",
			code: `package test
func foo() (int, error) {
	value := 1
	someErr := error(nil)
	return value, someErr
}`,
			original: "someErr",
			expected: true,
		},
		{
			name: "multi-result return, non-error operand",
			code: `package test
func foo() (int, error) {
	value := 1
	someErr := error(nil)
	return value, someErr
}`,
			original: "value",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()

			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Uses:  make(map[*ast.Ident]types.Object),
				Defs:  make(map[*ast.Ident]types.Object),
			}

			config := &types.Config{
				Error: func(_ error) {},
			}

			_, err = config.Check("test", fset, []*ast.File{f}, info)
			if err != nil {
				t.Fatalf("failed to type check: %v", err)
			}

			tc := NewTypeChecker(info)

			var retStmt *ast.ReturnStmt

			ast.Inspect(f, func(n ast.Node) bool {
				rs, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}

				retStmt = rs

				return false
			})

			if retStmt == nil {
				t.Fatal("no return statement found")
			}

			mutant := Mutant{Type: errorNilifyType, Original: tt.original, Mutated: nilIdentName}

			result := tc.IsValidMutation(retStmt, mutant)
			if result != tt.expected {
				t.Errorf("IsValidMutation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTypeChecker_IsValidErrorNilifyMutation_NilTypeInfo(t *testing.T) {
	tc := NewTypeChecker(nil)

	findReturnStmt := func(t *testing.T, src string) *ast.ReturnStmt {
		t.Helper()

		fset := token.NewFileSet()

		f, err := parser.ParseFile(fset, "test.go", src, 0)
		if err != nil {
			t.Fatalf("failed to parse code: %v", err)
		}

		var retStmt *ast.ReturnStmt

		ast.Inspect(f, func(n ast.Node) bool {
			rs, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}

			retStmt = rs

			return false
		})

		if retStmt == nil {
			t.Fatal("no return statement found")
		}

		return retStmt
	}

	acquireErrStmt := findReturnStmt(t, "package main\nfunc f() error { acquireErr := error(nil); return acquireErr }")
	if tc.IsValidMutation(acquireErrStmt, Mutant{Type: errorNilifyType, Original: "acquireErr", Mutated: nilIdentName}) {
		t.Error("expected non-err-named identifier to be rejected without type info")
	}

	errStmt := findReturnStmt(t, "package main\nfunc f() error { err := error(nil); return err }")
	if !tc.IsValidMutation(errStmt, Mutant{Type: errorNilifyType, Original: errIdentName, Mutated: nilIdentName}) {
		t.Error("expected err-named identifier to remain valid without type info")
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
		be, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		binaryExpr = be

		return false
	})

	mutants := []Mutant{
		{Type: arithmeticBinaryType, Mutated: "+"}, // Valid for string
		{Type: arithmeticBinaryType, Mutated: "-"}, // Invalid for string
		{Type: arithmeticBinaryType, Mutated: "*"}, // Invalid for string
		{Type: arithmeticBinaryType, Mutated: "/"}, // Invalid for string
	}

	filtered := FilterMutants(mutants, binaryExpr, info)

	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered mutant, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].Mutated != "+" {
		t.Errorf("expected mutated to be '+', got '%s'", filtered[0].Mutated)
	}
}
