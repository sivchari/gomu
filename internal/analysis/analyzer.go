// Package analysis provides mutation testing analysis functionality.
package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sivchari/gomu/internal/config"
)

// Analyzer handles code analysis and file discovery.
type Analyzer struct {
	config  *config.Config
	fileSet *token.FileSet
}

// New creates a new analyzer.
func New(cfg *config.Config) (*Analyzer, error) {
	return &Analyzer{
		config:  cfg,
		fileSet: token.NewFileSet(),
	}, nil
}

// FileInfo represents information about a Go source file.
type FileInfo struct {
	Path    string
	FileAST *ast.File
	Hash    string
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
			// Skip excluded directories
			for _, exclude := range a.config.ExcludeFiles {
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

			for _, exclude := range a.config.ExcludeFiles {
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
	if !a.config.UseGitDiff {
		return allFiles, nil
	}

	// Get changed files from git
	// Validate base branch name to prevent command injection
	if !isValidBranchName(a.config.BaseBranch) {
		return nil, fmt.Errorf("invalid base branch name: %s", a.config.BaseBranch)
	}

	cmd := exec.Command("git", "diff", "--name-only", a.config.BaseBranch+"...HEAD")

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

	return &FileInfo{
		Path:    filePath,
		FileAST: fileAST,
		Hash:    hash,
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
