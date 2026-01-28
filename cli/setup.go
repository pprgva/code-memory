package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/detect"
	"github.com/pprgva/code-memory/embedder"
	"github.com/pprgva/code-memory/indexer"
	"github.com/pprgva/code-memory/trace"
	"github.com/spf13/cobra"
)

var (
	setupAuto    bool
	setupJSON    bool
	setupName    string
	setupBackend string
	setupNoIndex bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize project with auto-detection",
	Long: `Automatically detect project type and initialize grepai.

This command will:
- Detect programming languages and frameworks
- Create .grepai/config.yaml if needed
- Register the project in ~/.grepai/projects.yaml
- Index all files (unless --no-index is specified)

Use --auto for completely non-interactive setup (ideal for AI agents).

Examples:
  grepai setup                    # Interactive setup
  grepai setup --auto             # Full auto, no prompts
  grepai setup --auto --json      # Auto setup with JSON output
  grepai setup --name myproject   # Override detected project name`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().BoolVar(&setupAuto, "auto", false, "Full automatic setup, no prompts")
	setupCmd.Flags().BoolVar(&setupJSON, "json", false, "Output result as JSON")
	setupCmd.Flags().StringVar(&setupName, "name", "", "Override auto-detected project name")
	setupCmd.Flags().StringVarP(&setupBackend, "backend", "b", "", "Storage backend (gob, postgres)")
	setupCmd.Flags().BoolVar(&setupNoIndex, "no-index", false, "Skip indexation after setup")

	rootCmd.AddCommand(setupCmd)
}

// SetupResult contains the result of the setup command.
type SetupResult struct {
	Project       SetupProjectInfo `json:"project"`
	Status        string           `json:"status"`
	FilesIndexed  int              `json:"files_indexed"`
	ChunksCreated int              `json:"chunks_created"`
	SymbolCount   int              `json:"symbol_count"`
	Commands      SetupCommands    `json:"commands"`
	Error         string           `json:"error,omitempty"`
}

// SetupProjectInfo contains project information for JSON output.
type SetupProjectInfo struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Languages []string `json:"languages"`
	Framework string   `json:"framework,omitempty"`
	BuildTool string   `json:"build_tool,omitempty"`
}

// SetupCommands contains available commands for the project.
type SetupCommands struct {
	Search  string `json:"search"`
	Trace   string `json:"trace"`
	Refresh string `json:"refresh"`
	Watch   string `json:"watch"`
}

func runSetup(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return outputSetupError(fmt.Errorf("failed to get current directory: %w", err))
	}

	// Step 1: Detect project type
	projectInfo, err := detect.DetectProject(cwd)
	if err != nil {
		return outputSetupError(fmt.Errorf("failed to detect project: %w", err))
	}

	// Override name if specified
	if setupName != "" {
		projectInfo.Name = setupName
	}

	if !setupJSON && !setupAuto {
		fmt.Printf("Detected project: %s\n", projectInfo.Name)
		if len(projectInfo.Languages) > 0 {
			fmt.Printf("Languages: %s\n", strings.Join(projectInfo.Languages, ", "))
		}
		if projectInfo.Framework != "" {
			fmt.Printf("Framework: %s\n", projectInfo.Framework)
		}
		if projectInfo.BuildTool != "" {
			fmt.Printf("Build tool: %s\n", projectInfo.BuildTool)
		}
		fmt.Println()
	}

	// Step 2: Initialize grepai if needed
	if !config.Exists(cwd) {
		if !setupJSON && !setupAuto {
			fmt.Println("Initializing grepai configuration...")
		}

		cfg := config.DefaultConfig()

		// Set backend
		backend := setupBackend
		if backend == "" {
			// Check machine config for DSN
			if machineCfg, err := config.LoadMachineConfig(); err == nil && machineCfg.Database.DSN != "" {
				backend = "postgres"
				cfg.Store.Postgres.DSN = machineCfg.Database.DSN
			} else {
				backend = "gob"
			}
		}
		cfg.Store.Backend = backend

		// Set model path from machine config or default
		if machineCfg, err := config.LoadMachineConfig(); err == nil && machineCfg.Embedder.ModelPath != "" {
			cfg.Embedder.ModelPath = machineCfg.Embedder.ModelPath
		} else {
			cfg.Embedder.ModelPath = embedder.DefaultModelDir()
		}

		// Handle postgres backend
		if backend == "postgres" && cfg.Store.Postgres.DSN == "" {
			if machineCfg, err := config.LoadMachineConfig(); err == nil && machineCfg.Database.DSN != "" {
				cfg.Store.Postgres.DSN = machineCfg.Database.DSN
			} else if setupAuto {
				// Fall back to gob if no DSN and auto mode
				cfg.Store.Backend = "gob"
				backend = "gob"
			} else {
				return outputSetupError(fmt.Errorf("postgres backend requires DSN (configure with 'grepai machine setup' or use --backend gob)"))
			}
		}

		if err := cfg.Save(cwd); err != nil {
			return outputSetupError(fmt.Errorf("failed to save configuration: %w", err))
		}

		if !setupJSON && !setupAuto {
			fmt.Printf("Created configuration at %s\n", config.GetConfigPath(cwd))
			fmt.Printf("Backend: %s\n", backend)
		}

		// Ensure venv exists
		venvDir := config.GetVenvDir()
		if !embedder.VenvExists(venvDir) {
			if !setupJSON && !setupAuto {
				fmt.Println("\nCreating Python virtual environment...")
			}
			if err := embedder.SetupVenv(venvDir, "python3"); err != nil {
				return outputSetupError(fmt.Errorf("failed to create Python venv: %w", err))
			}
			if !setupJSON && !setupAuto {
				fmt.Println("Installing dependencies (torch, transformers)...")
			}
			if err := embedder.InstallDeps(venvDir); err != nil {
				return outputSetupError(fmt.Errorf("failed to install Python dependencies: %w", err))
			}
		}

		// Download model if needed
		if cfg.Embedder.ModelPath == "" || !embedder.ModelExists(cfg.Embedder.ModelPath) {
			modelDir := embedder.DefaultModelDir()
			if !embedder.ModelExists(modelDir) {
				if !setupJSON && !setupAuto {
					fmt.Printf("\nDownloading model %s...\n", embedder.DefaultModelName)
					fmt.Println("(this may take a few minutes on first run)")
				}
				if err := embedder.DownloadModel(venvDir, modelDir, embedder.DefaultModelName); err != nil {
					return outputSetupError(fmt.Errorf("failed to download model: %w", err))
				}
			}
			cfg.Embedder.ModelPath = modelDir
			if err := cfg.Save(cwd); err != nil {
				return outputSetupError(fmt.Errorf("failed to update config with model path: %w", err))
			}
		}

		// Add .grepai/ to .gitignore
		gitignorePath := filepath.Join(cwd, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			if err := indexer.AddToGitignore(cwd, ".grepai/"); err != nil && !setupJSON && !setupAuto {
				fmt.Printf("Warning: could not update .gitignore: %v\n", err)
			}
		}
	} else if !setupJSON && !setupAuto {
		fmt.Println("grepai already initialized in this directory.")
	}

	// Step 3: Register project in global registry
	projectsCfg, err := config.LoadProjectsConfig()
	if err != nil {
		return outputSetupError(fmt.Errorf("failed to load projects config: %w", err))
	}

	absPath, _ := filepath.Abs(cwd)

	// Check if already registered
	_, existingProject, _ := projectsCfg.FindProjectByPath(absPath)
	if existingProject == nil {
		if err := projectsCfg.AddProject(projectInfo.Name, absPath); err != nil {
			// Project name might exist with different path, try with suffix
			if strings.Contains(err.Error(), "already exists") {
				projectInfo.Name = projectInfo.Name + "-" + filepath.Base(filepath.Dir(absPath))
				if err := projectsCfg.AddProject(projectInfo.Name, absPath); err != nil {
					return outputSetupError(fmt.Errorf("failed to register project: %w", err))
				}
			} else {
				return outputSetupError(fmt.Errorf("failed to register project: %w", err))
			}
		}
	}

	// Update languages and framework
	if len(projectInfo.Languages) > 0 {
		_ = projectsCfg.UpdateProjectLanguages(projectInfo.Name, projectInfo.Languages)
	}
	if projectInfo.Framework != "" {
		_ = projectsCfg.UpdateProjectFramework(projectInfo.Name, projectInfo.Framework)
	}

	if err := config.SaveProjectsConfig(projectsCfg); err != nil {
		return outputSetupError(fmt.Errorf("failed to save projects config: %w", err))
	}

	if !setupJSON && !setupAuto {
		fmt.Printf("\nRegistered project %q in global registry\n", projectInfo.Name)
	}

	// Step 4: Index files
	var filesIndexed, chunksCreated, symbolCount int
	if !setupNoIndex {
		if !setupJSON && !setupAuto {
			fmt.Println("\nIndexing project files...")
		}
		filesIndexed, chunksCreated, symbolCount, err = runIndexation(cwd, setupJSON || setupAuto)
		if err != nil {
			return outputSetupError(fmt.Errorf("indexation failed: %w", err))
		}
	}

	// Output result
	result := SetupResult{
		Project: SetupProjectInfo{
			Name:      projectInfo.Name,
			Path:      config.ContractPath(absPath),
			Languages: projectInfo.Languages,
			Framework: projectInfo.Framework,
			BuildTool: projectInfo.BuildTool,
		},
		Status:        "ready",
		FilesIndexed:  filesIndexed,
		ChunksCreated: chunksCreated,
		SymbolCount:   symbolCount,
		Commands: SetupCommands{
			Search:  "grepai search \"query\" --json",
			Trace:   "grepai trace callers \"symbol\" --json",
			Refresh: "grepai refresh",
			Watch:   "grepai watch --background",
		},
	}

	if setupJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else if !setupAuto {
		fmt.Printf("\nSetup complete!\n")
		fmt.Printf("  Files indexed:  %d\n", filesIndexed)
		fmt.Printf("  Chunks created: %d\n", chunksCreated)
		fmt.Printf("  Symbols found:  %d\n", symbolCount)
		fmt.Println("\nNext steps:")
		fmt.Println("  - Search your code: grepai search \"your query\"")
		fmt.Println("  - Start watcher:    grepai watch --background")
		fmt.Println("  - Re-index:         grepai refresh")
	} else {
		// Auto mode, minimal output
		fmt.Printf("Project %q setup complete: %d files, %d chunks, %d symbols\n",
			projectInfo.Name, filesIndexed, chunksCreated, symbolCount)
	}

	return nil
}

func outputSetupError(err error) error {
	if setupJSON {
		result := SetupResult{
			Status: "error",
			Error:  err.Error(),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	return err
}

// runIndexation performs the actual indexation and returns stats.
func runIndexation(projectRoot string, silent bool) (filesIndexed, chunksCreated, symbolCount int, err error) {
	ctx := context.Background()

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to load config: %w", err)
	}

	emb, err := initializeEmbedder(cfg)
	if err != nil {
		return 0, 0, 0, err
	}
	defer emb.Close()

	st, err := initializeStore(ctx, cfg, projectRoot)
	if err != nil {
		return 0, 0, 0, err
	}
	defer st.Close()

	ignoreMatcher, err := indexer.NewIgnoreMatcher(projectRoot, cfg.Ignore, cfg.ExternalGitignore)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to initialize ignore matcher: %w", err)
	}

	scanner := indexer.NewScanner(projectRoot, ignoreMatcher)
	chunker := indexer.NewChunker(cfg.Chunking.Size, cfg.Chunking.Overlap)
	idx := indexer.NewIndexer(projectRoot, st, emb, chunker, scanner, time.Time{})

	var stats *indexer.IndexStats
	if silent {
		stats, err = idx.IndexAllWithBatchProgress(ctx, nil, nil)
	} else {
		stats, err = idx.IndexAllWithBatchProgress(ctx,
			func(info indexer.ProgressInfo) {
				printProgress(info.Current, info.Total, info.CurrentFile)
			},
			func(info indexer.BatchProgressInfo) {
				printBatchProgress(info)
			},
		)
		fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	}
	if err != nil {
		return 0, 0, 0, err
	}

	if err := st.Persist(ctx); err != nil {
		log.Printf("Warning: failed to persist index: %v", err)
	}

	// Build symbol index
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		log.Printf("Warning: failed to load symbol index: %v", err)
	}
	defer symbolStore.Close()

	extractor := trace.NewRegexExtractor()
	tracedLanguages := cfg.Trace.EnabledLanguages
	if len(tracedLanguages) == 0 {
		tracedLanguages = trace.DefaultTracedExtensions()
	}

	files, _, _ := scanner.Scan()
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Path))
		matched := false
		for _, lang := range tracedLanguages {
			if ext == lang {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		symbols, refs, extractErr := extractor.ExtractAll(ctx, file.Path, file.Content)
		if extractErr != nil {
			continue
		}
		if saveErr := symbolStore.SaveFile(ctx, file.Path, symbols, refs); saveErr != nil {
			continue
		}
		symbolCount += len(symbols)
	}

	if err := symbolStore.Persist(ctx); err != nil {
		log.Printf("Warning: failed to persist symbol index: %v", err)
	}

	cfg.Watch.LastIndexTime = time.Now()
	if err := cfg.Save(projectRoot); err != nil {
		log.Printf("Warning: failed to save config: %v", err)
	}

	return stats.FilesIndexed, stats.ChunksCreated, symbolCount, nil
}
