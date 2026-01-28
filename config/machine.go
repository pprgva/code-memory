package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	MachineConfigFileName = "machine.yaml"
)

// MachineConfig holds machine-level configuration that applies to all projects.
// This configuration is stored in ~/.grepai/machine.yaml
type MachineConfig struct {
	Version     int               `yaml:"version"`
	Database    DatabaseConfig    `yaml:"database"`
	Embedder    MachineEmbedder   `yaml:"embedder"`
	Preferences PreferencesConfig `yaml:"preferences"`
	Paths       PathsConfig       `yaml:"paths"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	DSN string `yaml:"dsn"` // PostgreSQL connection string
}

// MachineEmbedder holds machine-level embedder settings
type MachineEmbedder struct {
	ModelPath string `yaml:"model_path"` // Path to E5 model directory
}

// PreferencesConfig holds user preferences
type PreferencesConfig struct {
	DefaultLimit int  `yaml:"default_limit"` // Default number of search results
	JSONOutput   bool `yaml:"json_output"`   // Default output format
}

// PathsConfig holds system paths
type PathsConfig struct {
	Venv  string `yaml:"venv"`  // Path to Python venv
	Cache string `yaml:"cache"` // Path to cache directory
}

// DefaultMachineConfig returns a MachineConfig with sensible defaults
func DefaultMachineConfig() *MachineConfig {
	return &MachineConfig{
		Version: 1,
		Database: DatabaseConfig{
			DSN: "",
		},
		Embedder: MachineEmbedder{
			ModelPath: "", // Will be set during setup
		},
		Preferences: PreferencesConfig{
			DefaultLimit: 10,
			JSONOutput:   false,
		},
		Paths: PathsConfig{
			Venv:  "~/.grepai/venv",
			Cache: "~/.grepai/cache",
		},
	}
}

// GetMachineConfigPath returns the path to the machine configuration file
func GetMachineConfigPath() string {
	return filepath.Join(GetHomeDir(), MachineConfigFileName)
}

// LoadMachineConfig loads the machine configuration from ~/.grepai/machine.yaml
func LoadMachineConfig() (*MachineConfig, error) {
	configPath := GetMachineConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("machine config not found (run 'grepai machine setup' first)")
		}
		return nil, fmt.Errorf("failed to read machine config: %w", err)
	}

	var cfg MachineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse machine config: %w", err)
	}

	return &cfg, nil
}

// SaveMachineConfig saves the machine configuration to ~/.grepai/machine.yaml
func SaveMachineConfig(cfg *MachineConfig) error {
	homeDir := GetHomeDir()

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		return fmt.Errorf("failed to create grepai home directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal machine config: %w", err)
	}

	configPath := GetMachineConfigPath()
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write machine config: %w", err)
	}

	return nil
}

// MachineConfigExists returns true if the machine configuration file exists
func MachineConfigExists() bool {
	configPath := GetMachineConfigPath()
	_, err := os.Stat(configPath)
	return err == nil
}

// ContractPath replaces home directory with ~ for portability.
func ContractPath(absPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return absPath
	}

	if strings.HasPrefix(absPath, home) {
		return "~" + absPath[len(home):]
	}
	return absPath
}

// ExpandPath expands ~ to the actual home directory.
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return home + path[1:]
}

// MaskDSN masks the password in a PostgreSQL DSN for display
// Example: postgres://user:secret@localhost:5432/db -> postgres://user:****@localhost:5432/db
func MaskDSN(dsn string) string {
	if dsn == "" {
		return "(not configured)"
	}

	// Handle postgres:// or postgresql:// URLs
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// Find the @ symbol that separates auth from host
		atIndex := strings.Index(dsn, "@")
		if atIndex == -1 {
			return dsn // No auth part
		}

		// Find the scheme separator
		schemeEnd := strings.Index(dsn, "://")
		if schemeEnd == -1 {
			return dsn
		}

		authPart := dsn[schemeEnd+3 : atIndex]
		colonIndex := strings.Index(authPart, ":")
		if colonIndex == -1 {
			return dsn // No password
		}

		// Reconstruct with masked password
		user := authPart[:colonIndex]
		return dsn[:schemeEnd+3] + user + ":****" + dsn[atIndex:]
	}

	// Handle key=value format DSN
	if strings.Contains(dsn, "password=") {
		// Find password= and mask the value
		parts := strings.Split(dsn, " ")
		for i, part := range parts {
			if strings.HasPrefix(part, "password=") {
				parts[i] = "password=****"
			}
		}
		return strings.Join(parts, " ")
	}

	return dsn
}
