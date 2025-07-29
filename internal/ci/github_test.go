package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestGitHubIntegration_ListPRComments(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectError    bool
		expectComments int
	}{
		{
			name:           "successful list",
			responseStatus: http.StatusOK,
			responseBody: `[
				{"id": 1, "body": "First comment"},
				{"id": 2, "body": "Second comment"}
			]`,
			expectError:    false,
			expectComments: 2,
		},
		{
			name:           "authentication failure",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"message": "Bad credentials"}`,
			expectError:    true,
			expectComments: 0,
		},
		{
			name:           "empty list",
			responseStatus: http.StatusOK,
			responseBody:   `[]`,
			expectError:    false,
			expectComments: 0,
		},
		{
			name:           "invalid JSON response",
			responseStatus: http.StatusOK,
			responseBody:   `{invalid json`,
			expectError:    true,
			expectComments: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "token test-token" {
					t.Errorf("Expected authorization header 'token test-token', got %s", r.Header.Get("Authorization"))
				}
				expectedPath := "/repos/owner/repo/issues/123/comments"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Send response
				w.WriteHeader(tt.responseStatus)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			// Create GitHub integration with test server
			github := NewGitHubIntegration("test-token", "owner/repo", 123)
			github.apiBase = server.URL

			// Execute test
			comments, err := github.ListPRComments(context.Background())

			// Verify results
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(comments) != tt.expectComments {
				t.Errorf("Expected %d comments, got %d", tt.expectComments, len(comments))
			}
			
			// Verify comment contents for successful case
			if tt.name == "successful list" && len(comments) == 2 {
				if comments[0].ID != 1 || comments[0].Body != "First comment" {
					t.Errorf("First comment mismatch: got %+v", comments[0])
				}
				if comments[1].ID != 2 || comments[1].Body != "Second comment" {
					t.Errorf("Second comment mismatch: got %+v", comments[1])
				}
			}
		})
	}
}

func TestGitHubIntegration_DeletePRComment(t *testing.T) {
	tests := []struct {
		name           string
		commentID      int
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful deletion",
			commentID:      456,
			responseStatus: http.StatusNoContent,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "comment not found",
			commentID:      999,
			responseStatus: http.StatusNotFound,
			responseBody:   `{"message": "Not Found"}`,
			expectError:    true,
		},
		{
			name:           "unauthorized",
			commentID:      123,
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"message": "Bad credentials"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "token test-token" {
					t.Errorf("Expected authorization header 'token test-token', got %s", r.Header.Get("Authorization"))
				}
				expectedPath := fmt.Sprintf("/repos/owner/repo/issues/comments/%d", tt.commentID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Send response
				w.WriteHeader(tt.responseStatus)
				if tt.responseBody != "" {
					fmt.Fprint(w, tt.responseBody)
				}
			}))
			defer server.Close()

			// Create GitHub integration with test server
			github := NewGitHubIntegration("test-token", "owner/repo", 123)
			github.apiBase = server.URL

			// Execute test
			err := github.DeletePRComment(context.Background(), tt.commentID)

			// Verify results
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGitHubIntegration_deleteExistingMutationComments(t *testing.T) {
	tests := []struct {
		name        string
		comments    []Comment
		expectError bool
		deleteCount int
	}{
		{
			name: "deletes mutation testing comments",
			comments: []Comment{
				{ID: 1, Body: "Regular comment"},
				{ID: 2, Body: "🧬 Mutation Testing Results\nTest data"},
				{ID: 3, Body: "Another comment"},
				{ID: 4, Body: "Some text\nGenerated by gomu mutation testing"},
			},
			expectError: false,
			deleteCount: 2, // Should delete comments 2 and 4
		},
		{
			name: "no mutation comments to delete",
			comments: []Comment{
				{ID: 1, Body: "Regular comment"},
				{ID: 2, Body: "Another regular comment"},
			},
			expectError: false,
			deleteCount: 0,
		},
		{
			name:        "empty comment list",
			comments:    []Comment{},
			expectError: false,
			deleteCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCallCount := 0
			
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments") {
					// List comments endpoint
					data, _ := json.Marshal(tt.comments)
					w.WriteHeader(http.StatusOK)
					w.Write(data)
				} else if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/comments/") {
					// Delete comment endpoint
					deleteCallCount++
					w.WriteHeader(http.StatusNoContent)
				} else {
					t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Create GitHub integration with test server
			github := NewGitHubIntegration("test-token", "owner/repo", 123)
			github.apiBase = server.URL

			// Execute test
			err := github.deleteExistingMutationComments(context.Background())

			// Verify results
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if deleteCallCount != tt.deleteCount {
				t.Errorf("Expected %d delete calls, got %d", tt.deleteCount, deleteCallCount)
			}
		})
	}
}
