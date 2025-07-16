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
