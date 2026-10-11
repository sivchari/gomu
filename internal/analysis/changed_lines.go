package analysis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LineRange is an inclusive range of physical source lines.
type LineRange struct {
	Start int
	End   int
}

const defaultBaseBranch = "main"

var changedLineHunk = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// GetChangedLines selects committed additions and modifications since the
// merge base with baseBranch. Renames are treated as deletion plus addition.
// Selected source files must have no staged or unstaged changes.
func (g *GitIntegration) GetChangedLines(baseBranch string) (map[string][]LineRange, error) {
	if baseBranch == "" {
		baseBranch = defaultBaseBranch
	}

	return g.getChangedLines(baseBranch, true)
}

// GetChangedLinesSince selects committed line ranges by comparing baseSHA
// directly with HEAD, without requiring a merge base.
func (g *GitIntegration) GetChangedLinesSince(baseSHA string) (map[string][]LineRange, error) {
	return g.getChangedLines(baseSHA, false)
}

func (g *GitIntegration) getChangedLines(baseRef string, findMergeBase bool) (map[string][]LineRange, error) {
	gitDir := g.workDir

	info, err := os.Stat(gitDir)
	if err != nil {
		return nil, fmt.Errorf("resolve changed-lines target: %w", err)
	}

	if !info.IsDir() {
		gitDir = filepath.Dir(gitDir)
	}

	rootOutput, err := changedLinesGit(gitDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("changed-lines requires a Git repository and git on PATH: %w", err)
	}

	root := strings.TrimSuffix(string(rootOutput), "\n")

	target, err := filepath.Abs(g.workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve changed-lines target: %w", err)
	}
	// Resolve symlinks so Git's repository root and the requested target use
	// the same coordinates (notably /var versus /private/var on macOS).
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return nil, fmt.Errorf("resolve changed-lines target: %w", err)
	}

	relTarget, err := filepath.Rel(root, target)
	if err != nil {
		return nil, fmt.Errorf("resolve changed-lines target relative to Git root: %w", err)
	}

	var baseOutput []byte

	if findMergeBase {
		baseOutput, err = changedLinesGit(root, "merge-base", "--", "HEAD", baseRef)
		if err != nil {
			return nil, fmt.Errorf("find merge base of HEAD and %q; ensure the base ref is available: %w", baseRef, err)
		}
	} else {
		baseOutput, err = changedLinesGit(root, "rev-parse", "--verify", "--end-of-options", baseRef+"^{commit}")
		if err != nil {
			return nil, fmt.Errorf("resolve base commit %q: %w", baseRef, err)
		}
	}

	base := strings.TrimSpace(string(baseOutput))
	diffOptions := []string{"diff", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-color", "--text"}
	nameArgs := append(append([]string{}, diffOptions...), "--name-only", "-z", "--diff-filter=AM", base, "HEAD", "--", filepath.ToSlash(relTarget))

	names, err := changedLinesGit(root, nameArgs...)
	if err != nil {
		return nil, fmt.Errorf("discover changed source files: %w", err)
	}

	selection := make(map[string][]LineRange)
	for _, name := range bytes.Split(names, []byte{0}) {
		file := string(name)
		if !IsGoSourceFile(file) || IsExcludedPath(file) {
			continue
		}

		absPath := filepath.Join(root, filepath.FromSlash(file))
		if !g.isMutationTarget(GetRelativePath(target, absPath), absPath) {
			continue
		}

		diffArgs := append(append([]string{}, diffOptions...), "--unified=0", "--inter-hunk-context=0", base, "HEAD", "--", file)

		patch, err := changedLinesGit(root, diffArgs...)
		if err != nil {
			return nil, fmt.Errorf("read changed lines for %q: %w", file, err)
		}

		ranges, err := parseChangedLineRanges(patch)
		if err != nil {
			return nil, fmt.Errorf("read changed lines for %q: %w", file, err)
		}

		if len(ranges) == 0 {
			continue
		}

		status, err := changedLinesGit(root, "status", "--porcelain=v1", "-z", "--untracked-files=no", "--", file)
		if err != nil {
			return nil, fmt.Errorf("check source file %q: %w", file, err)
		}

		if len(status) != 0 {
			return nil, fmt.Errorf("changed-lines source file %q has staged or unstaged changes; commit or restore it so its lines match HEAD", file)
		}

		selection[absPath] = ranges
	}

	return selection, nil
}

func changedLinesGit(dir string, args ...string) ([]byte, error) {
	args = append([]string{"--literal-pathspecs"}, args...)
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git %s: %w: %s", args[1], err, strings.TrimSpace(string(exitErr.Stderr)))
		}

		return nil, fmt.Errorf("git %s: %w", args[1], err)
	}

	return output, nil
}

func parseChangedLineRanges(patch []byte) ([]LineRange, error) {
	var ranges []LineRange

	for _, line := range bytes.Split(patch, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte("@@ ")) {
			continue
		}

		match := changedLineHunk.FindSubmatch(line)
		if match == nil {
			return nil, fmt.Errorf("invalid diff hunk header %q", line)
		}

		start, err := strconv.Atoi(string(match[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid diff line number: %w", err)
		}

		count := 1
		if len(match[2]) != 0 {
			count, err = strconv.Atoi(string(match[2]))
			if err != nil {
				return nil, fmt.Errorf("invalid diff line count: %w", err)
			}
		}

		if count > 0 {
			ranges = append(ranges, LineRange{Start: start, End: start + count - 1})
		}
	}

	return ranges, nil
}
