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
				if mutant.Type != logicalBinaryType {
					t.Errorf("Expected type %q, got %s", logicalBinaryType, mutant.Type)
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

func TestLogicalMutator_Mutate_UnaryExpr(t *testing.T) {
	t.Parallel()

	mutator := &LogicalMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc test() { _ = !a }"

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

	if m.Type != logicalNotRemovalType {
		t.Errorf("Type = %q, want %q", m.Type, logicalNotRemovalType)
	}

	if m.Original != "!" {
		t.Errorf("Original = %q, want %q", m.Original, "!")
	}

	if m.Mutated != "" {
		t.Errorf("Mutated = %q, want %q", m.Mutated, "")
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}

	if m.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestLogicalMutator_ApplyWithCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        string
		mutantType  string
		expected    bool
		wantReplace bool
	}{
		{
			name:        "NOT removal applies and calls replaceFunc",
			code:        "!a",
			mutantType:  logicalNotRemovalType,
			expected:    true,
			wantReplace: true,
		},
		{
			name:        "non-NOT-removal type returns false",
			code:        "!a",
			mutantType:  logicalBinaryType,
			expected:    false,
			wantReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &LogicalMutator{}

			expr, err := parser.ParseExpr(tt.code)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			replaced := false
			replaceFunc := func(ast.Node) {
				replaced = true
			}

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: "!",
				Mutated:  "",
			}

			result := mutator.ApplyWithCursor(expr, replaceFunc, mutant)

			if result != tt.expected {
				t.Errorf("ApplyWithCursor() = %v, want %v", result, tt.expected)
			}

			if replaced != tt.wantReplace {
				t.Errorf("replaceFunc called = %v, want %v", replaced, tt.wantReplace)
			}
		})
	}
}

func TestLogicalMutator_Apply_NotRemoval(t *testing.T) {
	t.Parallel()

	mutator := &LogicalMutator{}

	expr, err := parser.ParseExpr("!a")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:     logicalNotRemovalType,
		Original: "!",
		Mutated:  "",
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for logicalNotRemovalType")
	}
}

func TestLogicalMutator_CanMutate_NonNotUnary(t *testing.T) {
	t.Parallel()

	mutator := &LogicalMutator{}

	expr, err := parser.ParseExpr("-a")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for unary minus")
	}
}

func TestLogicalMutator_Apply_WrongOriginal(t *testing.T) {
	t.Parallel()

	mutator := &LogicalMutator{}
	fset := token.NewFileSet()

	src := "package main\nfunc test() { _ = a && b }"

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
		t.Fatal("BinaryExpr not found")
	}

	mutant := Mutant{
		Type:     logicalBinaryType,
		Original: "||",
		Mutated:  "&&",
	}

	if mutator.Apply(node, mutant) {
		t.Error("Apply() = true, want false when Original doesn't match node operator")
	}
}

func TestLogicalMutator_Apply_NonBinaryNode(t *testing.T) {
	t.Parallel()

	mutator := &LogicalMutator{}

	expr, err := parser.ParseExpr("!a")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{
		Type:     logicalBinaryType,
		Original: "&&",
		Mutated:  "||",
	}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-BinaryExpr node with logicalBinaryType")
	}
}

func TestLogicalMutator_Apply(t *testing.T) {
	mutator := &LogicalMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name          string
		code          string
		mutantType    string
		originalValue string
		mutantValue   string
		expected      bool
	}{
		{
			name:          "apply logical binary mutation LAND to LOR",
			code:          "a && b",
			mutantType:    logicalBinaryType,
			originalValue: "&&",
			mutantValue:   "||",
			expected:      true,
		},
		{
			name:          "apply logical binary mutation LOR to LAND",
			code:          "a || b",
			mutantType:    logicalBinaryType,
			originalValue: "||",
			mutantValue:   "&&",
			expected:      true,
		},
		{
			name:          "unknown mutation type",
			code:          "a && b",
			mutantType:    "unknown",
			originalValue: "&&",
			mutantValue:   "||",
			expected:      false,
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
				Type:     tt.mutantType,
				Original: tt.originalValue,
				Mutated:  tt.mutantValue,
			}

			result := mutator.Apply(node, mutant)
			if result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
