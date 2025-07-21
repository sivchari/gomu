// Package analysis provides mutation testing analysis functionality.
package analysis

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Analyzer handles code analysis and file discovery.
type Analyzer struct {
	fileSet  *token.FileSet
	typeInfo *types.Info
}

// New creates a new analyzer.
func New() (*Analyzer, error) {
	return &Analyzer{
		fileSet: token.NewFileSet(),
		typeInfo: &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Uses:  make(map[*ast.Ident]types.Object),
			Defs:  make(map[*ast.Ident]types.Object),
		},
	}, nil
}

// FileInfo represents information about a Go source file.
type FileInfo struct {
	Path     string
	FileAST  *ast.File
	Hash     string
	TypeInfo *types.Info
}

// FindTargetFiles discovers Go source files to be tested.
func (a *Analyzer) FindTargetFiles(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip standard excluded directories
			excludeDirs := []string{"vendor/", "testdata/"}
			for _, exclude := range excludeDirs {
				if strings.Contains(path, exclude) {
					return filepath.SkipDir
				}
			}

			return nil
		}

		// Only process Go source files (not test files)
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			// Skip excluded files
			excluded := false
			excludeDirs := []string{"vendor/", "testdata/"}

			for _, exclude := range excludeDirs {
				if strings.Contains(path, exclude) {
					excluded = true

					break
				}
			}

			if !excluded {
				files = append(files, path)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// FindChangedFiles returns files that have changed compared to the base branch.
func (a *Analyzer) FindChangedFiles(allFiles []string) ([]string, error) {
	// Always enable incremental analysis by default
	incrementalEnabled := true
	if !incrementalEnabled {
		return allFiles, nil
	}

	// Use intelligent default for base branch
	baseBranch := "main"

	// Get changed files from git
	// Validate base branch name to prevent command injection
	if !isValidBranchName(baseBranch) {
		return nil, fmt.Errorf("invalid base branch name: %s", baseBranch)
	}

	cmd := exec.Command("git", "diff", "--name-only", baseBranch+"...HEAD")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git diff: %w", err)
	}

	changedPaths := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(changedPaths) == 1 && changedPaths[0] == "" {
		// No changes
		return []string{}, nil
	}

	// Convert to absolute paths and filter
	var changedFiles []string

	for _, file := range allFiles {
		for _, changedPath := range changedPaths {
			if strings.HasSuffix(file, changedPath) {
				changedFiles = append(changedFiles, file)

				break
			}
		}
	}

	return changedFiles, nil
}

// ParseFile parses a Go source file and returns its AST.
func (a *Analyzer) ParseFile(filePath string) (*FileInfo, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	fileAST, err := parser.ParseFile(a.fileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	// Calculate file hash for incremental analysis
	hash := calculateFileHash(src)

	// Try to get type information for the file
	typeInfo := a.getTypeInfo(fileAST, filePath)

	return &FileInfo{
		Path:     filePath,
		FileAST:  fileAST,
		Hash:     hash,
		TypeInfo: typeInfo,
	}, nil
}

// GetPosition returns the position information for a token.
func (a *Analyzer) GetPosition(pos token.Pos) token.Position {
	return a.fileSet.Position(pos)
}

// GetFileSet returns the file set used by the analyzer.
func (a *Analyzer) GetFileSet() *token.FileSet {
	return a.fileSet
}

// getTypeInfo attempts to get type information for a file.
func (a *Analyzer) getTypeInfo(fileAST *ast.File, filePath string) *types.Info {
	// Create a new type info for this file
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	// Get the package directory
	pkgDir := filepath.Dir(filePath)

	// Parse all files in the package
	pkgFiles, err := a.parsePackageFiles(pkgDir)
	if err != nil {
		// If we can't parse the package, return nil (fall back to syntax-only)
		return nil
	}

	// Create type checker config
	config := &types.Config{
		Importer: importer.ForCompiler(a.fileSet, "source", nil),
		Error: func(_ error) {
			// Ignore type errors for now - we want to be permissive
		},
	}

	// Type check the package
	_, err = config.Check(fileAST.Name.Name, a.fileSet, pkgFiles, info)
	if err != nil {
		// If type checking fails, return nil (fall back to syntax-only)
		return nil
	}

	return info
}

// parsePackageFiles parses all Go files in a package directory.
func (a *Analyzer) parsePackageFiles(pkgDir string) ([]*ast.File, error) {
	files, err := filepath.Glob(filepath.Join(pkgDir, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	astFiles := make([]*ast.File, 0, len(files))

	for _, file := range files {
		// Skip test files for type checking
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		astFile, err := parser.ParseFile(a.fileSet, file, nil, parser.ParseComments)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}

		astFiles = append(astFiles, astFile)
	}

	return astFiles, nil
}

// CalculateFileHash calculates a hash for the given file content.
func calculateFileHash(content []byte) string {
	// Simple hash implementation - in production, use crypto/sha256
	return fmt.Sprintf("%x", len(content))
}

// isValidBranchName validates branch names to prevent command injection.
func isValidBranchName(name string) bool {
	// Git branch names can contain letters, numbers, hyphens, underscores, dots, and forward slashes
	// but cannot start with a hyphen or contain special characters that could be interpreted as options
	validBranchPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

	return validBranchPattern.MatchString(name) && !strings.HasPrefix(name, "-")
}
