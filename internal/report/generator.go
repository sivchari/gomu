// Package report provides report generation functionality.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
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
	TotalFiles     int                    `json:"totalFiles"`
	ProcessedFiles int                    `json:"processedFiles"`
	TotalMutants   int                    `json:"totalMutants"`
	KilledMutants  int                    `json:"killedMutants"`
	Results        []mutation.Result      `json:"results"`
	Files          map[string]*FileReport `json:"files"`
	Duration       time.Duration          `json:"duration"`
	Config         *config.Config         `json:"config,omitempty"`
	Statistics     Statistics             `json:"statistics"`
	Timestamp      time.Time              `json:"timestamp"`
}

// FileReport represents a report for a single file.
type FileReport struct {
	FilePath      string  `json:"filePath"`
	TotalMutants  int     `json:"totalMutants"`
	KilledMutants int     `json:"killedMutants"`
	MutationScore float64 `json:"mutationScore"`
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

func (g *Generator) generateHTML(summary *Summary) error {
	funcMap := template.FuncMap{
		"percentage": percentage,
	}

	tmpl, err := template.New("html_report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var output strings.Builder
	if err := tmpl.Execute(&output, summary); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	if g.config.OutputFile != "" {
		if err := os.WriteFile(g.config.OutputFile, []byte(output.String()), 0600); err != nil {
			return fmt.Errorf("failed to write HTML output file: %w", err)
		}

		return nil
	}

	fmt.Print(output.String())

	return nil
}

// htmlTemplate is the template for HTML reports.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mutation Testing Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 {
            color: #2c3e50;
            margin-bottom: 30px;
            text-align: center;
            border-bottom: 2px solid #3498db;
            padding-bottom: 15px;
        }
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 6px;
            border-left: 4px solid #3498db;
        }
        .summary-card h3 {
            margin: 0 0 10px 0;
            color: #2c3e50;
            font-size: 16px;
        }
        .summary-card .value {
            font-size: 24px;
            font-weight: bold;
            color: #2c3e50;
        }
        .statistics {
            margin-bottom: 30px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }
        .stat-item {
            background: white;
            padding: 15px;
            border-radius: 6px;
            border: 1px solid #e0e0e0;
            text-align: center;
        }
        .stat-item.killed { border-left: 4px solid #27ae60; }
        .stat-item.survived { border-left: 4px solid #e74c3c; }
        .stat-item.timed-out { border-left: 4px solid #f39c12; }
        .stat-item.error { border-left: 4px solid #e67e22; }
        .stat-item.not-covered { border-left: 4px solid #95a5a6; }
        .stat-number {
            font-size: 28px;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .stat-label {
            font-size: 14px;
            color: #7f8c8d;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .mutation-score {
            text-align: center;
            margin: 30px 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border-radius: 8px;
        }
        .mutation-score h2 {
            margin: 0 0 10px 0;
            font-size: 18px;
        }
        .score-value {
            font-size: 48px;
            font-weight: bold;
            margin: 0;
        }
        .survived-mutants {
            margin-top: 30px;
        }
        .survived-mutants h2 {
            color: #e74c3c;
            border-bottom: 2px solid #e74c3c;
            padding-bottom: 10px;
        }
        .mutant-list {
            background: #fff5f5;
            border-radius: 6px;
            padding: 20px;
            margin-top: 15px;
        }
        .mutant-item {
            background: white;
            margin-bottom: 15px;
            padding: 15px;
            border-radius: 4px;
            border-left: 4px solid #e74c3c;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .mutant-item:last-child {
            margin-bottom: 0;
        }
        .mutant-location {
            font-family: 'Monaco', 'Consolas', monospace;
            background: #f8f9fa;
            padding: 4px 8px;
            border-radius: 3px;
            font-size: 12px;
            color: #e74c3c;
        }
        .mutant-description {
            margin: 8px 0;
            color: #2c3e50;
        }
        .mutant-change {
            font-family: 'Monaco', 'Consolas', monospace;
            background: #f8f9fa;
            padding: 8px;
            border-radius: 3px;
            font-size: 13px;
            margin-top: 8px;
        }
        .original { color: #e74c3c; }
        .mutated { color: #27ae60; }
        .timestamp {
            text-align: center;
            color: #7f8c8d;
            font-size: 12px;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #e0e0e0;
        }
        @media (max-width: 768px) {
            .summary-grid, .stats-grid {
                grid-template-columns: 1fr;
            }
            .score-value {
                font-size: 36px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Mutation Testing Report</h1>
        
        <div class="summary-grid">
            <div class="summary-card">
                <h3>Files Processed</h3>
                <div class="value">{{.ProcessedFiles}}/{{.TotalFiles}}</div>
            </div>
            <div class="summary-card">
                <h3>Total Mutants</h3>
                <div class="value">{{.TotalMutants}}</div>
            </div>
            <div class="summary-card">
                <h3>Duration</h3>
                <div class="value">{{.Duration}}</div>
            </div>
        </div>
        
        <div class="mutation-score">
            <h2>Mutation Score</h2>
            <div class="score-value">{{printf "%.1f" .Statistics.Score}}%</div>
        </div>
        
        <div class="statistics">
            <div class="stats-grid">
                <div class="stat-item killed">
                    <div class="stat-number">{{.Statistics.Killed}}</div>
                    <div class="stat-label">Killed ({{printf "%.1f" (percentage .Statistics.Killed .TotalMutants)}}%)</div>
                </div>
                <div class="stat-item survived">
                    <div class="stat-number">{{.Statistics.Survived}}</div>
                    <div class="stat-label">Survived ({{printf "%.1f" (percentage .Statistics.Survived .TotalMutants)}}%)</div>
                </div>
                <div class="stat-item timed-out">
                    <div class="stat-number">{{.Statistics.TimedOut}}</div>
                    <div class="stat-label">Timed Out ({{printf "%.1f" (percentage .Statistics.TimedOut .TotalMutants)}}%)</div>
                </div>
                <div class="stat-item error">
                    <div class="stat-number">{{.Statistics.Errors}}</div>
                    <div class="stat-label">Errors ({{printf "%.1f" (percentage .Statistics.Errors .TotalMutants)}}%)</div>
                </div>
                <div class="stat-item not-covered">
                    <div class="stat-number">{{.Statistics.NotCovered}}</div>
                    <div class="stat-label">Not Covered ({{printf "%.1f" (percentage .Statistics.NotCovered .TotalMutants)}}%)</div>
                </div>
            </div>
        </div>
        {{if gt .Statistics.Survived 0}}
        <div class="survived-mutants">
            <h2>Survived Mutants</h2>
            <div class="mutant-list">
                {{range .Results}}
                {{if eq .Status "SURVIVED"}}
                <div class="mutant-item">
                    <div class="mutant-location">{{.Mutant.FilePath}}:{{.Mutant.Line}}:{{.Mutant.Column}}</div>
                    <div class="mutant-description">{{.Mutant.Description}}</div>
                    <div class="mutant-change">
                        <span class="original">{{.Mutant.Original}}</span> → <span class="mutated">{{.Mutant.Mutated}}</span>
                    </div>
                </div>
                {{end}}
                {{end}}
            </div>
        </div>
        {{end}}
        <div class="timestamp">
            Generated on {{.Timestamp.Format "2006-01-02 15:04:05 MST"}}
        </div>
    </div>
</body>
</html>`

func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}

	return float64(part) / float64(total) * 100
}
