package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/pprgva/code-memory/config"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete all indexed data for this project from the store",
	Long: `Delete all chunks and documents for this project from the storage backend.

This does NOT remove the .grepai/ configuration directory.
Use 'grepai remove' to also delete the local configuration.`,
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()
	st, err := initializeStore(ctx, cfg, cwd)
	if err != nil {
		return fmt.Errorf("failed to connect to store: %w", err)
	}
	defer st.Close()

	docs, err := st.ListDocuments(ctx)
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	total := len(docs)
	if total == 0 {
		fmt.Println("No data to delete.")
		return nil
	}

	for i, doc := range docs {
		_ = st.DeleteByFile(ctx, doc)
		_ = st.DeleteDocument(ctx, doc)
		fmt.Printf("\rDeleting [%d/%d] %s", i+1, total, doc)
	}
	fmt.Println()

	if err := st.Persist(ctx); err != nil {
		return fmt.Errorf("failed to persist: %w", err)
	}

	fmt.Printf("Deleted %d files from index.\n", total)
	return nil
}
