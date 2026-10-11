package gomu

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sivchari/gomu/internal/analysis"
	"github.com/sivchari/gomu/internal/buildflags"
	"github.com/sivchari/gomu/internal/mutation"
)

const noChangedLineMutants = "No mutants selected from changed lines; skipped"

func (e *Engine) discoverChangedLines(path string, opts *RunOptions, ignore analysis.IgnoreParser) ([]analysis.FileAnalysisResult, []string, map[string][]analysis.LineRange, error) {
	buildDir, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("resolve changed-lines target: %w", err)
	}

	info, err := os.Stat(buildDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stat changed-lines target: %w", err)
	}

	if !info.IsDir() {
		buildDir = filepath.Dir(buildDir)
	}

	buildCtx, err := analysis.LoadBuildContext(buildDir, buildflags.Discovery(opts.TestArgs), os.Environ())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load build configuration: %w", err)
	}

	e.mutator.SetBuildContext(buildCtx)

	git := analysis.NewGitIntegration(path)
	git.SetIgnoreParser(ignore)
	git.SetIncludeGenerated(opts.IncludeGenerated)
	git.SetBuildContext(buildCtx)

	var ranges map[string][]analysis.LineRange

	if opts.BaseSHA != "" {
		ranges, err = git.GetChangedLinesSince(opts.BaseSHA)
	} else {
		ranges, err = git.GetChangedLines(opts.BaseBranch)
	}

	if err != nil {
		return nil, nil, nil, fmt.Errorf("discover changed lines: %w", err)
	}

	files := make([]string, 0, len(ranges))
	for file := range ranges {
		files = append(files, file)
	}

	sort.Strings(files)

	results := make([]analysis.FileAnalysisResult, 0, len(files))
	for _, file := range files {
		results = append(results, analysis.FileAnalysisResult{FilePath: file, NeedsUpdate: true, Reason: "Changed lines"})
	}

	return results, files, ranges, nil
}

// generateSelectedMutants preserves ordinary generation when ranges is nil.
// Filtering after generation retains the original identities and order.
func (e *Engine) generateSelectedMutants(file string, ranges map[string][]analysis.LineRange) ([]mutation.Mutant, error) {
	mutants, err := e.mutator.GenerateMutants(file)
	if err != nil {
		return nil, fmt.Errorf("generate mutants: %w", err)
	}

	if ranges == nil {
		return mutants, nil
	}

	selected := make([]mutation.Mutant, 0, len(mutants))
	for _, m := range mutants {
		for _, r := range ranges[file] {
			if m.SelectionStartLine <= r.End && r.Start <= m.SelectionEndLine {
				selected = append(selected, m)

				break
			}
		}
	}

	return selected, nil
}
