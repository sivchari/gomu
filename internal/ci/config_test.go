package ci

import (
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Clear any existing environment variables that might affect the test
	t.Setenv("CI_MODE", "")
	t.Setenv("GITHUB_PR_NUMBER", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_ACTOR", "")
	t.Setenv("GITHUB_EVENT_NAME", "")
	t.Setenv("GITHUB_WORKSPACE", "")

	// Test default values
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
	t.Setenv("CI_MODE", "push")
	t.Setenv("GITHUB_PR_NUMBER", "123")
	t.Setenv("GITHUB_BASE_REF", "develop")
	t.Setenv("GITHUB_HEAD_REF", "feature-branch")

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
	t.Setenv("GITHUB_PR_NUMBER", "invalid")

	config := LoadConfigFromEnv()

	if config.PRNumber != 0 {
		t.Errorf("Expected PR number 0 for invalid input, got %d", config.PRNumber)
	}
}

func TestNewConfigFromEnv(t *testing.T) {
	// Clear any existing environment variables first
	t.Setenv("CI_MODE", "")
	t.Setenv("GITHUB_PR_NUMBER", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_ACTOR", "")
	t.Setenv("GITHUB_EVENT_NAME", "")
	t.Setenv("GITHUB_WORKSPACE", "")

	// Test with custom values
	t.Setenv("CI_MODE", "true")
	t.Setenv("GITHUB_PR_NUMBER", "456")
	t.Setenv("GITHUB_BASE_REF", "master")
	t.Setenv("GITHUB_HEAD_REF", "feature/test")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_ACTOR", "testuser")
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_WORKSPACE", "/workspace")

	config := NewConfigFromEnv()

	if config.Mode != "true" {
		t.Errorf("Expected mode 'true', got '%s'", config.Mode)
	}

	if config.PRNumber != 456 {
		t.Errorf("Expected PR number 456, got %d", config.PRNumber)
	}

	if config.BaseRef != "master" {
		t.Errorf("Expected base ref 'master', got '%s'", config.BaseRef)
	}

	if config.HeadRef != "feature/test" {
		t.Errorf("Expected head ref 'feature/test', got '%s'", config.HeadRef)
	}

	if config.Repository != "owner/repo" {
		t.Errorf("Expected repository 'owner/repo', got '%s'", config.Repository)
	}

	if config.Actor != "testuser" {
		t.Errorf("Expected actor 'testuser', got '%s'", config.Actor)
	}

	if config.EventName != "pull_request" {
		t.Errorf("Expected event name 'pull_request', got '%s'", config.EventName)
	}

	if config.Workspace != "/workspace" {
		t.Errorf("Expected workspace '/workspace', got '%s'", config.Workspace)
	}
}

func TestIsCIMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{
			name: "true in lowercase",
			mode: "true",
			want: true,
		},
		{
			name: "TRUE in uppercase",
			mode: "TRUE",
			want: true,
		},
		{
			name: "True in mixed case",
			mode: "True",
			want: true,
		},
		{
			name: "false",
			mode: "false",
			want: false,
		},
		{
			name: "pr mode",
			mode: "pr",
			want: false,
		},
		{
			name: "empty",
			mode: "",
			want: false,
		},
		{
			name: "1 as string",
			mode: "1",
			want: true,
		},
		{
			name: "on",
			mode: "on",
			want: true,
		},
		{
			name: "yes",
			mode: "yes",
			want: true,
		},
		{
			name: "random value",
			mode: "random",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{Mode: tt.mode}
			if got := config.IsCIMode(); got != tt.want {
				t.Errorf("IsCIMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPullRequest(t *testing.T) {
	tests := []struct {
		name      string
		eventName string
		prNumber  int
		want      bool
	}{
		{
			name:      "valid pull request",
			eventName: "pull_request",
			prNumber:  123,
			want:      true,
		},
		{
			name:      "pull request with zero PR number",
			eventName: "pull_request",
			prNumber:  0,
			want:      false,
		},
		{
			name:      "push event",
			eventName: "push",
			prNumber:  123,
			want:      false,
		},
		{
			name:      "empty event name",
			eventName: "",
			prNumber:  123,
			want:      false,
		},
		{
			name:      "negative PR number",
			eventName: "pull_request",
			prNumber:  -1,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				EventName: tt.eventName,
				PRNumber:  tt.prNumber,
			}
			if got := config.IsPullRequest(); got != tt.want {
				t.Errorf("IsPullRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBaseBranch(t *testing.T) {
	tests := []struct {
		name    string
		baseRef string
		want    string
	}{
		{
			name:    "custom base ref",
			baseRef: "develop",
			want:    "develop",
		},
		{
			name:    "empty base ref returns main",
			baseRef: "",
			want:    "main",
		},
		{
			name:    "master branch",
			baseRef: "master",
			want:    "master",
		},
		{
			name:    "feature branch",
			baseRef: "feature/new-feature",
			want:    "feature/new-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{BaseRef: tt.baseRef}
			if got := config.GetBaseBranch(); got != tt.want {
				t.Errorf("GetBaseBranch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue string
		want         string
	}{
		{
			name:         "env var exists",
			key:          "TEST_ENV_VAR",
			envValue:     "test-value",
			defaultValue: "default",
			want:         "test-value",
		},
		{
			name:         "env var empty returns default",
			key:          "TEST_ENV_VAR",
			envValue:     "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "env var single char",
			key:          "TEST_ENV_VAR",
			envValue:     "a",
			defaultValue: "default",
			want:         "a",
		},
		{
			name:         "default value empty",
			key:          "TEST_ENV_VAR",
			envValue:     "",
			defaultValue: "",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			if got := getEnv(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
