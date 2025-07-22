package ci

import (
	"encoding/json"
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

	err := github.CreatePRComment(t.Context(), summary, qualityResult)
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
		"## 🧬 Mutation Testing Results",
		"✅ **Quality Gate: PASSED**",
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

	if !strings.Contains(comment, "❌ **Quality Gate: FAILED**") {
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

func TestNewGitHubIntegration(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		repository string
		prNumber   int
	}{
		{
			name:       "creates integration with valid parameters",
			token:      "ghp_test123",
			repository: "owner/repo",
			prNumber:   42,
		},
		{
			name:       "creates integration with empty token",
			token:      "",
			repository: "owner/repo",
			prNumber:   1,
		},
		{
			name:       "creates integration with zero PR number",
			token:      "token",
			repository: "owner/repo",
			prNumber:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			github := NewGitHubIntegration(tt.token, tt.repository, tt.prNumber)

			if github == nil {
				t.Error("NewGitHubIntegration returned nil")

				return
			}

			if github.token != tt.token {
				t.Errorf("Expected token %s, got %s", tt.token, github.token)
			}

			if github.repository != tt.repository {
				t.Errorf("Expected repository %s, got %s", tt.repository, github.repository)
			}

			if github.prNumber != tt.prNumber {
				t.Errorf("Expected PR number %d, got %d", tt.prNumber, github.prNumber)
			}

			if github.client == nil {
				t.Error("HTTP client should not be nil")
			}

			if github.apiBase != "https://api.github.com" {
				t.Errorf("Expected API base https://api.github.com, got %s", github.apiBase)
			}
		})
	}
}

func TestPRComment_JSON(t *testing.T) {
	comment := PRComment{Body: "Test comment body"}

	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("Failed to marshal PRComment: %v", err)
	}

	var decoded PRComment

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal PRComment: %v", err)
	}

	if decoded.Body != comment.Body {
		t.Errorf("Expected body %s, got %s", comment.Body, decoded.Body)
	}
}

func TestComment_JSON(t *testing.T) {
	comment := Comment{ID: 123, Body: "Test comment"}

	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("Failed to marshal Comment: %v", err)
	}

	var decoded Comment

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Comment: %v", err)
	}

	if decoded.ID != comment.ID {
		t.Errorf("Expected ID %d, got %d", comment.ID, decoded.ID)
	}

	if decoded.Body != comment.Body {
		t.Errorf("Expected body %s, got %s", comment.Body, decoded.Body)
	}
}

func TestGitHubIntegration_formatPRComment_NilQualityResult(t *testing.T) {
	github := NewGitHubIntegration("token", "owner/repo", 123)

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 75,
		Files: map[string]*report.FileReport{
			"test.go": {
				FilePath:      "test.go",
				TotalMutants:  100,
				KilledMutants: 75,
				MutationScore: 75.0,
			},
		},
	}

	comment := github.formatPRComment(summary, nil)

	// Should handle nil quality result gracefully
	if !strings.Contains(comment, "75.0%") {
		t.Error("Expected calculated mutation score in comment")
	}

	if !strings.Contains(comment, "## 🧬 Mutation Testing Results") {
		t.Error("Expected mutation testing results header")
	}
}

func TestGitHubIntegration_formatPRComment_EdgeCases(t *testing.T) {
	github := NewGitHubIntegration("token", "owner/repo", 123)

	tests := []struct {
		name          string
		summary       *report.Summary
		qualityResult *QualityGateResult
		expectContain string
	}{
		{
			name: "zero mutation score",
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 0,
			},
			qualityResult: &QualityGateResult{
				Pass:          false,
				MutationScore: 0.0,
				Reason:        "No mutants killed",
			},
			expectContain: "0.0%",
		},
		{
			name: "perfect mutation score",
			summary: &report.Summary{
				TotalMutants:  50,
				KilledMutants: 50,
			},
			qualityResult: &QualityGateResult{
				Pass:          true,
				MutationScore: 100.0,
				Reason:        "Perfect score",
			},
			expectContain: "100.0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := github.formatPRComment(tt.summary, tt.qualityResult)

			if !strings.Contains(comment, tt.expectContain) {
				t.Errorf("Expected comment to contain '%s'", tt.expectContain)
			}
		})
	}
}
