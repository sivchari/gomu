package mutation

import (
	"go/ast"
	"testing"
)

func TestApplyMutantToNodeWithNilNode(t *testing.T) {
	tests := []struct {
		name         string
		mutationType string
		expectResult bool
	}{
		{"arithmetic_binary", "arithmetic_binary", false},
		{"conditional_binary", "conditional_binary", false},
		{"logical_binary", "logical_binary", false},
		{"unknown_type", "unknown_type", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use nil node - method checks type anyway
			result := applyMutantToNode(getAllMutators(), nil, func(_ ast.Node) {}, Mutant{
				Type:    tt.mutationType,
				Mutated: "+",
			})

			if result != tt.expectResult {
				t.Errorf("expected %v, got %v", tt.expectResult, result)
			}
		})
	}
}
