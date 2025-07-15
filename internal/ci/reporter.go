package ci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sivchari/gomu/internal/report"
)

// CIReporter generates CI-specific reports.
type CIReporter struct {
	outputDir    string
	outputFormat string
}

// NewCIReporter creates a new CI reporter.
func NewCIReporter(outputDir, outputFormat string) *CIReporter {
	return &CIReporter{
		outputDir:    outputDir,
		outputFormat: outputFormat,
	}
}

// CIReport represents a CI-specific mutation testing report.
type CIReport struct {
	Summary            *report.Summary     `json:"summary"`
	QualityGate        *QualityGateResult  `json:"qualityGate"`
	ChangedFiles       []ChangedFileReport `json:"changedFiles"`
	Recommendations    []string            `json:"recommendations"`
	MutationScore      float64             `json:"mutationScore"`
	TotalMutants       int                 `json:"totalMutants"`
	Killed             int                 `json:"killed"`
	Survived           int                 `json:"survived"`
	KilledPercentage   float64             `json:"killedPercentage"`
	SurvivedPercentage float64             `json:"survivedPercentage"`
	Duration           string              `json:"duration"`
	ProcessedFiles     int                 `json:"processedFiles"`
	TotalFiles         int                 `json:"totalFiles"`
	Timestamp          time.Time           `json:"timestamp"`
	CI                 CIMetadata          `json:"ci"`
}

// ChangedFileReport represents a report for a changed file.
type ChangedFileReport struct {
	Path   string  `json:"path"`
	Score  float64 `json:"score"`
	Status string  `json:"status"`
}

// CIMetadata contains CI-specific metadata.
type CIMetadata struct {
	Provider   string `json:"provider"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	PRNumber   int    `json:"prNumber,omitempty"`
	Actor      string `json:"actor"`
	RunID      string `json:"runId"`
	EventName  string `json:"eventName"`
}

// Generate generates and outputs CI-specific reports.
func (r *CIReporter) Generate(summary *report.Summary, qualityGate *QualityGateEvaluator) error {
	// Evaluate quality gate
	var qualityResult *QualityGateResult
	if qualityGate != nil {
		qualityResult = qualityGate.Evaluate(summary)
	}

	switch r.outputFormat {
	case "json":
		return r.generateJSONReport(summary, qualityResult)
	case "html":
		return r.generateHTMLReport(summary, qualityResult)
	case "console":
		r.generateConsoleReport(summary, qualityResult)
		return nil
	default:
		return r.generateJSONReport(summary, qualityResult)
	}
}

// generateJSONReport generates a JSON report.
func (r *CIReporter) generateJSONReport(summary *report.Summary, qualityResult *QualityGateResult) error {
	data := map[string]interface{}{
		"totalMutants":      summary.TotalMutants,
		"killedMutants":     summary.KilledMutants,
		"mutationScore":     qualityResult.MutationScore,
		"qualityGatePassed": qualityResult.Pass,
		"qualityGateReason": qualityResult.Reason,
		"files":             summary.Files,
		"timestamp":         time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	filename := filepath.Join(r.outputDir, "mutation-report.json")
	return os.WriteFile(filename, jsonData, 0644)
}

// generateHTMLReport generates an HTML report.
func (r *CIReporter) generateHTMLReport(summary *report.Summary, qualityResult *QualityGateResult) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Mutation Testing Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .score { font-size: 24px; font-weight: bold; color: %s; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Mutation Testing Report</h1>
        <div class="score">Overall Score: %.1f%%</div>
        <p>Quality Gate: %s</p>
    </div>
    <h2>Summary</h2>
    <p>Total Mutants: %d</p>
    <p>Killed: %d</p>
    <p>Files Analyzed: %d</p>
</body>
</html>`,
		r.getScoreColor(qualityResult.MutationScore),
		qualityResult.MutationScore,
		map[bool]string{true: "PASSED", false: "FAILED"}[qualityResult.Pass],
		summary.TotalMutants,
		summary.KilledMutants,
		len(summary.Files),
	)

	filename := filepath.Join(r.outputDir, "mutation-report.html")
	return os.WriteFile(filename, []byte(html), 0644)
}

// generateConsoleReport prints a console report.
func (r *CIReporter) generateConsoleReport(summary *report.Summary, qualityResult *QualityGateResult) {
	fmt.Printf("Mutation Score: %.1f%%\n", qualityResult.MutationScore)
	fmt.Printf("Quality Gate: %s\n", map[bool]string{true: "PASSED", false: "FAILED"}[qualityResult.Pass])
	fmt.Printf("Total Mutants: %d\n", summary.TotalMutants)
	fmt.Printf("Killed: %d\n", summary.KilledMutants)
}

// getScoreColor returns color based on score.
func (r *CIReporter) getScoreColor(score float64) string {
	if score >= 80 {
		return "#28a745"
	}
	if score >= 60 {
		return "#ffc107"
	}
	return "#dc3545"
}
