package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Workers != 4 {
		t.Errorf("Expected Workers to be 4, got %d", cfg.Workers)
	}

	if cfg.Test.Command != "go test" {
		t.Errorf("Expected TestCommand to be 'go test', got %s", cfg.Test.Command)
	}

	if cfg.Test.Timeout != 30 {
		t.Errorf("Expected TestTimeout to be 30, got %d", cfg.Test.Timeout)
	}

	expectedPatterns := []string{"*_test.go"}
	if diff := cmp.Diff(cfg.Test.Patterns, expectedPatterns); diff != "" {
		t.Errorf("TestPatterns mismatch (-want +got):\n%s", diff)
	}

	expectedExcludes := []string{"vendor/", ".git/"}
	if diff := cmp.Diff(cfg.Test.Exclude, expectedExcludes); diff != "" {
		t.Errorf("ExcludeFiles mismatch (-want +got):\n%s", diff)
	}

	expectedMutators := []string{"arithmetic", "conditional", "logical"}
	if diff := cmp.Diff(cfg.Mutation.Types, expectedMutators); diff != "" {
		t.Errorf("Mutators mismatch (-want +got):\n%s", diff)
	}

	if cfg.Mutation.Limit != 1000 {
		t.Errorf("Expected MutationLimit to be 1000, got %d", cfg.Mutation.Limit)
	}

	if cfg.Incremental.HistoryFile != ".gomu_history.json" {
		t.Errorf("Expected HistoryFile to be '.gomu_history.json', got %s", cfg.Incremental.HistoryFile)
	}

	if !cfg.Incremental.UseGitDiff {
		t.Error("Expected UseGitDiff to be true")
	}

	if cfg.Incremental.BaseBranch != "main" {
		t.Errorf("Expected BaseBranch to be 'main', got %s", cfg.Incremental.BaseBranch)
	}

	if cfg.Output.Format != "json" {
		t.Errorf("Expected OutputFormat to be 'json', got %s", cfg.Output.Format)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Test loading with no config file (should return defaults)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should be equal to defaults
	expected := Default()
	if diff := cmp.Diff(cfg, expected); diff != "" {
		t.Errorf("Config loaded without file mismatch (-want +got):\n%s", diff)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.json")

	configContent := `
verbose: true
workers: 8
test:
  command: "go test -v"
  timeout: 60
  patterns:
    - "*_test.go"
    - "*_integration_test.go"
  exclude:
    - "vendor/"
    - ".git/"
    - "generated/"
mutation:
  types:
    - "arithmetic"
    - "conditional"
  limit: 500
incremental:
  historyFile: ".custom_history.json"
  useGitDiff: false
  baseBranch: "develop"
output:
  format: "text"`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check all values were loaded correctly
	if !cfg.Verbose {
		t.Error("Expected Verbose to be true")
	}

	if cfg.Workers != 8 {
		t.Errorf("Expected Workers to be 8, got %d", cfg.Workers)
	}

	if cfg.Test.Command != "go test -v" {
		t.Errorf("Expected TestCommand to be 'go test -v', got %s", cfg.Test.Command)
	}

	if cfg.Test.Timeout != 60 {
		t.Errorf("Expected TestTimeout to be 60, got %d", cfg.Test.Timeout)
	}

	expectedPatterns := []string{"*_test.go", "*_integration_test.go"}
	if diff := cmp.Diff(cfg.Test.Patterns, expectedPatterns); diff != "" {
		t.Errorf("TestPatterns mismatch (-want +got):\n%s", diff)
	}

	expectedExcludes := []string{"vendor/", ".git/", "generated/"}
	if diff := cmp.Diff(cfg.Test.Exclude, expectedExcludes); diff != "" {
		t.Errorf("ExcludeFiles mismatch (-want +got):\n%s", diff)
	}

	expectedMutators := []string{"arithmetic", "conditional"}
	if diff := cmp.Diff(cfg.Mutation.Types, expectedMutators); diff != "" {
		t.Errorf("Mutators mismatch (-want +got):\n%s", diff)
	}

	if cfg.Mutation.Limit != 500 {
		t.Errorf("Expected MutationLimit to be 500, got %d", cfg.Mutation.Limit)
	}

	if cfg.Incremental.HistoryFile != ".custom_history.json" {
		t.Errorf("Expected HistoryFile to be '.custom_history.json', got %s", cfg.Incremental.HistoryFile)
	}

	if cfg.Incremental.UseGitDiff {
		t.Error("Expected UseGitDiff to be false")
	}

	if cfg.Incremental.BaseBranch != "develop" {
		t.Errorf("Expected BaseBranch to be 'develop', got %s", cfg.Incremental.BaseBranch)
	}

	if cfg.Output.Format != "text" {
		t.Errorf("Expected OutputFormat to be 'text', got %s", cfg.Output.Format)
	}
}
