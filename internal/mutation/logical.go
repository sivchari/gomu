package mutation

import (
	"fmt"
	"go/ast"
	"go/token"
)

const (
	logicalMutatorName    = "logical"
	logicalBinaryType     = "logical_binary"
	logicalNotRemovalType = "logical_not_removal"
)

// LogicalMutator mutates logical operators.
type LogicalMutator struct {
}

// Name returns the name of the mutator.
func (m *LogicalMutator) Name() string {
	return logicalMutatorName
}

// CanMutate returns true if the node can be mutated by this mutator.
func (m *LogicalMutator) CanMutate(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		return m.isLogicalOp(n.Op)
	case *ast.UnaryExpr:
		return n.Op == token.NOT
	}

	return false
}

// Mutate generates mutants for the given node.
func (m *LogicalMutator) Mutate(node ast.Node, fset *token.FileSet) []Mutant {
	var mutants []Mutant

	pos := fset.Position(node.Pos())

	switch n := node.(type) {
	case *ast.BinaryExpr:
		mutants = append(mutants, m.mutateBinaryExpr(n, pos)...)
	case *ast.UnaryExpr:
		mutants = append(mutants, m.mutateUnaryExpr(n, pos)...)
	}

	return mutants
}

func (m *LogicalMutator) mutateBinaryExpr(expr *ast.BinaryExpr, pos token.Position) []Mutant {
	mutations := m.getLogicalMutations(expr.Op)

	// Generate all mutations - validation will be done at compile time
	// No pre-filtering based on type safety

	mutants := make([]Mutant, 0, len(mutations))

	for _, newOp := range mutations {
		mutants = append(mutants, Mutant{
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "logical_binary",
			Original:    expr.Op.String(),
			Mutated:     newOp.String(),
			Description: fmt.Sprintf("Replace %s with %s", expr.Op.String(), newOp.String()),
		})
	}

	return mutants
}

func (m *LogicalMutator) mutateUnaryExpr(expr *ast.UnaryExpr, pos token.Position) []Mutant {
	var mutants []Mutant

	if expr.Op == token.NOT {
		// Generate all mutations - validation will be done at compile time
		// Remove the NOT operator
		mutants = append(mutants, Mutant{
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "logical_not_removal",
			Original:    "!",
			Mutated:     "",
			Description: "Remove ! (NOT) operator",
		})
	}

	return mutants
}

func (m *LogicalMutator) isLogicalOp(op token.Token) bool {
	switch op {
	case token.LAND, token.LOR:
		return true
	default:
		return false
	}
}

func (m *LogicalMutator) getLogicalMutations(op token.Token) []token.Token {
	switch op {
	case token.LAND: // &&
		return []token.Token{token.LOR}
	case token.LOR: // ||
		return []token.Token{token.LAND}
	default:
		return nil
	}
}

// Apply applies the mutation to the given AST node.
func (m *LogicalMutator) Apply(node ast.Node, mutant Mutant) bool {
	switch mutant.Type {
	case logicalBinaryType:
		return m.applyBinary(node, mutant)
	case logicalNotRemovalType:
		return m.applyNotRemoval(node, mutant)
	}

	return false
}

// applyBinary applies binary operator mutation.
func (m *LogicalMutator) applyBinary(node ast.Node, mutant Mutant) bool {
	if expr, ok := node.(*ast.BinaryExpr); ok {
		newOp := m.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			expr.Op = newOp

			return true
		}
	}

	return false
}

// applyNotRemoval applies NOT removal mutation.
func (m *LogicalMutator) applyNotRemoval(_ ast.Node, _ Mutant) bool {
	// For NOT removal, we need to replace the unary expression with its operand
	// This is more complex and requires parent node manipulation
	// For now, we'll return false to indicate this mutation type isn't fully implemented
	return false
}

// stringToToken converts string representation to token.Token for logical operations.
func (m *LogicalMutator) stringToToken(s string) token.Token {
	switch s {
	case "&&":
		return token.LAND
	case "||":
		return token.LOR
	default:
		return token.ILLEGAL
	}
}
