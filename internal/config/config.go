package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the configuration for gomu
type Config struct {
	// General settings
	Verbose bool `json:"verbose,omitempty"`
	Workers int  `json:"workers,omitempty"`

	// Test settings
	TestCommand   string   `json:"test_command,omitempty"`
	TestTimeout   int      `json:"test_timeout,omitempty"`
	TestPatterns  []string `json:"test_patterns,omitempty"`
	ExcludeFiles  []string `json:"exclude_files,omitempty"`

	// Mutation settings
	Mutators      []string `json:"mutators,omitempty"`
	MutationLimit int      `json:"mutation_limit,omitempty"`

	// Incremental analysis
	HistoryFile     string `json:"history_file,omitempty"`
	UseGitDiff      bool   `json:"use_git_diff,omitempty"`
	BaseBranch      string `json:"base_branch,omitempty"`

	// Output settings
	OutputFormat string `json:"output_format,omitempty"`
	OutputFile   string `json:"output_file,omitempty"`
}

// Default returns a config with default values
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

// Load loads configuration from file, falling back to defaults
func Load(configFile string) (*Config, error) {
	cfg := Default()

	// If no config file specified, try default locations
	if configFile == "" {
		candidates := []string{".gomu.json", "gomu.json", ".gomu/config.json"}
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
		return err
	}

	return json.Unmarshal(data, c)
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

// Save saves the configuration to a file
func (c *Config) Save(filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}