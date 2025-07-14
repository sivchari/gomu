// Package report provides report generation functionality.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sivchari/gomu/internal/config"
	"github.com/sivchari/gomu/internal/mutation"
)

// Generator handles report generation.
type Generator struct {
	config *config.Config
}

// Summary contains the complete results of a mutation testing run.
type Summary struct {
	TotalFiles     int               `json:"totalFiles"`
	ProcessedFiles int               `json:"processedFiles"`
	TotalMutants   int               `json:"totalMutants"`
	Results        []mutation.Result `json:"results"`
	Duration       time.Duration     `json:"duration"`
	Config         *config.Config    `json:"config,omitempty"`
	Statistics     Statistics        `json:"statistics"`
	Timestamp      time.Time         `json:"timestamp"`
}

// Statistics contains aggregated mutation testing statistics.
type Statistics struct {
	Killed     int     `json:"killed"`
	Survived   int     `json:"survived"`
	TimedOut   int     `json:"timedOut"`
	Errors     int     `json:"errors"`
	NotCovered int     `json:"notCovered"`
	Score      float64 `json:"mutationScore"`
	Coverage   float64 `json:"lineCoverage,omitempty"`
}

// New creates a new report generator.
func New(cfg *config.Config) (*Generator, error) {
	return &Generator{
		config: cfg,
	}, nil
}

// Generate creates and outputs the mutation testing report.
func (g *Generator) Generate(summary *Summary) error {
	// Calculate statistics
	summary.Statistics = g.calculateStatistics(summary.Results)
	summary.Timestamp = time.Now()

	switch g.config.OutputFormat {
	case "json":
		return g.generateJSON(summary)
	case "text":
		return g.generateText(summary)
	case "html":
		return g.generateHTML(summary)
	default:
		return g.generateJSON(summary)
	}
}

func (g *Generator) calculateStatistics(results []mutation.Result) Statistics {
	stats := Statistics{}

	for _, result := range results {
		switch result.Status {
		case mutation.StatusKilled:
			stats.Killed++
		case mutation.StatusSurvived:
			stats.Survived++
		case mutation.StatusTimedOut:
			stats.TimedOut++
		case mutation.StatusError:
			stats.Errors++
		case mutation.StatusNotCovered:
			stats.NotCovered++
		}
	}

	total := len(results)
	if total > 0 {
		stats.Score = float64(stats.Killed) / float64(total) * 100
	}

	return stats
}

func (g *Generator) generateJSON(summary *Summary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	if g.config.OutputFile != "" {
		if err := os.WriteFile(g.config.OutputFile, data, 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		return nil
	}

	fmt.Println(string(data))

	return nil
}

func (g *Generator) generateText(summary *Summary) error {
	output := g.formatTextReport(summary)

	if g.config.OutputFile != "" {
		if err := os.WriteFile(g.config.OutputFile, []byte(output), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		return nil
	}

	fmt.Print(output)

	return nil
}

func (g *Generator) formatTextReport(summary *Summary) string {
	stats := summary.Statistics

	report := fmt.Sprintf(`
Mutation Testing Report
=======================

Summary:
  Files processed: %d/%d
  Total mutants:   %d
  Duration:        %v

Results:
  Killed:     %d (%.1f%%)
  Survived:   %d (%.1f%%)
  Timed out:  %d (%.1f%%)
  Errors:     %d (%.1f%%)
  Not covered:%d (%.1f%%)

Mutation Score: %.1f%%

`,
		summary.ProcessedFiles, summary.TotalFiles,
		summary.TotalMutants,
		summary.Duration,
		stats.Killed, percentage(stats.Killed, summary.TotalMutants),
		stats.Survived, percentage(stats.Survived, summary.TotalMutants),
		stats.TimedOut, percentage(stats.TimedOut, summary.TotalMutants),
		stats.Errors, percentage(stats.Errors, summary.TotalMutants),
		stats.NotCovered, percentage(stats.NotCovered, summary.TotalMutants),
		stats.Score,
	)

	// Add details for survived mutants
	if stats.Survived > 0 {
		report += "\nSurvived Mutants:\n"
		report += "=================\n"

		for _, result := range summary.Results {
			if result.Status == mutation.StatusSurvived {
				report += fmt.Sprintf("  %s:%d:%d - %s (%s -> %s)\n",
					result.Mutant.FilePath,
					result.Mutant.Line,
					result.Mutant.Column,
					result.Mutant.Description,
					result.Mutant.Original,
					result.Mutant.Mutated,
				)
			}
		}
	}

	return report
}

func (g *Generator) generateHTML(_ *Summary) error {
	// TODO: Implement HTML report generation
	return fmt.Errorf("HTML report format not yet implemented")
}

func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}

	return float64(part) / float64(total) * 100
}
