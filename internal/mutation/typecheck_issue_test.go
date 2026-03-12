package mutation

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIssue33_ErrNilOrderedComparisons verifies that mutants of the form
// "err < nil", "err <= nil", "err > nil", "err >= nil" are never generated
// when err is of type error (an interface). These mutations are invalid because
// ordered comparison operators are not defined on interface types.
//
// Regression test for: https://github.com/sivchari/gomu/issues/33
func TestIssue33_ErrNilOrderedComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
	}{
		{
			name: "if err != nil return err",
			code: "package example\n\nfunc process(err error) error {\n\tif err != nil {\n\t\treturn err\n\t}\n\treturn nil\n}\n",
		},
		{
			name: "err == nil early return",
			code: "package example\n\nfunc process(err error) bool {\n\treturn err == nil\n}\n",
		},
	}

	forbiddenMutations := []string{"<", "<=", ">", ">="}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "example.go")

			if err := os.WriteFile(testFile, []byte(tt.code), 0600); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create mutation engine: %v", err)
			}

			mutants, err := engine.GenerateMutants(testFile)
			if err != nil {
				t.Fatalf("failed to generate mutants: %v", err)
			}

			for _, m := range mutants {
				t.Logf("mutant: original=%s mutated=%s type=%s", m.Original, m.Mutated, m.Type)
			}

			for _, m := range mutants {
				if m.Type != conditionalBinaryType {
					continue
				}

				for _, forbidden := range forbiddenMutations {
					if m.Mutated == forbidden {
						t.Errorf(
							"issue #33: found forbidden ordered comparison mutant %q -> %q for interface (error) type; this mutation is invalid",
							m.Original, m.Mutated,
						)
					}
				}
			}
		})
	}
}

// TestIssue34_LogicalAndNotMutatedToComparison verifies that the && operator in
// "err == nil && !force" is only mutated to || and never to comparison operators
// such as >, <, >=, <=, ==, or !=. The operands of && are boolean expressions,
// and comparison operators are not defined between bool values.
//
// Regression test for: https://github.com/sivchari/gomu/issues/34
func TestIssue34_LogicalAndNotMutatedToComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		code            string
		wantLogicalMuts []string
	}{
		{
			name:            "err == nil && !force",
			code:            "package example\n\nfunc check(err error, force bool) bool {\n\tresult := err == nil && !force\n\treturn result\n}\n",
			wantLogicalMuts: []string{"||"},
		},
		{
			name:            "multi-condition with err and bool",
			code:            "package example\n\nfunc check(err error, ready bool) bool {\n\treturn err == nil && ready\n}\n",
			wantLogicalMuts: []string{"||"},
		},
	}

	// Comparison operators must never appear as mutations of a logical && node.
	forbiddenFromLogical := []string{">", "<", ">=", "<=", "==", "!="}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "example.go")

			if err := os.WriteFile(testFile, []byte(tt.code), 0600); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			engine, err := New()
			if err != nil {
				t.Fatalf("failed to create mutation engine: %v", err)
			}

			mutants, err := engine.GenerateMutants(testFile)
			if err != nil {
				t.Fatalf("failed to generate mutants: %v", err)
			}

			for _, m := range mutants {
				t.Logf("mutant: original=%s mutated=%s type=%s", m.Original, m.Mutated, m.Type)
			}

			// Logical mutants (from &&) must only produce ||.
			for _, m := range mutants {
				if m.Type != logicalBinaryType {
					continue
				}

				if m.Original != "&&" {
					continue
				}

				for _, forbidden := range forbiddenFromLogical {
					if m.Mutated == forbidden {
						t.Errorf(
							"issue #34: logical && mutated to comparison operator %q -> %q; this mutation is type-invalid",
							m.Original, m.Mutated,
						)
					}
				}
			}

			// Confirm that the expected || mutation is present for &&.
			logicalMuts := make(map[string]bool)
			for _, m := range mutants {
				if m.Type == logicalBinaryType && m.Original == "&&" {
					logicalMuts[m.Mutated] = true
				}
			}

			for _, want := range tt.wantLogicalMuts {
				if !logicalMuts[want] {
					t.Errorf("expected logical mutation && -> %s to be present, but it was not", want)
				}
			}
		})
	}
}
