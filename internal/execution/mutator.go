package execution

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"time"

	"github.com/sivchari/gomu/internal/mutation"
)

// SourceMutator handles actual source code mutation.
type SourceMutator struct {
	backupDir string
}

// NewSourceMutator creates a new source mutator.
func NewSourceMutator() (*SourceMutator, error) {
	// Create unique backup directory per mutator instance
	backupDir := filepath.Join(os.TempDir(), fmt.Sprintf("gomu_backup_%d_%d", os.Getpid(), time.Now().UnixNano()))
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &SourceMutator{
		backupDir: backupDir,
	}, nil
}

// ApplyMutation applies a mutation to the source file.
func (sm *SourceMutator) ApplyMutation(mutant mutation.Mutant) error {
	// 1. Backup original file with mutant ID for uniqueness
	if err := sm.backupFile(mutant.FilePath, mutant.ID); err != nil {
		return fmt.Errorf("failed to backup file: %w", err)
	}

	// 2. Apply mutation
	if err := sm.mutateFile(mutant); err != nil {
		// Restore original if mutation fails
		if err := sm.RestoreOriginal(mutant.FilePath, mutant.ID); err != nil {
			// Log error but continue with next mutant
			fmt.Printf("Warning: failed to restore original file: %v\n", err)
		}

		return fmt.Errorf("failed to apply mutation: %w", err)
	}

	return nil
}

// RestoreOriginal restores the original file from backup.
func (sm *SourceMutator) RestoreOriginal(filePath, mutantID string) error {
	backupPath := sm.getBackupPath(filePath, mutantID)

	// Check if backup exists to prevent errors
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Read backup content
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write back to original location
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore original file: %w", err)
	}

	// Clean up the backup file after successful restoration
	if err := os.Remove(backupPath); err != nil {
		fmt.Printf("Warning: failed to remove backup file %s: %v\n", backupPath, err)
	}

	return nil
}

// Cleanup removes backup files.
func (sm *SourceMutator) Cleanup() error {
	if err := os.RemoveAll(sm.backupDir); err != nil {
		return fmt.Errorf("failed to remove backup directory: %w", err)
	}

	return nil
}

// backupFile creates a backup of the original file with unique mutant ID.
func (sm *SourceMutator) backupFile(filePath, mutantID string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	backupPath := sm.getBackupPath(filePath, mutantID)
	backupDir := filepath.Dir(backupPath)

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Remove existing backup if it exists (idempotent operation)
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to remove existing backup: %w", err)
		}
	}

	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// getBackupPath returns the backup path for a given file with mutant ID for uniqueness.
func (sm *SourceMutator) getBackupPath(filePath, mutantID string) string {
	// Create a shorter, unique backup name using just the filename and mutant ID
	backupName := fmt.Sprintf("%s_%s_original", filepath.Base(filePath), mutantID)

	return filepath.Join(sm.backupDir, backupName)
}

// mutateFile applies the actual mutation to the file.
func (sm *SourceMutator) mutateFile(mutant mutation.Mutant) error {
	// Parse the source file
	fset := token.NewFileSet()

	src, err := os.ReadFile(mutant.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	file, err := parser.ParseFile(fset, mutant.FilePath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Apply mutation
	mutated := false

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil || mutated {
			return false // Stop if node is nil or after first mutation
		}

		pos := fset.Position(node.Pos())
		if pos.Line == mutant.Line && pos.Column == mutant.Column {
			mutated = sm.applyMutationToNode(node, mutant)
		}

		return !mutated
	})

	if !mutated {
		return fmt.Errorf("failed to find mutation target at %s:%d:%d", mutant.FilePath, mutant.Line, mutant.Column)
	}

	// Write mutated code back to file
	return sm.writeModifiedAST(file, fset, mutant.FilePath)
}

// applyMutationToNode applies the mutation to a specific AST node.
func (sm *SourceMutator) applyMutationToNode(node ast.Node, mutant mutation.Mutant) bool {
	switch mutant.Type {
	case "arithmetic_binary":
		return sm.mutateArithmeticBinary(node, mutant)
	case "arithmetic_assign":
		return sm.mutateArithmeticAssign(node, mutant)
	case "arithmetic_incdec":
		return sm.mutateIncDec(node, mutant)
	case "conditional_binary":
		return sm.mutateConditional(node, mutant)
	case "logical_binary":
		return sm.mutateLogicalBinary(node, mutant)
	case "logical_not_removal":
		return sm.mutateLogicalNot(node, mutant)
	case "bitwise_binary":
		return sm.mutateBitwiseBinary(node, mutant)
	case "bitwise_assign":
		return sm.mutateBitwiseAssign(node, mutant)
	}

	return false
}

// mutateArithmeticBinary mutates arithmetic binary expressions.
func (sm *SourceMutator) mutateArithmeticBinary(node ast.Node, mutant mutation.Mutant) bool {
	if expr, ok := node.(*ast.BinaryExpr); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			expr.Op = newOp

			return true
		}
	}

	return false
}

// mutateArithmeticAssign mutates assignment operators.
func (sm *SourceMutator) mutateArithmeticAssign(node ast.Node, mutant mutation.Mutant) bool {
	if stmt, ok := node.(*ast.AssignStmt); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			stmt.Tok = newOp

			return true
		}
	}

	return false
}

// mutateIncDec mutates increment/decrement operators.
func (sm *SourceMutator) mutateIncDec(node ast.Node, mutant mutation.Mutant) bool {
	if stmt, ok := node.(*ast.IncDecStmt); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			stmt.Tok = newOp

			return true
		}
	}

	return false
}

// mutateConditional mutates conditional operators.
func (sm *SourceMutator) mutateConditional(node ast.Node, mutant mutation.Mutant) bool {
	if expr, ok := node.(*ast.BinaryExpr); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			expr.Op = newOp

			return true
		}
	}

	return false
}

// mutateLogicalBinary mutates logical binary operators.
func (sm *SourceMutator) mutateLogicalBinary(node ast.Node, mutant mutation.Mutant) bool {
	if expr, ok := node.(*ast.BinaryExpr); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			expr.Op = newOp

			return true
		}
	}

	return false
}

// mutateLogicalNot removes NOT operators.
func (sm *SourceMutator) mutateLogicalNot(_ ast.Node, _ mutation.Mutant) bool {
	// For NOT removal, we need to replace the unary expression with its operand
	// This is more complex and requires parent node manipulation
	// For now, we'll return false to indicate this mutation type isn't fully implemented
	return false
}

// mutateBitwiseBinary mutates bitwise binary operators.
func (sm *SourceMutator) mutateBitwiseBinary(node ast.Node, mutant mutation.Mutant) bool {
	if expr, ok := node.(*ast.BinaryExpr); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			expr.Op = newOp

			return true
		}
	}

	return false
}

// mutateBitwiseAssign mutates bitwise assignment operators.
func (sm *SourceMutator) mutateBitwiseAssign(node ast.Node, mutant mutation.Mutant) bool {
	if stmt, ok := node.(*ast.AssignStmt); ok {
		newOp := sm.stringToToken(mutant.Mutated)
		if newOp != token.ILLEGAL {
			stmt.Tok = newOp

			return true
		}
	}

	return false
}

// stringToToken converts string representation to token.Token.
func (sm *SourceMutator) stringToToken(s string) token.Token {
	if tok := sm.getArithmeticToken(s); tok != token.ILLEGAL {
		return tok
	}

	if tok := sm.getComparisonToken(s); tok != token.ILLEGAL {
		return tok
	}

	if tok := sm.getLogicalToken(s); tok != token.ILLEGAL {
		return tok
	}

	if tok := sm.getBitwiseToken(s); tok != token.ILLEGAL {
		return tok
	}

	if tok := sm.getAssignmentToken(s); tok != token.ILLEGAL {
		return tok
	}

	return token.ILLEGAL
}

// getArithmeticToken returns arithmetic tokens.
func (sm *SourceMutator) getArithmeticToken(s string) token.Token {
	switch s {
	case "+":
		return token.ADD
	case "-":
		return token.SUB
	case "*":
		return token.MUL
	case "/":
		return token.QUO
	case "%":
		return token.REM
	case "++":
		return token.INC
	case "--":
		return token.DEC
	default:
		return token.ILLEGAL
	}
}

// getComparisonToken returns comparison tokens.
func (sm *SourceMutator) getComparisonToken(s string) token.Token {
	switch s {
	case "==":
		return token.EQL
	case "!=":
		return token.NEQ
	case "<":
		return token.LSS
	case "<=":
		return token.LEQ
	case ">":
		return token.GTR
	case ">=":
		return token.GEQ
	default:
		return token.ILLEGAL
	}
}

// getLogicalToken returns logical tokens.
func (sm *SourceMutator) getLogicalToken(s string) token.Token {
	switch s {
	case "&&":
		return token.LAND
	case "||":
		return token.LOR
	default:
		return token.ILLEGAL
	}
}

// getBitwiseToken returns bitwise tokens.
func (sm *SourceMutator) getBitwiseToken(s string) token.Token {
	switch s {
	case "&":
		return token.AND
	case "|":
		return token.OR
	case "^":
		return token.XOR
	case "&^":
		return token.AND_NOT
	case "<<":
		return token.SHL
	case ">>":
		return token.SHR
	default:
		return token.ILLEGAL
	}
}

// getAssignmentToken returns assignment tokens.
func (sm *SourceMutator) getAssignmentToken(s string) token.Token {
	switch s {
	case "+=":
		return token.ADD_ASSIGN
	case "-=":
		return token.SUB_ASSIGN
	case "*=":
		return token.MUL_ASSIGN
	case "/=":
		return token.QUO_ASSIGN
	case "&=":
		return token.AND_ASSIGN
	case "|=":
		return token.OR_ASSIGN
	case "^=":
		return token.XOR_ASSIGN
	case "<<=":
		return token.SHL_ASSIGN
	case ">>=":
		return token.SHR_ASSIGN
	default:
		return token.ILLEGAL
	}
}

// writeModifiedAST writes the modified AST back to the file.
func (sm *SourceMutator) writeModifiedAST(file *ast.File, fset *token.FileSet, filePath string) error {
	// Create a temporary file first
	tmpFile := filePath + ".tmp"

	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Format and write the AST
	if err := format.Node(f, fset, file); err != nil {
		f.Close()

		if _, statErr := os.Stat(tmpFile); statErr == nil {
			if removeErr := os.Remove(tmpFile); removeErr != nil {
				fmt.Printf("Warning: failed to remove temporary file: %v\n", removeErr)
			}
		}

		return fmt.Errorf("failed to format and write AST: %w", err)
	}

	// Close file before rename
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Replace original file with temporary file
	if err := os.Rename(tmpFile, filePath); err != nil {
		if _, statErr := os.Stat(tmpFile); statErr == nil {
			if removeErr := os.Remove(tmpFile); removeErr != nil {
				fmt.Printf("Warning: failed to remove temporary file: %v\n", removeErr)
			}
		}

		return fmt.Errorf("failed to replace original file: %w", err)
	}

	return nil
}
