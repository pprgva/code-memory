package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	if cfg.Embedder.ModelPath != "" {
		t.Errorf("expected empty model_path, got %s", cfg.Embedder.ModelPath)
	}

	if cfg.Embedder.Dimensions != 1024 {
		t.Errorf("expected dimensions 1024, got %d", cfg.Embedder.Dimensions)
	}

	if cfg.Store.Backend != "gob" {
		t.Errorf("expected backend gob, got %s", cfg.Store.Backend)
	}

	if cfg.Chunking.Size != 512 {
		t.Errorf("expected chunk size 512, got %d", cfg.Chunking.Size)
	}

	if cfg.Chunking.Overlap != 50 {
		t.Errorf("expected chunk overlap 50, got %d", cfg.Chunking.Overlap)
	}

	if cfg.Watch.DebounceMs != 500 {
		t.Errorf("expected debounce 500ms, got %d", cfg.Watch.DebounceMs)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Embedder.ModelPath = "/path/to/e5-model"
	cfg.Embedder.Dimensions = 768
	cfg.Store.Backend = "postgres"

	err := cfg.Save(tmpDir)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Check file exists
	configPath := GetConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load config
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Embedder.ModelPath != "/path/to/e5-model" {
		t.Errorf("expected model_path /path/to/e5-model, got %s", loaded.Embedder.ModelPath)
	}

	if loaded.Embedder.Dimensions != 768 {
		t.Errorf("expected dimensions 768, got %d", loaded.Embedder.Dimensions)
	}

	if loaded.Store.Backend != "postgres" {
		t.Errorf("expected backend postgres, got %s", loaded.Store.Backend)
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if Exists(tmpDir) {
		t.Error("config should not exist initially")
	}

	// Create config
	cfg := DefaultConfig()
	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Should exist now
	if !Exists(tmpDir) {
		t.Error("config should exist after saving")
	}
}

func TestGetConfigDir(t *testing.T) {
	result := GetConfigDir("/test/path")
	expected := filepath.Join("/test/path", ConfigDir)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetConfigPath(t *testing.T) {
	result := GetConfigPath("/test/path")
	expected := filepath.Join("/test/path", ConfigDir, ConfigFileName)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetIndexPath(t *testing.T) {
	result := GetIndexPath("/test/path")
	expected := filepath.Join("/test/path", ConfigDir, IndexFileName)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

// TestBackwardCompatibility verifies that old configs without some fields
// still work correctly by applying sensible defaults.
func TestBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name               string
		configYAML         string
		expectedDimensions int
	}{
		{
			name: "missing dimensions defaults to 1024",
			configYAML: `version: 1
embedder:
  model_path: /path/to/model
store:
  backend: gob
`,
			expectedDimensions: 1024,
		},
		{
			name: "custom dimensions preserved",
			configYAML: `version: 1
embedder:
  model_path: /path/to/model
  dimensions: 768
store:
  backend: gob
`,
			expectedDimensions: 768,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ConfigDir)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatalf("failed to create config dir: %v", err)
			}

			configPath := filepath.Join(configDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			loaded, err := Load(tmpDir)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if loaded.Embedder.Dimensions != tt.expectedDimensions {
				t.Errorf("expected dimensions %d, got %d", tt.expectedDimensions, loaded.Embedder.Dimensions)
			}
		})
	}
}
