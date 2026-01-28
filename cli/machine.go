package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pprgva/code-memory/config"
	"github.com/spf13/cobra"
)

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Manage machine-level configuration",
	Long: `Manage machine-level configuration stored in ~/.grepai/machine.yaml.

Machine configuration includes settings that apply to all projects on this machine:
- Database connection (PostgreSQL DSN)
- Embedder model path
- User preferences (default limit, output format)
- System paths (venv, cache)`,
}

var machineSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive machine setup",
	Long: `Run an interactive setup to configure machine-level settings.

This command will prompt for:
- PostgreSQL DSN (optional, for shared index storage)
- Model path (optional, uses default if not provided)
- Default search limit
- Default output format`,
	RunE: runMachineSetup,
}

var machineShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display machine configuration",
	Long:  `Display the current machine configuration from ~/.grepai/machine.yaml.`,
	RunE:  runMachineShow,
}

func init() {
	machineCmd.AddCommand(machineSetupCmd)
	machineCmd.AddCommand(machineShowCmd)
}

func runMachineSetup(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Load existing config or create new one
	var cfg *config.MachineConfig
	if config.MachineConfigExists() {
		existing, err := config.LoadMachineConfig()
		if err != nil {
			fmt.Printf("Warning: could not load existing config: %v\n", err)
			cfg = config.DefaultMachineConfig()
		} else {
			cfg = existing
			fmt.Println("Updating existing machine configuration...")
		}
	} else {
		cfg = config.DefaultMachineConfig()
		fmt.Println("Creating new machine configuration...")
	}

	fmt.Println()

	// Database DSN
	fmt.Println("=== Database Configuration ===")
	fmt.Println("PostgreSQL DSN is used for shared vector storage across projects.")
	fmt.Println("Leave empty to use local GOB storage per project.")
	currentDSN := ""
	if cfg.Database.DSN != "" {
		currentDSN = config.MaskDSN(cfg.Database.DSN)
	}
	if currentDSN != "" {
		fmt.Printf("Current: %s\n", currentDSN)
	}
	fmt.Print("PostgreSQL DSN (or press Enter to skip): ")
	dsn, _ := reader.ReadString('\n')
	dsn = strings.TrimSpace(dsn)
	if dsn != "" {
		cfg.Database.DSN = dsn
	}

	fmt.Println()

	// Model path
	fmt.Println("=== Embedder Configuration ===")
	fmt.Println("Path to the E5 embedding model directory.")
	fmt.Println("Leave empty to use the default model path (~/.grepai/models/e5-base-v2).")
	if cfg.Embedder.ModelPath != "" {
		fmt.Printf("Current: %s\n", config.ContractPath(cfg.Embedder.ModelPath))
	}
	fmt.Print("Model path (or press Enter to skip): ")
	modelPath, _ := reader.ReadString('\n')
	modelPath = strings.TrimSpace(modelPath)
	if modelPath != "" {
		cfg.Embedder.ModelPath = config.ExpandPath(modelPath)
	}

	fmt.Println()

	// Preferences
	fmt.Println("=== Preferences ===")

	// Default limit
	fmt.Printf("Default search result limit [%d]: ", cfg.Preferences.DefaultLimit)
	limitStr, _ := reader.ReadString('\n')
	limitStr = strings.TrimSpace(limitStr)
	if limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 {
			cfg.Preferences.DefaultLimit = limit
		} else {
			fmt.Println("Invalid number, keeping current value.")
		}
	}

	// JSON output
	currentJSON := "no"
	if cfg.Preferences.JSONOutput {
		currentJSON = "yes"
	}
	fmt.Printf("Default to JSON output? (yes/no) [%s]: ", currentJSON)
	jsonStr, _ := reader.ReadString('\n')
	jsonStr = strings.TrimSpace(strings.ToLower(jsonStr))
	if jsonStr != "" {
		cfg.Preferences.JSONOutput = jsonStr == "yes" || jsonStr == "y"
	}

	fmt.Println()

	// Paths
	fmt.Println("=== Paths ===")

	// Venv path
	if cfg.Paths.Venv != "" {
		fmt.Printf("Python venv path [%s]: ", cfg.Paths.Venv)
	} else {
		fmt.Print("Python venv path [~/.grepai/venv]: ")
	}
	venvPath, _ := reader.ReadString('\n')
	venvPath = strings.TrimSpace(venvPath)
	if venvPath != "" {
		cfg.Paths.Venv = venvPath
	} else if cfg.Paths.Venv == "" {
		cfg.Paths.Venv = "~/.grepai/venv"
	}

	// Cache path
	if cfg.Paths.Cache != "" {
		fmt.Printf("Cache path [%s]: ", cfg.Paths.Cache)
	} else {
		fmt.Print("Cache path [~/.grepai/cache]: ")
	}
	cachePath, _ := reader.ReadString('\n')
	cachePath = strings.TrimSpace(cachePath)
	if cachePath != "" {
		cfg.Paths.Cache = cachePath
	} else if cfg.Paths.Cache == "" {
		cfg.Paths.Cache = "~/.grepai/cache"
	}

	fmt.Println()

	// Save configuration
	if err := config.SaveMachineConfig(cfg); err != nil {
		return fmt.Errorf("failed to save machine configuration: %w", err)
	}

	fmt.Printf("Machine configuration saved to %s\n", config.GetMachineConfigPath())

	return nil
}

func runMachineShow(cmd *cobra.Command, args []string) error {
	if !config.MachineConfigExists() {
		fmt.Println("No machine configuration found.")
		fmt.Println("Run 'grepai machine setup' to create one.")
		return nil
	}

	cfg, err := config.LoadMachineConfig()
	if err != nil {
		return fmt.Errorf("failed to load machine configuration: %w", err)
	}

	fmt.Printf("Machine Configuration (%s)\n", config.GetMachineConfigPath())
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	fmt.Println("Database:")
	fmt.Printf("  DSN: %s\n", config.MaskDSN(cfg.Database.DSN))
	fmt.Println()

	fmt.Println("Embedder:")
	modelPath := cfg.Embedder.ModelPath
	if modelPath == "" {
		modelPath = "(using default)"
	} else {
		modelPath = config.ContractPath(modelPath)
	}
	fmt.Printf("  Model path: %s\n", modelPath)
	fmt.Println()

	fmt.Println("Preferences:")
	fmt.Printf("  Default limit: %d\n", cfg.Preferences.DefaultLimit)
	fmt.Printf("  JSON output: %v\n", cfg.Preferences.JSONOutput)
	fmt.Println()

	fmt.Println("Paths:")
	fmt.Printf("  Venv: %s\n", cfg.Paths.Venv)
	fmt.Printf("  Cache: %s\n", cfg.Paths.Cache)

	return nil
}
