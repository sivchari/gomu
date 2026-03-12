// Package analysis provides mutation testing analysis functionality.
package analysis

import (
	"context"
	"crypto/sha256"
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

	"github.com/sivchari/gomu/internal/ignore"
)

// Analyzer handles code analysis and file discovery.
type Analyzer struct {
	fileSet      *token.FileSet
	typeInfo     *types.Info
	ignoreParser *ignore.Parser
}

// Option is a functional option for configuring an Analyzer.
type Option func(*Analyzer)

// WithIgnoreParser sets the ignore parser for the analyzer.
func WithIgnoreParser(parser *ignore.Parser) Option {
	return func(a *Analyzer) {
		a.ignoreParser = parser
	}
}

// New creates a new analyzer with optional configuration.
func New(opts ...Option) (*Analyzer, error) {
	a := &Analyzer{
		fileSet: token.NewFileSet(),
		typeInfo: &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Uses:  make(map[*ast.Ident]types.Object),
			Defs:  make(map[*ast.Ident]types.Object),
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	return a, nil
}

// FileInfo represents information about a Go source file.
type FileInfo struct {
	Path     string
	FileAST  *ast.File
	Hash     string
	TypeInfo *types.Info
}

// shouldSkipDirectory checks if a directory should be skipped.
func (a *Analyzer) shouldSkipDirectory(rootPath, path string) bool {
	// Check if directory should be ignored by .gomuignore
	if a.ignoreParser != nil {
		relPath := GetRelativePath(rootPath, path)
		if a.ignoreParser.ShouldIgnore(relPath) {
			return true
		}
	}

	// Skip standard excluded directories
	excludeDirs := []string{"vendor", "testdata"}
	for _, exclude := range excludeDirs {
		if strings.Contains(path, exclude) {
			return true
		}
	}

	return false
}

// shouldIgnoreFile checks if a file should be ignored.
func (a *Analyzer) shouldIgnoreFile(rootPath, path string) bool {
	if a.ignoreParser == nil {
		return false
	}

	relPath := GetRelativePath(rootPath, path)

	return a.ignoreParser.ShouldIgnore(relPath)
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
			if skip := a.shouldSkipDirectory(rootPath, path); skip {
				return filepath.SkipDir
			}

			return nil
		}

		// Only process Go source files (not test files)
		if !IsGoSourceFile(path) {
			return nil
		}

		// Check if file should be ignored by .gomuignore (for file-specific patterns)
		if a.shouldIgnoreFile(rootPath, path) {
			return nil
		}

		files = append(files, path)

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

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", baseBranch+"...HEAD")

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

	// Calculate file hash for incremental analysis
	hash := calculateFileHash(src)

	// Try to get type information by parsing all package files together
	fileAST, typeInfo, err := a.parseAndTypeCheck(filePath)
	if err != nil {
		// Fall back to syntax-only parsing
		fileAST, err = parser.ParseFile(a.fileSet, filePath, src, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
		}

		typeInfo = nil
	}

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

// parseAndTypeCheck parses all package files and type checks them, returning the AST and type info.
func (a *Analyzer) parseAndTypeCheck(filePath string) (*ast.File, *types.Info, error) {
	// Get the package directory
	pkgDir := filepath.Dir(filePath)

	// Parse all files in the package
	files, err := filepath.Glob(filepath.Join(pkgDir, "*.go"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to glob files: %w", err)
	}

	astFiles := make([]*ast.File, 0, len(files))

	var targetAST *ast.File

	for _, file := range files {
		// Skip test files for type checking
		if IsGoTestFile(file) {
			continue
		}

		astFile, err := parser.ParseFile(a.fileSet, file, nil, parser.ParseComments)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}

		astFiles = append(astFiles, astFile)

		// Remember the target file's AST
		if file == filePath {
			targetAST = astFile
		}
	}

	if targetAST == nil {
		return nil, nil, fmt.Errorf("target file not found in parsed files")
	}

	if len(astFiles) == 0 {
		return nil, nil, fmt.Errorf("no files to type check")
	}

	// Create type info
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	// Create type checker config
	config := &types.Config{
		Importer: importer.Default(),
		Error: func(_ error) {
			// Ignore type errors for now - we want to be permissive
		},
	}

	// Type check the package
	// We ignore the error because we want to return partial type info even if
	// type checking fails (e.g., due to missing imports)
	_, _ = config.Check(targetAST.Name.Name, a.fileSet, astFiles, info)

	return targetAST, info, nil
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
		if IsGoTestFile(file) {
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

// calculateFileHash calculates a SHA256 hash for the given file content.
func calculateFileHash(content []byte) string {
	h := sha256.Sum256(content)

	return fmt.Sprintf("%x", h)
}

// isValidBranchName validates branch names to prevent command injection.
func isValidBranchName(name string) bool {
	// Git branch names can contain letters, numbers, hyphens, underscores, dots, and forward slashes
	// but cannot start with a hyphen or contain special characters that could be interpreted as options
	validBranchPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

	return validBranchPattern.MatchString(name) && !strings.HasPrefix(name, "-")
}
