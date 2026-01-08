package report

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sivchari/gomu/internal/mutation"
)

func abs(x float64) float64 {
	return math.Abs(x)
}

func TestCalculateStatistics(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	results := []mutation.Result{
		{Mutant: mutation.Mutant{ID: "1", Type: "arithmetic"}, Status: mutation.StatusKilled},
		{Mutant: mutation.Mutant{ID: "2", Type: "arithmetic"}, Status: mutation.StatusKilled},
		{Mutant: mutation.Mutant{ID: "3", Type: "conditional"}, Status: mutation.StatusSurvived},
		{Mutant: mutation.Mutant{ID: "4", Type: "logical"}, Status: mutation.StatusTimedOut},
		{Mutant: mutation.Mutant{ID: "5", Type: "arithmetic"}, Status: mutation.StatusError},
		{Mutant: mutation.Mutant{ID: "6", Type: "conditional"}, Status: mutation.StatusNotViable},
	}

	stats := generator.calculateStatistics(results)

	if stats.Killed != 2 {
		t.Errorf("Expected Killed 2, got %d", stats.Killed)
	}

	if stats.Survived != 1 {
		t.Errorf("Expected Survived 1, got %d", stats.Survived)
	}

	if stats.TimedOut != 1 {
		t.Errorf("Expected TimedOut 1, got %d", stats.TimedOut)
	}

	if stats.Errors != 1 {
		t.Errorf("Expected Errors 1, got %d", stats.Errors)
	}

	if stats.NotViable != 1 {
		t.Errorf("Expected NotViable 1, got %d", stats.NotViable)
	}

	// Score should be 2/5 * 100 = 40.0 (excluding NOT_VIABLE)
	expectedScore := 2.0 / 5.0 * 100
	if abs(stats.Score-expectedScore) > 0.000001 {
		t.Errorf("Expected Score %f, got %f", expectedScore, stats.Score)
	}

	// Test mutation type statistics
	if stats.MutationTypes == nil {
		t.Error("Expected MutationTypes to be initialized")
	}

	if len(stats.MutationTypes) != 3 {
		t.Errorf("Expected 3 mutation types, got %d", len(stats.MutationTypes))
	}

	// Test arithmetic mutations
	if arithmeticStats, ok := stats.MutationTypes["arithmetic"]; ok {
		if arithmeticStats.Total != 3 {
			t.Errorf("Expected arithmetic total 3, got %d", arithmeticStats.Total)
		}

		if arithmeticStats.Killed != 2 {
			t.Errorf("Expected arithmetic killed 2, got %d", arithmeticStats.Killed)
		}

		if arithmeticStats.Survived != 0 {
			t.Errorf("Expected arithmetic survived 0, got %d", arithmeticStats.Survived)
		}
	} else {
		t.Error("Expected arithmetic mutation type to exist")
	}

	// Test conditional mutations
	if conditionalStats, ok := stats.MutationTypes["conditional"]; ok {
		if conditionalStats.Total != 2 {
			t.Errorf("Expected conditional total 2, got %d", conditionalStats.Total)
		}

		if conditionalStats.Killed != 0 {
			t.Errorf("Expected conditional killed 0, got %d", conditionalStats.Killed)
		}

		if conditionalStats.Survived != 1 {
			t.Errorf("Expected conditional survived 1, got %d", conditionalStats.Survived)
		}
	} else {
		t.Error("Expected conditional mutation type to exist")
	}

	// Test logical mutations
	if logicalStats, ok := stats.MutationTypes["logical"]; ok {
		if logicalStats.Total != 1 {
			t.Errorf("Expected logical total 1, got %d", logicalStats.Total)
		}

		if logicalStats.Killed != 0 {
			t.Errorf("Expected logical killed 0, got %d", logicalStats.Killed)
		}

		if logicalStats.Survived != 0 {
			t.Errorf("Expected logical survived 0, got %d", logicalStats.Survived)
		}
	} else {
		t.Error("Expected logical mutation type to exist")
	}
}

func TestCalculateStatistics_EmptyResults(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	stats := generator.calculateStatistics([]mutation.Result{})

	if stats.Killed != 0 {
		t.Errorf("Expected Killed 0, got %d", stats.Killed)
	}

	if stats.Survived != 0 {
		t.Errorf("Expected Survived 0, got %d", stats.Survived)
	}

	if stats.TimedOut != 0 {
		t.Errorf("Expected TimedOut 0, got %d", stats.TimedOut)
	}

	if stats.Errors != 0 {
		t.Errorf("Expected Errors 0, got %d", stats.Errors)
	}

	if stats.NotViable != 0 {
		t.Errorf("Expected NotViable 0, got %d", stats.NotViable)
	}

	if stats.Score != 0 {
		t.Errorf("Expected Score 0, got %f", stats.Score)
	}

	if stats.Coverage != 0 {
		t.Errorf("Expected Coverage 0, got %f", stats.Coverage)
	}

	if stats.MutationTypes == nil {
		t.Error("Expected MutationTypes to be initialized")
	}

	if len(stats.MutationTypes) != 0 {
		t.Errorf("Expected MutationTypes to be empty, got %d entries", len(stats.MutationTypes))
	}
}

func TestGenerateJSON(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	summary := &Summary{
		TotalFiles:     2,
		ProcessedFiles: 2,
		TotalMutants:   5,
		Results: []mutation.Result{
			{
				Mutant: mutation.Mutant{
					ID:          "test1",
					FilePath:    "test.go",
					Line:        10,
					Column:      5,
					Type:        "arithmetic",
					Original:    "+",
					Mutated:     "-",
					Description: "Replace + with -",
				},
				Status: mutation.StatusKilled,
			},
		},
		Duration: time.Second * 5,
		Statistics: Statistics{
			Killed:   1,
			Survived: 0,
			Score:    100.0,
		},
	}

	err = generator.Generate(summary)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify standard output file was created
	standardFile := "mutation-report.json"
	if _, err := os.Stat(standardFile); os.IsNotExist(err) {
		t.Error("Standard output file was not created")
	}

	// Verify JSON content
	data, err := os.ReadFile(standardFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Cleanup
	defer os.Remove(standardFile)

	var parsedSummary Summary

	err = json.Unmarshal(data, &parsedSummary)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsedSummary.TotalFiles != summary.TotalFiles {
		t.Errorf("Expected TotalFiles %d, got %d", summary.TotalFiles, parsedSummary.TotalFiles)
	}

	if parsedSummary.Statistics.Score != summary.Statistics.Score {
		t.Errorf("Expected Score %f, got %f", summary.Statistics.Score, parsedSummary.Statistics.Score)
	}
}

func TestGenerateText(t *testing.T) {
	generator, err := New("text")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	summary := &Summary{
		TotalFiles:     2,
		ProcessedFiles: 2,
		TotalMutants:   3,
		Results: []mutation.Result{
			{
				Mutant: mutation.Mutant{
					ID:          "test1",
					FilePath:    "test.go",
					Line:        10,
					Column:      5,
					Type:        "arithmetic",
					Original:    "+",
					Mutated:     "-",
					Description: "Replace + with -",
				},
				Status: mutation.StatusKilled,
			},
			{
				Mutant: mutation.Mutant{
					ID:          "test2",
					FilePath:    "test.go",
					Line:        15,
					Column:      8,
					Type:        "conditional",
					Original:    "==",
					Mutated:     "!=",
					Description: "Replace == with !=",
				},
				Status: mutation.StatusSurvived,
			},
			{
				Mutant: mutation.Mutant{
					ID:          "test3",
					FilePath:    "test.go",
					Line:        20,
					Column:      3,
					Type:        "logical",
					Original:    "&&",
					Mutated:     "||",
					Description: "Replace && with ||",
				},
				Status: mutation.StatusSurvived,
			},
		},
		Duration: time.Second * 5,
	}

	err = generator.Generate(summary)
	if err != nil {
		t.Fatalf("Failed to generate text report: %v", err)
	}

	// Verify standard output file was created
	standardFile := "mutation-report.txt"
	if _, err := os.Stat(standardFile); os.IsNotExist(err) {
		t.Error("Standard output file was not created")
	}

	// Verify text content
	data, err := os.ReadFile(standardFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Cleanup
	defer os.Remove(standardFile)

	content := string(data)

	// Check for key elements in the report
	expectedElements := []string{
		"Mutation Testing Report",
		"Files processed: 2/2",
		"Total mutants:   3",
		"Duration:        5s",
		"Killed:     1 (33.3%)",
		"Survived:   2 (66.7%)",
		"Mutation Score: 33.3%",
		"Survived Mutants:",
		"test.go:15:8 - Replace == with != (== -> !=)",
		"test.go:20:3 - Replace && with || (&& -> ||)",
	}

	for _, element := range expectedElements {
		if !strings.Contains(content, element) {
			t.Errorf("Expected element '%s' not found in report\nActual content:\n%s", element, content)
		}
	}
}

func TestFormatTextReport(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	summary := &Summary{
		TotalFiles:     1,
		ProcessedFiles: 1,
		TotalMutants:   2,
		Results: []mutation.Result{
			{
				Mutant: mutation.Mutant{
					ID:          "test1",
					FilePath:    "test.go",
					Line:        10,
					Column:      5,
					Type:        "arithmetic",
					Original:    "+",
					Mutated:     "-",
					Description: "Replace + with -",
				},
				Status: mutation.StatusSurvived,
			},
		},
		Duration: time.Millisecond * 1500,
		Statistics: Statistics{
			Killed:   0,
			Survived: 1,
			Score:    0.0,
		},
	}

	report := generator.formatTextReport(summary)

	// Check basic structure
	if !strings.Contains(report, "Mutation Testing Report") {
		t.Error("Report should contain title")
	}

	if !strings.Contains(report, "Files processed: 1/1") {
		t.Error("Report should contain file count")
	}

	if !strings.Contains(report, "Total mutants:   2") {
		t.Error("Report should contain mutant count")
	}

	if !strings.Contains(report, "Duration:        1.5s") {
		t.Error("Report should contain duration")
	}

	if !strings.Contains(report, "Mutation Score: 0.0%") {
		t.Error("Report should contain mutation score")
	}

	// Check survived mutants section
	if !strings.Contains(report, "Survived Mutants:") {
		t.Error("Report should contain survived mutants section")
	}

	if !strings.Contains(report, "test.go:10:5 - Replace + with - (+ -> -)") {
		t.Error("Report should contain survived mutant details")
	}
}

func TestFormatTextReport_NoSurvivedMutants(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	summary := &Summary{
		TotalFiles:     1,
		ProcessedFiles: 1,
		TotalMutants:   1,
		Results: []mutation.Result{
			{
				Mutant: mutation.Mutant{ID: "test1"},
				Status: mutation.StatusKilled,
			},
		},
		Duration: time.Second,
		Statistics: Statistics{
			Killed:   1,
			Survived: 0,
			Score:    100.0,
		},
	}

	report := generator.formatTextReport(summary)

	// Should not contain survived mutants section
	if strings.Contains(report, "Survived Mutants:") {
		t.Error("Report should not contain survived mutants section when there are none")
	}
}

func TestGenerateHTML(t *testing.T) {
	generator, err := New("html")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	summary := &Summary{
		TotalFiles:     2,
		ProcessedFiles: 2,
		TotalMutants:   3,
		Results: []mutation.Result{
			{
				Mutant: mutation.Mutant{
					ID:          "test1",
					FilePath:    "test.go",
					Line:        10,
					Column:      5,
					Type:        "arithmetic",
					Original:    "+",
					Mutated:     "-",
					Description: "Replace + with -",
					Function:    "calculateSum",
					Context:     "func calculateSum(a, b int) int {\n    return a + b\n}",
				},
				Status:        mutation.StatusKilled,
				ExecutionTime: 120,
				TestsRun:      3,
				TestsFailed:   1,
				TestOutput: []mutation.TestInfo{
					{
						Name:     "TestCalculateSum",
						Package:  "calculator",
						Status:   "PASS",
						Duration: 45,
					},
					{
						Name:     "TestCalculateSumNegative",
						Package:  "calculator",
						Status:   "FAIL",
						Duration: 32,
						Output:   "Expected: 0, Got: 4",
					},
					{
						Name:     "TestCalculateSumZero",
						Package:  "calculator",
						Status:   "PASS",
						Duration: 18,
					},
				},
			},
			{
				Mutant: mutation.Mutant{
					ID:          "test2",
					FilePath:    "test.go",
					Line:        15,
					Column:      8,
					Type:        "conditional",
					Original:    "==",
					Mutated:     "!=",
					Description: "Replace == with !=",
					Function:    "checkEqual",
					Context:     "func checkEqual(a, b int) bool {\n    return a == b\n}",
				},
				Status:        mutation.StatusSurvived,
				ExecutionTime: 85,
				TestsRun:      2,
				TestsFailed:   0,
				TestOutput: []mutation.TestInfo{
					{
						Name:     "TestCheckEqual",
						Package:  "calculator",
						Status:   "PASS",
						Duration: 25,
					},
					{
						Name:     "TestCheckEqualFalse",
						Package:  "calculator",
						Status:   "PASS",
						Duration: 18,
					},
				},
			},
		},
		Duration: time.Second * 5,
		Files: map[string]*FileReport{
			"test.go": {
				FilePath:      "test.go",
				TotalMutants:  3,
				KilledMutants: 1,
				MutationScore: 33.3,
			},
		},
	}

	err = generator.Generate(summary)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	// Verify standard output file was created
	standardFile := "mutation-report.html"
	if _, err := os.Stat(standardFile); os.IsNotExist(err) {
		t.Error("Standard output file was not created")
	}

	// Verify HTML content
	data, err := os.ReadFile(standardFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Cleanup
	defer os.Remove(standardFile)

	content := string(data)

	// Check for key elements in the HTML report
	expectedElements := []string{
		"<!DOCTYPE html>",
		"<title>Gomu Mutation Testing Report</title>",
		"<h1>🧬 Gomu Mutation Testing Report</h1>",
		"Files Processed",
		"2/2",
		"Total Mutants",
		"Duration",
		"Mutation Score",
		"Killed",
		"Survived",
		"test.go:15:8",
		"Replace == with !=",
	}

	for _, element := range expectedElements {
		if !strings.Contains(content, element) {
			t.Errorf("Expected element '%s' not found in HTML report", element)
		}
	}
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		part     int
		total    int
		expected float64
	}{
		{0, 0, 0},
		{0, 10, 0},
		{5, 10, 50},
		{10, 10, 100},
		{3, 7, 42.857142857142854},
	}

	for _, tt := range tests {
		result := percentage(tt.part, tt.total)
		if result != tt.expected {
			t.Errorf("percentage(%d, %d) = %f, expected %f", tt.part, tt.total, result, tt.expected)
		}
	}
}

func TestCalculateStatistics_UnknownMutationType(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	results := []mutation.Result{
		{
			Mutant: mutation.Mutant{
				ID:   "1",
				Type: "", // Empty type should become "unknown"
			},
			Status: mutation.StatusKilled,
		},
		{
			Mutant: mutation.Mutant{
				ID:   "2",
				Type: "", // Another empty type
			},
			Status: mutation.StatusSurvived,
		},
	}

	stats := generator.calculateStatistics(results)

	// Check that unknown type is tracked
	if _, ok := stats.MutationTypes["unknown"]; !ok {
		t.Error("Expected 'unknown' mutation type to be tracked")
	}

	unknownStats := stats.MutationTypes["unknown"]
	if unknownStats.Total != 2 {
		t.Errorf("Expected unknown total 2, got %d", unknownStats.Total)
	}

	if unknownStats.Killed != 1 {
		t.Errorf("Expected unknown killed 1, got %d", unknownStats.Killed)
	}

	if unknownStats.Survived != 1 {
		t.Errorf("Expected unknown survived 1, got %d", unknownStats.Survived)
	}
}

func TestCalculateStatistics_AllNotViable(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	results := []mutation.Result{
		{
			Mutant: mutation.Mutant{ID: "1"},
			Status: mutation.StatusNotViable,
		},
		{
			Mutant: mutation.Mutant{ID: "2"},
			Status: mutation.StatusNotViable,
		},
	}

	stats := generator.calculateStatistics(results)

	// When all mutants are not viable, score should be 0
	if stats.Score != 0 {
		t.Errorf("Expected Score 0 when all mutants are not viable, got %f", stats.Score)
	}
}

func TestGenerateJSON_WriteError(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a read-only directory to force write error
	tempDir := t.TempDir()

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	os.Chdir(tempDir)

	os.Chmod(tempDir, 0555) // Read-only
	defer os.Chmod(tempDir, 0755)

	summary := &Summary{
		TotalFiles:   1,
		TotalMutants: 1,
		Results:      []mutation.Result{},
		Duration:     time.Second,
	}

	err = generator.Generate(summary)
	if err == nil {
		t.Error("Expected error when unable to write file")
	}
}

func TestGenerateText_LargeSummary(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a large summary with many survived mutants
	var results []mutation.Result
	for i := 0; i < 100; i++ {
		results = append(results, mutation.Result{
			Mutant: mutation.Mutant{
				ID:          fmt.Sprintf("test%d", i),
				FilePath:    fmt.Sprintf("file%d.go", i/10),
				Line:        i + 1,
				Column:      5,
				Type:        "arithmetic",
				Original:    "+",
				Mutated:     "-",
				Description: fmt.Sprintf("Mutation %d", i),
			},
			Status: mutation.StatusSurvived,
		})
	}

	summary := &Summary{
		TotalFiles:     10,
		ProcessedFiles: 10,
		TotalMutants:   100,
		Results:        results,
		Duration:       time.Minute * 5,
	}

	// This should not panic or error
	err = generator.Generate(summary)
	if err != nil {
		t.Fatalf("Failed to generate report for large summary: %v", err)
	}

	// Cleanup
	defer os.Remove("mutation-report.json")
}

func TestGenerateHTML_ComplexData(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Test with special characters in strings that could break HTML
	summary := &Summary{
		TotalFiles:     1,
		ProcessedFiles: 1,
		TotalMutants:   2,
		Results: []mutation.Result{
			{
				Mutant: mutation.Mutant{
					ID:          "test1",
					FilePath:    "test<script>alert('xss')</script>.go",
					Line:        10,
					Column:      5,
					Type:        "arithmetic",
					Original:    "<",
					Mutated:     ">",
					Description: "Replace < with >",
					Function:    "Compare<T>",
					Context:     "if a < b { return \"less\" }",
				},
				Status: mutation.StatusKilled,
				TestOutput: []mutation.TestInfo{
					{
						Name:   "TestCompare<>",
						Output: "Error: Expected '<', got '>'",
					},
				},
			},
		},
		Duration: time.Second,
		Files: map[string]*FileReport{
			"test.go": {
				FilePath:      "test.go",
				TotalMutants:  2,
				KilledMutants: 1,
				MutationScore: 50.0,
			},
		},
	}

	err = generator.generateHTML(summary)
	if err != nil {
		t.Fatalf("Failed to generate HTML with special characters: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat("mutation-report.html"); os.IsNotExist(err) {
		t.Error("HTML file was not created")
	}

	// Cleanup
	defer os.Remove("mutation-report.html")
}

func TestFormatTextReport_EdgeCases(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	tests := []struct {
		name     string
		summary  *Summary
		contains []string
	}{
		{
			name: "zero duration",
			summary: &Summary{
				TotalFiles:     1,
				ProcessedFiles: 0,
				TotalMutants:   0,
				Results:        []mutation.Result{},
				Duration:       0,
				Statistics: Statistics{
					Killed:   0,
					Survived: 0,
					Score:    0,
				},
			},
			contains: []string{
				"Duration:        0s",
				"Files processed: 0/1",
			},
		},
		{
			name: "all mutation types present",
			summary: &Summary{
				TotalFiles:     1,
				ProcessedFiles: 1,
				TotalMutants:   5,
				Results: []mutation.Result{
					{Status: mutation.StatusKilled},
					{Status: mutation.StatusSurvived},
					{Status: mutation.StatusTimedOut},
					{Status: mutation.StatusError},
					{Status: mutation.StatusNotViable},
				},
				Duration: time.Hour + time.Minute + time.Second,
				Statistics: Statistics{
					Killed:    1,
					Survived:  1,
					TimedOut:  1,
					Errors:    1,
					NotViable: 1,
					Score:     25.0,
				},
			},
			contains: []string{
				"Killed:     1 (20.0%)",
				"Survived:   1 (20.0%)",
				"Timed out:  1 (20.0%)",
				"Errors:     1 (20.0%)",
				"Not viable: 1 (20.0%)",
				"Mutation Score: 25.0%",
			},
		},
		{
			name: "long file paths",
			summary: &Summary{
				TotalFiles:     1,
				ProcessedFiles: 1,
				TotalMutants:   1,
				Results: []mutation.Result{
					{
						Mutant: mutation.Mutant{
							FilePath:    "/very/long/path/to/some/deeply/nested/source/file/that/might/break/formatting.go",
							Line:        12345,
							Column:      999,
							Description: "Very long description that goes on and on and on",
							Original:    "veryLongOriginalValueThatMightBreakFormatting",
							Mutated:     "veryLongMutatedValueThatMightBreakFormatting",
						},
						Status: mutation.StatusSurvived,
					},
				},
				Duration: time.Millisecond * 123,
				Statistics: Statistics{
					Survived: 1,
					Score:    0,
				},
			},
			contains: []string{
				"Survived Mutants:",
				"/very/long/path/to/some/deeply/nested/source/file/that/might/break/formatting.go:12345:999",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := generator.formatTextReport(tt.summary)

			for _, expected := range tt.contains {
				if !strings.Contains(report, expected) {
					t.Errorf("Expected report to contain %q", expected)
				}
			}
		})
	}
}

func TestGenerate_Timestamp(t *testing.T) {
	generator, err := New("json")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	beforeTime := time.Now()

	summary := &Summary{
		TotalFiles:   1,
		TotalMutants: 1,
		Results:      []mutation.Result{},
		Duration:     time.Second,
	}

	err = generator.Generate(summary)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	afterTime := time.Now()

	// Verify timestamp was set
	if summary.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// Verify timestamp is within expected range
	if summary.Timestamp.Before(beforeTime) || summary.Timestamp.After(afterTime) {
		t.Error("Timestamp should be between test start and end times")
	}

	// Verify version is set
	if summary.Version == "" {
		t.Error("Version should be set")
	}

	// Cleanup
	defer os.Remove("mutation-report.json")
}

func TestNew_AlwaysSucceeds(t *testing.T) {
	// Test that New() always returns a valid generator
	for i := 0; i < 10; i++ {
		generator, err := New("json")
		if err != nil {
			t.Errorf("New() should never return error, got: %v", err)
		}

		if generator == nil {
			t.Error("New() should always return a non-nil generator")
		}
	}
}
