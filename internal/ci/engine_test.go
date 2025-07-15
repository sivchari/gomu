package ci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sivchari/gomu/internal/config"
)

func TestNewCIEngine(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a basic config file
	cfg := config.Default()
	configPath := filepath.Join(tmpDir, ".gomu.yaml")
	configData, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath, configData, 0600)

	// Set CI environment variables
	os.Setenv("CI_MODE", "pr")
	os.Setenv("GITHUB_PR_NUMBER", "123")
	os.Setenv("GITHUB_BASE_REF", "main")
	defer func() {
		os.Unsetenv("CI_MODE")
		os.Unsetenv("GITHUB_PR_NUMBER")
		os.Unsetenv("GITHUB_BASE_REF")
	}()

	engine, err := NewCIEngine(configPath, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create CI engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	if engine.workDir != tmpDir {
		t.Errorf("Expected workDir %s, got %s", tmpDir, engine.workDir)
	}

	if engine.ciConfig.Mode != "pr" {
		t.Errorf("Expected CI mode 'pr', got '%s'", engine.ciConfig.Mode)
	}

	if engine.ciConfig.PRNumber != 123 {
		t.Errorf("Expected PR number 123, got %d", engine.ciConfig.PRNumber)
	}
}

func TestNewCIEngine_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Try with non-existent config file
	_, err := NewCIEngine("/non/existent/config.json", tmpDir)
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

func TestCIEngine_shouldFailOnQualityGate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a basic config file
	cfg := config.Default()
	configPath := filepath.Join(tmpDir, ".gomu.yaml")
	configData, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath, configData, 0600)

	engine, err := NewCIEngine(configPath, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create CI engine: %v", err)
	}

	// For now, this should always return true
	if !engine.shouldFailOnQualityGate() {
		t.Error("Expected shouldFailOnQualityGate to return true")
	}
}

// Integration test that would require more setup
func TestCIEngine_Run_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a basic config file
	cfg := config.Default()
	configPath := filepath.Join(tmpDir, ".gomu.yaml")
	configData, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath, configData, 0600)

	// Initialize git repo (mock)
	gitDir := filepath.Join(tmpDir, ".git")
	os.Mkdir(gitDir, 0755)

	engine, err := NewCIEngine(configPath, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create CI engine: %v", err)
	}

	// This test would require a more complete setup with actual Go files
	// For now, we just test that the engine can be created
	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
}
