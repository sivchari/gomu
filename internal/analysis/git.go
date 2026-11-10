package analysis

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitIntegration provides Git integration functionality.
type GitIntegration struct {
	workDir          string
	ignoreParser     IgnoreParser
	includeGenerated bool
	buildContext     *BuildContext
}

// NewGitIntegration creates a new Git integration.
func NewGitIntegration(workDir string) *GitIntegration {
	return &GitIntegration{
		workDir: workDir,
	}
}

// SetIgnoreParser sets the ignore parser for the Git integration.
func (g *GitIntegration) SetIgnoreParser(parser IgnoreParser) {
	g.ignoreParser = parser
}

// SetIncludeGenerated controls whether files carrying the standard Go
// generated-code marker are eligible as mutation targets. It is false by
// default.
func (g *GitIntegration) SetIncludeGenerated(include bool) {
	g.includeGenerated = include
}

// SetBuildContext restricts discovered files to those the active Go build
// configuration includes. A nil context imposes no restriction.
func (g *GitIntegration) SetBuildContext(bc *BuildContext) {
	g.buildContext = bc
}

// IsGitRepository checks if the current directory is a Git repository.
func (g *GitIntegration) IsGitRepository() bool {
	gitDir := filepath.Join(g.workDir, ".git")
	_, err := os.Stat(gitDir)

	return err == nil
}

// GetChangedFiles returns the list of changed files compared to the base branch.
func (g *GitIntegration) GetChangedFiles(baseBranch string) ([]string, error) {
	if !g.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	// Get the merge base with the base branch
	ctx := context.Background()
	mergeBaseCmd := exec.CommandContext(ctx, "git", "merge-base", "HEAD", baseBranch)
	mergeBaseCmd.Dir = g.workDir

	mergeBaseOutput, err := mergeBaseCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	mergeBase := strings.TrimSpace(string(mergeBaseOutput))

	return g.changedGoFiles(ctx, mergeBase)
}

// GetChangedFilesSince returns the Go files changed between baseSHA and HEAD.
//
// It compares the two trees directly, so it works on a shallow clone as long
// as baseSHA itself has been fetched.
func (g *GitIntegration) GetChangedFilesSince(baseSHA string) ([]string, error) {
	if !g.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	return g.changedGoFiles(context.Background(), baseSHA)
}

// changedGoFiles returns the eligible Go source files changed between rev and HEAD
// as absolute paths.
func (g *GitIntegration) changedGoFiles(ctx context.Context, rev string) ([]string, error) {
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--name-only", rev, "HEAD")
	diffCmd.Dir = g.workDir

	output, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}

	// Filter for Go files and apply ignore patterns
	var goFiles []string

	for _, file := range files {
		if IsGoSourceFile(file) {
			// Skip standard excluded directories (vendor, testdata)
			if IsExcludedPath(file) {
				continue
			}

			absPath := filepath.Join(g.workDir, file)

			if g.isMutationTarget(file, absPath) {
				goFiles = append(goFiles, absPath)
			}
		}
	}

	return goFiles, nil
}

// isMutationTarget reports whether a Go source file passes .gomuignore, the
// generated-file exclusion and the active build configuration. relPath is
// relative to the work directory; absPath is the same file's absolute path.
func (g *GitIntegration) isMutationTarget(relPath, absPath string) bool {
	if g.ignoreParser != nil && g.ignoreParser.ShouldIgnore(relPath) {
		return false
	}

	if !g.includeGenerated && IsGeneratedFile(absPath) {
		return false
	}

	return g.buildContext.IsActive(absPath)
}

// GetCurrentBranch returns the current Git branch name.
func (g *GitIntegration) GetCurrentBranch() (string, error) {
	if !g.IsGitRepository() {
		return "", fmt.Errorf("not a git repository")
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetAllGoFiles returns all Go files in the repository.
func (g *GitIntegration) GetAllGoFiles() ([]string, error) {
	var goFiles []string

	err := filepath.Walk(g.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and standard excluded directories (vendor, testdata)
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || IsExcludedPath(GetRelativePath(g.workDir, path)) {
				return filepath.SkipDir
			}

			// Check if directory should be ignored by .gomuignore
			if g.ignoreParser != nil {
				relPath := GetRelativePath(g.workDir, path)
				if g.ignoreParser.ShouldIgnore(relPath) {
					return filepath.SkipDir
				}
			}

			return nil
		}

		// Only include Go source files (not test files)
		if IsGoSourceFile(path) && g.isMutationTarget(GetRelativePath(g.workDir, path), path) {
			goFiles = append(goFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return goFiles, nil
}

// HasUncommittedChanges checks if there are uncommitted changes.
func (g *GitIntegration) HasUncommittedChanges() (bool, error) {
	if !g.IsGitRepository() {
		return false, fmt.Errorf("not a git repository")
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = g.workDir

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}
