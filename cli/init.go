package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/embedder"
	"github.com/pprgva/code-memory/indexer"
)

var (
	initModelPath      string
	initBackend        string
	initNonInteractive bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize grepai in the current directory",
	Long: `Initialize grepai by creating a .grepai directory with configuration.

This command will:
- Create .grepai/config.yaml with default settings
- Create a Python virtual environment in .grepai/venv/
- Install torch and transformers automatically
- Download the E5 model if no --model-path is provided
- Add .grepai/ to .gitignore if present`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initModelPath, "model-path", "", "Path to E5 model directory")
	initCmd.Flags().StringVarP(&initBackend, "backend", "b", "", "Storage backend (gob, postgres, or qdrant)")
	initCmd.Flags().BoolVar(&initNonInteractive, "yes", false, "Use defaults without prompting")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if config.Exists(cwd) {
		fmt.Println("grepai is already initialized in this directory.")
		fmt.Printf("Configuration: %s\n", config.GetConfigPath(cwd))
		return nil
	}

	cfg := config.DefaultConfig()

	if !initNonInteractive {
		reader := bufio.NewReader(os.Stdin)

		// Model path
		if initModelPath == "" {
			fmt.Print("\nPath to E5 model directory: ")
			input, _ := reader.ReadString('\n')
			initModelPath = strings.TrimSpace(input)
		}
		cfg.Embedder.ModelPath = initModelPath

		// Backend selection
		if initBackend == "" {
			fmt.Println("\nSelect storage backend:")
			fmt.Println("  1) gob (local file, recommended for most projects)")
			fmt.Println("  2) postgres (pgvector, for large monorepos or shared index)")
			fmt.Println("  3) qdrant (Docker-based vector database)")
			fmt.Print("Choice [1]: ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "2", "postgres":
				cfg.Store.Backend = "postgres"
				fmt.Print("PostgreSQL DSN: ")
				dsn, _ := reader.ReadString('\n')
				cfg.Store.Postgres.DSN = strings.TrimSpace(dsn)
			case "3", "qdrant":
				cfg.Store.Backend = "qdrant"
				fmt.Print("Qdrant endpoint [localhost]: ")
				endpoint, _ := reader.ReadString('\n')
				endpoint = strings.TrimSpace(endpoint)
				if endpoint == "" {
					endpoint = "localhost"
				}
				cfg.Store.Qdrant.Endpoint = endpoint

				fmt.Print("Qdrant port [6334]: ")
				port, _ := reader.ReadString('\n')
				port = strings.TrimSpace(port)
				if port == "" {
					cfg.Store.Qdrant.Port = 6334
				} else {
					var portInt int
					if _, err := fmt.Sscanf(port, "%d", &portInt); err != nil {
						return fmt.Errorf("invalid port number: %w", err)
					}
					cfg.Store.Qdrant.Port = portInt
				}
			default:
				cfg.Store.Backend = "gob"
			}
		} else {
			cfg.Store.Backend = initBackend
		}
	} else {
		cfg.Embedder.ModelPath = initModelPath
		if initBackend != "" {
			cfg.Store.Backend = initBackend
		}
	}

	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\nCreated configuration at %s\n", config.GetConfigPath(cwd))

	// Setup Python venv
	venvDir := config.GetVenvDir(cwd)
	if embedder.VenvExists(venvDir) {
		fmt.Println("Python venv already exists, skipping setup.")
	} else {
		fmt.Println("\nCreating Python virtual environment...")
		if err := embedder.SetupVenv(venvDir, "python3"); err != nil {
			return fmt.Errorf("failed to create Python venv: %w", err)
		}
		fmt.Println("Installing dependencies (torch, transformers)...")
		if err := embedder.InstallDeps(venvDir); err != nil {
			return fmt.Errorf("failed to install Python dependencies: %w", err)
		}
		fmt.Println("Python environment ready.")
	}

	// Download model if no path was provided
	if cfg.Embedder.ModelPath == "" {
		modelDir := embedder.DefaultModelDir()
		if embedder.ModelExists(modelDir) {
			fmt.Printf("Model already available at %s\n", modelDir)
		} else {
			fmt.Printf("\nDownloading model %s...\n", embedder.DefaultModelName)
			fmt.Println("(this may take a few minutes on first run)")
			if err := embedder.DownloadModel(venvDir, modelDir, embedder.DefaultModelName); err != nil {
				return fmt.Errorf("failed to download model: %w", err)
			}
			fmt.Println("Model downloaded successfully.")
		}
		cfg.Embedder.ModelPath = modelDir
		if err := cfg.Save(cwd); err != nil {
			return fmt.Errorf("failed to update configuration with model path: %w", err)
		}
	}

	// Add .grepai/ to .gitignore
	gitignorePath := cwd + "/.gitignore"
	if _, err := os.Stat(gitignorePath); err == nil {
		if err := indexer.AddToGitignore(cwd, ".grepai/"); err != nil {
			fmt.Printf("Warning: could not update .gitignore: %v\n", err)
		} else {
			fmt.Println("Added .grepai/ to .gitignore")
		}
	}

	fmt.Println("\ngrepai initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Start the indexing daemon: grepai watch")
	fmt.Println("  2. Search your code: grepai search \"your query\"")

	return nil
}
