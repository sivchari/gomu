package ci

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original env vars
	origMode := os.Getenv("CI_MODE")
	origPR := os.Getenv("GITHUB_PR_NUMBER")
	origBase := os.Getenv("GITHUB_BASE_REF")
	origHead := os.Getenv("GITHUB_HEAD_REF")

	// Restore after test
	defer func() {
		os.Setenv("CI_MODE", origMode)
		os.Setenv("GITHUB_PR_NUMBER", origPR)
		os.Setenv("GITHUB_BASE_REF", origBase)
		os.Setenv("GITHUB_HEAD_REF", origHead)
	}()

	// Test default values
	os.Unsetenv("CI_MODE")
	os.Unsetenv("GITHUB_PR_NUMBER")
	os.Unsetenv("GITHUB_BASE_REF")
	os.Unsetenv("GITHUB_HEAD_REF")

	config := LoadConfigFromEnv()

	if config.Mode != "pr" {
		t.Errorf("Expected default mode 'pr', got '%s'", config.Mode)
	}

	if config.PRNumber != 0 {
		t.Errorf("Expected default PR number 0, got %d", config.PRNumber)
	}

	if config.BaseRef != "main" {
		t.Errorf("Expected default base ref 'main', got '%s'", config.BaseRef)
	}

	if config.HeadRef != "" {
		t.Errorf("Expected default head ref '', got '%s'", config.HeadRef)
	}

	// Test with environment variables
	os.Setenv("CI_MODE", "push")
	os.Setenv("GITHUB_PR_NUMBER", "123")
	os.Setenv("GITHUB_BASE_REF", "develop")
	os.Setenv("GITHUB_HEAD_REF", "feature-branch")

	config = LoadConfigFromEnv()

	if config.Mode != "push" {
		t.Errorf("Expected mode 'push', got '%s'", config.Mode)
	}

	if config.PRNumber != 123 {
		t.Errorf("Expected PR number 123, got %d", config.PRNumber)
	}

	if config.BaseRef != "develop" {
		t.Errorf("Expected base ref 'develop', got '%s'", config.BaseRef)
	}

	if config.HeadRef != "feature-branch" {
		t.Errorf("Expected head ref 'feature-branch', got '%s'", config.HeadRef)
	}
}

func TestLoadConfigFromEnv_InvalidPRNumber(t *testing.T) {
	origPR := os.Getenv("GITHUB_PR_NUMBER")
	defer os.Setenv("GITHUB_PR_NUMBER", origPR)

	os.Setenv("GITHUB_PR_NUMBER", "invalid")

	config := LoadConfigFromEnv()

	if config.PRNumber != 0 {
		t.Errorf("Expected PR number 0 for invalid input, got %d", config.PRNumber)
	}
}
