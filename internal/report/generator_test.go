package report

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sivchari/gomu/internal/config"
	"github.com/sivchari/gomu/internal/mutation"
)

func abs(x float64) float64 {
	return math.Abs(x)
}

func TestNew(t *testing.T) {
	cfg := config.Default()

	generator, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	if generator == nil {
		t.Fatal("Expected generator to be non-nil")
	}

	if generator.config != cfg {
		t.Error("Generator config does not match provided config")
	}
}

func TestCalculateStatistics(t *testing.T) {
	cfg := config.Default()

	generator, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	results := []mutation.Result{
		{Mutant: mutation.Mutant{ID: "1"}, Status: mutation.StatusKilled},
		{Mutant: mutation.Mutant{ID: "2"}, Status: mutation.StatusKilled},
		{Mutant: mutation.Mutant{ID: "3"}, Status: mutation.StatusSurvived},
		{Mutant: mutation.Mutant{ID: "4"}, Status: mutation.StatusTimedOut},
		{Mutant: mutation.Mutant{ID: "5"}, Status: mutation.StatusError},
		{Mutant: mutation.Mutant{ID: "6"}, Status: mutation.StatusNotCovered},
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

	if stats.NotCovered != 1 {
		t.Errorf("Expected NotCovered 1, got %d", stats.NotCovered)
	}

	// Score should be 2/6 * 100 = 33.33...
	expectedScore := 2.0 / 6.0 * 100
	if abs(stats.Score-expectedScore) > 0.000001 {
		t.Errorf("Expected Score %f, got %f", expectedScore, stats.Score)
	}
}

func TestCalculateStatistics_EmptyResults(t *testing.T) {
	cfg := config.Default()

	generator, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	stats := generator.calculateStatistics([]mutation.Result{})

	expectedStats := Statistics{
		Killed:     0,
		Survived:   0,
		TimedOut:   0,
		Errors:     0,
		NotCovered: 0,
		Score:      0,
		Coverage:   0,
	}

	if stats != expectedStats {
		t.Errorf("Expected %+v, got %+v", expectedStats, stats)
	}
}

func TestGenerateJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	cfg := config.Default()
	cfg.OutputFormat = "json"
	cfg.OutputFile = outputFile

	generator, err := New(cfg)
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

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Verify JSON content
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

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
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	cfg := config.Default()
	cfg.OutputFormat = "text"
	cfg.OutputFile = outputFile

	generator, err := New(cfg)
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

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Verify text content
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

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
	cfg := config.Default()

	generator, err := New(cfg)
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
	cfg := config.Default()

	generator, err := New(cfg)
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

func TestGenerateHTML_NotImplemented(t *testing.T) {
	cfg := config.Default()
	cfg.OutputFormat = "html"

	generator, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	summary := &Summary{}

	err = generator.Generate(summary)
	if err == nil {
		t.Error("Expected error for HTML generation, got nil")
	}

	if !strings.Contains(err.Error(), "HTML report format not yet implemented") {
		t.Errorf("Expected HTML not implemented error, got: %v", err)
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
