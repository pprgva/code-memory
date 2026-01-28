package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/pprgva/code-memory/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Delete indexed data and remove .grepai/ from this project",
	Long: `Completely remove grepai from this project:

1. Delete all chunks and documents from the storage backend
2. Remove the .grepai/ configuration directory

After this, you will need to run 'grepai init' to use grepai again.`,
	RunE: runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 1. Purger les données du store
	ctx := context.Background()
	st, err := initializeStore(ctx, cfg, projectRoot)
	if err != nil {
		fmt.Printf("Warning: could not connect to store: %v\n", err)
	} else {
		docs, err := st.ListDocuments(ctx)
		if err != nil {
			fmt.Printf("Warning: could not list documents: %v\n", err)
		} else {
			total := len(docs)
			for i, doc := range docs {
				_ = st.DeleteByFile(ctx, doc)
				_ = st.DeleteDocument(ctx, doc)
				fmt.Printf("\rDeleting [%d/%d] %s", i+1, total, doc)
			}
			if total > 0 {
				fmt.Println()
			}
			if err := st.Persist(ctx); err != nil {
				fmt.Printf("Warning: could not persist: %v\n", err)
			}
			fmt.Printf("Deleted %d files from index.\n", total)
		}
		st.Close()
	}

	// 2. Supprimer le dossier .grepai/
	configDir := config.GetConfigDir(projectRoot)
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", configDir, err)
	}

	fmt.Printf("Removed %s\n", configDir)
	fmt.Println("grepai has been completely removed from this project.")
	return nil
}
