package ci

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
	"github.com/sivchari/gomu/internal/report"
)

// errorSummary mirrors a run of 20 mutants: 8 killed, 2 timed out, 4 survived,
// 4 errors and 2 not viable. Score is (8+2)/(20-2-4) = 71.4%.
func errorSummary() *report.Summary {
	in := report.ScoreInput{Killed: 8, TimedOut: 2, Total: 20, NotViable: 2, Errors: 4}
	score := report.MutationScore(in)

	return &report.Summary{
		TotalMutants:  in.Total,
		KilledMutants: in.Killed,
		Files: map[string]*report.FileReport{
			"internal/foo.go": {
				FilePath:      "internal/foo.go",
				TotalMutants:  10,
				KilledMutants: 5,
				TimedOut:      1,
				Errors:        2,
				NotViable:     1,
				MutationScore: report.MutationScore(report.ScoreInput{Killed: 5, TimedOut: 1, Total: 10, NotViable: 1, Errors: 2}),
			},
			"internal/bar.go": {
				FilePath:      "internal/bar.go",
				TotalMutants:  10,
				KilledMutants: 3,
				TimedOut:      1,
				Errors:        2,
				NotViable:     1,
				MutationScore: report.MutationScore(report.ScoreInput{Killed: 3, TimedOut: 1, Total: 10, NotViable: 1, Errors: 2}),
			},
		},
		Statistics: report.Statistics{
			Killed:    in.Killed,
			TimedOut:  in.TimedOut,
			Errors:    in.Errors,
			NotViable: in.NotViable,
			Score:     score,
		},
	}
}

// statusSummary runs the report generator over results so Statistics.Score is
// computed by the production code path.
func statusSummary(t *testing.T, statuses ...mutation.Status) *report.Summary {
	t.Helper()

	t.Chdir(t.TempDir())

	results := make([]mutation.Result, 0, len(statuses))
	for _, status := range statuses {
		results = append(results, mutation.Result{Status: status})
	}

	gen, err := report.New("json")
	if err != nil {
		t.Fatalf("report.New: %v", err)
	}

	summary := &report.Summary{TotalMutants: len(results), Results: results}
	if err := gen.Generate(summary); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	return summary
}

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
				MutationScore: report.MutationScore(report.ScoreInput{Killed: 71, Total: 191, NotViable: 12}),
			},
		},
		Statistics: report.Statistics{
			Killed:    71,
			NotViable: 12,
			Score:     report.MutationScore(report.ScoreInput{Killed: 71, Total: 191, NotViable: 12}),
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

func TestQualityGate_ScoreFormula(t *testing.T) {
	tests := []struct {
		name     string
		statuses []mutation.Status
		minScore float64
		wantPass bool
		want     float64
	}{
		{
			name:     "errors leave the denominator",
			statuses: []mutation.Status{mutation.StatusKilled, mutation.StatusKilled, mutation.StatusSurvived, mutation.StatusError, mutation.StatusError},
			minScore: 60,
			wantPass: true,
			want:     200.0 / 3.0,
		},
		{
			name:     "counting errors in the denominator would fail this gate",
			statuses: []mutation.Status{mutation.StatusKilled, mutation.StatusError, mutation.StatusError, mutation.StatusError},
			minScore: 90,
			wantPass: true,
			want:     100,
		},
		{
			name:     "timed out counts as detected",
			statuses: []mutation.Status{mutation.StatusKilled, mutation.StatusTimedOut, mutation.StatusSurvived, mutation.StatusSurvived},
			minScore: 50,
			wantPass: true,
			want:     50,
		},
		{
			name:     "timed out stays in the denominator",
			statuses: []mutation.Status{mutation.StatusKilled, mutation.StatusTimedOut, mutation.StatusSurvived, mutation.StatusSurvived},
			minScore: 51,
			wantPass: false,
			want:     50,
		},
		{
			name:     "every mutant errored or not viable fails",
			statuses: []mutation.Status{mutation.StatusError, mutation.StatusError, mutation.StatusNotViable},
			minScore: 1,
			wantPass: false,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := statusSummary(t, tt.statuses...)

			gate := NewQualityGateEvaluator(true, tt.minScore).Evaluate(summary)
			if gate.Pass != tt.wantPass {
				t.Errorf("Pass = %v, want %v (score %.2f)", gate.Pass, tt.wantPass, gate.MutationScore)
			}

			if math.Abs(gate.MutationScore-tt.want) > 1e-9 {
				t.Errorf("MutationScore = %v, want %v", gate.MutationScore, tt.want)
			}
		})
	}
}

func TestFormatPRComment_ScoreFormula(t *testing.T) {
	gate := NewQualityGateEvaluator(true, 70).Evaluate(errorSummary())
	if !gate.Pass {
		t.Fatalf("gate must pass at %.2f%% against threshold 70", gate.MutationScore)
	}

	tests := []struct {
		name string
		gate *QualityGateResult
	}{
		{name: "with quality gate", gate: gate},
		{name: "without quality gate", gate: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := NewGitHubIntegration("token", "owner/repo", 1).formatPRComment(errorSummary(), tt.gate)

			for _, want := range []string{
				"**Overall Mutation Score:** 71.4%",
				"(killed + timed out) / (total - not viable - errors)",
				"**Killed:** 8",
				"**Timed out:** 2",
				"**Not viable:** 2",
				"**Errors:** 4",
				"| File | Score | Mutants | Killed | Timed Out | Not Viable | Errors |",
				"| internal/foo.go | 85.7% | 10 | 5 | 1 | 1 | 2 |",
				"| internal/bar.go | 57.1% | 10 | 3 | 1 | 1 | 2 |",
			} {
				if !strings.Contains(comment, want) {
					t.Errorf("expected %q in comment, got:\n%s", want, comment)
				}
			}
		})
	}
}

func TestFormatPRComment_ScoreFormulaWithoutFiles(t *testing.T) {
	summary := errorSummary()
	summary.Files = nil

	comment := NewGitHubIntegration("token", "owner/repo", 1).formatPRComment(summary, nil)

	for _, want := range []string{"**Overall Mutation Score:** 71.4%", "**Timed out:** 2", "**Errors:** 4"} {
		if !strings.Contains(comment, want) {
			t.Errorf("expected %q in comment, got:\n%s", want, comment)
		}
	}
}

func TestFormatPRComment_AllErroredScoresZero(t *testing.T) {
	summary := &report.Summary{
		TotalMutants: 5,
		Files: map[string]*report.FileReport{
			"a.go": {FilePath: "a.go", TotalMutants: 5, Errors: 3, NotViable: 2},
		},
		Statistics: report.Statistics{Errors: 3, NotViable: 2},
	}

	comment := NewGitHubIntegration("token", "owner/repo", 1).formatPRComment(summary, nil)

	for _, want := range []string{"**Overall Mutation Score:** 0.0%", "Quality Gate: FAILED", "**Errors:** 3"} {
		if !strings.Contains(comment, want) {
			t.Errorf("expected %q in comment, got:\n%s", want, comment)
		}
	}
}

func TestReporter_ScoreCountsErrorsAndTimeoutsWithoutGate(t *testing.T) {
	dir := t.TempDir()
	summary := errorSummary()

	if err := NewReporter(dir, "json").Generate(summary, nil); err != nil {
		t.Fatalf("json Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "mutation-report.json"))
	if err != nil {
		t.Fatalf("read json report: %v", err)
	}

	var got struct {
		MutationScore float64                       `json:"mutationScore"`
		Files         map[string]*report.FileReport `json:"files"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal json report: %v", err)
	}

	if want := 10.0 / 14.0 * 100; math.Abs(got.MutationScore-want) > 1e-9 {
		t.Errorf("json score = %v, want %v", got.MutationScore, want)
	}

	if f := got.Files["internal/foo.go"]; f == nil || f.Errors != 2 || f.TimedOut != 1 {
		t.Errorf("json file report must carry errors and timedOut, got %+v", f)
	}

	if err := NewReporter(dir, "html").Generate(summary, nil); err != nil {
		t.Fatalf("html Generate: %v", err)
	}

	html, err := os.ReadFile(filepath.Join(dir, "mutation-report.html"))
	if err != nil {
		t.Fatalf("read html report: %v", err)
	}

	// foo.go: (5+1)/(10-1-2) = 85.7%; bar.go: (3+1)/7 = 57.1%; overall 71.4%.
	for _, want := range []string{
		"Overall Score: 71.4%",
		"(killed + timed out) / (total - not viable - errors)",
		"<p>Killed: 8</p>",
		"<p>Timed out: 2</p>",
		"<p>Not viable: 2</p>",
		"<p>Errors: 4</p>",
		"<th>Killed</th>",
		"<th>Timed Out</th>",
		"<th>Not Viable</th>",
		"<th>Errors</th>",
		"<td>10</td>\n            <td>5</td>\n            <td>1</td>\n            <td>1</td>\n            <td>2</td>\n            <td>85.7%</td>",
		"<td>57.1%</td>",
	} {
		if !strings.Contains(string(html), want) {
			t.Errorf("html report must contain %q, got:\n%s", want, html)
		}
	}
}

func TestReporter_ConsoleShowsScoreInputs(t *testing.T) {
	summary := errorSummary()
	gate := NewQualityGateEvaluator(true, 70).Evaluate(summary)

	tests := []struct {
		name string
		gate *QualityGateResult
		want []string
	}{
		{
			name: "with quality gate",
			gate: gate,
			want: []string{"Quality Gate: PASSED"},
		},
		{
			name: "without quality gate",
			gate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder

			writeConsoleReport(&buf, summary, tt.gate)

			want := append([]string{
				"Mutation Score: 71.4% ((killed + timed out) / (total - not viable - errors))",
				"Total Mutants: 20",
				"Killed: 8",
				"Timed out: 2",
				"Not viable: 2",
				"Errors: 4",
			}, tt.want...)

			for _, w := range want {
				if !strings.Contains(buf.String(), w) {
					t.Errorf("console report must contain %q, got:\n%s", w, buf.String())
				}
			}
		})
	}
}
