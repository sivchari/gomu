// Package gomu provides the main API for mutation testing.
package gomu

import (
	"fmt"
	"log"
	"time"

	"github.com/sivchari/gomu/internal/analysis"
	"github.com/sivchari/gomu/internal/config"
	"github.com/sivchari/gomu/internal/execution"
	"github.com/sivchari/gomu/internal/history"
	"github.com/sivchari/gomu/internal/mutation"
	"github.com/sivchari/gomu/internal/report"
)

// Engine is the main mutation testing engine.
type Engine struct {
	config   *config.Config
	analyzer *analysis.Analyzer
	mutator  *mutation.Engine
	executor *execution.Engine
	history  *history.Store
	reporter *report.Generator
}

// NewEngine creates a new mutation testing engine.
func NewEngine(cfg *config.Config) (*Engine, error) {
	analyzer, err := analysis.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	mutator, err := mutation.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create mutator: %w", err)
	}

	executor, err := execution.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	historyStore, err := history.New(cfg.HistoryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create history store: %w", err)
	}

	reporter, err := report.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create reporter: %w", err)
	}

	return &Engine{
		config:   cfg,
		analyzer: analyzer,
		mutator:  mutator,
		executor: executor,
		history:  historyStore,
		reporter: reporter,
	}, nil
}

// Run executes mutation testing on the specified path.
func (e *Engine) Run(path string) error {
	start := time.Now()

	if e.config.Verbose {
		log.Printf("Starting mutation testing on path: %s", path)
	}

	// 1. Analyze target files
	files, err := e.analyzer.FindTargetFiles(path)
	if err != nil {
		return fmt.Errorf("failed to find target files: %w", err)
	}

	if e.config.Verbose {
		log.Printf("Found %d target files", len(files))
	}

	// 2. Filter changed files if using incremental analysis
	if e.config.UseGitDiff {
		changedFiles, err := e.analyzer.FindChangedFiles(files)
		if err != nil {
			return fmt.Errorf("failed to get git diff: %w", err)
		}

		files = changedFiles
		if e.config.Verbose {
			log.Printf("Using incremental analysis: %d changed files", len(files))
		}
	}

	var allResults []mutation.Result

	totalMutants := 0

	// 3. Process each file
	for _, file := range files {
		if e.config.Verbose {
			log.Printf("Processing file: %s", file)
		}

		// Check history for unchanged files
		if !e.shouldProcessFile(file) {
			if e.config.Verbose {
				log.Printf("Skipping unchanged file: %s", file)
			}

			continue
		}

		// Generate mutations
		mutants, err := e.mutator.GenerateMutants(file)
		if err != nil {
			log.Printf("Warning: failed to generate mutants for %s: %v", file, err)

			continue
		}

		if len(mutants) == 0 {
			if e.config.Verbose {
				log.Printf("No mutants generated for file: %s", file)
			}

			continue
		}

		totalMutants += len(mutants)

		if e.config.Verbose {
			log.Printf("Generated %d mutants for %s", len(mutants), file)
		}

		// Execute mutations
		results, err := e.executor.RunMutations(mutants)
		if err != nil {
			log.Printf("Warning: failed to execute mutations for %s: %v", file, err)

			continue
		}

		allResults = append(allResults, results...)

		// Update history
		e.history.UpdateFile(file, mutants, results)
	}

	// 4. Cleanup execution engine
	if err := e.executor.Close(); err != nil {
		log.Printf("Warning: failed to cleanup execution engine: %v", err)
	}

	// 5. Save history
	if err := e.history.Save(); err != nil {
		log.Printf("Warning: failed to save history: %v", err)
	}

	// 6. Generate report
	summary := &report.Summary{
		TotalFiles:     len(files),
		TotalMutants:   totalMutants,
		Results:        allResults,
		Duration:       time.Since(start),
		Config:         e.config,
		ProcessedFiles: len(files),
	}

	if err := e.reporter.Generate(summary); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if e.config.Verbose {
		log.Printf("Mutation testing completed in %v", time.Since(start))
	}

	return nil
}

func (e *Engine) shouldProcessFile(_ string) bool {
	// For now, always process files
	// TODO: implement hash-based comparison with history
	return true
}
