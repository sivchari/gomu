package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestConditionalMutator_Mutate_AllOperators verifies the pre-filter
// over-generation contract: Mutate on a comparison always returns all five
// other comparison operators, valid or not; the TypeChecker prunes the
// invalid ones later (see testdata/issue33_err_nil_ordered in golden_test.go).
func TestConditionalMutator_Mutate_AllOperators(t *testing.T) {
	t.Parallel()

	expr, err := parser.ParseExpr("x == nil")
	if err != nil {
		t.Fatalf("failed to parse expression: %v", err)
	}

	be, ok := expr.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected *ast.BinaryExpr, got %T", expr)
	}

	mutator := &ConditionalMutator{}

	mutants := mutator.Mutate(be, token.NewFileSet())

	got := make(map[string]bool, len(mutants))
	for _, m := range mutants {
		got[m.Mutated] = true
	}

	for _, want := range []string{"!=", "<", "<=", ">", ">="} {
		if !got[want] {
			t.Errorf("Mutate(x == nil) missing operator %q", want)
		}
	}

	if len(mutants) != 5 {
		t.Errorf("Mutate(x == nil) = %d mutants, want 5", len(mutants))
	}
}
