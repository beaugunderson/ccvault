// ABOUTME: Tests for the config package
// ABOUTME: Verifies configuration loading and default values

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultClaudeHome(t *testing.T) {
	home := DefaultClaudeHome()

	if home == "" {
		t.Error("DefaultClaudeHome returned empty string")
	}

	// Should end with .claude
	if filepath.Base(home) != ".claude" {
		t.Errorf("DefaultClaudeHome should end with .claude, got %s", home)
	}

	// Should be under home directory
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home dir: %v", err)
	}
	expected := filepath.Join(userHome, ".claude")
	if home != expected {
		t.Errorf("Expected %s, got %s", expected, home)
	}
}

func TestDefaultDataDir(t *testing.T) {
	dataDir := DefaultDataDir()

	if dataDir == "" {
		t.Error("DefaultDataDir returned empty string")
	}

	// Should end with .ccvault
	if filepath.Base(dataDir) != ".ccvault" {
		t.Errorf("DefaultDataDir should end with .ccvault, got %s", dataDir)
	}

	// Should be under home directory
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home dir: %v", err)
	}
	expected := filepath.Join(userHome, ".ccvault")
	if dataDir != expected {
		t.Errorf("Expected %s, got %s", expected, dataDir)
	}
}

func TestLoad(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Should have default values
	if cfg.ClaudeHome == "" {
		t.Error("ClaudeHome is empty")
	}
	if cfg.DataDir == "" {
		t.Error("DataDir is empty")
	}
}

func TestEnsureDataDir(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	testDataDir := filepath.Join(tmpDir, "test-ccvault")

	cfg := &Config{
		DataDir: testDataDir,
	}

	err := EnsureDataDir(cfg)
	if err != nil {
		t.Fatalf("EnsureDataDir failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(testDataDir)
	if err != nil {
		t.Fatalf("Failed to stat created directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("EnsureDataDir did not create a directory")
	}
}
