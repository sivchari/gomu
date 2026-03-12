package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

const branchIfSimpleSrc = "package main\nfunc f(x int) bool {\n\tif x > 0 {\n\t\treturn true\n\t}\n\treturn false\n}"

func TestBranchMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}

	if mutator.Name() != branchMutatorName {
		t.Errorf("Name() = %q, want %q", mutator.Name(), branchMutatorName)
	}
}

func TestBranchMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}

	tests := []struct {
		name     string
		src      string
		expected bool
	}{
		{
			name:     "if with comparison",
			src:      branchIfSimpleSrc,
			expected: true,
		},
		{
			name:     "if with true literal",
			src:      "package main\nfunc f() {\n\tif true {\n\t}\n}",
			expected: false,
		},
		{
			name:     "if with false literal",
			src:      "package main\nfunc f() {\n\tif false {\n\t}\n}",
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

			var ifStmt ast.Node

			ast.Inspect(file, func(n ast.Node) bool {
				if s, ok := n.(*ast.IfStmt); ok {
					ifStmt = s

					return false
				}

				return true
			})

			if ifStmt == nil {
				t.Fatalf("IfStmt not found in: %s", tt.src)
			}

			if got := mutator.CanMutate(ifStmt); got != tt.expected {
				t.Errorf("CanMutate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBranchMutator_CanMutate_NonIfNode(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-IfStmt node")
	}
}

func TestBranchMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "test.go", branchIfSimpleSrc, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var ifStmt ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if s, ok := n.(*ast.IfStmt); ok {
			ifStmt = s

			return false
		}

		return true
	})

	if ifStmt == nil {
		t.Fatal("IfStmt not found")
	}

	mutants := mutator.Mutate(ifStmt, fset)

	if len(mutants) != 2 {
		t.Fatalf("Expected 2 mutants, got %d", len(mutants))
	}

	mutatedValues := make(map[string]bool)

	for _, m := range mutants {
		if m.Type != branchConditionType {
			t.Errorf("Type = %q, want %q", m.Type, branchConditionType)
		}

		if m.Original == "" {
			t.Error("Expected non-empty Original")
		}

		if m.Mutated == "" {
			t.Error("Expected non-empty Mutated")
		}

		if m.Description == "" {
			t.Error("Expected non-empty Description")
		}

		if m.Line <= 0 {
			t.Errorf("Expected positive line number, got %d", m.Line)
		}

		mutatedValues[m.Mutated] = true
	}

	if !mutatedValues[boolTrue] {
		t.Error("Expected a mutant with Mutated=true")
	}

	if !mutatedValues[boolFalse] {
		t.Error("Expected a mutant with Mutated=false")
	}
}

func TestBranchMutator_Apply(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "test.go", branchIfSimpleSrc, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var ifStmt *ast.IfStmt

	ast.Inspect(file, func(n ast.Node) bool {
		if s, ok := n.(*ast.IfStmt); ok {
			ifStmt = s

			return false
		}

		return true
	})

	if ifStmt == nil {
		t.Fatal("IfStmt not found")
	}

	mutant := Mutant{
		Type:    branchConditionType,
		Mutated: boolTrue,
	}

	if !mutator.Apply(ifStmt, mutant) {
		t.Error("Apply() = false, want true")
	}

	ident, ok := ifStmt.Cond.(*ast.Ident)
	if !ok {
		t.Fatal("Expected *ast.Ident as Cond after Apply")
	}

	if ident.Name != boolTrue {
		t.Errorf("ident.Name = %q, want %q", ident.Name, boolTrue)
	}
}

func TestBranchMutator_Apply_NonIfStmt(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:    branchConditionType,
		Mutated: boolTrue,
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-IfStmt node")
	}
}

func TestBranchMutator_Apply_WrongType(t *testing.T) {
	t.Parallel()

	mutator := &BranchMutator{}
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "test.go", branchIfSimpleSrc, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var ifStmt *ast.IfStmt

	ast.Inspect(file, func(n ast.Node) bool {
		if s, ok := n.(*ast.IfStmt); ok {
			ifStmt = s

			return false
		}

		return true
	})

	if ifStmt == nil {
		t.Fatal("IfStmt not found")
	}

	mutant := Mutant{
		Type:    "unknown_type",
		Mutated: boolTrue,
	}

	if mutator.Apply(ifStmt, mutant) {
		t.Error("Apply() = true, want false for unknown mutation type")
	}
}
