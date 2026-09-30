package execution

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sivchari/gomu/internal/mutation"
)

//go:embed testdata/branch.go
var branchSrc string

//go:embed testdata/branch_true.go
var branchTrueSrc string

//go:embed testdata/errorhandling.go
var errorhandlingSrc string

//go:embed testdata/errorhandling_nil.go
var errorhandlingNilSrc string

//go:embed testdata/return.go
var returnSrc string

//go:embed testdata/return_false.go
var returnFalseSrc string

//go:embed testdata/arithmetic.go
var arithmeticSrc string

//go:embed testdata/arithmetic_sub.go
var arithmeticSubSrc string

//go:embed testdata/bitwise.go
var bitwiseSrc string

//go:embed testdata/bitwise_or.go
var bitwiseOrSrc string

//go:embed testdata/conditional.go
var conditionalSrc string

//go:embed testdata/conditional_neq.go
var conditionalNeqSrc string

//go:embed testdata/logical.go
var logicalSrc string

//go:embed testdata/logical_or.go
var logicalOrSrc string

//go:embed testdata/logical_not.go
var logicalNotSrc string

//go:embed testdata/logical_not_removed.go
var logicalNotRemovedSrc string

//go:embed testdata/invertnegatives.go
var invertNegativesSrc string

//go:embed testdata/invertnegatives_plus.go
var invertNegativesPlusSrc string

//go:embed testdata/selfassign.go
var selfAssignSrc string

//go:embed testdata/selfassign_simple.go
var selfAssignSimpleSrc string

//go:embed testdata/breakcontinue.go
var breakContinueSrc string

//go:embed testdata/breakcontinue_continue.go
var breakContinueContinueSrc string

//go:embed testdata/boundary.go
var boundarySrc string

//go:embed testdata/boundary_inc.go
var boundaryIncSrc string

//go:embed testdata/stringliteral.go
var stringLiteralSrc string

//go:embed testdata/stringliteral_empty.go
var stringLiteralEmptySrc string

//go:embed testdata/loopcondition.go
var loopConditionSrc string

//go:embed testdata/loopcondition_false.go
var loopConditionFalseSrc string

//go:embed testdata/emptyblock.go
var emptyBlockSrc string

//go:embed testdata/emptyblock_empty.go
var emptyBlockEmptySrc string

//go:embed testdata/assignremoval.go
var assignRemovalSrc string

//go:embed testdata/assignremoval_removed.go
var assignRemovalRemovedSrc string

//go:embed testdata/exprremoval.go
var exprRemovalSrc string

//go:embed testdata/exprremoval_removed.go
var exprRemovalRemovedSrc string

//go:embed testdata/stmtremoval.go
var stmtRemovalSrc string

//go:embed testdata/stmtremoval_removed.go
var stmtRemovalRemovedSrc string

//go:embed testdata/switch.go
var switchSrc string

//go:embed testdata/switch_default_removed.go
var switchDefaultRemovedSrc string

func TestOverlayMutatorLifecycle(t *testing.T) {
	mutator, err := NewOverlayMutator()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mutator == nil {
		t.Fatal("mutator should not be nil")
	}

	if mutator.baseDir == "" {
		t.Error("base directory should not be empty")
	}

	if _, err := os.Stat(mutator.baseDir); err != nil {
		t.Errorf("base directory should exist before cleanup: %v", err)
	}

	if err := mutator.Cleanup(); err != nil {
		t.Errorf("cleanup failed: %v", err)
	}

	if _, err := os.Stat(mutator.baseDir); !os.IsNotExist(err) {
		t.Error("base directory should not exist after cleanup")
	}
}

func TestPrepareMutation(t *testing.T) {
	tempDir := createOverlayTestProject(t)

	tests := []struct {
		name        string
		mutant      mutation.Mutant
		expectError bool
		validate    func(t *testing.T, ctx *MutationContext)
	}{
		{
			name: "prepares mutation successfully",
			mutant: mutation.Mutant{
				ID:       "test-1",
				Type:     "arithmetic_binary",
				FilePath: filepath.Join(tempDir, "calc.go"),
				Line:     4,
				Column:   9,
				Original: "+",
				Mutated:  "-",
			},
			expectError: false,
			validate: func(t *testing.T, ctx *MutationContext) {
				// Verify mutated file exists
				if _, err := os.Stat(ctx.MutatedPath); err != nil {
					t.Errorf("mutated file should exist: %v", err)
				}

				// Verify overlay.json exists
				if _, err := os.Stat(ctx.OverlayPath); err != nil {
					t.Errorf("overlay.json should exist: %v", err)
				}

				// Verify overlay.json content
				data, err := os.ReadFile(ctx.OverlayPath)
				if err != nil {
					t.Errorf("failed to read overlay.json: %v", err)
				}

				var config OverlayConfig
				if err := json.Unmarshal(data, &config); err != nil {
					t.Errorf("failed to parse overlay.json: %v", err)
				}

				if len(config.Replace) != 1 {
					t.Errorf("expected 1 replacement, got %d", len(config.Replace))
				}

				// Verify original file is not modified
				originalContent, _ := os.ReadFile(ctx.OriginalPath)
				if strings.Contains(string(originalContent), "a - b") {
					t.Error("original file should not be modified")
				}

				// Verify mutated file has the mutation
				mutatedContent, _ := os.ReadFile(ctx.MutatedPath)
				if !strings.Contains(string(mutatedContent), "a - b") {
					t.Error("mutated file should contain the mutation")
				}
			},
		},
		{
			name: "fails with non-existent file",
			mutant: mutation.Mutant{
				ID:       "test-2",
				Type:     "arithmetic_binary",
				FilePath: "/nonexistent/file.go",
				Line:     1,
				Column:   1,
				Original: "+",
				Mutated:  "-",
			},
			expectError: true,
		},
		{
			name: "fails with invalid mutation target",
			mutant: mutation.Mutant{
				ID:       "test-3",
				Type:     "arithmetic_binary",
				FilePath: filepath.Join(tempDir, "calc.go"),
				Line:     999, // Non-existent line
				Column:   1,
				Original: "+",
				Mutated:  "-",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewOverlayMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			ctx, err := mutator.PrepareMutation(tt.mutant)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")

				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if tt.validate != nil && ctx != nil {
				tt.validate(t, ctx)
			}

			if ctx != nil {
				mutator.CleanupMutation(ctx)
			}
		})
	}
}

func TestCleanupMutation(t *testing.T) {
	tempDir := createOverlayTestProject(t)

	tests := []struct {
		name        string
		setupCtx    func(t *testing.T, mutator *OverlayMutator) *MutationContext
		expectError bool
	}{
		{
			name: "cleanup removes mutant directory",
			setupCtx: func(t *testing.T, mutator *OverlayMutator) *MutationContext {
				mutant := mutation.Mutant{
					ID:       "cleanup-test",
					Type:     "arithmetic_binary",
					FilePath: filepath.Join(tempDir, "calc.go"),
					Line:     4,
					Column:   9,
					Original: "+",
					Mutated:  "-",
				}

				ctx, err := mutator.PrepareMutation(mutant)
				if err != nil {
					t.Fatalf("failed to prepare mutation: %v", err)
				}

				return ctx
			},
			expectError: false,
		},
		{
			name: "cleanup with nil context succeeds",
			setupCtx: func(_ *testing.T, _ *OverlayMutator) *MutationContext {
				return nil
			},
			expectError: false,
		},
		{
			name: "cleanup with empty mutant dir succeeds",
			setupCtx: func(_ *testing.T, _ *OverlayMutator) *MutationContext {
				return &MutationContext{}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewOverlayMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			ctx := tt.setupCtx(t, mutator)

			var mutantDir string

			if ctx != nil {
				mutantDir = ctx.MutantDir
			}

			err = mutator.CleanupMutation(ctx)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify mutant directory is removed
			if mutantDir != "" {
				if _, err := os.Stat(mutantDir); !os.IsNotExist(err) {
					t.Error("mutant directory should be removed after cleanup")
				}
			}
		})
	}
}

func TestCreateMutatedFileVariousMutations(t *testing.T) {
	tempDir := createOverlayTestProject(t)

	tests := []struct {
		name           string
		mutant         mutation.Mutant
		expectError    bool
		expectedChange string
	}{
		{
			name: "arithmetic binary mutation",
			mutant: mutation.Mutant{
				ID:       "arith-1",
				Type:     "arithmetic_binary",
				FilePath: filepath.Join(tempDir, "calc.go"),
				Line:     4,
				Column:   9,
				Original: "+",
				Mutated:  "-",
			},
			expectError:    false,
			expectedChange: "a - b",
		},
		{
			name: "conditional mutation",
			mutant: mutation.Mutant{
				ID:       "cond-1",
				Type:     "conditional_binary",
				FilePath: filepath.Join(tempDir, "compare.go"),
				Line:     4,
				Column:   9,
				Original: ">",
				Mutated:  "<",
			},
			expectError:    false,
			expectedChange: "a < b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := NewOverlayMutator()
			if err != nil {
				t.Fatalf("failed to create mutator: %v", err)
			}
			defer mutator.Cleanup()

			ctx, err := mutator.PrepareMutation(tt.mutant)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")

				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if ctx != nil {
				defer mutator.CleanupMutation(ctx)

				// Verify mutation was applied to mutated file
				mutatedContent, err := os.ReadFile(ctx.MutatedPath)
				if err != nil {
					t.Errorf("failed to read mutated file: %v", err)

					return
				}

				if !strings.Contains(string(mutatedContent), tt.expectedChange) {
					t.Errorf("expected mutated content to contain %q, got: %s", tt.expectedChange, string(mutatedContent))
				}

				// Verify original file is unchanged
				originalContent, err := os.ReadFile(tt.mutant.FilePath)
				if err != nil {
					t.Errorf("failed to read original file: %v", err)

					return
				}

				if strings.Contains(string(originalContent), tt.expectedChange) {
					t.Error("original file should not contain the mutation")
				}
			}
		})
	}
}

func TestMultipleMutationsOnSameFile(t *testing.T) {
	tempDir := createOverlayTestProject(t)

	mutator, err := NewOverlayMutator()
	if err != nil {
		t.Fatalf("failed to create mutator: %v", err)
	}
	defer mutator.Cleanup()

	// Create multiple mutations on the same file
	mutants := []mutation.Mutant{
		{
			ID:       "multi-1",
			Type:     "arithmetic_binary",
			FilePath: filepath.Join(tempDir, "calc.go"),
			Line:     4,
			Column:   9,
			Original: "+",
			Mutated:  "-",
		},
		{
			ID:       "multi-2",
			Type:     "arithmetic_binary",
			FilePath: filepath.Join(tempDir, "calc.go"),
			Line:     4,
			Column:   9,
			Original: "+",
			Mutated:  "*",
		},
	}

	contexts := make([]*MutationContext, len(mutants))

	// Prepare all mutations
	for i, mutant := range mutants {
		ctx, err := mutator.PrepareMutation(mutant)
		if err != nil {
			t.Errorf("failed to prepare mutation %d: %v", i, err)

			continue
		}

		contexts[i] = ctx
	}

	// Verify each has independent mutated files
	for i, ctx := range contexts {
		if ctx == nil {
			continue
		}

		// Each should have its own mutant directory
		if _, err := os.Stat(ctx.MutantDir); err != nil {
			t.Errorf("mutant %d directory should exist: %v", i, err)
		}

		// Each should have its own overlay.json
		if _, err := os.Stat(ctx.OverlayPath); err != nil {
			t.Errorf("mutant %d overlay.json should exist: %v", i, err)
		}
	}

	// Cleanup all
	for _, ctx := range contexts {
		if ctx != nil {
			mutator.CleanupMutation(ctx)
		}
	}

	// Verify original file is still unchanged
	originalContent, _ := os.ReadFile(filepath.Join(tempDir, "calc.go"))
	if !strings.Contains(string(originalContent), "a + b") {
		t.Error("original file should still contain 'a + b'")
	}
}

// TestMutateAndApplyIntegration verifies that mutants generated by Mutate()
// can be successfully applied by PrepareMutation(), and that the mutated file
// matches the expected output exactly. This is the core integration test for
// the two-pass architecture: positions recorded in Pass 1 must match nodes
// found in Pass 2.
func TestMutateAndApplyIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		src        string
		mutantType string
		original   string
		mutated    string
		want       string
	}{
		{
			name:       "branch condition replaced with true",
			src:        branchSrc,
			mutantType: "branch_condition",
			original:   "n == 0",
			mutated:    "true",
			want:       branchTrueSrc,
		},
		{
			name:       "error return replaced with nil",
			src:        errorhandlingSrc,
			mutantType: "error_nilify",
			original:   "err",
			mutated:    "nil",
			want:       errorhandlingNilSrc,
		},
		{
			name:       "return true replaced with false",
			src:        returnSrc,
			mutantType: "return_bool_literal",
			original:   "true",
			mutated:    "false",
			want:       returnFalseSrc,
		},
		{
			name:       "arithmetic add replaced with sub",
			src:        arithmeticSrc,
			mutantType: "arithmetic_binary",
			original:   "+",
			mutated:    "-",
			want:       arithmeticSubSrc,
		},
		{
			name:       "bitwise and replaced with or",
			src:        bitwiseSrc,
			mutantType: "bitwise_binary",
			original:   "&",
			mutated:    "|",
			want:       bitwiseOrSrc,
		},
		{
			name:       "conditional eq replaced with neq",
			src:        conditionalSrc,
			mutantType: "conditional_binary",
			original:   "==",
			mutated:    "!=",
			want:       conditionalNeqSrc,
		},
		{
			name:       "logical and replaced with or",
			src:        logicalSrc,
			mutantType: "logical_binary",
			original:   "&&",
			mutated:    "||",
			want:       logicalOrSrc,
		},
		{
			name:       "logical not removed",
			src:        logicalNotSrc,
			mutantType: "logical_not_removal",
			original:   "!",
			mutated:    "",
			want:       logicalNotRemovedSrc,
		},
		{
			name:       "unary minus inverted to plus",
			src:        invertNegativesSrc,
			mutantType: "invert_negatives",
			original:   "-",
			mutated:    "+",
			want:       invertNegativesPlusSrc,
		},
		{
			name:       "compound assignment replaced with simple assignment",
			src:        selfAssignSrc,
			mutantType: "remove_self_assignments",
			original:   "+=",
			mutated:    "=",
			want:       selfAssignSimpleSrc,
		},
		{
			name:       "break replaced with continue",
			src:        breakContinueSrc,
			mutantType: "break_continue",
			original:   "break",
			mutated:    "continue",
			want:       breakContinueContinueSrc,
		},
		{
			name:       "integer literal incremented to boundary",
			src:        boundarySrc,
			mutantType: "boundary_value",
			original:   "10",
			mutated:    "11",
			want:       boundaryIncSrc,
		},
		{
			name:       "non-empty string replaced with empty string",
			src:        stringLiteralSrc,
			mutantType: "string_literal",
			original:   `"hello"`,
			mutated:    `""`,
			want:       stringLiteralEmptySrc,
		},
		{
			name:       "loop condition replaced with false",
			src:        loopConditionSrc,
			mutantType: "loop_condition",
			original:   "i < n",
			mutated:    "false",
			want:       loopConditionFalseSrc,
		},
		{
			name:       "block body emptied",
			src:        emptyBlockSrc,
			mutantType: "empty_block",
			original:   "{...}",
			mutated:    "{}",
			want:       emptyBlockEmptySrc,
		},
		{
			name:       "assignment statement removed",
			src:        assignRemovalSrc,
			mutantType: "assignment_removal",
			original:   "total",
			mutated:    "<removed>",
			want:       assignRemovalRemovedSrc,
		},
		{
			name:       "expression statement removed",
			src:        exprRemovalSrc,
			mutantType: "expression_removal",
			original:   "ping()",
			mutated:    "<removed>",
			want:       exprRemovalRemovedSrc,
		},
		{
			name:       "statement removed",
			src:        stmtRemovalSrc,
			mutantType: "statement_removal",
			original:   "*counter++",
			mutated:    "<removed>",
			want:       stmtRemovalRemovedSrc,
		},
		{
			name:       "switch case body emptied",
			src:        switchSrc,
			mutantType: "switch_case_body_removal",
			original:   "default",
			mutated:    "<removed>",
			want:       switchDefaultRemovedSrc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()

			if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0600); err != nil {
				t.Fatalf("failed to create go.mod: %v", err)
			}

			srcPath := filepath.Join(tempDir, "main.go")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0600); err != nil {
				t.Fatalf("failed to create main.go: %v", err)
			}

			// Pass 1: generate mutants
			engine, err := mutation.New()
			if err != nil {
				t.Fatalf("failed to create mutation engine: %v", err)
			}

			mutants, err := engine.GenerateMutants(srcPath)
			if err != nil {
				t.Fatalf("failed to generate mutants: %v", err)
			}

			// Find the target mutant
			var target *mutation.Mutant

			for i := range mutants {
				if mutants[i].Type == tt.mutantType && mutants[i].Original == tt.original && mutants[i].Mutated == tt.mutated {
					target = &mutants[i]

					break
				}
			}

			if target == nil {
				t.Fatalf("mutant not found: type=%s original=%s mutated=%s\navailable mutants:", tt.mutantType, tt.original, tt.mutated)
			}

			// Pass 2: apply and compare
			overlay, err := NewOverlayMutator()
			if err != nil {
				t.Fatalf("failed to create overlay mutator: %v", err)
			}
			defer overlay.Cleanup()

			ctx, err := overlay.PrepareMutation(*target)
			if err != nil {
				t.Fatalf("PrepareMutation failed: %v", err)
			}
			defer overlay.CleanupMutation(ctx)

			got, err := os.ReadFile(ctx.MutatedPath)
			if err != nil {
				t.Fatalf("failed to read mutated file: %v", err)
			}

			if diff := cmp.Diff(tt.want, string(got)); diff != "" {
				t.Errorf("mutated file mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

const overlayCalcFileFixture = `package main

func Add(a, b int) int {
	return a + b
}
`

const overlayCompareFileFixture = `package main

func IsGreater(a, b int) bool {
	return a > b
}
`

const overlayMainTestFileFixture = `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2)
	if result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
}

func TestIsGreater(t *testing.T) {
	if !IsGreater(5, 3) {
		t.Error("Expected 5 > 3 to be true")
	}
}
`

// createOverlayTestProject creates a temporary test project for overlay tests.
func createOverlayTestProject(t *testing.T) string {
	return newTestModule(t, map[string]string{
		"calc.go":      overlayCalcFileFixture,
		"compare.go":   overlayCompareFileFixture,
		"main_test.go": overlayMainTestFileFixture,
	})
}
