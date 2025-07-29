package ci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/report"
)

func TestReporter_Generate(t *testing.T) {
	tmpDir := t.TempDir()

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

	qualityGate := NewQualityGateEvaluator(true, 80.0)

	testCases := []struct {
		name         string
		format       string
		expectedFile string
		checkContent func(string) bool
	}{
		{
			name:         "JSON format",
			format:       "json",
			expectedFile: "mutation-report.json",
			checkContent: func(content string) bool {
				var result map[string]any

				return json.Unmarshal([]byte(content), &result) == nil
			},
		},
		{
			name:         "HTML format",
			format:       "html",
			expectedFile: "mutation-report.html",
			checkContent: func(content string) bool {
				return strings.Contains(content, "<html>") &&
					strings.Contains(content, "Mutation Testing Report")
			},
		},
		{
			name:         "Console format",
			format:       "console",
			expectedFile: "",
			checkContent: func(_ string) bool {
				return true // Console output doesn't create a file
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reporter := NewReporter(tmpDir, tc.format)

			err := reporter.Generate(summary, qualityGate)
			if err != nil {
				t.Fatalf("Failed to generate report: %v", err)
			}

			if tc.expectedFile != "" {
				reportPath := filepath.Join(tmpDir, tc.expectedFile)
				if _, err := os.Stat(reportPath); os.IsNotExist(err) {
					t.Errorf("Expected report file %s was not created", tc.expectedFile)

					return
				}

				content, err := os.ReadFile(reportPath)
				if err != nil {
					t.Fatalf("Failed to read report file: %v", err)
				}

				if !tc.checkContent(string(content)) {
					t.Errorf("Report content validation failed for format %s", tc.format)
				}
			}
		})
	}
}

func TestReporter_generateJSONReport(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "json")

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

	err := reporter.generateJSONReport(summary, qualityResult)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(tmpDir, "mutation-report.json")

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read JSON report: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check required fields
	if result["totalMutants"] != float64(100) {
		t.Errorf("Expected totalMutants=100, got %v", result["totalMutants"])
	}

	if result["killedMutants"] != float64(85) {
		t.Errorf("Expected killedMutants=85, got %v", result["killedMutants"])
	}

	if result["mutationScore"] != 85.0 {
		t.Errorf("Expected mutationScore=85.0, got %v", result["mutationScore"])
	}
}

func TestReporter_generateHTMLReport(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "html")

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

	err := reporter.generateHTMLReport(summary, qualityResult)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(tmpDir, "mutation-report.html")

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read HTML report: %v", err)
	}

	contentStr := string(content)

	// Check HTML structure
	expectedElements := []string{
		"<html>",
		"<head>",
		"<body>",
		"Mutation Testing Report",
		"Overall Score: 85.0%",
		"Quality Gate: PASSED",
		"example.go",
		"90.0%",
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Expected HTML to contain '%s'", element)
		}
	}
}

func TestReporter_generateHTMLReport_WithFailedQualityGate(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "html")

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 50,
		Files: map[string]*report.FileReport{
			"example.go": {
				FilePath:      "example.go",
				TotalMutants:  100,
				KilledMutants: 50,
				MutationScore: 50.0,
			},
		},
	}

	qualityResult := &QualityGateResult{
		Pass:          false,
		MutationScore: 50.0,
		Reason:        "Mutation score below minimum threshold",
	}

	err := reporter.generateHTMLReport(summary, qualityResult)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(tmpDir, "mutation-report.html")

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read HTML report: %v", err)
	}

	contentStr := string(content)

	// Check HTML structure for failed quality gate
	expectedElements := []string{
		"Quality Gate: FAILED - Mutation score below minimum threshold",
		"50.0%",
		"#dc3545", // Red color for low score
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Expected HTML to contain '%s'", element)
		}
	}
}

func TestReporter_generateHTMLReport_WithNilQualityGate(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "html")

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

	// Test with nil quality gate
	err := reporter.generateHTMLReport(summary, nil)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(tmpDir, "mutation-report.html")

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read HTML report: %v", err)
	}

	contentStr := string(content)

	// Check HTML structure for nil quality gate
	expectedElements := []string{
		"Overall Score: 85.0%",
		"Quality Gate: No quality gate configured",
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Expected HTML to contain '%s'", element)
		}
	}
}

func TestReporter_generateJSONReport_WithNilQualityGate(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "json")

	tests := []struct {
		name          string
		summary       *report.Summary
		expectedScore float64
	}{
		{
			name: "with mutants",
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 75,
			},
			expectedScore: 75.0,
		},
		{
			name: "zero mutants",
			summary: &report.Summary{
				TotalMutants:  0,
				KilledMutants: 0,
			},
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up previous test file if exists
			os.Remove(filepath.Join(tmpDir, "mutation-report.json"))
			
			err := reporter.generateJSONReport(tt.summary, nil)
			if err != nil {
				t.Fatalf("Failed to generate JSON report: %v", err)
			}

			// Verify file was created
			reportPath := filepath.Join(tmpDir, "mutation-report.json")
			content, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("Failed to read JSON report: %v", err)
			}

			// Verify it's valid JSON
			var result map[string]any
			if err := json.Unmarshal(content, &result); err != nil {
				t.Fatalf("Generated JSON is invalid: %v", err)
			}

			// Check fields for nil quality gate
			if result["mutationScore"] != tt.expectedScore {
				t.Errorf("Expected mutationScore=%v, got %v", tt.expectedScore, result["mutationScore"])
			}

			if result["qualityGatePassed"] != "N/A" {
				t.Errorf("Expected qualityGatePassed=N/A, got %v", result["qualityGatePassed"])
			}

			if result["qualityGateReason"] != noQualityGateMessage {
				t.Errorf("Expected qualityGateReason=%s, got %v", noQualityGateMessage, result["qualityGateReason"])
			}
		})
	}
}

func TestReporter_Generate_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "invalid-format")

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 85,
	}

	// Invalid format should default to JSON
	err := reporter.Generate(summary, nil)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Verify JSON file was created (default behavior)
	reportPath := filepath.Join(tmpDir, "mutation-report.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("Expected JSON report file to be created for invalid format")
	}
}

func TestReporter_getScoreColor(t *testing.T) {
	reporter := NewReporter("", "")

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{
			name:     "high score (>= 80)",
			score:    85.0,
			expected: "#28a745",
		},
		{
			name:     "exact 80",
			score:    80.0,
			expected: "#28a745",
		},
		{
			name:     "medium score (60-79)",
			score:    70.0,
			expected: "#ffc107",
		},
		{
			name:     "exact 60",
			score:    60.0,
			expected: "#ffc107",
		},
		{
			name:     "low score (< 60)",
			score:    50.0,
			expected: "#dc3545",
		},
		{
			name:     "very low score",
			score:    10.0,
			expected: "#dc3545",
		},
		{
			name:     "zero score",
			score:    0.0,
			expected: "#dc3545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := reporter.getScoreColor(tt.score)
			if color != tt.expected {
				t.Errorf("Expected color %s for score %.1f, got %s", tt.expected, tt.score, color)
			}
		})
	}
}

func TestReporter_generateJSONReport_WriteError(t *testing.T) {
	// Use a directory that doesn't exist to trigger write error
	reporter := NewReporter("/nonexistent/path", "json")

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 85,
	}

	err := reporter.generateJSONReport(summary, nil)
	if err == nil {
		t.Error("Expected error when writing to non-existent directory")
	}
	if !strings.Contains(err.Error(), "failed to write JSON report") {
		t.Errorf("Expected error to contain 'failed to write JSON report', got: %v", err)
	}
}

func TestReporter_generateHTMLReport_WriteError(t *testing.T) {
	// Use a directory that doesn't exist to trigger write error
	reporter := NewReporter("/nonexistent/path", "html")

	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 85,
		Files:         map[string]*report.FileReport{},
	}

	err := reporter.generateHTMLReport(summary, nil)
	if err == nil {
		t.Error("Expected error when writing to non-existent directory")
	}
	if !strings.Contains(err.Error(), "failed to write HTML report") {
		t.Errorf("Expected error to contain 'failed to write HTML report', got: %v", err)
	}
}

func TestReporter_generateHTMLReport_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	reporter := NewReporter(tmpDir, "html")

	tests := []struct {
		name    string
		summary *report.Summary
		quality *QualityGateResult
	}{
		{
			name: "empty files map",
			summary: &report.Summary{
				TotalMutants:  0,
				KilledMutants: 0,
				Files:         map[string]*report.FileReport{},
			},
			quality: nil,
		},
		{
			name: "file with zero mutants",
			summary: &report.Summary{
				TotalMutants:  0,
				KilledMutants: 0,
				Files: map[string]*report.FileReport{
					"empty.go": {
						FilePath:      "empty.go",
						TotalMutants:  0,
						KilledMutants: 0,
						MutationScore: 0.0,
					},
				},
			},
			quality: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reporter.generateHTMLReport(tt.summary, tt.quality)
			if err != nil {
				t.Fatalf("Failed to generate HTML report: %v", err)
			}

			// Verify file was created
			reportPath := filepath.Join(tmpDir, "mutation-report.html")
			if _, err := os.Stat(reportPath); os.IsNotExist(err) {
				t.Error("Expected HTML report file to be created")
			}
		})
	}
}
