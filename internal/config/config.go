// Package config provides configuration management for gomu.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// Load loads configuration from file, falling back to defaults.
func Load(configFile string) (*Config, error) {
	cfg := Default()

	// If no config file specified, try default locations
	if configFile == "" {
		candidates := []string{".gomu.json"}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configFile = candidate

				break
			}
		}
	}

	// If config file exists, load it
	if configFile != "" {
		if err := cfg.loadFromFile(configFile); err != nil {
			return nil, err
		}
	}

	// Validate and set defaults
	cfg.validate()

	return cfg, nil
}

func (c *Config) loadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

func (c *Config) validate() {
	if c.Workers <= 0 {
		c.Workers = 4
	}

	if c.TestTimeout <= 0 {
		c.TestTimeout = 30
	}

	if len(c.TestPatterns) == 0 {
		c.TestPatterns = []string{"*_test.go"}
	}

	if c.HistoryFile == "" {
		c.HistoryFile = ".gomu_history.json"
	}

	if c.BaseBranch == "" {
		c.BaseBranch = "main"
	}

	if c.OutputFormat == "" {
		c.OutputFormat = "json"
	}
}

// Save saves the configuration to a file.
func (c *Config) Save(filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
