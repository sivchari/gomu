package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sivchari/gomu/internal/config"
)

// HistoryStore defines the interface for history storage.
type HistoryStore interface {
	GetEntry(filePath string) (HistoryEntry, bool)
	HasChanged(filePath, currentHash string) bool
}

// HistoryEntry represents a history entry.
type HistoryEntry struct {
	FileHash      string
	TestHash      string
	MutationScore float64
}

// IncrementalAnalyzer provides incremental analysis functionality.
type IncrementalAnalyzer struct {
	config  *config.Config
	hasher  *FileHasher
	git     *GitIntegration
	history HistoryStore
	workDir string
}

// NewIncrementalAnalyzer creates a new incremental analyzer.
func NewIncrementalAnalyzer(cfg *config.Config, workDir string, historyStore HistoryStore) (*IncrementalAnalyzer, error) {
	return &IncrementalAnalyzer{
		config:  cfg,
		hasher:  NewFileHasher(),
		git:     NewGitIntegration(workDir),
		history: historyStore,
		workDir: workDir,
	}, nil
}

// FileAnalysisResult represents the result of file analysis.
type FileAnalysisResult struct {
	FilePath     string
	NeedsUpdate  bool
	Reason       string
	CurrentHash  string
	PreviousHash string
}

// AnalyzeFiles determines which files need mutation testing.
func (a *IncrementalAnalyzer) AnalyzeFiles() ([]FileAnalysisResult, error) {
	// Get files to analyze
	files, err := a.getFilesToAnalyze()
	if err != nil {
		return nil, fmt.Errorf("failed to get files to analyze: %w", err)
	}

	// Analyze each file
	results := make([]FileAnalysisResult, 0, len(files))

	for _, file := range files {
		result, err := a.analyzeFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze file %s: %w", file, err)
		}

		results = append(results, result)
	}

	return results, nil
}

// getFilesToAnalyze returns the list of files that should be analyzed.
func (a *IncrementalAnalyzer) getFilesToAnalyze() ([]string, error) {
	if a.config.Incremental.UseGitDiff && a.git.IsGitRepository() {
		// Use Git diff to get changed files
		return a.git.GetChangedFiles(a.config.Incremental.BaseBranch)
	}

	// Fallback to all Go files
	return a.git.GetAllGoFiles()
}

// analyzeFile analyzes a single file to determine if it needs mutation testing.
func (a *IncrementalAnalyzer) analyzeFile(filePath string) (FileAnalysisResult, error) {
	result := FileAnalysisResult{
		FilePath: filePath,
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		result.NeedsUpdate = false
		result.Reason = "File does not exist"

		return result, nil
	}

	// Calculate current hash
	currentHash, err := a.hasher.HashFile(filePath)
	if err != nil {
		return result, fmt.Errorf("failed to hash file: %w", err)
	}

	result.CurrentHash = currentHash

	// Get previous entry from history
	entry, exists := a.history.GetEntry(filePath)
	if !exists {
		result.NeedsUpdate = true
		result.Reason = "No previous history"

		return result, nil
	}

	result.PreviousHash = entry.FileHash

	// Check if file has changed
	if a.history.HasChanged(filePath, currentHash) {
		result.NeedsUpdate = true
		result.Reason = "File content changed"

		return result, nil
	}

	// Check if related test files have changed
	if a.hasTestFilesChanged(filePath) {
		result.NeedsUpdate = true
		result.Reason = "Related test files changed"

		return result, nil
	}

	result.NeedsUpdate = false
	result.Reason = "No changes detected"

	return result, nil
}

// hasTestFilesChanged checks if test files related to the given file have changed.
func (a *IncrementalAnalyzer) hasTestFilesChanged(filePath string) bool {
	// Find related test files
	testFiles := a.findRelatedTestFiles(filePath)

	for _, testFile := range testFiles {
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			continue
		}

		currentHash, err := a.hasher.HashFile(testFile)
		if err != nil {
			continue
		}

		// Check if test file hash has changed
		entry, exists := a.history.GetEntry(filePath)
		if !exists || entry.TestHash != currentHash {
			return true
		}
	}

	return false
}

// findRelatedTestFiles finds test files related to the given file.
func (a *IncrementalAnalyzer) findRelatedTestFiles(filePath string) []string {
	var testFiles []string

	// Get directory and base name
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	// Remove .go extension
	nameWithoutExt := strings.TrimSuffix(base, ".go")

	// Common test file patterns
	patterns := []string{
		nameWithoutExt + "_test.go",
		"test_" + nameWithoutExt + ".go",
	}

	for _, pattern := range patterns {
		testFile := filepath.Join(dir, pattern)
		if _, err := os.Stat(testFile); err == nil {
			testFiles = append(testFiles, testFile)
		}
	}

	return testFiles
}

// GetFilesNeedingUpdate returns only the files that need mutation testing.
func (a *IncrementalAnalyzer) GetFilesNeedingUpdate() ([]string, error) {
	results, err := a.AnalyzeFiles()
	if err != nil {
		return nil, err
	}

	var needsUpdate []string

	for _, result := range results {
		if result.NeedsUpdate {
			needsUpdate = append(needsUpdate, result.FilePath)
		}
	}

	return needsUpdate, nil
}

// PrintAnalysisReport prints a summary of the analysis results.
func (a *IncrementalAnalyzer) PrintAnalysisReport(results []FileAnalysisResult) {
	needsUpdate := 0
	skipped := 0

	fmt.Println("Incremental Analysis Report")
	fmt.Println("==========================")

	for _, result := range results {
		if result.NeedsUpdate {
			needsUpdate++

			fmt.Printf("✓ %s - %s\n", result.FilePath, result.Reason)
		} else {
			skipped++

			fmt.Printf("- %s - %s\n", result.FilePath, result.Reason)
		}
	}

	fmt.Printf("\nSummary: %d files need testing, %d files skipped\n", needsUpdate, skipped)

	if needsUpdate > 0 {
		fmt.Printf("Performance improvement: %.1f%% files skipped\n",
			float64(skipped)/float64(len(results))*100)
	}
}
