package ci

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sivchari/gomu/internal/analysis"
	"github.com/sivchari/gomu/internal/config"
	"github.com/sivchari/gomu/internal/history"
	"github.com/sivchari/gomu/internal/mutation"
	"github.com/sivchari/gomu/internal/report"
)

// Engine integrates mutation testing with CI/CD environments.
type Engine struct {
	config       *config.Config
	ciConfig     *Config
	workDir      string
	historyStore *history.Store
	incremental  *analysis.IncrementalAnalyzer
	qualityGate  *QualityGateEvaluator
	reporter     *Reporter
	github       *GitHubIntegration
}

// NewEngine creates a new Engine.
func NewEngine(configPath, workDir string) (*Engine, error) {
	// Load unified YAML/JSON config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load CI config from environment
	ciConfig := LoadConfigFromEnv()

	// Initialize history store
	historyPath := filepath.Join(workDir, cfg.Incremental.HistoryFile)

	historyStore, err := history.NewStore(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history store: %w", err)
	}

	// Initialize incremental analyzer with simplified adapter
	incremental, err := analysis.NewIncrementalAnalyzer(cfg, workDir, &historyStoreWrapper{store: historyStore})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize incremental analyzer: %w", err)
	}

	// Initialize quality gate from YAML config
	qualityGate := NewQualityGateEvaluator(
		cfg.CI.QualityGate.Enabled,
		cfg.CI.QualityGate.MinMutationScore,
	)

	// Initialize reporter with config from YAML
	outputFormat := "json"
	if len(cfg.CI.Reports.Formats) > 0 {
		outputFormat = cfg.CI.Reports.Formats[0]
	}

	reporter := NewReporter(cfg.CI.Reports.OutputDir, outputFormat)

	// Initialize GitHub integration
	var github *GitHubIntegration

	if cfg.CI.GitHub.Enabled && ciConfig.Mode == "pr" && ciConfig.PRNumber > 0 {
		token := os.Getenv("GITHUB_TOKEN")
		repo := os.Getenv("GITHUB_REPOSITORY")

		if token != "" && repo != "" {
			github = NewGitHubIntegration(token, repo, ciConfig.PRNumber)
		}
	}

	return &Engine{
		config:       cfg,
		ciConfig:     ciConfig,
		workDir:      workDir,
		historyStore: historyStore,
		incremental:  incremental,
		qualityGate:  qualityGate,
		reporter:     reporter,
		github:       github,
	}, nil
}

// Run executes the CI mutation testing workflow.
func (e *Engine) Run(ctx context.Context) error {
	fmt.Println("Starting CI mutation testing...")

	// Step 1: Analyze files for incremental testing
	fmt.Println("Analyzing files for changes...")

	results, err := e.incremental.AnalyzeFiles()
	if err != nil {
		return fmt.Errorf("failed to analyze files: %w", err)
	}

	e.incremental.PrintAnalysisReport(results)

	// Get files that need testing
	filesToTest, err := e.incremental.GetFilesNeedingUpdate()
	if err != nil {
		return fmt.Errorf("failed to get files needing update: %w", err)
	}

	if len(filesToTest) == 0 {
		fmt.Println("No files need mutation testing. Skipping...")

		return nil
	}

	fmt.Printf("Running mutation testing on %d files...\n", len(filesToTest))

	// Step 2: Run mutation testing
	summary, err := e.runMutationTesting(filesToTest)
	if err != nil {
		return fmt.Errorf("failed to run mutation testing: %w", err)
	}

	// Step 3: Evaluate quality gate
	qualityResult := e.qualityGate.Evaluate(summary)
	fmt.Printf("Quality Gate: %s (Score: %.1f%%)\n",
		map[bool]string{true: "PASSED", false: "FAILED"}[qualityResult.Pass],
		qualityResult.MutationScore)

	if !qualityResult.Pass {
		fmt.Printf("Reason: %s\n", qualityResult.Reason)
	}

	// Step 4: Generate reports
	if err := e.reporter.Generate(summary, e.qualityGate); err != nil {
		return fmt.Errorf("failed to generate reports: %w", err)
	}

	// Step 5: Create PR comment if in PR mode
	if e.github != nil {
		if err := e.github.CreatePRComment(ctx, summary, qualityResult); err != nil {
			fmt.Printf("Warning: Failed to create PR comment: %v\n", err)
		} else {
			fmt.Println("Created PR comment with mutation testing results")
		}
	}

	// Step 6: Update history
	if err := e.updateHistory(summary); err != nil {
		fmt.Printf("Warning: Failed to update history: %v\n", err)
	}

	// Step 7: Exit with appropriate code based on quality gate
	if !qualityResult.Pass && e.shouldFailOnQualityGate() {
		return fmt.Errorf("quality gate failed: %s", qualityResult.Reason)
	}

	fmt.Println("CI mutation testing completed successfully")

	return nil
}

// runMutationTesting executes mutation testing on the specified files.
func (e *Engine) runMutationTesting(files []string) (*report.Summary, error) {
	// Create mutation engine
	engine, err := mutation.NewEngine(e.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create mutation engine: %w", err)
	}

	// Run mutation testing
	results, err := engine.RunOnFiles(files)
	if err != nil {
		return nil, fmt.Errorf("failed to run mutations: %w", err)
	}

	// Create summary
	summary := &report.Summary{
		Files: make(map[string]*report.FileReport),
	}

	totalMutants := 0
	killedMutants := 0

	for _, result := range results {
		fileReport := &report.FileReport{
			FilePath:      result.FilePath,
			TotalMutants:  len(result.Mutations),
			KilledMutants: 0,
		}

		for _, mut := range result.Mutations {
			if mut.Status == "killed" {
				fileReport.KilledMutants++
				killedMutants++
			}

			totalMutants++
		}

		if fileReport.TotalMutants > 0 {
			fileReport.MutationScore = float64(fileReport.KilledMutants) / float64(fileReport.TotalMutants) * 100
		}

		summary.Files[result.FilePath] = fileReport
	}

	summary.TotalMutants = totalMutants
	summary.KilledMutants = killedMutants

	return summary, nil
}

// updateHistory updates the mutation testing history with new results.
func (e *Engine) updateHistory(summary *report.Summary) error {
	hasher := analysis.NewFileHasher()

	for filePath, fileReport := range summary.Files {
		// Calculate current file hash
		currentHash, err := hasher.HashFile(filePath)
		if err != nil {
			continue // Skip files that can't be hashed
		}

		// Create history entry
		entry := history.Entry{
			FileHash:      currentHash,
			TestHash:      "", // Would need to calculate test file hash
			MutationScore: fileReport.MutationScore,
			Timestamp:     time.Now(),
			Mutants:       []mutation.Mutant{},
			Results:       []mutation.Result{},
		}

		// Update history
		if err := e.historyStore.UpdateEntry(filePath, entry); err != nil {
			return fmt.Errorf("failed to update history for %s: %w", filePath, err)
		}
	}

	if err := e.historyStore.Save(); err != nil {
		return fmt.Errorf("failed to save history store: %w", err)
	}

	return nil
}

// shouldFailOnQualityGate determines if the build should fail on quality gate failure.
func (e *Engine) shouldFailOnQualityGate() bool {
	// This would be configurable via .gomu.yaml
	// For now, return true to fail the build on quality gate failure
	return true
}

// historyStoreWrapper wraps history.Store to implement analysis.HistoryStore.
type historyStoreWrapper struct {
	store *history.Store
}

// GetEntry implements analysis.HistoryStore.GetEntry.
func (w *historyStoreWrapper) GetEntry(filePath string) (analysis.HistoryEntry, bool) {
	entry, exists := w.store.GetEntry(filePath)
	if !exists {
		return analysis.HistoryEntry{}, false
	}

	return analysis.HistoryEntry{
		FileHash:      entry.FileHash,
		TestHash:      entry.TestHash,
		MutationScore: entry.MutationScore,
	}, true
}

// HasChanged implements analysis.HistoryStore.HasChanged.
func (w *historyStoreWrapper) HasChanged(filePath, currentHash string) bool {
	return w.store.HasChanged(filePath, currentHash)
}
