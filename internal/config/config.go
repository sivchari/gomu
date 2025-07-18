// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Empty - all configuration via CLI flags and environment
}

func Default() *Config {
	return &Config{}
}

// Load loads configuration from file with fallback support.
// Since config is now zero-config, this always returns default empty config.
func Load(configFile string) (*Config, error) {
	cfg := Default()

	// Config files are optional - ignore errors and just use defaults
	if configFile != "" {
		if err := cfg.loadFromFile(configFile); err != nil {
			// Just log and continue with defaults - config files are optional
		}
	}

	// Validate and set defaults
	cfg.validate()

	return cfg, nil
}

// loadFromFile loads config from  file.
func (c *Config) loadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse  config file: %w", err)
	}

	return nil
}

// validate ensures the configuration has sensible values.
func (c *Config) validate() {
	// No validation needed - using intelligent defaults
}

// Save saves the configuration to a  file.
func (c *Config) Save(filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal  config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write  config file: %w", err)
	}

	return nil
}
