package config

import (
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// Config should be empty - all defaults are handled by intelligent defaults
	if cfg == nil {
		t.Error("Expected config to be non-nil")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Test loading with no config file (should return defaults)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should be equal to defaults
	if cfg == nil {
		t.Error("Expected config to be non-nil")
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Config files are no longer used - should gracefully handle missing files
	cfg, err := Load("nonexistent.yaml")
	if err != nil {
		// This is expected - config files are optional now
		t.Logf("Expected error for nonexistent config file: %v", err)
	}

	if cfg == nil {
		t.Error("Expected config to be non-nil even with missing file")
	}
}