package gomu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/report"
)

func runChangedLinesGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)

	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}

	return strings.TrimSpace(string(out))
}

func setupChangedLinesRun(t *testing.T, updated string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)

	git := func(args ...string) { runChangedLinesGit(t, dir, args...) }
	git("init", "-b", "main")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	writeFile(t, filepath.Join(dir, "go.mod"), testModuleContent)

	source := "package example\n\nfunc Legacy(x int) int {\n return x * 2\n}\n\nfunc Updated(x int) int {\n return x + 1\n}\n"
	writeFile(t, filepath.Join(dir, "source.go"), source)
	writeFile(t, filepath.Join(dir, "source_test.go"), `package example
import "testing"
func TestUpdated(t *testing.T) {
 for _, x := range []int{0, 2} {
  if got := Updated(x); got != x + 2 { t.Fatalf("Updated(%d) = %d", x, got) }
 }
}
`)
	git("add", ".")
	git("commit", "-m", "base fixture")
	git("checkout", "-b", "feature")
	writeFile(t, filepath.Join(dir, "source.go"), strings.Replace(source, "return x + 1", updated, 1))
	git("add", ".")
	git("commit", "-m", "update fixture")

	return dir
}

func runChangedLinesEngine(t *testing.T, opts *RunOptions) (string, error) {
	t.Helper()

	engine, err := NewEngine(opts)
	if err != nil {
		t.Fatal(err)
	}

	var runErr error

	out := captureStdout(t, func() { runErr = engine.Run(context.Background(), ".", opts) })

	return out, runErr
}

func readChangedLinesSummary(t *testing.T, dir string) report.Summary {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "mutation-report.json"))
	if err != nil {
		t.Fatal(err)
	}

	var summary report.Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}

	return summary
}

func TestChangedLinesRunSelectionAndHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the Go toolchain")
	}

	for _, warmHistory := range []bool{false, true} {
		t.Run(fmt.Sprintf("warm_history_%t", warmHistory), func(t *testing.T) {
			dir := setupChangedLinesRun(t, "return x + 2")

			var before []byte

			if warmHistory {
				_, err := runChangedLinesEngine(t, &RunOptions{Output: "json", Workers: 1, Timeout: 10})
				if err != nil {
					t.Fatal(err)
				}

				before, err = os.ReadFile(filepath.Join(dir, ".gomu_history.json"))
				if err != nil {
					t.Fatal(err)
				}
			}

			opts := &RunOptions{ChangedLines: true, Incremental: true, DryRun: true, Output: "json", Workers: 1, Timeout: 10, Threshold: 100, FailOnGate: true}

			dry, err := runChangedLinesEngine(t, opts)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(dry, "Changed lines: 8-8") || !strings.Contains(dry, "L7-9") {
				t.Errorf("missing selection ranges: %s", dry)
			}

			opts.DryRun = false

			out, err := runChangedLinesEngine(t, opts)
			if err != nil {
				t.Fatalf("selected mutants should pass gate: %v\n%s", err, out)
			}

			summary := readChangedLinesSummary(t, dir)
			if len(summary.Results) == 0 || summary.Statistics.Score != 100 {
				t.Fatalf("unexpected summary: %+v", summary)
			}

			for _, r := range summary.Results {
				if r.Mutant.Line < 7 {
					t.Errorf("Legacy mutant executed: %+v", r.Mutant)
				}

				if !strings.Contains(dry, fmt.Sprintf("L%d:%d", r.Mutant.Line, r.Mutant.Column)) || !strings.Contains(dry, r.Mutant.Description) {
					t.Errorf("executed mutant missing from dry-run: %+v", r.Mutant)
				}
			}

			if !strings.Contains(dry, fmt.Sprintf("Total: %d mutant(s)", len(summary.Results))) {
				t.Errorf("dry-run selection differs from execution: %s", dry)
			}

			opts.CIMode = true
			opts.Output = ""

			out, err = runChangedLinesEngine(t, opts)
			if err != nil || !strings.Contains(out, "Quality Gate: PASSED (Score: 100.0%, "+report.ScoreFormula+")") {
				t.Errorf("excluded Legacy survivors affected gate: %v\n%s", err, out)
			}

			after, err := os.ReadFile(filepath.Join(dir, ".gomu_history.json"))
			if warmHistory {
				if err != nil || string(before) != string(after) {
					t.Errorf("history changed: %v", err)
				}
			} else if !os.IsNotExist(err) {
				t.Errorf("partial run created history: %v", err)
			}

			if !warmHistory {
				_, err := runChangedLinesEngine(t, &RunOptions{Output: "json", Workers: 1, Timeout: 10})
				if err != nil {
					t.Fatal(err)
				}

				summary = readChangedLinesSummary(t, dir)

				foundLegacy := false
				for _, r := range summary.Results {
					foundLegacy = foundLegacy || r.Mutant.Line < 7
				}

				if !foundLegacy {
					t.Error("partial run suppressed later ordinary analysis")
				}
			}
		})
	}
}

func TestChangedLinesSelectedSurvivorsFailGate(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the Go toolchain")
	}

	dir := setupChangedLinesRun(t, "return x + 0")
	writeFile(t, filepath.Join(dir, "source_test.go"), `package example
import "testing"
func TestUpdated(t *testing.T) {
 if got := Updated(0); got != 0 { t.Fatalf("Updated(0) = %d", got) }
}
`)

	out, err := runChangedLinesEngine(t, &RunOptions{
		ChangedLines: true, Incremental: true, Workers: 1, Timeout: 10,
		CIMode: true, Threshold: 100, FailOnGate: true,
	})
	if err == nil || !strings.Contains(err.Error(), "quality gate failed") || !strings.Contains(out, "Quality Gate: FAILED") {
		t.Errorf("selected survivors did not fail gate: %v\n%s", err, out)
	}
}

func TestChangedLinesErrorsAndEmpty(t *testing.T) {
	t.Run("conflicting flags", func(t *testing.T) {
		t.Chdir(t.TempDir())

		_, err := runChangedLinesEngine(t, &RunOptions{ChangedLines: true})
		if err == nil || !strings.Contains(err.Error(), "--incremental=false") {
			t.Errorf("conflicting flags error = %v", err)
		}
	})

	for _, dry := range []bool{false, true} {
		t.Run(fmt.Sprintf("no_eligible_mutants_dry_%t", dry), func(t *testing.T) {
			dir := setupChangedLinesRun(t, "return x + 2")
			path := filepath.Join(dir, "source.go")

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			writeFile(t, path, "// Package example demonstrates selection.\n"+string(data))
			runChangedLinesGit(t, dir, "add", "source.go")
			runChangedLinesGit(t, dir, "commit", "-m", "documentation fixture")

			opts := &RunOptions{ChangedLines: true, Incremental: true, DryRun: dry, BaseBranch: "HEAD~1", CIMode: true, Threshold: 100, FailOnGate: true}

			out, err := runChangedLinesEngine(t, opts)
			if err != nil || !strings.Contains(out, "No mutants selected from changed lines; skipped") {
				t.Errorf("no eligible mutants: %v\n%s", err, out)
			}

			if strings.Contains(out, "0.0%") {
				t.Errorf("empty selection presented as a measured score: %s", out)
			}
		})
		t.Run(fmt.Sprintf("empty_dry_%t", dry), func(t *testing.T) {
			dir := setupChangedLinesRun(t, "return x + 2")
			opts := &RunOptions{ChangedLines: true, Incremental: true, DryRun: dry, BaseBranch: "HEAD", CIMode: true, Threshold: 100, FailOnGate: true}

			out, err := runChangedLinesEngine(t, opts)
			if err != nil || !strings.Contains(out, "No mutants selected from changed lines; skipped") {
				t.Errorf("empty run: %v\n%s", err, out)
			}

			if _, err := os.Stat(filepath.Join(dir, ".gomu_history.json")); !os.IsNotExist(err) {
				t.Errorf("empty run created history: %v", err)
			}
		})
	}

	for _, dry := range []bool{false, true} {
		t.Run(fmt.Sprintf("invalid_source_dry_%t", dry), func(t *testing.T) {
			dir := setupChangedLinesRun(t, "return )")

			out, err := runChangedLinesEngine(t, &RunOptions{ChangedLines: true, Incremental: true, DryRun: dry})
			if err == nil || strings.Contains(out, "skipped") {
				t.Errorf("invalid source treated as empty: %v\n%s", err, out)
			}

			if _, err := os.Stat(filepath.Join(dir, ".gomu_history.json")); !os.IsNotExist(err) {
				t.Errorf("failed run created history: %v", err)
			}
		})
	}
}

func TestChangedLinesUsesBaseSHAAndBuildTags(t *testing.T) {
	dir := setupChangedLinesRun(t, "return x + 2")
	base := runChangedLinesGit(t, dir, "rev-parse", "HEAD")

	writeFile(t, filepath.Join(dir, "tagged.go"), "//go:build changedlines\n\npackage example\n\nfunc Tagged(x int) int { return x + 2 }\n")
	runChangedLinesGit(t, dir, "add", "tagged.go")
	runChangedLinesGit(t, dir, "commit", "-m", "add tagged source")

	for _, tt := range []struct {
		name     string
		testArgs []string
		want     string
	}{
		{name: "inactive", want: noChangedLineMutants},
		{name: "active", testArgs: []string{"-tags=changedlines"}, want: "/tagged.go ("},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := &RunOptions{
				ChangedLines: true, Incremental: true, DryRun: true,
				BaseBranch: "missing-branch", BaseSHA: base, TestArgs: tt.testArgs,
			}

			out, err := runChangedLinesEngine(t, opts)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(out, tt.want) || strings.Contains(out, "/source.go (") {
				t.Errorf("dry run = %q, want %q and no source.go mutations", out, tt.want)
			}
		})
	}
}

func TestChangedLinesConservativeOverlap(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dir := setupChangedLinesRun(t, "return x + 2")
	// Match Git's physical paths when the temporary directory uses a symlink.
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)

	base := `package example

func Legacy(x int) int {
 return x * 2
}

func Updated(x int) int {
 if x > 0 {
  return x +
   1
 }
 return x
}
`
	path := filepath.Join(dir, "source.go")
	writeFile(t, path, base)
	runChangedLinesGit(t, dir, "add", "source.go")
	runChangedLinesGit(t, dir, "commit", "-m", "multiline base fixture")
	writeFile(t, path, strings.Replace(base, "   1", "   2", 1))
	runChangedLinesGit(t, dir, "add", "source.go")
	runChangedLinesGit(t, dir, "commit", "-m", "multiline change fixture")

	opts := &RunOptions{ChangedLines: true, Incremental: true, DryRun: true, BaseBranch: "HEAD~1"}

	multilineOut, err := runChangedLinesEngine(t, opts)
	if err != nil {
		t.Fatal(err)
	}

	// A comment-only edit inside the if selects its containing nodes, but
	// does not select the unchanged expression on the following lines.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, path, strings.Replace(string(data), "  return x +", "  // Explain the branch.\n  return x +", 1))
	runChangedLinesGit(t, dir, "add", "source.go")
	runChangedLinesGit(t, dir, "commit", "-m", "comment fixture")

	commentOut, err := runChangedLinesEngine(t, opts)
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(packageDir)
	t.Run("multiline", func(t *testing.T) {
		assertGolden(t, filepath.Join("testdata", "dryrun", "changed_lines_multiline.golden"), []byte(multilineOut))
	})
	t.Run("comment", func(t *testing.T) {
		assertGolden(t, filepath.Join("testdata", "dryrun", "changed_lines_comment.golden"), []byte(commentOut))
	})
}
