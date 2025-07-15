// Package config provides configuration management for gomu.
package config

import (
	"fmt"
)

// Config represents the configuration for gomu.
type Config struct {
	// General settings
	Verbose bool `json:"verbose,omitempty"`
	Workers int  `json:"workers,omitempty"`

	// Test settings
	TestCommand  string   `json:"testCommand,omitempty"`
	TestTimeout  int      `json:"testTimeout,omitempty"`
	TestPatterns []string `json:"testPatterns,omitempty"`
	ExcludeFiles []string `json:"excludeFiles,omitempty"`

	// Mutation settings
	Mutators      []string `json:"mutators,omitempty"`
	MutationLimit int      `json:"mutationLimit,omitempty"`

	// Incremental analysis
	HistoryFile string `json:"historyFile,omitempty"`
	UseGitDiff  bool   `json:"useGitDiff"`
	BaseBranch  string `json:"baseBranch,omitempty"`

	// Output settings
	OutputFormat string `json:"outputFormat,omitempty"`
	OutputFile   string `json:"outputFile,omitempty"`
}

// Default returns a config with default values.
func Default() *Config {
	return &Config{
		Workers:       4,
		TestCommand:   "go test",
		TestTimeout:   30,
		TestPatterns:  []string{"*_test.go"},
		ExcludeFiles:  []string{"vendor/", ".git/"},
		Mutators:      []string{"arithmetic", "conditional", "logical"},
		MutationLimit: 1000,
		HistoryFile:   ".gomu_history.json",
		UseGitDiff:    true,
		BaseBranch:    "main",
		OutputFormat:  "json",
	}
}

// Load loads configuration from YAML file, falling back to defaults.
func Load(configFile string) (*Config, error) {
	// Load as unified YAML config
	yamlCfg, err := LoadYAML(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	return yamlCfg.ToLegacyConfig(), nil
}
