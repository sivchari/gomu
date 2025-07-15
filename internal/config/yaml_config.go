package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the unified YAML configuration structure.
type YAMLConfig struct {
	// General settings
	Verbose bool `yaml:"verbose,omitempty" json:"verbose,omitempty"`
	Workers int  `yaml:"workers,omitempty" json:"workers,omitempty"`

	// Test settings
	Test TestConfig `yaml:"test,omitempty" json:"test,omitempty"`

	// Mutation settings
	Mutation MutationConfig `yaml:"mutation,omitempty" json:"mutation,omitempty"`

	// Incremental analysis
	Incremental IncrementalConfig `yaml:"incremental,omitempty" json:"incremental,omitempty"`

	// Output settings
	Output OutputConfig `yaml:"output,omitempty" json:"output,omitempty"`

	// CI/CD settings
	CI CIConfig `yaml:"ci,omitempty" json:"ci,omitempty"`
}

// TestConfig contains test-related configuration.
type TestConfig struct {
	Command  string   `yaml:"command,omitempty" json:"command,omitempty"`
	Timeout  int      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Patterns []string `yaml:"patterns,omitempty" json:"patterns,omitempty"`
	Exclude  []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// MutationConfig contains mutation-related configuration.
type MutationConfig struct {
	Types []string `yaml:"types,omitempty" json:"types,omitempty"`
	Limit int      `yaml:"limit,omitempty" json:"limit,omitempty"`
}

// IncrementalConfig contains incremental analysis configuration.
type IncrementalConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	HistoryFile string `yaml:"historyFile,omitempty" json:"historyFile,omitempty"`
	UseGitDiff  bool   `yaml:"useGitDiff" json:"useGitDiff"`
	BaseBranch  string `yaml:"baseBranch,omitempty" json:"baseBranch,omitempty"`
}

// OutputConfig contains output-related configuration.
type OutputConfig struct {
	Format string           `yaml:"format,omitempty" json:"format,omitempty"`
	File   string           `yaml:"file,omitempty" json:"file,omitempty"`
	HTML   HTMLOutputConfig `yaml:"html,omitempty" json:"html,omitempty"`
}

// HTMLOutputConfig contains HTML-specific output configuration.
type HTMLOutputConfig struct {
	Template string `yaml:"template,omitempty" json:"template,omitempty"`
	CSS      string `yaml:"css,omitempty" json:"css,omitempty"`
}

// CIConfig contains CI/CD-related configuration.
type CIConfig struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	QualityGate QualityGateConfig `yaml:"qualityGate,omitempty" json:"qualityGate,omitempty"`
	GitHub      GitHubConfig      `yaml:"github,omitempty" json:"github,omitempty"`
	Reports     CIReportsConfig   `yaml:"reports,omitempty" json:"reports,omitempty"`
}

// QualityGateConfig contains quality gate configuration.
type QualityGateConfig struct {
	Enabled            bool    `yaml:"enabled" json:"enabled"`
	MinMutationScore   float64 `yaml:"minMutationScore,omitempty" json:"minMutationScore,omitempty"`
	MaxSurvivors       int     `yaml:"maxSurvivors,omitempty" json:"maxSurvivors,omitempty"`
	FailOnQualityGate  bool    `yaml:"failOnQualityGate" json:"failOnQualityGate"`
	GradualEnforcement bool    `yaml:"gradualEnforcement,omitempty" json:"gradualEnforcement,omitempty"`
	BaselineFile       string  `yaml:"baselineFile,omitempty" json:"baselineFile,omitempty"`
}

// GitHubConfig contains GitHub-specific configuration.
type GitHubConfig struct {
	Enabled    bool `yaml:"enabled" json:"enabled"`
	PRComments bool `yaml:"prComments" json:"prComments"`
	Badges     bool `yaml:"badges,omitempty" json:"badges,omitempty"`
}

// CIReportsConfig contains CI report configuration.
type CIReportsConfig struct {
	Formats   []string `yaml:"formats,omitempty" json:"formats,omitempty"`
	OutputDir string   `yaml:"outputDir,omitempty" json:"outputDir,omitempty"`
	Artifacts bool     `yaml:"artifacts" json:"artifacts"`
}

// DefaultYAML returns a YAML config with default values.
func DefaultYAML() *YAMLConfig {
	return &YAMLConfig{
		Workers: 4,
		Test: TestConfig{
			Command:  "go test",
			Timeout:  30,
			Patterns: []string{"*_test.go"},
			Exclude:  []string{"vendor/", ".git/"},
		},
		Mutation: MutationConfig{
			Types: []string{"arithmetic", "conditional", "logical"},
			Limit: 1000,
		},
		Incremental: IncrementalConfig{
			Enabled:     true,
			HistoryFile: ".gomu_history.json",
			UseGitDiff:  true,
			BaseBranch:  "main",
		},
		Output: OutputConfig{
			Format: "json",
		},
		CI: CIConfig{
			Enabled: true,
			QualityGate: QualityGateConfig{
				Enabled:           true,
				MinMutationScore:  80.0,
				FailOnQualityGate: true,
			},
			GitHub: GitHubConfig{
				Enabled:    true,
				PRComments: true,
			},
			Reports: CIReportsConfig{
				Formats:   []string{"json", "html"},
				OutputDir: ".",
				Artifacts: true,
			},
		},
	}
}

// LoadYAML loads configuration from YAML file with fallback support.
func LoadYAML(configFile string) (*YAMLConfig, error) {
	cfg := DefaultYAML()

	// If no config file specified, try default locations
	if configFile == "" {
		candidates := []string{".gomu.yaml", ".gomu.yml"}
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

// loadFromFile loads config from YAML file.
func (c *YAMLConfig) loadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	return nil
}

// validate ensures the configuration has sensible values.
func (c *YAMLConfig) validate() {
	if c.Workers <= 0 {
		c.Workers = 4
	}

	if c.Test.Timeout <= 0 {
		c.Test.Timeout = 30
	}

	if len(c.Test.Patterns) == 0 {
		c.Test.Patterns = []string{"*_test.go"}
	}

	if c.Test.Command == "" {
		c.Test.Command = "go test"
	}

	if len(c.Mutation.Types) == 0 {
		c.Mutation.Types = []string{"arithmetic", "conditional", "logical"}
	}

	if c.Incremental.HistoryFile == "" {
		c.Incremental.HistoryFile = ".gomu_history.json"
	}

	if c.Incremental.BaseBranch == "" {
		c.Incremental.BaseBranch = "main"
	}

	if c.Output.Format == "" {
		c.Output.Format = "json"
	}

	if c.CI.QualityGate.MinMutationScore == 0 {
		c.CI.QualityGate.MinMutationScore = 80.0
	}

	if len(c.CI.Reports.Formats) == 0 {
		c.CI.Reports.Formats = []string{"json", "html"}
	}

	if c.CI.Reports.OutputDir == "" {
		c.CI.Reports.OutputDir = "."
	}
}

// ToLegacyConfig converts YAMLConfig to the legacy Config format for backward compatibility.
func (c *YAMLConfig) ToLegacyConfig() *Config {
	return &Config{
		Verbose:       c.Verbose,
		Workers:       c.Workers,
		TestCommand:   c.Test.Command,
		TestTimeout:   c.Test.Timeout,
		TestPatterns:  c.Test.Patterns,
		ExcludeFiles:  c.Test.Exclude,
		Mutators:      c.Mutation.Types,
		MutationLimit: c.Mutation.Limit,
		HistoryFile:   c.Incremental.HistoryFile,
		UseGitDiff:    c.Incremental.UseGitDiff,
		BaseBranch:    c.Incremental.BaseBranch,
		OutputFormat:  c.Output.Format,
		OutputFile:    c.Output.File,
	}
}

// SaveYAML saves the configuration to a YAML file.
func (c *YAMLConfig) SaveYAML(filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write YAML config file: %w", err)
	}

	return nil
}
