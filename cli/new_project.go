package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/embedder"
	"github.com/pprgva/code-memory/indexer"
	"github.com/spf13/cobra"
)

var (
	newProjectBackend string
)

var newProjectCmd = &cobra.Command{
	Use:   "new-project",
	Short: "Re-initialize config with latest defaults and force full re-index",
	Long: `Reset the project configuration to latest defaults and trigger a full re-index.

This command will:
- Recreate .grepai/config.yaml with the latest default settings
- Preserve your backend and DSN settings
- Reset last_index_time to force a full re-scan
- Start the watcher in force mode

Use this when your config is outdated or you want a clean re-index.`,
	RunE: runNewProject,
}

func init() {
	newProjectCmd.Flags().StringVarP(&newProjectBackend, "backend", "b", "", "Storage backend (gob, postgres, or qdrant)")
}

func runNewProject(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Charger l'ancienne config si elle existe (pour préserver backend/DSN)
	var oldCfg *config.Config
	if config.Exists(cwd) {
		oldCfg, err = config.Load(cwd)
		if err != nil {
			fmt.Printf("Warning: could not load existing config, starting fresh: %v\n", err)
		}
	}

	// Nouvelle config avec tous les derniers défauts
	cfg := config.DefaultConfig()

	// Préserver les settings de l'ancienne config
	if oldCfg != nil {
		cfg.Embedder.ModelPath = oldCfg.Embedder.ModelPath
		cfg.Embedder.Dimensions = oldCfg.Embedder.Dimensions
		cfg.Store = oldCfg.Store
	}

	// Override du backend si spécifié en flag
	if newProjectBackend != "" {
		cfg.Store.Backend = newProjectBackend
	}

	// Model path par défaut si vide
	if cfg.Embedder.ModelPath == "" {
		cfg.Embedder.ModelPath = embedder.DefaultModelDir()
	}

	// Sauvegarder la nouvelle config
	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Config reset with latest defaults at %s\n", config.GetConfigPath(cwd))
	fmt.Printf("Backend: %s\n", cfg.Store.Backend)

	// Purger les données existantes dans le store
	fmt.Println("Purging existing index data...")
	ctx := context.Background()
	st, err := initializeStore(ctx, cfg, cwd)
	if err != nil {
		fmt.Printf("Warning: could not connect to store for purge: %v\n", err)
	} else {
		docs, err := st.ListDocuments(ctx)
		if err != nil {
			fmt.Printf("Warning: could not list documents: %v\n", err)
		} else {
			total := len(docs)
			for i, doc := range docs {
				_ = st.DeleteByFile(ctx, doc)
				_ = st.DeleteDocument(ctx, doc)
				fmt.Printf("\rPurging [%d/%d] %s", i+1, total, doc)
			}
			if total > 0 {
				fmt.Println()
			}
			if err := st.Persist(ctx); err != nil {
				fmt.Printf("Warning: could not persist after purge: %v\n", err)
			}
			fmt.Printf("Purged %d files from index.\n", total)
		}
		st.Close()
	}

	fmt.Println("\nStarting full re-index...")

	// Réindexer sans rester en mode watch
	ctx2, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg2, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	emb, err := initializeEmbedder(cfg2)
	if err != nil {
		return err
	}
	defer emb.Close()

	st2, err := initializeStore(ctx2, cfg2, cwd)
	if err != nil {
		return err
	}
	defer st2.Close()

	ignoreMatcher, err := indexer.NewIgnoreMatcher(cwd, cfg2.Ignore, cfg2.ExternalGitignore)
	if err != nil {
		return fmt.Errorf("failed to initialize ignore matcher: %w", err)
	}

	scanner := indexer.NewScanner(cwd, ignoreMatcher)
	chunker := indexer.NewChunker(cfg2.Chunking.Size, cfg2.Chunking.Overlap)
	idx := indexer.NewIndexer(cwd, st2, emb, chunker, scanner, time.Time{})

	stats, err := idx.IndexAllWithBatchProgress(ctx2,
		func(info indexer.ProgressInfo) {
			printProgress(info.Current, info.Total, info.CurrentFile)
		},
		func(info indexer.BatchProgressInfo) {
			printBatchProgress(info)
		},
	)
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	if err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	cfg2.Watch.LastIndexTime = time.Now()
	if err := cfg2.Save(cwd); err != nil {
		fmt.Printf("Warning: failed to save config: %v\n", err)
	}

	if err := st2.Persist(ctx2); err != nil {
		fmt.Printf("Warning: failed to persist index: %v\n", err)
	}

	fmt.Printf("Done: %d files indexed, %d chunks created, %d skipped.\n",
		stats.FilesIndexed, stats.ChunksCreated, stats.FilesSkipped)
	return nil
}
