package mutation

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"strconv"
)

const (
	boundaryValueMutatorName = "boundary_value"
	boundaryValueType        = "boundary_value"
)

// BoundaryValueMutator mutates integer literals to their boundary neighbours
// (N -> N+1 and N -> N-1), surfacing weak off-by-one / boundary tests.
//
// Relational operator boundary shifts (e.g. < -> <=) are already covered by
// the conditional mutator, so this mutator focuses on the literal side of the
// boundary to avoid generating duplicate mutants.
type BoundaryValueMutator struct {
}

// Name returns the name of the mutator.
func (m *BoundaryValueMutator) Name() string {
	return boundaryValueMutatorName
}

// CanMutate returns true if the node can be mutated by this mutator.
func (m *BoundaryValueMutator) CanMutate(node ast.Node) bool {
	lit, ok := node.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}

	_, ok = parseIntLit(lit.Value)

	return ok
}

// Mutate generates mutants for the given node.
func (m *BoundaryValueMutator) Mutate(node ast.Node, fset *token.FileSet) []Mutant {
	lit, ok := node.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return nil
	}

	value, ok := parseIntLit(lit.Value)
	if !ok {
		return nil
	}

	pos := fset.Position(node.Pos())

	var mutants []Mutant

	// N -> N+1 (skip on overflow).
	if value != math.MaxInt64 {
		mutants = append(mutants, m.newMutant(lit.Value, strconv.FormatInt(value+1, 10), pos))
	}

	// N -> N-1 (skip when the result would become a negative literal, which is
	// not representable as a single integer literal in the AST).
	if value >= 1 {
		mutants = append(mutants, m.newMutant(lit.Value, strconv.FormatInt(value-1, 10), pos))
	}

	return mutants
}

func (m *BoundaryValueMutator) newMutant(original, mutated string, pos token.Position) Mutant {
	return Mutant{
		Line:        pos.Line,
		Column:      pos.Column,
		Type:        boundaryValueType,
		Original:    original,
		Mutated:     mutated,
		Description: fmt.Sprintf("Replace integer literal %s with %s", original, mutated),
	}
}

// Apply applies the mutation to the given AST node.
func (m *BoundaryValueMutator) Apply(node ast.Node, mutant Mutant) bool {
	if mutant.Type != boundaryValueType {
		return false
	}

	lit, ok := node.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}

	if lit.Value != mutant.Original {
		return false
	}

	lit.Value = mutant.Mutated

	return true
}

// parseIntLit parses a Go integer literal (supporting base prefixes and digit
// separators) into an int64. It reports false when the literal cannot be
// represented as an int64.
func parseIntLit(s string) (int64, bool) {
	value, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}
