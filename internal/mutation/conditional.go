package mutation

import (
	"fmt"
	"go/ast"
	"go/token"
)

// ConditionalMutator mutates conditional operators
type ConditionalMutator struct{}

func (m *ConditionalMutator) Name() string {
	return "conditional"
}

func (m *ConditionalMutator) CanMutate(node ast.Node) bool {
	if expr, ok := node.(*ast.BinaryExpr); ok {
		return m.isConditionalOp(expr.Op)
	}
	return false
}

func (m *ConditionalMutator) Mutate(node ast.Node, fset *token.FileSet) []Mutant {
	var mutants []Mutant
	pos := fset.Position(node.Pos())

	if expr, ok := node.(*ast.BinaryExpr); ok {
		mutants = append(mutants, m.mutateBinaryExpr(expr, pos)...)
	}

	return mutants
}

func (m *ConditionalMutator) mutateBinaryExpr(expr *ast.BinaryExpr, pos token.Position) []Mutant {
	var mutants []Mutant

	mutations := m.getConditionalMutations(expr.Op)
	for _, newOp := range mutations {
		mutants = append(mutants, Mutant{
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "conditional",
			Original:    expr.Op.String(),
			Mutated:     newOp.String(),
			Description: fmt.Sprintf("Replace %s with %s", expr.Op.String(), newOp.String()),
		})
	}

	return mutants
}

func (m *ConditionalMutator) isConditionalOp(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func (m *ConditionalMutator) getConditionalMutations(op token.Token) []token.Token {
	switch op {
	case token.EQL: // ==
		return []token.Token{token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ}
	case token.NEQ: // !=
		return []token.Token{token.EQL, token.LSS, token.LEQ, token.GTR, token.GEQ}
	case token.LSS: // <
		return []token.Token{token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ}
	case token.LEQ: // <=
		return []token.Token{token.LSS, token.GTR, token.GEQ, token.EQL, token.NEQ}
	case token.GTR: // >
		return []token.Token{token.GEQ, token.LSS, token.LEQ, token.EQL, token.NEQ}
	case token.GEQ: // >=
		return []token.Token{token.GTR, token.LSS, token.LEQ, token.EQL, token.NEQ}
	default:
		return nil
	}
}