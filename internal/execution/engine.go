// Package execution provides mutation testing execution functionality.
package execution

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sivchari/gomu/internal/mutation"
)

const maxCommandOutputBytes = 1 << 20
const defaultMaxWorkers = 4
const defaultChildMaxRSSMiB = 2048

var outputTruncatedMarker = []byte("\n[gomu: command output truncated]\n")
var errChildMemoryLimit = errors.New("child process exceeded memory limit")

// Engine handles test execution using overlay-based mutation.
type Engine struct {
	overlay *OverlayMutator
}

// New creates a new execution engine.
func New() (*Engine, error) {
	overlay, err := NewOverlayMutator()
	if err != nil {
		return nil, fmt.Errorf("failed to create overlay mutator: %w", err)
	}

	return &Engine{
		overlay: overlay,
	}, nil
}

// Close cleans up the execution engine.
func (e *Engine) Close() error {
	if e.overlay != nil {
		return e.overlay.Cleanup()
	}

	return nil
}

// RunMutations executes tests for all mutants in parallel.
func (e *Engine) RunMutations(mutants []mutation.Mutant) ([]mutation.Result, error) {
	return e.RunMutationsWithOptions(mutants, 4, 30)
}

// RunMutationsWithOptions executes tests for all mutants in parallel with custom options.
func (e *Engine) RunMutationsWithOptions(mutants []mutation.Mutant, workers, timeout int) ([]mutation.Result, error) {
	return e.RunMutationsWithContext(context.Background(), mutants, workers, timeout)
}

// RunMutationsWithContext executes tests for all mutants while honoring caller cancellation.
func (e *Engine) RunMutationsWithContext(
	ctx context.Context,
	mutants []mutation.Mutant,
	workers, timeout int,
) ([]mutation.Result, error) {
	if len(mutants) == 0 {
		return nil, nil
	}

	if workers < 1 {
		workers = 1
	}

	if workers > configuredMaxWorkers() {
		workers = configuredMaxWorkers()
	}

	seen := make(map[string]struct{}, len(mutants))
	for _, mutant := range mutants {
		if _, ok := seen[mutant.ID]; ok {
			return nil, fmt.Errorf("duplicate mutant id %q", mutant.ID)
		}

		seen[mutant.ID] = struct{}{}
	}

	results := make([]mutation.Result, len(mutants))
	resultsChan := make(chan indexedResult, workers)
	jobs := make(chan indexedMutant)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for job := range jobs {
				result := e.runSingleMutationWithContext(ctx, job.mutant, timeout)

				select {
				case resultsChan <- indexedResult{index: job.index, result: result}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)

		for index, mutant := range mutants {
			select {
			case jobs <- indexedMutant{index: index, mutant: mutant}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for indexedRes := range resultsChan {
		results[indexedRes.index] = indexedRes.result
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("mutation execution canceled: %w", err)
	}

	return results, nil
}

type indexedMutant struct {
	index  int
	mutant mutation.Mutant
}

type indexedResult struct {
	index  int
	result mutation.Result
}

// runSingleMutation executes tests for a single mutant using overlay.
func (e *Engine) runSingleMutation(mutant mutation.Mutant, timeout int) mutation.Result {
	return e.runSingleMutationWithContext(context.Background(), mutant, timeout)
}

func (e *Engine) runSingleMutationWithContext(
	ctx context.Context,
	mutant mutation.Mutant,
	timeout int,
) mutation.Result {
	result := mutation.Result{
		Mutant: mutant,
		Status: mutation.StatusError,
	}

	mutationCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 1. Prepare mutation (create mutated file + overlay.json)
	mutCtx, err := e.overlay.PrepareMutation(mutant)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to prepare mutation: %v", err)

		return result
	}

	defer func() {
		if cleanupErr := e.overlay.CleanupMutation(mutCtx); cleanupErr != nil {
			fmt.Printf("Warning: failed to cleanup mutation: %v\n", cleanupErr)
		}
	}()

	// 2. Check if the mutated code compiles using overlay
	if err := e.checkCompilationWithOverlay(mutationCtx, mutCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errChildMemoryLimit) {
			result.Status = mutation.StatusTimedOut
			result.Error = "Mutation execution timed out during compilation"

			return result
		}

		result.Status = mutation.StatusNotViable
		result.Error = fmt.Sprintf("Compilation failed: %v", err)
		result.Output = err.Error()

		return result
	}

	// 3. Run tests using overlay
	return e.runTestWithOverlay(mutationCtx, mutCtx, mutant)
}

// checkCompilationWithOverlay verifies that the mutated code compiles using overlay.
func (e *Engine) checkCompilationWithOverlay(ctx context.Context, mutCtx *MutationContext) error {
	// Get the directory containing the original file for compilation
	compileDir := filepath.Dir(mutCtx.OriginalPath)

	// Build the entire package with overlay to properly resolve dependencies
	output, err := runBoundedCommand(ctx, compileDir, "go", "build", "-overlay="+mutCtx.OverlayPath, ".")
	if err != nil {
		return fmt.Errorf("compilation error: %s: %w", output, err)
	}

	return nil
}

// runTestWithOverlay runs tests using the overlay configuration.
func (e *Engine) runTestWithOverlay(
	ctx context.Context,
	mutCtx *MutationContext,
	mutant mutation.Mutant,
) mutation.Result {
	result := mutation.Result{
		Mutant: mutant,
		Status: mutation.StatusError,
	}

	// Get the directory containing the original file for running tests
	testDir := filepath.Dir(mutCtx.OriginalPath)

	output, err := runBoundedCommand(ctx, testDir, "go", "test", "-overlay="+mutCtx.OverlayPath, ".")

	// Analyze test results
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, errChildMemoryLimit) {
		result.Status = mutation.StatusTimedOut
		result.Error = "Test execution timed out"

		return result
	}

	result.Output = output

	if err != nil {
		// Tests failed - check if it's because the mutant was killed
		if !errors.Is(err, context.Canceled) {
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

type limitedBuffer struct {
	buffer    bytes.Buffer
	remaining int
	truncated bool
}

func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{remaining: limit}
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	written := len(data)
	if len(data) > b.remaining {
		data = data[:b.remaining]
		b.truncated = true
	}

	_, _ = b.buffer.Write(data)
	b.remaining -= len(data)

	return written, nil
}

func (b *limitedBuffer) String() string {
	if b.truncated {
		return b.buffer.String() + string(outputTruncatedMarker)
	}

	return b.buffer.String()
}

func runBoundedCommand(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := newProcessGroupCommand(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PWD="+dir, "GOMEMLIMIT="+childMemoryLimit())
	output := newLimitedBuffer(maxCommandOutputBytes)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		return output.String(), err
	}

	done := make(chan error, 1)
	stopMonitor := make(chan struct{})
	memoryLimit := childMaxRSSBytes()
	memoryExceeded := make(chan struct{}, 1)

	go monitorProcessGroup(cmd.Process.Pid, memoryLimit, stopMonitor, memoryExceeded)

	defer close(stopMonitor)

	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return output.String(), err
	case <-ctx.Done():
		killProcessGroup(cmd)
		<-done

		return output.String(), fmt.Errorf("command canceled: %w", ctx.Err())
	case <-memoryExceeded:
		killProcessGroup(cmd)
		<-done

		return output.String(), errChildMemoryLimit
	}
}

func configuredMaxWorkers() int {
	value, err := strconv.Atoi(os.Getenv("GOMU_MAX_WORKERS"))
	if err == nil && value > 0 {
		return value
	}

	return defaultMaxWorkers
}

func childMaxRSSBytes() int64 {
	value, err := strconv.ParseInt(os.Getenv("GOMU_CHILD_MAX_RSS_MIB"), 10, 64)
	if err == nil && value > 0 {
		return value * 1024 * 1024
	}

	return defaultChildMaxRSSMiB * 1024 * 1024
}

func monitorProcessGroup(pid int, limit int64, stop <-chan struct{}, exceeded chan<- struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			rss, err := processGroupRSS(pid)
			if err == nil && rss > limit {
				exceeded <- struct{}{}

				return
			}
		}
	}
}

func processGroupRSS(pgid int) (int64, error) {
	output, err := exec.Command("ps", "-axo", "pgid=,rss=").Output()
	if err != nil {
		return 0, fmt.Errorf("read process RSS: %w", err)
	}

	var total int64

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) != 2 {
			continue
		}

		group, groupErr := strconv.Atoi(fields[0])

		rss, rssErr := strconv.ParseInt(fields[1], 10, 64)
		if groupErr == nil && rssErr == nil && group == pgid {
			total += rss * 1024
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan process RSS: %w", err)
	}

	return total, nil
}

func childMemoryLimit() string {
	if value := os.Getenv("GOMU_CHILD_GOMEMLIMIT"); value != "" {
		return value
	}

	return "2GiB"
}
