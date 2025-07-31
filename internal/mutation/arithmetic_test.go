package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestArithmeticMutator_Name(t *testing.T) {
	mutator := &ArithmeticMutator{}
	if mutator.Name() != arithmeticMutatorName {
		t.Errorf("Expected name 'arithmetic', got %s", mutator.Name())
	}
}

func TestArithmeticMutator_CanMutate(t *testing.T) {
	mutator := &ArithmeticMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "binary expression",
			code:     "a + b",
			expected: true,
		},
		{
			name:     "assignment statement",
			code:     "a += b",
			expected: true,
		},
		{
			name:     "increment statement",
			code:     "a++",
			expected: true,
		},
		{
			name:     "decrement statement",
			code:     "a--",
			expected: true,
		},
		{
			name:     "function call",
			code:     "fmt.Println()",
			expected: false,
		},
		{
			name:     "variable declaration",
			code:     "var x int",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			if err != nil {
				// Try as statement if expression parsing fails
				stmt, err := parser.ParseExpr("func() { " + tt.code + " }")
				if err != nil {
					t.Fatalf("Failed to parse code: %v", err)
				}
				// Extract the statement from the function
				if fn, ok := stmt.(*ast.FuncLit); ok && len(fn.Body.List) > 0 {
					if canMutate := mutator.CanMutate(fn.Body.List[0]); canMutate != tt.expected {
						t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
					}
				}

				return
			}

			if canMutate := mutator.CanMutate(expr); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestArithmeticMutator_Mutate_BinaryExpr(t *testing.T) {
	mutator := &ArithmeticMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected []string // Expected mutated operators
	}{
		{
			name:     "addition",
			code:     "a + b",
			expected: []string{"-", "*", "/"},
		},
		{
			name:     "subtraction",
			code:     "a - b",
			expected: []string{"+", "*", "/"},
		},
		{
			name:     "multiplication",
			code:     "a * b",
			expected: []string{"+", "-", "/", "%"},
		},
		{
			name:     "division",
			code:     "a / b",
			expected: []string{"+", "-", "*", "%"},
		},
		{
			name:     "modulo",
			code:     "a % b",
			expected: []string{"+", "-", "*", "/"},
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
				if mutant.Type != "arithmetic_binary" {
					t.Errorf("Expected type 'arithmetic_binary', got %s", mutant.Type)
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

func TestArithmeticMutator_Mutate_AssignStmt(t *testing.T) {
	mutator := &ArithmeticMutator{}
	fset := token.NewFileSet()

	// Parse assignment statement
	src := `package main
func test() {
	a += b
}`

	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Find the assignment statement
	var assignStmt *ast.AssignStmt

	ast.Inspect(file, func(node ast.Node) bool {
		if stmt, ok := node.(*ast.AssignStmt); ok {
			assignStmt = stmt

			return false
		}

		return true
	})

	if assignStmt == nil {
		t.Fatal("Assignment statement not found")
	}

	mutants := mutator.Mutate(assignStmt, fset)

	expectedOps := []string{"-=", "*=", "/="}
	if len(mutants) != len(expectedOps) {
		t.Errorf("Expected %d mutants, got %d", len(expectedOps), len(mutants))
	}

	// Check that all expected mutations are present
	mutatedOps := make(map[string]bool)
	for _, mutant := range mutants {
		mutatedOps[mutant.Mutated] = true

		if mutant.Type != "arithmetic_assign" {
			t.Errorf("Expected type 'arithmetic_assign', got %s", mutant.Type)
		}
	}

	for _, expectedOp := range expectedOps {
		if !mutatedOps[expectedOp] {
			t.Errorf("Expected mutation to %s not found", expectedOp)
		}
	}
}

func TestArithmeticMutator_Mutate_IncDecStmt(t *testing.T) {
	mutator := &ArithmeticMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "increment",
			code:     "a++",
			expected: "--",
		},
		{
			name:     "decrement",
			code:     "a--",
			expected: "++",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := `package main
func test() {
	` + tt.code + `
}`

			file, err := parser.ParseFile(fset, "", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Find the inc/dec statement
			var incDecStmt *ast.IncDecStmt

			ast.Inspect(file, func(node ast.Node) bool {
				if stmt, ok := node.(*ast.IncDecStmt); ok {
					incDecStmt = stmt

					return false
				}

				return true
			})

			if incDecStmt == nil {
				t.Fatal("Inc/Dec statement not found")
			}

			mutants := mutator.Mutate(incDecStmt, fset)

			if len(mutants) != 1 {
				t.Errorf("Expected 1 mutant, got %d", len(mutants))
			}

			if len(mutants) > 0 {
				mutant := mutants[0]
				if mutant.Mutated != tt.expected {
					t.Errorf("Expected mutation to %s, got %s", tt.expected, mutant.Mutated)
				}

				if mutant.Type != "arithmetic_incdec" {
					t.Errorf("Expected type 'arithmetic_incdec', got %s", mutant.Type)
				}
			}
		})
	}
}

func TestArithmeticMutator_GetArithmeticMutations(t *testing.T) {
	mutator := &ArithmeticMutator{}

	tests := []struct {
		op       token.Token
		expected []token.Token
	}{
		{token.ADD, []token.Token{token.SUB, token.MUL, token.QUO}},
		{token.SUB, []token.Token{token.ADD, token.MUL, token.QUO}},
		{token.MUL, []token.Token{token.ADD, token.SUB, token.QUO, token.REM}},
		{token.QUO, []token.Token{token.ADD, token.SUB, token.MUL, token.REM}},
		{token.REM, []token.Token{token.ADD, token.SUB, token.MUL, token.QUO}},
		{token.LAND, nil}, // Not an arithmetic operator
	}

	for _, tt := range tests {
		result := mutator.getArithmeticMutations(tt.op)
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

func TestArithmeticMutator_GetAssignMutations(t *testing.T) {
	mutator := &ArithmeticMutator{}

	tests := []struct {
		op       token.Token
		expected []token.Token
	}{
		{token.ADD_ASSIGN, []token.Token{token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN}},
		{token.SUB_ASSIGN, []token.Token{token.ADD_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN}},
		{token.MUL_ASSIGN, []token.Token{token.ADD_ASSIGN, token.SUB_ASSIGN, token.QUO_ASSIGN}},
		{token.QUO_ASSIGN, []token.Token{token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN}},
		{token.ASSIGN, nil}, // Not a compound assignment
	}

	for _, tt := range tests {
		result := mutator.getAssignMutations(tt.op)
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

func TestArithmeticMutator_Apply(t *testing.T) {
	mutator := &ArithmeticMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name        string
		code        string
		mutantType  string
		mutantValue string
		expected    bool
	}{
		{
			name:        "apply binary mutation",
			code:        "a + b",
			mutantType:  "arithmetic_binary",
			mutantValue: "-",
			expected:    true,
		},
		{
			name:        "apply assign mutation",
			code:        "a += b",
			mutantType:  "arithmetic_assign",
			mutantValue: "-=",
			expected:    true,
		},
		{
			name:        "apply incdec mutation",
			code:        "a++",
			mutantType:  "arithmetic_incdec",
			mutantValue: "--",
			expected:    true,
		},
		{
			name:        "unknown mutation type",
			code:        "a + b",
			mutantType:  "unknown",
			mutantValue: "-",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var src string
			if tt.mutantType == "arithmetic_assign" || tt.mutantType == "arithmetic_incdec" {
				src = "package main\nfunc test() {\n\t" + tt.code + "\n}"
			} else {
				src = "package main\nfunc test() { _ = " + tt.code + " }"
			}

			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			var node ast.Node
			ast.Inspect(file, func(n ast.Node) bool {
				switch tt.mutantType {
				case "arithmetic_binary":
					if be, ok := n.(*ast.BinaryExpr); ok {
						node = be
						return false
					}
				case "arithmetic_assign":
					if as, ok := n.(*ast.AssignStmt); ok {
						node = as
						return false
					}
				case "arithmetic_incdec":
					if ids, ok := n.(*ast.IncDecStmt); ok {
						node = ids
						return false
					}
				default:
					if be, ok := n.(*ast.BinaryExpr); ok {
						node = be
						return false
					}
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
