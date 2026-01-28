package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pprgva/code-memory/config"
	"github.com/spf13/cobra"
)

var (
	refreshJSON  bool
	refreshForce bool
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-index the current project",
	Long: `Re-index all files in the current project.

This command is a convenient shortcut that:
- Loads the existing configuration
- Re-scans all files
- Updates the index with any changes

By default, it uses incremental indexing (only changed files).
Use --force to perform a full re-index.

Examples:
  grepai refresh                # Incremental re-index
  grepai refresh --force        # Full re-index
  grepai refresh --json         # Output result as JSON`,
	RunE: runRefresh,
}

func init() {
	refreshCmd.Flags().BoolVar(&refreshJSON, "json", false, "Output result as JSON")
	refreshCmd.Flags().BoolVar(&refreshForce, "force", false, "Force full re-index (ignore modification times)")

	rootCmd.AddCommand(refreshCmd)
}

// RefreshResult contains the result of the refresh command.
type RefreshResult struct {
	Status        string `json:"status"`
	FilesIndexed  int    `json:"files_indexed"`
	ChunksCreated int    `json:"chunks_created"`
	FilesSkipped  int    `json:"files_skipped"`
	FilesRemoved  int    `json:"files_removed"`
	SymbolCount   int    `json:"symbol_count"`
	Duration      string `json:"duration"`
	Error         string `json:"error,omitempty"`
}

func runRefresh(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return outputRefreshError(fmt.Errorf("failed to get current directory: %w", err))
	}

	// Check if grepai is initialized
	if !config.Exists(cwd) {
		return outputRefreshError(fmt.Errorf("grepai not initialized in this directory (run 'grepai setup' first)"))
	}

	if !refreshJSON {
		if refreshForce {
			fmt.Println("Starting full re-index...")
		} else {
			fmt.Println("Starting incremental re-index...")
		}
	}

	startTime := time.Now()

	// Set watchForce for the indexation
	oldWatchForce := watchForce
	watchForce = refreshForce
	defer func() { watchForce = oldWatchForce }()

	// Run indexation
	filesIndexed, chunksCreated, symbolCount, err := runIndexation(cwd, refreshJSON)
	if err != nil {
		return outputRefreshError(err)
	}

	duration := time.Since(startTime)

	result := RefreshResult{
		Status:        "complete",
		FilesIndexed:  filesIndexed,
		ChunksCreated: chunksCreated,
		SymbolCount:   symbolCount,
		Duration:      duration.Round(time.Millisecond).String(),
	}

	if refreshJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("\nRefresh complete!\n")
		fmt.Printf("  Files indexed:  %d\n", filesIndexed)
		fmt.Printf("  Chunks created: %d\n", chunksCreated)
		fmt.Printf("  Symbols found:  %d\n", symbolCount)
		fmt.Printf("  Duration:       %s\n", duration.Round(time.Millisecond))
	}

	return nil
}

func outputRefreshError(err error) error {
	if refreshJSON {
		result := RefreshResult{
			Status: "error",
			Error:  err.Error(),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	return err
}
