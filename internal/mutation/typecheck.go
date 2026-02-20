package mutation

import (
	"go/ast"
	"go/token"
	"go/types"
)

// TypeChecker validates mutations against type information.
type TypeChecker struct {
	typeInfo *types.Info
}

// NewTypeChecker creates a new type checker.
func NewTypeChecker(typeInfo *types.Info) *TypeChecker {
	return &TypeChecker{typeInfo: typeInfo}
}

// IsValidMutation checks if a mutation is valid for the given node.
func (tc *TypeChecker) IsValidMutation(node ast.Node, mutant Mutant) bool {
	if tc.typeInfo == nil {
		// No type info available, assume valid
		return true
	}

	switch mutant.Type {
	case "arithmetic_binary":
		return tc.isValidArithmeticBinaryMutation(node, mutant)
	case "arithmetic_assign":
		return tc.isValidArithmeticAssignMutation(node, mutant)
	case "conditional_binary":
		return tc.isValidConditionalMutation(node, mutant)
	default:
		// Other mutation types are assumed valid
		return true
	}
}

// isValidArithmeticBinaryMutation checks if an arithmetic binary mutation is valid.
func (tc *TypeChecker) isValidArithmeticBinaryMutation(node ast.Node, mutant Mutant) bool {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	// Get the type of the left operand
	leftType := tc.getExprType(expr.X)
	if leftType == nil {
		return true // Can't determine type, assume valid
	}

	// Check if the mutation is valid for this type
	return tc.isArithmeticOpValidForType(leftType, mutant.Mutated)
}

// isValidArithmeticAssignMutation checks if an arithmetic assignment mutation is valid.
func (tc *TypeChecker) isValidArithmeticAssignMutation(node ast.Node, mutant Mutant) bool {
	stmt, ok := node.(*ast.AssignStmt)
	if !ok || len(stmt.Lhs) == 0 {
		return false
	}

	// Get the type of the left-hand side
	lhsType := tc.getExprType(stmt.Lhs[0])
	if lhsType == nil {
		return true // Can't determine type, assume valid
	}

	// Convert assign operator to binary operator for validation
	binaryOp := tc.assignToBinaryOp(mutant.Mutated)
	return tc.isArithmeticOpValidForType(lhsType, binaryOp)
}

// isValidConditionalMutation checks if a conditional mutation is valid.
func (tc *TypeChecker) isValidConditionalMutation(node ast.Node, mutant Mutant) bool {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	// Get the type of the left operand
	leftType := tc.getExprType(expr.X)
	if leftType == nil {
		return true // Can't determine type, assume valid
	}

	// Check if the new operator is valid for this type
	return tc.isComparisonOpValidForType(leftType, mutant.Mutated)
}

// getExprType returns the type of an expression.
func (tc *TypeChecker) getExprType(expr ast.Expr) types.Type {
	if tc.typeInfo == nil || tc.typeInfo.Types == nil {
		return nil
	}

	tv, ok := tc.typeInfo.Types[expr]
	if !ok {
		return nil
	}

	return tv.Type
}

// isArithmeticOpValidForType checks if an arithmetic operator is valid for a type.
func (tc *TypeChecker) isArithmeticOpValidForType(t types.Type, op string) bool {
	underlying := t.Underlying()

	switch underlying := underlying.(type) {
	case *types.Basic:
		info := underlying.Info()

		// String type: only + is valid
		if info&types.IsString != 0 {
			return op == "+"
		}

		// Numeric types: all arithmetic operators are valid
		if info&types.IsNumeric != 0 {
			return true
		}

		return false

	default:
		// For non-basic types, arithmetic is generally not valid
		return false
	}
}

// isComparisonOpValidForType checks if a comparison operator is valid for a type.
func (tc *TypeChecker) isComparisonOpValidForType(t types.Type, op string) bool {
	underlying := t.Underlying()

	switch underlying := underlying.(type) {
	case *types.Basic:
		info := underlying.Info()

		// Ordered operators (<, <=, >, >=) require ordered types
		if op == "<" || op == "<=" || op == ">" || op == ">=" {
			return info&types.IsOrdered != 0
		}

		// Equality operators (==, !=) work on comparable types
		// Basic types (numeric, string, boolean) are always comparable
		if op == "==" || op == "!=" {
			return info&types.IsString != 0 || info&types.IsNumeric != 0 || info&types.IsBoolean != 0
		}

		return true

	case *types.Pointer, *types.Chan, *types.Interface:
		// These types only support == and !=
		return op == "==" || op == "!="

	case *types.Slice, *types.Map, *types.Signature:
		// These types can only be compared to nil with == and !=
		return op == "==" || op == "!="

	case *types.Struct, *types.Array:
		// Structs and arrays support == and != if all fields are comparable
		return op == "==" || op == "!="

	default:
		return true
	}
}

// assignToBinaryOp converts an assignment operator string to binary operator string.
func (tc *TypeChecker) assignToBinaryOp(op string) string {
	switch op {
	case "+=":
		return "+"
	case "-=":
		return "-"
	case "*=":
		return "*"
	case "/=":
		return "/"
	case "%=":
		return "%"
	default:
		return op
	}
}

// FilterMutants filters out invalid mutations based on type information.
func FilterMutants(mutants []Mutant, node ast.Node, typeInfo *types.Info) []Mutant {
	if typeInfo == nil {
		return mutants
	}

	tc := NewTypeChecker(typeInfo)
	validMutants := make([]Mutant, 0, len(mutants))

	for _, mutant := range mutants {
		if tc.IsValidMutation(node, mutant) {
			validMutants = append(validMutants, mutant)
		}
	}

	return validMutants
}

// GetNodeTypeInfo returns the type information for a specific node position.
func GetNodeTypeInfo(fset *token.FileSet, typeInfo *types.Info, line, column int) types.Type {
	if typeInfo == nil || typeInfo.Types == nil {
		return nil
	}

	// This is a simplified approach - in practice, we'd need to walk the AST
	// to find the exact node at the given position
	return nil
}
