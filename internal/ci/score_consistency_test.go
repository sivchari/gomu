package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/report"
)

// notViableSummary mirrors a run where 12 of 191 mutants do not compile.
func notViableSummary() *report.Summary {
	return &report.Summary{
		TotalMutants:  191,
		KilledMutants: 71,
		Files: map[string]*report.FileReport{
			"internal/foo.go": {
				FilePath:      "internal/foo.go",
				TotalMutants:  191,
				KilledMutants: 71,
				NotViable:     12,
				MutationScore: report.MutationScore(71, 191, 12),
			},
		},
		Statistics: report.Statistics{
			Killed:    71,
			NotViable: 12,
			Score:     report.MutationScore(71, 191, 12),
		},
	}
}

func TestFormatPRComment_ScoreMatchesQualityGate(t *testing.T) {
	summary := notViableSummary()
	gate := NewQualityGateEvaluator(true, 39).Evaluate(summary)

	if !gate.Pass {
		t.Fatalf("gate must pass at %.2f%% against threshold 39", gate.MutationScore)
	}

	comment := NewGitHubIntegration("token", "owner/repo", 1).formatPRComment(summary, gate)

	want := fmt.Sprintf("**Overall Mutation Score:** %.1f%%", gate.MutationScore)
	if !strings.Contains(comment, want) {
		t.Errorf("comment must show the gate score %q, got:\n%s", want, comment)
	}

	if !strings.Contains(comment, "**Overall Mutation Score:** 39.7%") {
		t.Errorf("expected 39.7%% in comment, got:\n%s", comment)
	}

	if !strings.Contains(comment, "Quality Gate: PASSED") {
		t.Errorf("expected PASSED in comment, got:\n%s", comment)
	}

	if !strings.Contains(comment, "**Not viable:** 12") {
		t.Errorf("expected not-viable count in comment, got:\n%s", comment)
	}
}

func TestFormatPRComment_NilQualityResultExcludesNotViable(t *testing.T) {
	comment := NewGitHubIntegration("token", "owner/repo", 1).formatPRComment(notViableSummary(), nil)

	if !strings.Contains(comment, "**Overall Mutation Score:** 39.7%") {
		t.Errorf("expected 39.7%% in comment, got:\n%s", comment)
	}
}

func TestFormatPRComment_DisplaysRepoRelativePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	rel := filepath.Join("internal", "execution", "very", "deeply", "nested", "package", "path", "example_file.go")
	abs := filepath.Join(wd, rel)

	summary := &report.Summary{
		TotalMutants:  10,
		KilledMutants: 8,
		Files: map[string]*report.FileReport{
			abs: {FilePath: abs, TotalMutants: 10, KilledMutants: 8, MutationScore: 80},
		},
	}
	gate := &QualityGateResult{Pass: true, MutationScore: 80}

	comment := NewGitHubIntegration("token", "owner/repo", 1).formatPRComment(summary, gate)

	if !strings.Contains(comment, "| "+rel+" |") {
		t.Errorf("expected repo-relative path %q in comment, got:\n%s", rel, comment)
	}

	if strings.Contains(comment, wd) || strings.Contains(comment, "...") {
		t.Errorf("comment must not contain the absolute or truncated path, got:\n%s", comment)
	}
}

func TestReporter_HTMLDisplaysRepoRelativePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	rel := filepath.Join("internal", "foo.go")
	abs := filepath.Join(wd, rel)

	summary := &report.Summary{
		TotalMutants:  10,
		KilledMutants: 8,
		Files: map[string]*report.FileReport{
			abs: {FilePath: abs, TotalMutants: 10, KilledMutants: 8, MutationScore: 80},
		},
	}
	dir := t.TempDir()

	if err := NewReporter(dir, "html").Generate(summary, nil); err != nil {
		t.Fatalf("html Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "mutation-report.html"))
	if err != nil {
		t.Fatalf("read html report: %v", err)
	}

	html := string(data)

	if !strings.Contains(html, "<td>"+rel+"</td>") {
		t.Errorf("expected repo-relative path %q in html report, got:\n%s", rel, html)
	}

	if strings.Contains(html, wd) {
		t.Errorf("html report must not contain the absolute path %q, got:\n%s", wd, html)
	}
}

func TestReporter_ScoreExcludesNotViableWithoutGate(t *testing.T) {
	summary := notViableSummary()
	dir := t.TempDir()

	if err := NewReporter(dir, "json").Generate(summary, nil); err != nil {
		t.Fatalf("json Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "mutation-report.json"))
	if err != nil {
		t.Fatalf("read json report: %v", err)
	}

	if !strings.Contains(string(data), `"mutationScore": 39.66`) {
		t.Errorf("json score must exclude not-viable mutants, got:\n%s", data)
	}

	if err := NewReporter(dir, "html").Generate(summary, nil); err != nil {
		t.Fatalf("html Generate: %v", err)
	}

	html, err := os.ReadFile(filepath.Join(dir, "mutation-report.html"))
	if err != nil {
		t.Fatalf("read html report: %v", err)
	}

	if got := strings.Count(string(html), "39.7%"); got != 2 {
		t.Errorf("html must show 39.7%% for the overall and the file score, found %d times:\n%s", got, html)
	}
}
