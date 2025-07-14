// Package gomu provides the main API for mutation testing.
package gomu

import (
	"fmt"
	"log"
	"path/filepath"
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
	config               *config.Config
	analyzer             *analysis.Analyzer
	mutator              *mutation.Engine
	executor             *execution.Engine
	history              *history.Store
	reporter             *report.Generator
	incrementalAnalyzer  *analysis.IncrementalAnalyzer
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
		config:               cfg,
		analyzer:             analyzer,
		mutator:              mutator,
		executor:             executor,
		history:              historyStore,
		reporter:             reporter,
		incrementalAnalyzer:  nil, // Will be initialized in Run method
	}, nil
}

// Run executes mutation testing on the specified path.
func (e *Engine) Run(path string) error {
	start := time.Now()

	if e.config.Verbose {
		log.Printf("Starting mutation testing on path: %s", path)
	}

	// Get absolute path for working directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Initialize incremental analyzer
	historyWrapper := &historyStoreWrapper{store: e.history}
	e.incrementalAnalyzer, err = analysis.NewIncrementalAnalyzer(e.config, absPath, historyWrapper)
	if err != nil {
		return fmt.Errorf("failed to create incremental analyzer: %w", err)
	}

	// 1. Perform incremental analysis
	analysisResults, err := e.incrementalAnalyzer.AnalyzeFiles()
	if err != nil {
		return fmt.Errorf("failed to analyze files: %w", err)
	}

	if e.config.Verbose {
		e.incrementalAnalyzer.PrintAnalysisReport(analysisResults)
	}

	// 2. Get files that need processing
	files, err := e.incrementalAnalyzer.GetFilesNeedingUpdate()
	if err != nil {
		return fmt.Errorf("failed to get files needing update: %w", err)
	}

	if e.config.Verbose {
		log.Printf("Processing %d files", len(files))
	}

	// If no files need processing, return early
	if len(files) == 0 {
		if e.config.Verbose {
			log.Println("No files need processing - all files are up to date")
		}
		return nil
	}

	var allResults []mutation.Result
	totalMutants := 0
	processedFiles := 0

	hasher := analysis.NewFileHasher()

	// 3. Process each file
	for _, file := range files {
		if e.config.Verbose {
			log.Printf("Processing file: %s", file)
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

		// Calculate file and test hashes
		fileHash, err := hasher.HashFile(file)
		if err != nil {
			log.Printf("Warning: failed to hash file %s: %v", file, err)
			fileHash = ""
		}

		testHash := e.calculateTestHash(file, hasher)

		// Update history with hashes
		e.history.UpdateFileWithHashes(file, mutants, results, fileHash, testHash)
		processedFiles++
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
		TotalFiles:     len(analysisResults),
		TotalMutants:   totalMutants,
		Results:        allResults,
		Duration:       time.Since(start),
		Config:         e.config,
		ProcessedFiles: processedFiles,
	}

	if err := e.reporter.Generate(summary); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if e.config.Verbose {
		log.Printf("Mutation testing completed in %v", time.Since(start))
	}

	return nil
}

// calculateTestHash calculates the combined hash of test files related to the given file.
func (e *Engine) calculateTestHash(filePath string, hasher *analysis.FileHasher) string {
	// Find related test files
	testFiles := e.findRelatedTestFiles(filePath)
	
	if len(testFiles) == 0 {
		return ""
	}
	
	// Calculate combined hash
	var combinedContent []byte
	for _, testFile := range testFiles {
		content, err := hasher.HashFile(testFile)
		if err != nil {
			continue
		}
		combinedContent = append(combinedContent, []byte(content)...)
	}
	
	if len(combinedContent) == 0 {
		return ""
	}
	
	return hasher.HashContent(combinedContent)
}

// findRelatedTestFiles finds test files related to the given file.
func (e *Engine) findRelatedTestFiles(filePath string) []string {
	var testFiles []string
	
	// Get directory and base name
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	
	// Remove .go extension
	nameWithoutExt := base[:len(base)-3]
	
	// Common test file patterns
	patterns := []string{
		nameWithoutExt + "_test.go",
		"test_" + nameWithoutExt + ".go",
	}
	
	for _, pattern := range patterns {
		testFile := filepath.Join(dir, pattern)
		if _, err := filepath.Abs(testFile); err == nil {
			testFiles = append(testFiles, testFile)
		}
	}
	
	return testFiles
}

// historyStoreWrapper wraps history.Store to implement analysis.HistoryStore interface.
type historyStoreWrapper struct {
	store *history.Store
}

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

func (w *historyStoreWrapper) HasChanged(filePath, currentHash string) bool {
	return w.store.HasChanged(filePath, currentHash)
}
