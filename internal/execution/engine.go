// Package execution provides mutation testing execution functionality.
package execution

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/sivchari/gomu/internal/config"
	"github.com/sivchari/gomu/internal/mutation"
)

// Engine handles test execution.
type Engine struct {
	config    *config.Config
	mutator   *SourceMutator
	fileLocks map[string]*sync.Mutex
	fileMapMu sync.Mutex
}

// New creates a new execution engine.
func New(cfg *config.Config) (*Engine, error) {
	mutator, err := NewSourceMutator()
	if err != nil {
		return nil, fmt.Errorf("failed to create source mutator: %w", err)
	}

	return &Engine{
		config:    cfg,
		mutator:   mutator,
		fileLocks: make(map[string]*sync.Mutex),
	}, nil
}

// Close cleans up the execution engine.
func (e *Engine) Close() error {
	if e.mutator != nil {
		return e.mutator.Cleanup()
	}

	return nil
}

// RunMutations executes tests for all mutants in parallel.
func (e *Engine) RunMutations(mutants []mutation.Mutant) ([]mutation.Result, error) {
	if len(mutants) == 0 {
		return nil, nil
	}

	results := make([]mutation.Result, len(mutants))
	resultsChan := make(chan indexedResult, len(mutants))

	// Create worker pool
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, e.config.Workers)

	// Start workers
	for i, mutant := range mutants {
		wg.Add(1)

		go func(index int, m mutation.Mutant) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			result := e.runSingleMutation(m)
			resultsChan <- indexedResult{index: index, result: result}
		}(i, mutant)
	}

	// Close results channel when all workers complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for indexedRes := range resultsChan {
		results[indexedRes.index] = indexedRes.result
	}

	return results, nil
}

type indexedResult struct {
	index  int
	result mutation.Result
}

// getFileLock returns a mutex for the given file path.
func (e *Engine) getFileLock(filePath string) *sync.Mutex {
	e.fileMapMu.Lock()
	defer e.fileMapMu.Unlock()

	if lock, exists := e.fileLocks[filePath]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	e.fileLocks[filePath] = lock

	return lock
}

// runSingleMutation executes tests for a single mutant.
func (e *Engine) runSingleMutation(mutant mutation.Mutant) mutation.Result {
	result := mutation.Result{
		Mutant: mutant,
		Status: mutation.StatusError,
	}

	// Acquire file lock to prevent concurrent mutations on the same file
	fileLock := e.getFileLock(mutant.FilePath)

	fileLock.Lock()
	defer fileLock.Unlock()

	// 1. Apply the mutation to the source code
	if err := e.mutator.ApplyMutation(mutant); err != nil {
		result.Error = fmt.Sprintf("Failed to apply mutation: %v", err)

		return result
	}

	// 2. Run the tests
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.config.Test.Timeout)*time.Second)
	defer cancel()

	// Get the directory containing the mutated file for running tests
	testDir := filepath.Dir(mutant.FilePath)
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// 3. Always restore the original code
	restoreErr := e.mutator.RestoreOriginal(mutant.FilePath)
	if restoreErr != nil {
		result.Error = fmt.Sprintf("Failed to restore original file: %v", restoreErr)

		return result
	}

	// 4. Analyze the test results
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = mutation.StatusTimedOut
		result.Error = "Test execution timed out"

		return result
	}

	result.Output = string(output)

	if err != nil {
		// Tests failed - check if it's because the mutant was killed
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 0 {
			// Test failure likely means the mutant was detected (killed)
			result.Status = mutation.StatusKilled
		} else {
			result.Status = mutation.StatusError
			result.Error = err.Error()
		}
	} else {
		// Tests passed - mutant survived
		result.Status = mutation.StatusSurvived
	}

	return result
}
