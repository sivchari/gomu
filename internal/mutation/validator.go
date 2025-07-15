package mutation

import (
	"go/ast"
	"go/token"
	"go/types"
)

// TypeValidator provides type-safe mutation validation.
type TypeValidator struct {
	info *types.Info
}

// NewTypeValidator creates a new type validator.
func NewTypeValidator(info *types.Info) *TypeValidator {
	return &TypeValidator{info: info}
}

// IsValidArithmeticMutation checks if an arithmetic mutation is type-safe.
func (v *TypeValidator) IsValidArithmeticMutation(expr *ast.BinaryExpr, newOp token.Token) bool {
	if v.info == nil {
		return true // Fall back to allow all mutations if no type info
	}

	// Get the type of the left operand
	leftType := v.info.TypeOf(expr.X)
	if leftType == nil {
		return true // Allow if type is unknown
	}

	// Get the underlying type
	underlying := leftType.Underlying()

	// Check for problematic combinations
	switch underlying := underlying.(type) {
	case *types.Basic:
		// Float types don't support modulo operator
		if underlying.Info()&types.IsFloat != 0 && newOp == token.REM {
			return false
		}

		// String types only support + operator
		if underlying.Info()&types.IsString != 0 && newOp != token.ADD {
			return false
		}

		// Complex types only support + - * /
		if underlying.Info()&types.IsComplex != 0 &&
			(newOp == token.REM) {
			return false
		}

	case *types.Slice, *types.Array:
		// Only + operator is valid for slices (concatenation)
		if newOp != token.ADD {
			return false
		}

	case *types.Pointer:
		// Pointers don't support arithmetic operations
		return false

	case *types.Interface, *types.Struct:
		// Interfaces and structs don't support arithmetic operations
		return false
	}

	return true
}

// IsValidConditionalMutation checks if a conditional mutation is type-safe.
func (v *TypeValidator) IsValidConditionalMutation(expr *ast.BinaryExpr, newOp token.Token) bool {
	if v.info == nil {
		return true // Fall back to allow all mutations if no type info
	}

	// Get types of both operands
	leftType := v.info.TypeOf(expr.X)
	rightType := v.info.TypeOf(expr.Y)

	if leftType == nil || rightType == nil {
		return true // Allow if types are unknown
	}

	// Check if types are comparable
	if !types.Comparable(leftType) || !types.Comparable(rightType) {
		// Only == and != are valid for non-comparable types
		return newOp == token.EQL || newOp == token.NEQ
	}

	// Check if ordered comparison is valid
	if newOp == token.LSS || newOp == token.LEQ || newOp == token.GTR || newOp == token.GEQ {
		return v.isOrderedType(leftType) && v.isOrderedType(rightType)
	}

	return true
}

// IsValidLogicalMutation checks if a logical mutation is type-safe.
func (v *TypeValidator) IsValidLogicalMutation(expr ast.Expr, _ token.Token) bool {
	if v.info == nil {
		return true // Fall back to allow all mutations if no type info
	}

	// Get the type of the expression
	exprType := v.info.TypeOf(expr)
	if exprType == nil {
		return true // Allow if type is unknown
	}

	// Check if the expression is boolean
	underlying := exprType.Underlying()
	if basic, ok := underlying.(*types.Basic); ok {
		return basic.Kind() == types.Bool
	}

	return false
}

// isOrderedType checks if a type supports ordering operations (<, <=, >, >=).
func (v *TypeValidator) isOrderedType(t types.Type) bool {
	underlying := t.Underlying()

	if basic, ok := underlying.(*types.Basic); ok {
		info := basic.Info()

		return info&types.IsOrdered != 0
	}

	return false
}

// IsValidUnaryMutation checks if a unary mutation is type-safe.
func (v *TypeValidator) IsValidUnaryMutation(expr *ast.UnaryExpr, remove bool) bool {
	if v.info == nil {
		return true // Fall back to allow all mutations if no type info
	}

	// For NOT operator removal, the operand must be boolean
	if expr.Op == token.NOT && remove {
		exprType := v.info.TypeOf(expr.X)
		if exprType == nil {
			return true // Allow if type is unknown
		}

		underlying := exprType.Underlying()
		if basic, ok := underlying.(*types.Basic); ok {
			return basic.Kind() == types.Bool
		}

		return false
	}

	return true
}

// GetSafeMutations returns only type-safe mutations for a given expression.
func (v *TypeValidator) GetSafeMutations(expr ast.Expr, _ token.Token, candidates []token.Token) []token.Token {
	if v.info == nil {
		return candidates // Return all if no type info
	}

	var safe []token.Token

	if e, ok := expr.(*ast.BinaryExpr); ok {
		for _, candidate := range candidates {
			switch {
			case v.isArithmeticOp(candidate):
				if v.IsValidArithmeticMutation(e, candidate) {
					safe = append(safe, candidate)
				}
			case v.isConditionalOp(candidate):
				if v.IsValidConditionalMutation(e, candidate) {
					safe = append(safe, candidate)
				}
			case v.isLogicalOp(candidate):
				if v.IsValidLogicalMutation(e, candidate) {
					safe = append(safe, candidate)
				}
			}
		}
	}

	return safe
}

// Helper methods to categorize operators.
func (v *TypeValidator) isArithmeticOp(op token.Token) bool {
	switch op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
		return true
	default:
		return false
	}
}

func (v *TypeValidator) isConditionalOp(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func (v *TypeValidator) isLogicalOp(op token.Token) bool {
	switch op {
	case token.LAND, token.LOR:
		return true
	default:
		return false
	}
}
