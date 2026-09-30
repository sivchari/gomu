package mutation

import (
	"path/filepath"
	"testing"
)

// TestIssueRegressions verifies mutants that are invalid under Go's type
// system are pruned before they reach the caller. A golden alone could be
// -updated past a regression, so the forbidden mutated operators are
// asserted explicitly here.
func TestIssueRegressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		caseDir    string
		mutantType string
		forbidden  []string
	}{
		{
			// https://github.com/sivchari/gomu/issues/33: ordered comparisons
			// are not defined on the error interface.
			name:       "issue33 ordered comparison on error interface",
			caseDir:    "issue33_err_nil_ordered",
			mutantType: conditionalBinaryType,
			forbidden:  []string{"<", "<=", ">", ">="},
		},
		{
			// https://github.com/sivchari/gomu/issues/34: comparison operators
			// are not defined between bool operands.
			name:       "issue34 logical && turned into comparison",
			caseDir:    "issue34_logical_and_bool",
			mutantType: logicalBinaryType,
			forbidden:  []string{"<", "<=", ">", ">=", "==", "!="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create mutation engine: %v", err)
			}

			mutants, err := engine.GenerateMutants(filepath.Join("testdata", tt.caseDir, "input.go"))
			if err != nil {
				t.Fatalf("failed to generate mutants: %v", err)
			}

			for _, m := range mutants {
				if m.Type != tt.mutantType {
					continue
				}

				for _, forbidden := range tt.forbidden {
					if m.Mutated == forbidden {
						t.Errorf("found forbidden mutant %q -> %q (type %s)", m.Original, m.Mutated, m.Type)
					}
				}
			}
		})
	}
}
