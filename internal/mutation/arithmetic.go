// Package mutation provides mutation testing functionality.
package mutation

import (
	"fmt"
	"go/ast"
	"go/token"
)

const arithmeticMutatorName = "arithmetic"

// ArithmeticMutator mutates arithmetic operators.
type ArithmeticMutator struct{}

// Name returns the name of the mutator.
func (m *ArithmeticMutator) Name() string {
	return arithmeticMutatorName
}

// CanMutate returns true if the node can be mutated by this mutator.
func (m *ArithmeticMutator) CanMutate(node ast.Node) bool {
	switch node.(type) {
	case *ast.BinaryExpr:
		return true
	case *ast.AssignStmt:
		return true
	case *ast.IncDecStmt:
		return true
	}

	return false
}

// Mutate generates mutants for the given node.
func (m *ArithmeticMutator) Mutate(node ast.Node, fset *token.FileSet) []Mutant {
	var mutants []Mutant

	pos := fset.Position(node.Pos())

	switch n := node.(type) {
	case *ast.BinaryExpr:
		mutants = append(mutants, m.mutateBinaryExpr(n, pos)...)
	case *ast.AssignStmt:
		mutants = append(mutants, m.mutateAssignStmt(n, pos)...)
	case *ast.IncDecStmt:
		mutants = append(mutants, m.mutateIncDecStmt(n, pos)...)
	}

	return mutants
}

func (m *ArithmeticMutator) mutateBinaryExpr(expr *ast.BinaryExpr, pos token.Position) []Mutant {
	mutations := m.getArithmeticMutations(expr.Op)
	mutants := make([]Mutant, 0, len(mutations))

	for _, newOp := range mutations {
		mutants = append(mutants, Mutant{
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "arithmetic_binary",
			Original:    expr.Op.String(),
			Mutated:     newOp.String(),
			Description: fmt.Sprintf("Replace %s with %s", expr.Op.String(), newOp.String()),
		})
	}

	return mutants
}

func (m *ArithmeticMutator) mutateAssignStmt(stmt *ast.AssignStmt, pos token.Position) []Mutant {
	op := stmt.Tok
	mutations := m.getAssignMutations(op)
	mutants := make([]Mutant, 0, len(mutations))

	for _, newOp := range mutations {
		mutants = append(mutants, Mutant{
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "arithmetic_assign",
			Original:    op.String(),
			Mutated:     newOp.String(),
			Description: fmt.Sprintf("Replace %s with %s", op.String(), newOp.String()),
		})
	}

	return mutants
}

func (m *ArithmeticMutator) mutateIncDecStmt(stmt *ast.IncDecStmt, pos token.Position) []Mutant {
	var mutants []Mutant

	var newOp token.Token

	var desc string

	if stmt.Tok == token.INC {
		newOp = token.DEC
		desc = "Replace ++ with --"
	} else {
		newOp = token.INC
		desc = "Replace -- with ++"
	}

	mutants = append(mutants, Mutant{
		Line:        pos.Line,
		Column:      pos.Column,
		Type:        "arithmetic_incdec",
		Original:    stmt.Tok.String(),
		Mutated:     newOp.String(),
		Description: desc,
	})

	return mutants
}

func (m *ArithmeticMutator) getArithmeticMutations(op token.Token) []token.Token {
	switch op {
	case token.ADD:
		return []token.Token{token.SUB, token.MUL, token.QUO}
	case token.SUB:
		return []token.Token{token.ADD, token.MUL, token.QUO}
	case token.MUL:
		return []token.Token{token.ADD, token.SUB, token.QUO, token.REM}
	case token.QUO:
		return []token.Token{token.ADD, token.SUB, token.MUL, token.REM}
	case token.REM:
		return []token.Token{token.ADD, token.SUB, token.MUL, token.QUO}
	default:
		return nil
	}
}

func (m *ArithmeticMutator) getAssignMutations(op token.Token) []token.Token {
	switch op {
	case token.ADD_ASSIGN:
		return []token.Token{token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN}
	case token.SUB_ASSIGN:
		return []token.Token{token.ADD_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN}
	case token.MUL_ASSIGN:
		return []token.Token{token.ADD_ASSIGN, token.SUB_ASSIGN, token.QUO_ASSIGN}
	case token.QUO_ASSIGN:
		return []token.Token{token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN}
	default:
		return nil
	}
}
