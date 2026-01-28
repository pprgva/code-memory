package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDir           = ".grepai"
	ConfigFileName      = "config.yaml"
	IndexFileName       = "index.gob"
	SymbolIndexFileName = "symbols.gob"
)

type Config struct {
	Version           int            `yaml:"version"`
	Embedder          EmbedderConfig `yaml:"embedder"`
	Store             StoreConfig    `yaml:"store"`
	Chunking          ChunkingConfig `yaml:"chunking"`
	Watch             WatchConfig    `yaml:"watch"`
	Search            SearchConfig   `yaml:"search"`
	Trace             TraceConfig    `yaml:"trace"`
	Update            UpdateConfig   `yaml:"update"`
	Ignore            []string       `yaml:"ignore"`
	ExternalGitignore string         `yaml:"external_gitignore,omitempty"`
}

// UpdateConfig holds auto-update settings
type UpdateConfig struct {
	CheckOnStartup bool `yaml:"check_on_startup"` // Check for updates when running commands
}

type SearchConfig struct {
	Boost  BoostConfig  `yaml:"boost"`
	Hybrid HybridConfig `yaml:"hybrid"`
}

type HybridConfig struct {
	Enabled bool    `yaml:"enabled"`
	K       float32 `yaml:"k"` // RRF constant (default: 60)
}

type BoostConfig struct {
	Enabled   bool        `yaml:"enabled"`
	Penalties []BoostRule `yaml:"penalties"`
	Bonuses   []BoostRule `yaml:"bonuses"`
}

type BoostRule struct {
	Pattern string  `yaml:"pattern"`
	Factor  float32 `yaml:"factor"`
}

type EmbedderConfig struct {
	ModelPath  string `yaml:"model_path"`           // Path to E5 model directory
	Dimensions int    `yaml:"dimensions,omitempty"` // Auto-detected from worker, default 1024
}

type StoreConfig struct {
	Backend  string         `yaml:"backend"` // gob | postgres | qdrant
	Postgres PostgresConfig `yaml:"postgres,omitempty"`
	Qdrant   QdrantConfig   `yaml:"qdrant,omitempty"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type QdrantConfig struct {
	Endpoint   string `yaml:"endpoint"`             // e.g., "http://localhost" or "localhost"
	Port       int    `yaml:"port,omitempty"`       // e.g., 6333
	Collection string `yaml:"collection,omitempty"` // Optional, defaults from project path
	APIKey     string `yaml:"api_key,omitempty"`    // Optional, for Qdrant Cloud
	UseTLS     bool   `yaml:"use_tls,omitempty"`    // Enable TLS (for Qdrant Cloud)
}

type ChunkingConfig struct {
	Size    int `yaml:"size"`
	Overlap int `yaml:"overlap"`
}

type WatchConfig struct {
	DebounceMs    int       `yaml:"debounce_ms"`
	LastIndexTime time.Time `yaml:"last_index_time,omitempty"`
}

type TraceConfig struct {
	Mode             string   `yaml:"mode"`              // fast or precise
	EnabledLanguages []string `yaml:"enabled_languages"` // File extensions to index
	ExcludePatterns  []string `yaml:"exclude_patterns"`  // Patterns to exclude
}

func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Embedder: EmbedderConfig{
			ModelPath:  "",
			Dimensions: 1024,
		},
		Store: StoreConfig{
			Backend: "gob",
		},
		Chunking: ChunkingConfig{
			Size:    512,
			Overlap: 50,
		},
		Watch: WatchConfig{
			DebounceMs: 500,
		},
		Search: SearchConfig{
			Hybrid: HybridConfig{
				Enabled: false,
				K:       60,
			},
			Boost: BoostConfig{
				Enabled: true,
				Penalties: []BoostRule{
					// Test files (multi-language)
					{Pattern: "/tests/", Factor: 0.5},
					{Pattern: "/test/", Factor: 0.5},
					{Pattern: "__tests__", Factor: 0.5},
					{Pattern: "_test.", Factor: 0.5},
					{Pattern: ".test.", Factor: 0.5},
					{Pattern: ".spec.", Factor: 0.5},
					{Pattern: "test_", Factor: 0.5},
					// Mocks
					{Pattern: "/mocks/", Factor: 0.4},
					{Pattern: "/mock/", Factor: 0.4},
					{Pattern: ".mock.", Factor: 0.4},
					// Fixtures & test data
					{Pattern: "/fixtures/", Factor: 0.4},
					{Pattern: "/testdata/", Factor: 0.4},
					// Generated code
					{Pattern: "/generated/", Factor: 0.4},
					{Pattern: ".generated.", Factor: 0.4},
					{Pattern: ".gen.", Factor: 0.4},
					// Documentation
					{Pattern: ".md", Factor: 0.6},
					{Pattern: "/docs/", Factor: 0.6},
				},
				Bonuses: []BoostRule{
					// Entry points (multi-language)
					{Pattern: "/src/", Factor: 1.1},
					{Pattern: "/lib/", Factor: 1.1},
					{Pattern: "/app/", Factor: 1.1},
				},
			},
		},
		Trace: TraceConfig{
			Mode: "fast",
			EnabledLanguages: []string{
				".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".php",
				".c", ".h", ".cpp", ".hpp", ".cc", ".cxx",
				".rs", ".zig", ".cs", ".java",
				".pas", ".dpr", // Pascal/Delphi
			},
			ExcludePatterns: []string{
				"*_test.go",
				"*.spec.ts",
				"*.spec.js",
				"*.test.ts",
				"*.test.js",
				"__tests__/*",
			},
		},
		Update: UpdateConfig{
			CheckOnStartup: false, // Opt-in by default for privacy
		},
		Ignore: []string{
			// VCS
			".git",
			".svn",
			".hg",

			// Outil grepai
			".grepai",
			"qdrant_storage",

			// IDE / éditeurs
			".idea",
			".vscode",
			".vs",
			".eclipse",
			".settings",

			// JavaScript / TypeScript
			"node_modules",
			".next",
			".nuxt",
			".output",
			".svelte-kit",
			"bower_components",

			// Build / output génériques
			"build",
			"out",
			"dist",
			"bin",
			"obj",
			"coverage",
			".cache",

			// Python
			"__pycache__",
			".venv",
			"venv",
			".tox",
			".mypy_cache",
			".pytest_cache",
			".ruff_cache",
			"*.egg-info",

			// Go
			"vendor",

			// Rust
			"target",

			// Zig
			".zig-cache",
			"zig-out",

			// Java / Kotlin
			".gradle",
			".m2",

			// .NET / C#
			"packages",

			// PHP / Symfony / Laravel
			"var",

			// iOS / macOS
			"Pods",
			"DerivedData",
			".build",
			".swiftpm",

			// Dart / Flutter
			".dart_tool",
			".pub-cache",

			// Infra / DevOps
			".terraform",
			".vagrant",

			// Divers
			".DS_Store",
			"Thumbs.db",
			"tmp",
			"temp",
			"logs",
		},
	}
}

func GetConfigDir(projectRoot string) string {
	return filepath.Join(projectRoot, ConfigDir)
}

func GetConfigPath(projectRoot string) string {
	return filepath.Join(GetConfigDir(projectRoot), ConfigFileName)
}

func GetIndexPath(projectRoot string) string {
	return filepath.Join(GetConfigDir(projectRoot), IndexFileName)
}

func GetSymbolIndexPath(projectRoot string) string {
	return filepath.Join(GetConfigDir(projectRoot), SymbolIndexFileName)
}

// GetHomeDir retourne le dossier global grepai : ~/.grepai
func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".grepai")
}

// GetVenvDir retourne le chemin global du venv Python : ~/.grepai/venv
func GetVenvDir() string {
	return filepath.Join(GetHomeDir(), "venv")
}

func Load(projectRoot string) (*Config, error) {
	configPath := GetConfigPath(projectRoot)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing values (backward compatibility)
	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults fills in missing configuration values with sensible defaults.
// This ensures backward compatibility with older config files that may not
// have newer fields like dimensions or endpoint.
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	// Embedder defaults
	if c.Embedder.Dimensions == 0 {
		c.Embedder.Dimensions = 1024
	}

	// Chunking defaults
	if c.Chunking.Size == 0 {
		c.Chunking.Size = defaults.Chunking.Size
	}
	if c.Chunking.Overlap == 0 {
		c.Chunking.Overlap = defaults.Chunking.Overlap
	}

	// Watch defaults
	if c.Watch.DebounceMs == 0 {
		c.Watch.DebounceMs = defaults.Watch.DebounceMs
	}

	// Qdrant defaults
	if c.Store.Backend == "qdrant" && c.Store.Qdrant.Port <= 0 {
		c.Store.Qdrant.Port = 6334
	}
}

func (c *Config) Save(projectRoot string) error {
	configDir := GetConfigDir(projectRoot)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := GetConfigPath(projectRoot)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func Exists(projectRoot string) bool {
	configPath := GetConfigPath(projectRoot)
	_, err := os.Stat(configPath)
	return err == nil
}

func FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	dir := cwd
	for {
		if Exists(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no grepai project found (run 'grepai init' first)")
}
