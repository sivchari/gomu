package gomu

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sivchari/gomu/internal/analysis"
	"github.com/sivchari/gomu/internal/ci"
	"github.com/sivchari/gomu/internal/history"
	"github.com/sivchari/gomu/internal/mutation"
	"github.com/sivchari/gomu/internal/report"
)

const testModuleContent = `module test

go 1.21
`

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name        string
		opts        *RunOptions
		expectError bool
		errContains string
	}{
		{
			name:        "creates engine successfully with nil options",
			opts:        nil,
			expectError: false,
		},
		{
			name: "creates engine with CI mode enabled",
			opts: &RunOptions{
				CIMode:     true,
				Threshold:  90.0,
				Output:     "xml",
				FailOnGate: true,
			},
			expectError: false,
		},
		{
			name: "creates engine with CI mode disabled",
			opts: &RunOptions{
				CIMode: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			engine, err := NewEngine(tt.opts)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if engine == nil {
				t.Fatal("engine should not be nil")
			}

			// Verify components are initialized
			for _, c := range []struct {
				name string
				got  any
			}{
				{"analyzer", engine.analyzer},
				{"mutator", engine.mutator},
				{"executor", engine.executor},
				{"history", engine.history},
				{"reporter", engine.reporter},
			} {
				if c.got == nil {
					t.Errorf("%s should not be nil", c.name)
				}
			}

			// Verify CI components when CI mode is enabled
			if tt.opts != nil && tt.opts.CIMode {
				if engine.qualityGate == nil {
					t.Error("quality gate should not be nil in CI mode")
				}

				if engine.ciReporter == nil {
					t.Error("CI reporter should not be nil in CI mode")
				}
			}
		})
	}
}

func TestInitializeCIComponents(t *testing.T) {
	tests := []struct {
		name       string
		opts       *RunOptions
		setupEnv   map[string]string
		verifyFunc func(t *testing.T, e *Engine)
	}{
		{
			name: "initializes with custom threshold",
			opts: &RunOptions{
				Threshold: 95.5,
				Output:    "json",
			},
			verifyFunc: func(t *testing.T, e *Engine) {
				if e.qualityGate == nil {
					t.Error("quality gate should not be nil")
				}

				if e.ciReporter == nil {
					t.Error("CI reporter should not be nil")
				}
			},
		},
		{
			name: "initializes with nil options",
			opts: nil,
			verifyFunc: func(t *testing.T, e *Engine) {
				if e.qualityGate == nil {
					t.Error("quality gate should not be nil")
				}
				// Should use default threshold of 80.0
			},
		},
		{
			name: "initializes with GitHub environment",
			opts: &RunOptions{
				Threshold: 85.0,
				Output:    "xml",
			},
			setupEnv: map[string]string{
				"CI_MODE":           "true",
				"GITHUB_PR_NUMBER":  "123",
				"GITHUB_TOKEN":      "test-token",
				"GITHUB_REPOSITORY": "owner/repo",
			},
			verifyFunc: func(t *testing.T, e *Engine) {
				if e.github == nil {
					t.Error("GitHub integration should not be nil")
				}
			},
		},
		{
			name: "skips GitHub when token missing",
			opts: &RunOptions{
				Threshold: 80.0,
			},
			setupEnv: map[string]string{
				"CI_MODE":           "true",
				"GITHUB_PR_NUMBER":  "123",
				"GITHUB_REPOSITORY": "owner/repo",
				// No GITHUB_TOKEN
			},
			verifyFunc: func(t *testing.T, e *Engine) {
				if e.github != nil {
					t.Error("GitHub integration should be nil without token")
				}
			},
		},
		{
			name: "skips GitHub when repo missing",
			opts: &RunOptions{
				Threshold: 80.0,
			},
			setupEnv: map[string]string{
				"CI_MODE":          "true",
				"GITHUB_PR_NUMBER": "123",
				"GITHUB_TOKEN":     "test-token",
				// No GITHUB_REPOSITORY
			},
			verifyFunc: func(t *testing.T, e *Engine) {
				if e.github != nil {
					t.Error("GitHub integration should be nil without repo")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the CI env vars this test cares about first: t.Setenv
			// restores whatever ambient value a key had (e.g. GITHUB_REPOSITORY
			// is always set on real GitHub Actions runners), unlike
			// os.Unsetenv, so a prior subtest's value could otherwise leak in.
			for _, k := range []string{"CI_MODE", "GITHUB_PR_NUMBER", "GITHUB_TOKEN", "GITHUB_REPOSITORY"} {
				t.Setenv(k, "")
			}

			for k, v := range tt.setupEnv {
				t.Setenv(k, v)
			}

			engine := &Engine{}
			engine.initializeCIComponents(tt.opts)

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, engine)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (string, func())
		opts         *RunOptions
		expectError  bool
		errContains  string
		setupContext func() context.Context
	}{
		{
			name: "successful run with no files to process",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				// Create empty Go module
				writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)

				return tempDir, func() {}
			},
			opts: &RunOptions{
				Workers:     2,
				Timeout:     10,
				Output:      "json",
				Incremental: true,
				BaseBranch:  "main",
				Threshold:   80.0,
				FailOnGate:  false,
				Verbose:     true,
			},
			expectError: false,
		},
		{
			name: "successful run with files to process",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()

				// Create go.mod
				writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)

				// Create a simple Go file
				content := `package main

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}
`
				writeFile(t, filepath.Join(tempDir, "math.go"), content)

				// Create test file
				testContent := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Error("Add failed")
	}
}
`
				writeFile(t, filepath.Join(tempDir, "math_test.go"), testContent)

				return tempDir, func() {}
			},
			opts: &RunOptions{
				Workers:     1,
				Timeout:     5,
				Output:      "json",
				Incremental: false,
				Verbose:     false,
			},
			expectError: false,
		},
		{
			name: "run with nil options uses defaults",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)

				return tempDir, func() {}
			},
			opts:        nil,
			expectError: false,
		},
		{
			name: "run with CI mode enabled",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)

				// Create a file with mutations
				content := `package main

func IsPositive(n int) bool {
	if n > 0 {
		return true
	}
	return false
}
`
				writeFile(t, filepath.Join(tempDir, "check.go"), content)

				return tempDir, func() {}
			},
			opts: &RunOptions{
				Workers:    1,
				Timeout:    5,
				Output:     "json",
				CIMode:     true,
				Threshold:  90.0,
				FailOnGate: false,
				Verbose:    true,
			},
			expectError: false,
		},
		{
			name: "invalid path returns error",
			setupFunc: func(_ *testing.T) (string, func()) {
				return "/nonexistent/invalid/path", func() {}
			},
			opts: &RunOptions{
				Workers: 1,
				Timeout: 5,
			},
			expectError: true,
			errContains: "failed to create incremental analyzer",
		},
		{
			name: "context cancellation",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)

				return tempDir, func() {}
			},
			opts: &RunOptions{
				Workers: 1,
			},
			setupContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				return ctx
			},
			expectError: false, // Should handle cancelled context gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			path, cleanup := tt.setupFunc(t)
			defer cleanup()

			engine, err := NewEngine(tt.opts)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			ctx := context.Background()
			if tt.setupContext != nil {
				ctx = tt.setupContext()
			}

			err = engine.Run(ctx, path, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Cleanup
			if engine.executor != nil {
				engine.executor.Close()
			}
		})
	}
}

func TestProcessCIWorkflow(t *testing.T) {
	t.Chdir(t.TempDir())

	tests := []struct {
		name        string
		setupEngine func() *Engine
		summary     *report.Summary
		opts        *RunOptions
		expectError bool
		errContains string
	}{
		{
			name: "successful CI workflow with passing quality gate",
			setupEngine: func() *Engine {
				e := &Engine{
					qualityGate: ci.NewQualityGateEvaluator(true, 80.0),
					ciReporter:  ci.NewReporter(".", "json"),
				}

				return e
			},
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 85,
				Statistics: report.Statistics{
					Score: 85.0,
				},
				Duration: time.Second,
				Results: []mutation.Result{
					{
						Mutant: mutation.Mutant{
							FilePath: "test.go",
						},
						Status: mutation.StatusKilled,
					},
				},
			},
			opts: &RunOptions{
				FailOnGate: true,
				Verbose:    true,
			},
			expectError: false,
		},
		{
			name: "CI workflow with failing quality gate",
			setupEngine: func() *Engine {
				e := &Engine{
					qualityGate: ci.NewQualityGateEvaluator(true, 90.0),
					ciReporter:  ci.NewReporter(".", "json"),
				}

				return e
			},
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 50,
				Statistics: report.Statistics{
					Score: 50.0,
				},
				Duration: time.Second,
				Results: []mutation.Result{
					{
						Mutant: mutation.Mutant{
							FilePath: "test.go",
						},
						Status: mutation.StatusSurvived,
					},
				},
			},
			opts: &RunOptions{
				FailOnGate: true,
				Verbose:    false,
			},
			expectError: true,
			errContains: "quality gate failed",
		},
		{
			name: "CI workflow with GitHub integration",
			setupEngine: func() *Engine {
				e := &Engine{
					qualityGate: ci.NewQualityGateEvaluator(true, 80.0),
					ciReporter:  ci.NewReporter(".", "json"),
					github:      &ci.GitHubIntegration{}, // Mock GitHub integration
				}

				return e
			},
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 85,
				Statistics: report.Statistics{
					Score: 85.0,
				},
				Duration: time.Second,
			},
			opts: &RunOptions{
				FailOnGate: false,
				Verbose:    true,
			},
			expectError: false,
		},
		{
			name: "CI workflow without quality gate evaluator",
			setupEngine: func() *Engine {
				e := &Engine{
					ciReporter: ci.NewReporter(".", "json"),
				}

				return e
			},
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 85,
				Statistics: report.Statistics{
					Score: 85.0,
				},
				Duration: time.Second,
				Results: []mutation.Result{
					{
						Mutant: mutation.Mutant{
							FilePath: "test.go",
						},
						Status: mutation.StatusKilled,
					},
				},
			},
			opts: &RunOptions{
				Verbose: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := tt.setupEngine()

			err := engine.processCIWorkflow(context.Background(), tt.summary, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConvertToCISummary(t *testing.T) {
	tests := []struct {
		name           string
		summary        *report.Summary
		expectedFiles  int
		expectedKilled int
		expectedTotal  int
		expectedScores map[string]float64
	}{
		{
			name: "converts summary with multiple files",
			summary: &report.Summary{
				Results: []mutation.Result{
					{
						Mutant: mutation.Mutant{
							FilePath: "file1.go",
						},
						Status: mutation.StatusKilled,
					},
					{
						Mutant: mutation.Mutant{
							FilePath: "file1.go",
						},
						Status: mutation.StatusSurvived,
					},
					{
						Mutant: mutation.Mutant{
							FilePath: "file2.go",
						},
						Status: mutation.StatusKilled,
					},
					{
						Mutant: mutation.Mutant{
							FilePath: "file2.go",
						},
						Status: mutation.StatusKilled,
					},
					{
						Mutant: mutation.Mutant{
							FilePath: "file2.go",
						},
						Status: mutation.StatusTimedOut,
					},
				},
				Duration: 5 * time.Second,
			},
			expectedFiles:  2,
			expectedKilled: 3,
			expectedTotal:  5,
			expectedScores: map[string]float64{
				"file1.go": 50.0,
				"file2.go": 66.66666666666667,
			},
		},
		{
			name: "handles empty results",
			summary: &report.Summary{
				Results:  []mutation.Result{},
				Duration: time.Second,
			},
			expectedFiles:  0,
			expectedKilled: 0,
			expectedTotal:  0,
			expectedScores: map[string]float64{},
		},
		{
			name: "all mutations killed",
			summary: &report.Summary{
				Results: []mutation.Result{
					{
						Mutant: mutation.Mutant{
							FilePath: "perfect.go",
						},
						Status: mutation.StatusKilled,
					},
					{
						Mutant: mutation.Mutant{
							FilePath: "perfect.go",
						},
						Status: mutation.StatusKilled,
					},
				},
			},
			expectedFiles:  1,
			expectedKilled: 2,
			expectedTotal:  2,
			expectedScores: map[string]float64{
				"perfect.go": 100.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{}

			ciSummary := engine.convertToCISummary(tt.summary)

			if ciSummary.TotalMutants != tt.expectedTotal {
				t.Errorf("expected total mutants %d, got %d", tt.expectedTotal, ciSummary.TotalMutants)
			}

			if ciSummary.KilledMutants != tt.expectedKilled {
				t.Errorf("expected killed mutants %d, got %d", tt.expectedKilled, ciSummary.KilledMutants)
			}

			if len(ciSummary.Files) != tt.expectedFiles {
				t.Errorf("expected %d files, got %d", tt.expectedFiles, len(ciSummary.Files))
			}

			// Verify file scores
			for filePath, expectedScore := range tt.expectedScores {
				if fileReport, ok := ciSummary.Files[filePath]; ok {
					if math.Abs(fileReport.MutationScore-expectedScore) > 0.01 {
						t.Errorf("file %s: expected score %.2f, got %.2f",
							filePath, expectedScore, fileReport.MutationScore)
					}
				} else {
					t.Errorf("expected file %s not found in summary", filePath)
				}
			}

			// Verify duration is preserved
			if ciSummary.Duration != tt.summary.Duration {
				t.Errorf("expected duration %v, got %v", tt.summary.Duration, ciSummary.Duration)
			}
		})
	}
}

func TestSetDefaultOptions(t *testing.T) {
	tests := []struct {
		name            string
		input           *RunOptions
		expectWorkers   int
		expectTimeout   int
		expectOutput    string
		expectThreshold float64
	}{
		{
			name:            "nil options returns defaults",
			input:           nil,
			expectWorkers:   4,
			expectTimeout:   30,
			expectOutput:    "console",
			expectThreshold: 80.0,
		},
		{
			name: "non-nil options returns as is",
			input: &RunOptions{
				Workers:   2,
				Timeout:   10,
				Output:    "html",
				Threshold: 90.0,
			},
			expectWorkers:   2,
			expectTimeout:   10,
			expectOutput:    "html",
			expectThreshold: 90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{}
			result := engine.setDefaultOptions(tt.input)

			if result.Workers != tt.expectWorkers {
				t.Errorf("expected workers %d, got %d", tt.expectWorkers, result.Workers)
			}

			if result.Timeout != tt.expectTimeout {
				t.Errorf("expected timeout %d, got %d", tt.expectTimeout, result.Timeout)
			}

			if result.Output != tt.expectOutput {
				t.Errorf("expected output %s, got %s", tt.expectOutput, result.Output)
			}

			if result.Threshold != tt.expectThreshold {
				t.Errorf("expected threshold %.1f, got %.1f", tt.expectThreshold, result.Threshold)
			}
		})
	}
}

func TestGetAbsolutePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		setupFunc   func() (string, func())
		expectError bool
	}{
		{
			name:  "relative path is converted to absolute",
			input: ".",
			setupFunc: func() (string, func()) {
				wd, _ := os.Getwd()

				return wd, func() {}
			},
			expectError: false,
		},
		{
			name:  "absolute path remains unchanged",
			input: "/usr/local/bin",
			setupFunc: func() (string, func()) {
				return "/usr/local/bin", func() {}
			},
			expectError: false,
		},
		{
			name:  "non-existent path still returns absolute path",
			input: "/this/path/does/not/exist/at/all",
			setupFunc: func() (string, func()) {
				return "/this/path/does/not/exist/at/all", func() {}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{}
			expectedPath, cleanup := tt.setupFunc()

			defer cleanup()

			result, err := engine.getAbsolutePath(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.input == "." && result != expectedPath {
				t.Errorf("expected absolute path %s, got %s", expectedPath, result)
			}

			if tt.input == "/usr/local/bin" && result != tt.input {
				t.Errorf("expected path %s, got %s", tt.input, result)
			}
		})
	}
}

func TestLogStartupInfo(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		opts      *RunOptions
		expectLog bool
	}{
		{
			name: "verbose mode logs info",
			path: "/test/path",
			opts: &RunOptions{
				Verbose:     true,
				Workers:     2,
				Timeout:     10,
				Output:      "json",
				Incremental: true,
			},
			expectLog: true,
		},
		{
			name: "non-verbose mode doesn't log",
			path: "/test/path",
			opts: &RunOptions{
				Verbose: false,
			},
			expectLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			engine := &Engine{}
			// Just verify the function doesn't panic
			engine.logStartupInfo(tt.path, tt.opts)
		})
	}
}

func TestCalculateTestHash(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(t *testing.T) (string, string)
		expectHash bool
	}{
		{
			name: "calculates hash for file with test",
			setupFiles: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				// Create main file
				mainFile := filepath.Join(tempDir, "calc.go")
				content := `package calc

func Add(a, b int) int {
	return a + b
}
`
				writeFile(t, mainFile, content)

				// Create test file
				testFile := filepath.Join(tempDir, "calc_test.go")
				testContent := `package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("Add failed")
	}
}
`
				writeFile(t, testFile, testContent)

				return mainFile, tempDir
			},
			expectHash: true,
		},
		{
			name: "returns empty for file without test",
			setupFiles: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				mainFile := filepath.Join(tempDir, "notested.go")
				content := `package main

func Untested() {}
`
				writeFile(t, mainFile, content)

				return mainFile, tempDir
			},
			expectHash: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, _ := tt.setupFiles(t)

			hasher := analysis.NewFileHasher()

			hash := calculateTestHash(filePath, hasher)

			if tt.expectHash && hash == "" {
				t.Error("expected non-empty hash")
			}

			if !tt.expectHash && hash != "" {
				t.Error("expected empty hash")
			}
		})
	}
}

func TestHistoryStoreWrapper(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(t *testing.T) *historyStoreWrapper
		filePath    string
		expectEntry bool
		expectHash  string
	}{
		{
			name: "returns existing entry",
			setupStore: func(t *testing.T) *historyStoreWrapper {
				tempFile := filepath.Join(t.TempDir(), "history.json")
				store, _ := history.New(tempFile)

				// Add an entry
				store.UpdateFileWithHashes("test.go", nil, nil, "hash123", "testhash456")

				return &historyStoreWrapper{store: store}
			},
			filePath:    "test.go",
			expectEntry: true,
			expectHash:  "hash123",
		},
		{
			name: "returns false for non-existent entry",
			setupStore: func(t *testing.T) *historyStoreWrapper {
				tempFile := filepath.Join(t.TempDir(), "history.json")
				store, _ := history.New(tempFile)

				return &historyStoreWrapper{store: store}
			},
			filePath:    "notfound.go",
			expectEntry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := tt.setupStore(t)

			entry, exists := wrapper.GetEntry(tt.filePath)

			if exists != tt.expectEntry {
				t.Errorf("expected exists=%v, got %v", tt.expectEntry, exists)
			}

			if tt.expectEntry && entry.FileHash != tt.expectHash {
				t.Errorf("expected hash %s, got %s", tt.expectHash, entry.FileHash)
			}
		})
	}
}

func TestHandleCIWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		summary     *report.Summary
		opts        *RunOptions
		expectError bool
		errContains string
	}{
		{
			name:        "nil options returns nil",
			summary:     &report.Summary{},
			opts:        nil,
			expectError: false,
		},
		{
			name:    "CI mode disabled returns nil",
			summary: &report.Summary{},
			opts: &RunOptions{
				CIMode: false,
			},
			expectError: false,
		},
		{
			name: "CI mode enabled with passing quality gate",
			summary: &report.Summary{
				TotalMutants:  10,
				KilledMutants: 9,
				Statistics: report.Statistics{
					Score: 90.0,
				},
				Results: []mutation.Result{
					{Mutant: mutation.Mutant{FilePath: "test.go"}, Status: mutation.StatusKilled},
				},
			},
			opts: &RunOptions{
				CIMode:     true,
				Threshold:  80.0,
				FailOnGate: true,
			},
			expectError: false,
		},
		{
			name: "CI mode enabled with failing quality gate and FailOnGate true",
			summary: &report.Summary{
				TotalMutants:  10,
				KilledMutants: 5,
				Statistics: report.Statistics{
					Score: 50.0,
				},
				Results: []mutation.Result{
					{Mutant: mutation.Mutant{FilePath: "test.go"}, Status: mutation.StatusKilled},
					{Mutant: mutation.Mutant{FilePath: "test.go"}, Status: mutation.StatusSurvived},
				},
			},
			opts: &RunOptions{
				CIMode:     true,
				Threshold:  80.0,
				FailOnGate: true,
			},
			expectError: true,
			errContains: "quality gate failed",
		},
		{
			name: "CI mode enabled with failing quality gate but FailOnGate false",
			summary: &report.Summary{
				TotalMutants:  10,
				KilledMutants: 5,
				Statistics: report.Statistics{
					Score: 50.0,
				},
				Results: []mutation.Result{
					{Mutant: mutation.Mutant{FilePath: "test.go"}, Status: mutation.StatusKilled},
					{Mutant: mutation.Mutant{FilePath: "test.go"}, Status: mutation.StatusSurvived},
				},
			},
			opts: &RunOptions{
				CIMode:     true,
				Threshold:  80.0,
				FailOnGate: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter, _ := report.New("json")
			engine := &Engine{
				reporter: reporter,
			}

			err := engine.handleCIWorkflow(context.Background(), tt.summary, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProcessCIWorkflowDetailed(t *testing.T) {
	summary := &report.Summary{
		TotalMutants:  10,
		KilledMutants: 10,
		Statistics: report.Statistics{
			Score: 100.0,
		},
		Results: []mutation.Result{
			{Mutant: mutation.Mutant{FilePath: "test.go"}, Status: mutation.StatusKilled},
		},
	}
	opts := &RunOptions{
		Verbose:    true,
		Threshold:  80.0,
		FailOnGate: false,
	}

	reporter, _ := report.New("json")
	engine := &Engine{
		reporter:    reporter,
		ciReporter:  ci.NewReporter(".", "json"),
		qualityGate: ci.NewQualityGateEvaluator(true, 80.0),
	}

	err := engine.processCIWorkflow(context.Background(), summary, opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHistoryStoreWrapperHasChanged(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "history.json")
	store, _ := history.New(tempFile)
	wrapper := &historyStoreWrapper{store: store}

	// Test changed file
	store.UpdateFileWithHashes("changed.go", nil, nil, "oldhash", "")

	if !wrapper.HasChanged("changed.go", "newhash") {
		t.Error("expected file to be detected as changed")
	}

	// Test unchanged file
	store.UpdateFileWithHashes("same.go", nil, nil, "samehash", "")

	if wrapper.HasChanged("same.go", "samehash") {
		t.Error("expected file to be detected as unchanged")
	}

	// Test new file
	if !wrapper.HasChanged("new.go", "newhash") {
		t.Error("expected new file to be detected as changed")
	}
}
