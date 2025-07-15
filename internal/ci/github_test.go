package ci

import (
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/report"
)

func TestGitHubIntegration_CreatePRComment(t *testing.T) {
	// Test without token (should return error)
	github := NewGitHubIntegration("", "owner/repo", 123)

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 85,
		Files: map[string]*report.FileReport{
			"example.go": {
				FilePath:      "example.go",
				TotalMutants:  50,
				KilledMutants: 45,
				MutationScore: 90.0,
			},
		},
	}

	qualityResult := &QualityGateResult{
		Pass:          true,
		MutationScore: 85.0,
		Reason:        "Mutation score meets minimum threshold",
	}

	err := github.CreatePRComment(summary, qualityResult)
	if err == nil {
		t.Error("Expected error when token is empty")
	}
}

func TestGitHubIntegration_formatPRComment(t *testing.T) {
	github := NewGitHubIntegration("token", "owner/repo", 123)

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 85,
		Files: map[string]*report.FileReport{
			"example.go": {
				FilePath:      "example.go",
				TotalMutants:  50,
				KilledMutants: 45,
				MutationScore: 90.0,
			},
			"utils.go": {
				FilePath:      "utils.go",
				TotalMutants:  50,
				KilledMutants: 40,
				MutationScore: 80.0,
			},
		},
	}

	qualityResult := &QualityGateResult{
		Pass:          true,
		MutationScore: 85.0,
		Reason:        "Mutation score meets minimum threshold",
	}

	comment := github.formatPRComment(summary, qualityResult)

	// Check that comment contains expected elements
	expectedElements := []string{
		"## Mutation Testing Results",
		"✅ Quality Gate: PASSED",
		"**Overall Mutation Score:** 85.0%",
		"| File | Score | Mutants | Killed |",
		"example.go",
		"utils.go",
		"90.0%",
		"80.0%",
	}

	for _, element := range expectedElements {
		if !strings.Contains(comment, element) {
			t.Errorf("Expected comment to contain '%s'", element)
		}
	}

	// Test failed quality gate
	qualityResult.Pass = false
	qualityResult.Reason = "Mutation score below threshold"

	comment = github.formatPRComment(summary, qualityResult)

	if !strings.Contains(comment, "❌ Quality Gate: FAILED") {
		t.Error("Expected failed quality gate indicator")
	}

	if !strings.Contains(comment, "Mutation score below threshold") {
		t.Error("Expected failure reason in comment")
	}
}

func TestGitHubIntegration_formatPRComment_EmptyFiles(t *testing.T) {
	github := NewGitHubIntegration("token", "owner/repo", 123)

	summary := &report.Summary{
		TotalMutants:  0,
		KilledMutants: 0,
		Files:         map[string]*report.FileReport{},
	}

	qualityResult := &QualityGateResult{
		Pass:          false,
		MutationScore: 0.0,
		Reason:        "No mutants generated",
	}

	comment := github.formatPRComment(summary, qualityResult)

	if !strings.Contains(comment, "No files analyzed") {
		t.Error("Expected 'No files analyzed' message for empty files")
	}
}
