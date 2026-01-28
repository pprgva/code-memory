package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/embedder"
	"github.com/pprgva/code-memory/indexer"
	"github.com/pprgva/code-memory/store"
	"github.com/spf13/cobra"
)

var (
	upgradeAllParallel int
	upgradeAllDryRun   bool
	upgradeAllForce    bool
)

var statusAllCmd = &cobra.Command{
	Use:   "status-all",
	Short: "Show status of all registered projects",
	Long: `Show the status of all projects registered in the global grepai registry.

Output columns:
  PROJECT     - Project name
  PATH        - Path to project (using ~ for home directory)
  STATUS      - Current status: indexed, watching, or not indexed
  FILES       - Number of indexed files (- if not indexed)
  LAST UPDATE - Time since last index update

Example output:
  PROJECT       PATH                                      STATUS      FILES    LAST UPDATE
  whisperclip   ~/Documents/development/SuperwhisperaPaul indexed     42       2 min ago
  grepai        ~/Documents/development/code-memory       watching    156      just now
  backend       ~/Projects/backend-api                    not indexed -        -`,
	RunE: runStatusAll,
}

var upgradeAllCmd = &cobra.Command{
	Use:   "upgrade-all",
	Short: "Re-index all registered projects",
	Long: `Re-index all projects registered in the global grepai registry.

This command iterates through all registered projects and performs a full
re-indexation of each one. Projects that are not initialized (no .grepai/)
are skipped.

Flags:
  --parallel N   Number of concurrent jobs (default: 1)
  --dry-run      Show what would be done without making changes
  --force        Force full re-index (ignore last index time)

Example:
  grepai upgrade-all              # Re-index all projects sequentially
  grepai upgrade-all --parallel 2 # Re-index 2 projects at a time
  grepai upgrade-all --dry-run    # Preview which projects would be indexed`,
	RunE: runUpgradeAll,
}

func init() {
	upgradeAllCmd.Flags().IntVarP(&upgradeAllParallel, "parallel", "p", 1, "Number of concurrent jobs")
	upgradeAllCmd.Flags().BoolVar(&upgradeAllDryRun, "dry-run", false, "Show what would be done")
	upgradeAllCmd.Flags().BoolVar(&upgradeAllForce, "force", false, "Force full re-index (ignore last index time)")
}

func runStatusAll(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadProjectsConfig()
	if err != nil {
		return fmt.Errorf("failed to load projects config: %w", err)
	}

	names := cfg.ListProjects()
	if len(names) == 0 {
		fmt.Println("No projects registered.")
		fmt.Println("\nRegister one with: grepai project add [path]")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tPATH\tSTATUS\tFILES\tLAST UPDATE")

	for _, name := range names {
		project, _ := cfg.GetProject(name)
		absPath := config.ExpandPath(project.Path)
		status := config.GetProjectStatus(absPath)

		// Get file count and last update time
		fileCount := "-"
		lastUpdate := "-"

		if status != config.ProjectStatusNotIndexed {
			// Try to get stats from the store
			if projectCfg, err := config.Load(absPath); err == nil {
				// Get file count from index
				count, err := getIndexedFileCount(absPath, projectCfg)
				if err == nil && count > 0 {
					fileCount = fmt.Sprintf("%d", count)
				}

				// Get last update time
				if !projectCfg.Watch.LastIndexTime.IsZero() {
					lastUpdate = formatTimeAgo(projectCfg.Watch.LastIndexTime)
				}
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, project.Path, status, fileCount, lastUpdate)
	}

	w.Flush()
	return nil
}

func runUpgradeAll(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadProjectsConfig()
	if err != nil {
		return fmt.Errorf("failed to load projects config: %w", err)
	}

	names := cfg.ListProjects()
	if len(names) == 0 {
		fmt.Println("No projects registered.")
		fmt.Println("\nRegister one with: grepai project add [path]")
		return nil
	}

	// Filter to only projects that can be indexed
	type projectToUpgrade struct {
		name    string
		project *config.RegisteredProject
		absPath string
	}
	projectsToUpgrade := make([]projectToUpgrade, 0, len(names))

	for _, name := range names {
		project, _ := cfg.GetProject(name)
		absPath := config.ExpandPath(project.Path)

		// Check if project directory exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			if !upgradeAllDryRun {
				fmt.Printf("Skipping %s: directory not found (%s)\n", name, project.Path)
			}
			continue
		}

		// Check if project is initialized (has .grepai/config.yaml)
		if !config.Exists(absPath) {
			if !upgradeAllDryRun {
				fmt.Printf("Skipping %s: not initialized (run 'cd %s && grepai init')\n", name, absPath)
			}
			continue
		}

		projectsToUpgrade = append(projectsToUpgrade, struct {
			name    string
			project *config.RegisteredProject
			absPath string
		}{name, project, absPath})
	}

	if len(projectsToUpgrade) == 0 {
		fmt.Println("No projects to upgrade.")
		return nil
	}

	// Dry run mode
	if upgradeAllDryRun {
		fmt.Printf("Would upgrade %d projects:\n", len(projectsToUpgrade))
		for i, p := range projectsToUpgrade {
			fmt.Printf("  [%d/%d] %s (%s)\n", i+1, len(projectsToUpgrade), p.name, p.project.Path)
		}
		return nil
	}

	fmt.Printf("Upgrading %d projects...\n", len(projectsToUpgrade))

	// For now, sequential processing (parallel can be added later if needed)
	// The parallel flag is parsed but we process sequentially to avoid
	// overloading the embedder
	successCount := 0
	for i, p := range projectsToUpgrade {
		fmt.Printf("[%d/%d] %s ", i+1, len(projectsToUpgrade), p.name)

		fileCount, err := upgradeProject(p.absPath)
		if err != nil {
			fmt.Printf("failed: %v\n", err)
			continue
		}

		fmt.Printf("done (%d files)\n", fileCount)
		successCount++
	}

	fmt.Printf("Done! %d projects upgraded.\n", successCount)
	return nil
}

// upgradeProject performs a full re-index of a single project
func upgradeProject(projectRoot string) (int, error) {
	ctx := context.Background()

	// Load project configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return 0, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize embedder
	emb, err := initializeEmbedderForUpgrade(cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize embedder: %w", err)
	}
	defer emb.Close()

	// Initialize store
	st, err := initializeStore(ctx, cfg, projectRoot)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize store: %w", err)
	}
	defer st.Close()

	// Initialize ignore matcher
	ignoreMatcher, err := indexer.NewIgnoreMatcher(projectRoot, cfg.Ignore, cfg.ExternalGitignore)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize ignore matcher: %w", err)
	}

	// Initialize scanner
	scanner := indexer.NewScanner(projectRoot, ignoreMatcher)

	// Initialize chunker
	chunker := indexer.NewChunker(cfg.Chunking.Size, cfg.Chunking.Overlap)

	// Initialize indexer with optional force flag
	lastIndexTime := cfg.Watch.LastIndexTime
	if upgradeAllForce {
		lastIndexTime = time.Time{}
	}
	idx := indexer.NewIndexer(projectRoot, st, emb, chunker, scanner, lastIndexTime)

	// Run full index
	stats, err := idx.IndexAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("indexing failed: %w", err)
	}

	// Persist index
	if err := st.Persist(ctx); err != nil {
		return 0, fmt.Errorf("failed to persist index: %w", err)
	}

	// Update last index time
	cfg.Watch.LastIndexTime = time.Now()
	if err := cfg.Save(projectRoot); err != nil {
		// Non-fatal, just log
		fmt.Printf("warning: failed to save config: %v\n", err)
	}

	return stats.FilesIndexed, nil
}

// initializeEmbedderForUpgrade creates an embedder for the upgrade command
// It tries to reuse the socket embedder if available, otherwise creates a new E5 embedder
func initializeEmbedderForUpgrade(cfg *config.Config) (embedder.Embedder, error) {
	// Try socket embedder first (if watch is running)
	socketEmb, err := embedder.NewSocketEmbedder()
	if err == nil {
		return socketEmb, nil
	}
	// Fall back to E5 embedder
	return embedder.NewE5Embedder(cfg.Embedder.ModelPath, config.GetVenvDir())
}

// getIndexedFileCount returns the number of indexed files for a project
func getIndexedFileCount(projectRoot string, cfg *config.Config) (int, error) {
	ctx := context.Background()

	switch cfg.Store.Backend {
	case "gob":
		indexPath := config.GetIndexPath(projectRoot)
		// Check if index file exists
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			return 0, nil
		}
		gobStore := store.NewGOBStore(indexPath)
		if err := gobStore.Load(ctx); err != nil {
			return 0, err
		}
		defer gobStore.Close()

		stats, err := gobStore.GetStats(ctx)
		if err != nil {
			return 0, err
		}
		return stats.TotalFiles, nil

	case "postgres":
		st, err := store.NewPostgresStore(ctx, cfg.Store.Postgres.DSN, projectRoot, cfg.Embedder.Dimensions)
		if err != nil {
			return 0, err
		}
		defer st.Close()

		stats, err := st.GetStats(ctx)
		if err != nil {
			return 0, err
		}
		return stats.TotalFiles, nil

	case "qdrant":
		collectionName := cfg.Store.Qdrant.Collection
		if collectionName == "" {
			collectionName = store.SanitizeCollectionName(projectRoot)
		}
		st, err := store.NewQdrantStore(ctx, cfg.Store.Qdrant.Endpoint, cfg.Store.Qdrant.Port, cfg.Store.Qdrant.UseTLS, collectionName, cfg.Store.Qdrant.APIKey, cfg.Embedder.Dimensions)
		if err != nil {
			return 0, err
		}
		defer st.Close()

		stats, err := st.GetStats(ctx)
		if err != nil {
			return 0, err
		}
		return stats.TotalFiles, nil

	default:
		return 0, fmt.Errorf("unknown backend: %s", cfg.Store.Backend)
	}
}

// formatTimeAgo formats a time as a human-readable relative time
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", minutes)
	}

	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}

	// More than 30 days, show the date
	return t.Format("2006-01-02")
}

