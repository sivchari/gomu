package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseBasicLit(t *testing.T, code string) ast.Node {
	t.Helper()

	expr, err := parser.ParseExpr(code)
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	var lit ast.Node

	ast.Inspect(expr, func(n ast.Node) bool {
		if bl, ok := n.(*ast.BasicLit); ok {
			lit = bl

			return false
		}

		return true
	})

	if lit == nil {
		return expr
	}

	return lit
}

func TestBoundaryValueMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &BoundaryValueMutator{}
	if mutator.Name() != boundaryValueMutatorName {
		t.Errorf("Expected name %q, got %s", boundaryValueMutatorName, mutator.Name())
	}
}

func TestBoundaryValueMutator_CanMutate(t *testing.T) {
	t.Parallel()

	mutator := &BoundaryValueMutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{name: "decimal int", code: "10", expected: true},
		{name: "zero", code: "0", expected: true},
		{name: "hex int", code: "0xFF", expected: true},
		{name: "binary int", code: "0b1010", expected: true},
		{name: "underscored int", code: "1_000", expected: true},
		{name: "float", code: "3.14", expected: false},
		{name: "string", code: `"hello"`, expected: false},
		{name: "identifier", code: "x", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := parseBasicLit(t, tt.code)

			if canMutate := mutator.CanMutate(node); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func TestBoundaryValueMutator_Mutate(t *testing.T) {
	t.Parallel()

	mutator := &BoundaryValueMutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name        string
		code        string
		wantMutated []string
	}{
		{name: "positive decimal", code: "10", wantMutated: []string{"11", "9"}},
		{name: "zero only increments", code: "0", wantMutated: []string{"1"}},
		{name: "one", code: "1", wantMutated: []string{"2", "0"}},
		{name: "hex", code: "0x10", wantMutated: []string{"17", "15"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := parseBasicLit(t, tt.code)

			mutants := mutator.Mutate(node, fset)

			if len(mutants) != len(tt.wantMutated) {
				t.Fatalf("Expected %d mutants, got %d", len(tt.wantMutated), len(mutants))
			}

			got := make([]string, len(mutants))
			for i, mut := range mutants {
				got[i] = mut.Mutated

				if mut.Type != boundaryValueType {
					t.Errorf("Type = %q, want %q", mut.Type, boundaryValueType)
				}

				if mut.Original != tt.code {
					t.Errorf("Original = %q, want %q", mut.Original, tt.code)
				}

				if mut.Description == "" {
					t.Error("Expected non-empty description")
				}
			}

			for i, want := range tt.wantMutated {
				if got[i] != want {
					t.Errorf("mutant[%d].Mutated = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestBoundaryValueMutator_Mutate_NonInt(t *testing.T) {
	t.Parallel()

	mutator := &BoundaryValueMutator{}
	fset := token.NewFileSet()

	node := parseBasicLit(t, "3.14")

	if mutants := mutator.Mutate(node, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for float literal", mutants)
	}
}

func TestBoundaryValueMutator_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		mutantType string
		original   string
		mutated    string
		expected   bool
		wantValue  string
	}{
		{
			name:       "increment",
			code:       "10",
			mutantType: boundaryValueType,
			original:   "10",
			mutated:    "11",
			expected:   true,
			wantValue:  "11",
		},
		{
			name:       "decrement",
			code:       "10",
			mutantType: boundaryValueType,
			original:   "10",
			mutated:    "9",
			expected:   true,
			wantValue:  "9",
		},
		{
			name:       "wrong mutant type",
			code:       "10",
			mutantType: "unknown",
			original:   "10",
			mutated:    "11",
			expected:   false,
			wantValue:  "10",
		},
		{
			name:       "wrong original",
			code:       "10",
			mutantType: boundaryValueType,
			original:   "20",
			mutated:    "21",
			expected:   false,
			wantValue:  "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &BoundaryValueMutator{}
			node := parseBasicLit(t, tt.code)

			mutant := Mutant{
				Type:     tt.mutantType,
				Original: tt.original,
				Mutated:  tt.mutated,
			}

			if result := mutator.Apply(node, mutant); result != tt.expected {
				t.Errorf("Apply() = %v, expected %v", result, tt.expected)
			}

			lit, ok := node.(*ast.BasicLit)
			if !ok {
				t.Fatalf("expected *ast.BasicLit, got %T", node)
			}

			if lit.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", lit.Value, tt.wantValue)
			}
		})
	}
}

func TestBoundaryValueMutator_Apply_NonLit(t *testing.T) {
	t.Parallel()

	mutator := &BoundaryValueMutator{}

	expr, err := parser.ParseExpr("a + b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{Type: boundaryValueType, Original: "10", Mutated: "11"}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-BasicLit node")
	}
}

func TestParseIntLit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int64
		ok    bool
	}{
		{input: "0", want: 0, ok: true},
		{input: "42", want: 42, ok: true},
		{input: "0xFF", want: 255, ok: true},
		{input: "0b101", want: 5, ok: true},
		{input: "0o17", want: 15, ok: true},
		{input: "1_000", want: 1000, ok: true},
		{input: "3.14", want: 0, ok: false},
		{input: "abc", want: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, ok := parseIntLit(tt.input)
			if ok != tt.ok {
				t.Fatalf("parseIntLit(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}

			if got != tt.want {
				t.Errorf("parseIntLit(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
