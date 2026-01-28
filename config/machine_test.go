package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultMachineConfig(t *testing.T) {
	cfg := DefaultMachineConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	if cfg.Database.DSN != "" {
		t.Errorf("expected empty DSN, got %s", cfg.Database.DSN)
	}

	if cfg.Preferences.DefaultLimit != 10 {
		t.Errorf("expected default limit 10, got %d", cfg.Preferences.DefaultLimit)
	}

	if cfg.Preferences.JSONOutput != false {
		t.Error("expected JSON output to be false")
	}

	if cfg.Paths.Venv != "~/.grepai/venv" {
		t.Errorf("expected venv path ~/.grepai/venv, got %s", cfg.Paths.Venv)
	}

	if cfg.Paths.Cache != "~/.grepai/cache" {
		t.Errorf("expected cache path ~/.grepai/cache, got %s", cfg.Paths.Cache)
	}
}

func TestMachineConfigSaveAndLoad(t *testing.T) {
	// Create a temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := DefaultMachineConfig()
	cfg.Database.DSN = "postgres://user:pass@localhost:5432/testdb"
	cfg.Embedder.ModelPath = "/custom/model/path"
	cfg.Preferences.DefaultLimit = 20
	cfg.Preferences.JSONOutput = true

	// Save
	err := SaveMachineConfig(cfg)
	if err != nil {
		t.Fatalf("failed to save machine config: %v", err)
	}

	// Check file exists
	configPath := filepath.Join(tmpDir, ".grepai", "machine.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("machine config file was not created")
	}

	// Load
	loaded, err := LoadMachineConfig()
	if err != nil {
		t.Fatalf("failed to load machine config: %v", err)
	}

	if loaded.Database.DSN != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("expected DSN postgres://user:pass@localhost:5432/testdb, got %s", loaded.Database.DSN)
	}

	if loaded.Embedder.ModelPath != "/custom/model/path" {
		t.Errorf("expected model path /custom/model/path, got %s", loaded.Embedder.ModelPath)
	}

	if loaded.Preferences.DefaultLimit != 20 {
		t.Errorf("expected default limit 20, got %d", loaded.Preferences.DefaultLimit)
	}

	if !loaded.Preferences.JSONOutput {
		t.Error("expected JSON output to be true")
	}
}

func TestMachineConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Should not exist initially
	if MachineConfigExists() {
		t.Error("machine config should not exist initially")
	}

	// Create config
	cfg := DefaultMachineConfig()
	if err := SaveMachineConfig(cfg); err != nil {
		t.Fatalf("failed to save machine config: %v", err)
	}

	// Should exist now
	if !MachineConfigExists() {
		t.Error("machine config should exist after saving")
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/projects/myapp", filepath.Join(home, "projects", "myapp")},
		{"/other/path", "/other/path"},
		{"~", home},
		{"", ""},
		{"relative/path", "relative/path"},
	}

	for _, tc := range tests {
		result := ExpandPath(tc.input)
		if result != tc.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestContractPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(home, "projects", "myapp"), "~/projects/myapp"},
		{"/other/path", "/other/path"},
		{home, "~"},
		{"", ""},
	}

	for _, tc := range tests {
		result := ContractPath(tc.input)
		if result != tc.expected {
			t.Errorf("ContractPath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestMaskDSN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty DSN",
			input:    "",
			expected: "(not configured)",
		},
		{
			name:     "postgres URL with password",
			input:    "postgres://user:secret@localhost:5432/db",
			expected: "postgres://user:****@localhost:5432/db",
		},
		{
			name:     "postgresql URL with password",
			input:    "postgresql://admin:mypassword123@server:5432/production",
			expected: "postgresql://admin:****@server:5432/production",
		},
		{
			name:     "postgres URL without password",
			input:    "postgres://user@localhost:5432/db",
			expected: "postgres://user@localhost:5432/db",
		},
		{
			name:     "postgres URL without auth",
			input:    "postgres://localhost:5432/db",
			expected: "postgres://localhost:5432/db",
		},
		{
			name:     "key-value DSN with password",
			input:    "host=localhost port=5432 user=admin password=secret dbname=mydb",
			expected: "host=localhost port=5432 user=admin password=**** dbname=mydb",
		},
		{
			name:     "key-value DSN without password",
			input:    "host=localhost port=5432 user=admin dbname=mydb",
			expected: "host=localhost port=5432 user=admin dbname=mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskDSN(tt.input)
			if result != tt.expected {
				t.Errorf("MaskDSN(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetMachineConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	result := GetMachineConfigPath()
	expected := filepath.Join(home, ".grepai", "machine.yaml")

	if result != expected {
		t.Errorf("GetMachineConfigPath() = %q, want %q", result, expected)
	}
}

func TestLoadMachineConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	_, err := LoadMachineConfig()
	if err == nil {
		t.Error("expected error when loading non-existent config")
	}

	if !strings.Contains(err.Error(), "machine config not found") {
		t.Errorf("expected 'machine config not found' error, got: %v", err)
	}
}
