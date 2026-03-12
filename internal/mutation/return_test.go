package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestReturnMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}
	if mutator.Name() != "return" {
		t.Errorf("Expected name 'return', got %s", mutator.Name())
	}
}

func TestReturnMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}

	tests := []struct {
		name     string
		src      string
		expected bool
	}{
		{
			name:     "return true",
			src:      "package main\nfunc f() bool { return true }",
			expected: true,
		},
		{
			name:     "return false",
			src:      "package main\nfunc f() bool { return false }",
			expected: true,
		},
		{
			name:     "return int literal",
			src:      "package main\nfunc f() int { return 42 }",
			expected: true,
		},
		{
			name:     "return string literal",
			src:      "package main\nfunc f() string { return \"hello\" }",
			expected: true,
		},
		{
			name:     "return variable",
			src:      "package main\nfunc f() int { x := 1; return x }",
			expected: false,
		},
		{
			name:     "naked return",
			src:      "package main\nfunc f() (x int) { return }",
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

			if canMutate := mutator.CanMutate(retStmt); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestReturnMutator_CanMutate_NonReturnNode(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-ReturnStmt node")
	}
}

func TestReturnMutator_Mutate_BoolLiteral(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc f() bool { return true }"

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

	if m.Type != returnBoolLiteralType {
		t.Errorf("Type = %q, want %q", m.Type, returnBoolLiteralType)
	}

	if m.Original != "true" {
		t.Errorf("Original = %q, want %q", m.Original, "true")
	}

	if m.Mutated != "false" {
		t.Errorf("Mutated = %q, want %q", m.Mutated, "false")
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestReturnMutator_Mutate_ZeroValue(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name         string
		src          string
		wantOriginal string
		wantMutated  string
	}{
		{
			name:         "INT literal",
			src:          "package main\nfunc f() int { return 42 }",
			wantOriginal: "42",
			wantMutated:  "0",
		},
		{
			name:         "FLOAT literal",
			src:          "package main\nfunc f() float64 { return 3.14 }",
			wantOriginal: "3.14",
			wantMutated:  "0",
		},
		{
			name:         "STRING literal",
			src:          "package main\nfunc f() string { return \"hello\" }",
			wantOriginal: `"hello"`,
			wantMutated:  `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

			mutants := mutator.Mutate(retStmt, fset)

			if len(mutants) != 1 {
				t.Fatalf("Expected 1 mutant, got %d", len(mutants))
			}

			m := mutants[0]

			if m.Type != returnZeroValueType {
				t.Errorf("Type = %q, want %q", m.Type, returnZeroValueType)
			}

			if m.Original != tt.wantOriginal {
				t.Errorf("Original = %q, want %q", m.Original, tt.wantOriginal)
			}

			if m.Mutated != tt.wantMutated {
				t.Errorf("Mutated = %q, want %q", m.Mutated, tt.wantMutated)
			}

			if m.Line <= 0 {
				t.Errorf("Expected positive line number, got %d", m.Line)
			}

			if m.Description == "" {
				t.Error("Expected non-empty description")
			}
		})
	}
}

func TestReturnMutator_Apply_BoolLiteral(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc f() bool { return true }"

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
		Type:     returnBoolLiteralType,
		Original: "true",
		Mutated:  "false",
	}

	if !mutator.Apply(retStmt, mutant) {
		t.Error("Apply() = false, want true")
	}

	ident, ok := retStmt.Results[0].(*ast.Ident)
	if !ok {
		t.Fatal("Expected *ast.Ident in return results")
	}

	if ident.Name != "false" {
		t.Errorf("ident.Name = %q, want %q", ident.Name, "false")
	}
}

func TestReturnMutator_Apply_ZeroValue(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name      string
		src       string
		original  string
		mutated   string
		wantValue string
	}{
		{
			name:      "INT literal to zero",
			src:       "package main\nfunc f() int { return 42 }",
			original:  "42",
			mutated:   "0",
			wantValue: "0",
		},
		{
			name:      "STRING literal to empty",
			src:       "package main\nfunc f() string { return \"hello\" }",
			original:  `"hello"`,
			mutated:   `""`,
			wantValue: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(fset, "test.go", tt.src, 0)
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
				t.Fatalf("ReturnStmt not found in: %s", tt.src)
			}

			mutant := Mutant{
				Type:     returnZeroValueType,
				Original: tt.original,
				Mutated:  tt.mutated,
			}

			if !mutator.Apply(retStmt, mutant) {
				t.Error("Apply() = false, want true")
			}

			lit, ok := retStmt.Results[0].(*ast.BasicLit)
			if !ok {
				t.Fatal("Expected *ast.BasicLit in return results")
			}

			if lit.Value != tt.wantValue {
				t.Errorf("lit.Value = %q, want %q", lit.Value, tt.wantValue)
			}
		})
	}
}

func TestReturnMutator_Apply_WrongType(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc f() bool { return true }"

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
		Type:     "unknown_type",
		Original: "true",
		Mutated:  "false",
	}

	if mutator.Apply(retStmt, mutant) {
		t.Error("Apply() = true, want false for unknown mutation type")
	}
}

func TestReturnMutator_Apply_NonReturnNode(t *testing.T) {
	t.Parallel()

	mutator := &ReturnMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:     returnBoolLiteralType,
		Original: "true",
		Mutated:  "false",
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-ReturnStmt node")
	}
}
