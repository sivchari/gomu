package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Workers != 4 {
		t.Errorf("Expected Workers to be 4, got %d", cfg.Workers)
	}

	if cfg.TestCommand != "go test" {
		t.Errorf("Expected TestCommand to be 'go test', got %s", cfg.TestCommand)
	}

	if cfg.TestTimeout != 30 {
		t.Errorf("Expected TestTimeout to be 30, got %d", cfg.TestTimeout)
	}

	expectedPatterns := []string{"*_test.go"}
	if !reflect.DeepEqual(cfg.TestPatterns, expectedPatterns) {
		t.Errorf("Expected TestPatterns to be %v, got %v", expectedPatterns, cfg.TestPatterns)
	}

	expectedExcludes := []string{"vendor/", ".git/"}
	if !reflect.DeepEqual(cfg.ExcludeFiles, expectedExcludes) {
		t.Errorf("Expected ExcludeFiles to be %v, got %v", expectedExcludes, cfg.ExcludeFiles)
	}

	expectedMutators := []string{"arithmetic", "conditional", "logical"}
	if !reflect.DeepEqual(cfg.Mutators, expectedMutators) {
		t.Errorf("Expected Mutators to be %v, got %v", expectedMutators, cfg.Mutators)
	}

	if cfg.MutationLimit != 1000 {
		t.Errorf("Expected MutationLimit to be 1000, got %d", cfg.MutationLimit)
	}

	if cfg.HistoryFile != ".gomu_history.json" {
		t.Errorf("Expected HistoryFile to be '.gomu_history.json', got %s", cfg.HistoryFile)
	}

	if !cfg.UseGitDiff {
		t.Error("Expected UseGitDiff to be true")
	}

	if cfg.BaseBranch != "main" {
		t.Errorf("Expected BaseBranch to be 'main', got %s", cfg.BaseBranch)
	}

	if cfg.OutputFormat != "json" {
		t.Errorf("Expected OutputFormat to be 'json', got %s", cfg.OutputFormat)
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
	if !reflect.DeepEqual(cfg, expected) {
		t.Error("Config loaded without file should equal defaults")
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
		"verbose": true,
		"workers": 8,
		"testCommand": "go test -v",
		"testTimeout": 60,
		"testPatterns": ["*_test.go", "*_integration_test.go"],
		"excludeFiles": ["vendor/", ".git/", "generated/"],
		"mutators": ["arithmetic", "conditional"],
		"mutationLimit": 500,
		"historyFile": ".custom_history.json",
		"useGitDiff": false,
		"baseBranch": "develop",
		"outputFormat": "text"
	}`

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

	if cfg.TestCommand != "go test -v" {
		t.Errorf("Expected TestCommand to be 'go test -v', got %s", cfg.TestCommand)
	}

	if cfg.TestTimeout != 60 {
		t.Errorf("Expected TestTimeout to be 60, got %d", cfg.TestTimeout)
	}

	expectedPatterns := []string{"*_test.go", "*_integration_test.go"}
	if !reflect.DeepEqual(cfg.TestPatterns, expectedPatterns) {
		t.Errorf("Expected TestPatterns to be %v, got %v", expectedPatterns, cfg.TestPatterns)
	}

	expectedExcludes := []string{"vendor/", ".git/", "generated/"}
	if !reflect.DeepEqual(cfg.ExcludeFiles, expectedExcludes) {
		t.Errorf("Expected ExcludeFiles to be %v, got %v", expectedExcludes, cfg.ExcludeFiles)
	}

	expectedMutators := []string{"arithmetic", "conditional"}
	if !reflect.DeepEqual(cfg.Mutators, expectedMutators) {
		t.Errorf("Expected Mutators to be %v, got %v", expectedMutators, cfg.Mutators)
	}

	if cfg.MutationLimit != 500 {
		t.Errorf("Expected MutationLimit to be 500, got %d", cfg.MutationLimit)
	}

	if cfg.HistoryFile != ".custom_history.json" {
		t.Errorf("Expected HistoryFile to be '.custom_history.json', got %s", cfg.HistoryFile)
	}

	if cfg.UseGitDiff {
		t.Error("Expected UseGitDiff to be false")
	}

	if cfg.BaseBranch != "develop" {
		t.Errorf("Expected BaseBranch to be 'develop', got %s", cfg.BaseBranch)
	}

	if cfg.OutputFormat != "text" {
		t.Errorf("Expected OutputFormat to be 'text', got %s", cfg.OutputFormat)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	// Create temporary config file with invalid JSON
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid_config.json")

	invalidContent := `{
		"workers": 8,
		"invalid": json
	}`

	err := os.WriteFile(configFile, []byte(invalidContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err = Load(configFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	// Test loading with nonexistent file path
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		expect *Config
	}{
		{
			name: "zero workers",
			config: &Config{
				Workers: 0,
			},
			expect: &Config{
				Workers:      4,
				TestTimeout:  30,
				TestPatterns: []string{"*_test.go"},
				HistoryFile:  ".gomu_history.json",
				BaseBranch:   "main",
				OutputFormat: "json",
			},
		},
		{
			name: "negative timeout",
			config: &Config{
				TestTimeout: -5,
			},
			expect: &Config{
				Workers:      4,
				TestTimeout:  30,
				TestPatterns: []string{"*_test.go"},
				HistoryFile:  ".gomu_history.json",
				BaseBranch:   "main",
				OutputFormat: "json",
			},
		},
		{
			name: "empty test patterns",
			config: &Config{
				TestPatterns: []string{},
			},
			expect: &Config{
				Workers:      4,
				TestTimeout:  30,
				TestPatterns: []string{"*_test.go"},
				HistoryFile:  ".gomu_history.json",
				BaseBranch:   "main",
				OutputFormat: "json",
			},
		},
		{
			name: "empty history file",
			config: &Config{
				HistoryFile: "",
			},
			expect: &Config{
				Workers:      4,
				TestTimeout:  30,
				TestPatterns: []string{"*_test.go"},
				HistoryFile:  ".gomu_history.json",
				BaseBranch:   "main",
				OutputFormat: "json",
			},
		},
		{
			name: "empty base branch",
			config: &Config{
				BaseBranch: "",
			},
			expect: &Config{
				Workers:      4,
				TestTimeout:  30,
				TestPatterns: []string{"*_test.go"},
				HistoryFile:  ".gomu_history.json",
				BaseBranch:   "main",
				OutputFormat: "json",
			},
		},
		{
			name: "empty output format",
			config: &Config{
				OutputFormat: "",
			},
			expect: &Config{
				Workers:      4,
				TestTimeout:  30,
				TestPatterns: []string{"*_test.go"},
				HistoryFile:  ".gomu_history.json",
				BaseBranch:   "main",
				OutputFormat: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.validate()

			if !reflect.DeepEqual(tt.config, tt.expect) {
				t.Errorf("After validation, config = %+v, want %+v", tt.config, tt.expect)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "save_test.json")

	cfg := &Config{
		Verbose:       true,
		Workers:       8,
		TestCommand:   "go test -v",
		TestTimeout:   60,
		TestPatterns:  []string{"*_test.go"},
		ExcludeFiles:  []string{"vendor/"},
		Mutators:      []string{"arithmetic"},
		MutationLimit: 500,
		HistoryFile:   ".test_history.json",
		UseGitDiff:    false,
		BaseBranch:    "develop",
		OutputFormat:  "text",
		OutputFile:    "output.txt",
	}

	err := cfg.Save(configFile)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load and verify content
	loadedCfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Validate will be called on load, so compare the validated version
	cfg.validate()

	if !reflect.DeepEqual(cfg, loadedCfg) {
		t.Errorf("Loaded config does not match saved config:\nSaved:  %+v\nLoaded: %+v", cfg, loadedCfg)
	}
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := Default()

	// Try to save to an invalid path
	err := cfg.Save("/invalid/path/that/does/not/exist/config.json")
	if err == nil {
		t.Error("Expected error for invalid save path, got nil")
	}
}
