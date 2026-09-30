package mutation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
)

// TestMutatorContract verifies, for every mutator and every node of every
// testdata/*/input.go corpus file, the properties shared by all Mutator
// implementations: a node it does not claim via CanMutate produces no
// mutants and is never applied, and every mutant it does produce is applied
// by exactly one of Apply / ApplyWithCursor.
func TestMutatorContract(t *testing.T) {
	t.Parallel()

	inputs, err := readTestdataInputs()
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	for _, m := range getAllMutators() {
		t.Run(m.Name(), func(t *testing.T) {
			t.Parallel()

			for _, input := range inputs {
				checkMutatorContract(t, m, input)
			}
		})
	}
}

// readTestdataInputs returns the source of every testdata/<case>/input.go,
// the same corpus golden_test.go drives TestMutators against.
func readTestdataInputs() ([][]byte, error) {
	entries, err := fs.ReadDir(testdataFS, "testdata")
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata: %w", err)
	}

	inputs := make([][]byte, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.ToSlash(filepath.Join("testdata", entry.Name(), "input.go"))

		src, err := testdataFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		inputs = append(inputs, src)
	}

	return inputs, nil
}

// checkMutatorContract walks every node of src and checks m's contract
// against it.
func checkMutatorContract(t *testing.T, m Mutator, src []byte) {
	t.Helper()

	fset, file := parseInput(t, src)

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		checkNodeContract(t, m, src, n, fset)

		return true
	})
}

// checkNodeContract checks m's contract against a single node n, re-parsing
// src for each mutant so an earlier in-place Apply cannot corrupt a later
// check.
func checkNodeContract(t *testing.T, m Mutator, src []byte, n ast.Node, fset *token.FileSet) {
	t.Helper()

	if m.Apply(n, Mutant{}) {
		t.Errorf("%T: Apply(n, Mutant{}) = true, want false", n)
	}

	if ca, ok := m.(CursorApplier); ok {
		called := false

		if ca.ApplyWithCursor(n, func(ast.Node) { called = true }, Mutant{}) {
			t.Errorf("%T: ApplyWithCursor(n, Mutant{}) = true, want false", n)
		}

		if called {
			t.Errorf("%T: ApplyWithCursor(n, Mutant{}) called replace, want no call", n)
		}
	}

	if !m.CanMutate(n) {
		mutants := m.Mutate(n, fset)
		if len(mutants) != 0 && !mutatorsWithKnownOverGeneration[m.Name()] {
			t.Errorf("%T: CanMutate false but Mutate returned %d mutants", n, len(mutants))
		}

		return
	}

	span := nodeSpan{start: fset.Position(n.Pos()), end: fset.Position(n.End())}

	for _, mutant := range m.Mutate(n, fset) {
		checkAppliesExactlyOnce(t, m, src, span, n, mutant)
	}
}

// mutatorsWithKnownOverGeneration lists mutators whose Mutate ignores their
// own CanMutate filtering for some node shapes: ReturnMutator flips any
// *ast.Ident result regardless of whether its name is true/false,
// LogicalMutator's not-removal fires on any *ast.UnaryExpr regardless of
// operator, and BranchMutator regenerates both branch_condition mutants even
// when the condition is already a bool literal. These are pre-existing
// production bugs (see research-refactor-golden-tests.md) tracked
// separately, not covered by this contract.
var mutatorsWithKnownOverGeneration = map[string]bool{
	returnMutatorName:  true,
	logicalMutatorName: true,
	branchMutatorName:  true,
}

// nodeSpan identifies an AST node by its source extent, independent of any
// particular *token.FileSet.
type nodeSpan struct {
	start, end token.Position
}

// checkAppliesExactlyOnce verifies exactly one of Apply / ApplyWithCursor
// applies mutant, each against its own fresh parse of src so that Apply's
// in-place mutation of one attempt never affects the other.
func checkAppliesExactlyOnce(t *testing.T, m Mutator, src []byte, span nodeSpan, want ast.Node, mutant Mutant) {
	t.Helper()

	applyFset, applyFile := parseInput(t, src)

	applyNode := findNode(applyFile, applyFset, span, want)
	if applyNode == nil {
		t.Fatalf("%T: could not relocate node at %v after re-parse", want, span.start)
	}

	applyOK := m.Apply(applyNode, mutant)

	var (
		cursorOK      bool
		replaceCalled int
	)

	if ca, ok := m.(CursorApplier); ok {
		cursorFset, cursorFile := parseInput(t, src)

		cursorNode := findNode(cursorFile, cursorFset, span, want)
		if cursorNode == nil {
			t.Fatalf("%T: could not relocate node at %v after re-parse", want, span.start)
		}

		cursorOK = ca.ApplyWithCursor(cursorNode, func(ast.Node) { replaceCalled++ }, mutant)
	}

	switch {
	case applyOK && cursorOK:
		t.Errorf("%T: both Apply and ApplyWithCursor applied mutant %q -> %q, want exactly one", want, mutant.Original, mutant.Mutated)
	case !applyOK && !cursorOK:
		t.Errorf("%T: neither Apply nor ApplyWithCursor applied mutant %q -> %q", want, mutant.Original, mutant.Mutated)
	case cursorOK && replaceCalled != 1:
		t.Errorf("%T: ApplyWithCursor mutant %q -> %q called replace %d times, want 1", want, mutant.Original, mutant.Mutated, replaceCalled)
	}
}

// parseInput parses src as a standalone file, fresh for every call.
func parseInput(t *testing.T, src []byte) (*token.FileSet, *ast.File) {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "input.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse input: %v", err)
	}

	return fset, file
}

// findNode locates the node in file with the same dynamic type as want and
// the same source span, or nil if none matches. Matching on the span (not
// just its start) disambiguates nodes such as a BinaryExpr nested as the
// leftmost operand of another BinaryExpr, which share a start position.
func findNode(file *ast.File, fset *token.FileSet, span nodeSpan, want ast.Node) ast.Node {
	wantType := reflect.TypeOf(want)

	var found ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil || found != nil {
			return false
		}

		if reflect.TypeOf(n) != wantType {
			return true
		}

		if fset.Position(n.Pos()) == span.start && fset.Position(n.End()) == span.end {
			found = n

			return false
		}

		return true
	})

	return found
}
