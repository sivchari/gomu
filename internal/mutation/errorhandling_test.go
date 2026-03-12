package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestErrorHandlingMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &ErrorHandlingMutator{}

	if mutator.Name() != errorHandlingMutatorName {
		t.Errorf("Name() = %q, want %q", mutator.Name(), errorHandlingMutatorName)
	}
}

func TestErrorHandlingMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &ErrorHandlingMutator{}

	tests := []struct {
		name     string
		src      string
		expected bool
	}{
		{
			name:     "return with err and other value",
			src:      "package main\nfunc f() (int, error) { err := error(nil); return 0, err }",
			expected: true,
		},
		{
			name:     "return nil only",
			src:      "package main\nfunc f() error { return nil }",
			expected: false,
		},
		{
			name:     "return err alone",
			src:      "package main\nfunc f() error { err := error(nil); return err }",
			expected: true,
		},
		{
			name:     "return x and y without err",
			src:      "package main\nfunc f() (int, int) { x, y := 1, 2; return x, y }",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()

			file, err := parser.ParseFile(fset, "test.go", tt.src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			var retStmt ast.Node

			ast.Inspect(file, func(n ast.Node) bool {
				if rs, ok := n.(*ast.ReturnStmt); ok {
					retStmt = rs

					return false
				}

				return true
			})

			if retStmt == nil {
				t.Fatalf("ReturnStmt not found in: %s", tt.src)
			}

			if got := mutator.CanMutate(retStmt); got != tt.expected {
				t.Errorf("CanMutate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorHandlingMutator_CanMutate_NonReturnNode(t *testing.T) {
	t.Parallel()

	mutator := &ErrorHandlingMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-ReturnStmt node")
	}
}

func TestErrorHandlingMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &ErrorHandlingMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc f() (int, error) { err := error(nil); return 0, err }"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var retStmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if rs, ok := n.(*ast.ReturnStmt); ok {
			retStmt = rs

			return false
		}

		return true
	})

	if retStmt == nil {
		t.Fatal("ReturnStmt not found")
	}

	mutants := mutator.Mutate(retStmt, fset)

	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant, got %d", len(mutants))
	}

	m := mutants[0]

	if m.Type != errorNilifyType {
		t.Errorf("Type = %q, want %q", m.Type, errorNilifyType)
	}

	if m.Original != "err" {
		t.Errorf("Original = %q, want %q", m.Original, "err")
	}

	if m.Mutated != "nil" {
		t.Errorf("Mutated = %q, want %q", m.Mutated, "nil")
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestErrorHandlingMutator_Apply(t *testing.T) {
	t.Parallel()

	mutator := &ErrorHandlingMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc f() (int, error) { err := error(nil); return 0, err }"

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var retStmt *ast.ReturnStmt

	ast.Inspect(file, func(n ast.Node) bool {
		if rs, ok := n.(*ast.ReturnStmt); ok {
			retStmt = rs

			return false
		}

		return true
	})

	if retStmt == nil {
		t.Fatal("ReturnStmt not found")
	}

	mutant := Mutant{
		Type:     errorNilifyType,
		Original: "err",
		Mutated:  "nil",
	}

	if !mutator.Apply(retStmt, mutant) {
		t.Error("Apply() = false, want true")
	}

	ident, ok := retStmt.Results[1].(*ast.Ident)
	if !ok {
		t.Fatal("Expected *ast.Ident in return results[1] after Apply")
	}

	if ident.Name != "nil" {
		t.Errorf("ident.Name = %q, want %q", ident.Name, "nil")
	}
}

func TestErrorHandlingMutator_Apply_NonReturnStmt(t *testing.T) {
	t.Parallel()

	mutator := &ErrorHandlingMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:     errorNilifyType,
		Original: "err",
		Mutated:  "nil",
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-ReturnStmt node")
	}
}
