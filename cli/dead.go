package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/trace"
)

var (
	deadType     string
	deadExported bool
	deadJSON     bool
)

var deadCmd = &cobra.Command{
	Use:   "dead",
	Short: "Find potentially unused functions (dead code)",
	Long: `Analyze the codebase to find functions that are never called.

Note: This may include false positives like:
- Interface implementations
- Reflection-based calls
- Entry points (HTTP handlers, etc.)
- Exported functions called by external packages`,
	RunE: runDead,
}

func init() {
	deadCmd.Flags().StringVar(&deadType, "type", "", "Filter by language (go, ts, py, etc.)")
	deadCmd.Flags().BoolVar(&deadExported, "exported", false, "Only show exported/public functions")
	deadCmd.Flags().BoolVar(&deadJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(deadCmd)
}

func runDead(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	store := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := store.Load(ctx); err != nil {
		return fmt.Errorf("failed to load symbol index: %w", err)
	}
	defer store.Close()

	symbols, err := store.GetAllSymbols(ctx)
	if err != nil {
		return fmt.Errorf("failed to get symbols: %w", err)
	}

	var deadFuncs []trace.Symbol
	for _, sym := range symbols {
		// Only functions and methods
		if sym.Kind != trace.KindFunction && sym.Kind != trace.KindMethod {
			continue
		}
		// Apply filters
		if deadType != "" && sym.Language != deadType {
			continue
		}
		if deadExported && !sym.Exported {
			continue
		}
		// Skip special functions
		if isSpecialFunction(sym.Name, sym.Language) {
			continue
		}
		// Check for callers
		callers, _ := store.LookupCallers(ctx, sym.Name)
		if len(callers) == 0 {
			deadFuncs = append(deadFuncs, sym)
		}
	}

	// Sort by file then line
	sort.Slice(deadFuncs, func(i, j int) bool {
		if deadFuncs[i].File != deadFuncs[j].File {
			return deadFuncs[i].File < deadFuncs[j].File
		}
		return deadFuncs[i].Line < deadFuncs[j].Line
	})

	if deadJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"total_scanned": len(symbols),
			"dead_count":    len(deadFuncs),
			"results":       deadFuncs,
		})
	}

	fmt.Printf("Dead Code Analysis\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Scanned: %d symbols\n", len(symbols))
	fmt.Printf("Dead functions: %d\n\n", len(deadFuncs))

	for i, sym := range deadFuncs {
		exported := ""
		if sym.Exported {
			exported = " (exported)"
		}
		fmt.Printf("%d. %s%s\n", i+1, sym.Name, exported)
		fmt.Printf("   %s:%d\n", sym.File, sym.Line)
	}

	return nil
}

func isSpecialFunction(name, lang string) bool {
	// Always exclude
	if name == "main" || name == "init" {
		return true
	}
	// Test functions
	switch lang {
	case "go":
		if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Example") {
			return true
		}
	case "python":
		if strings.HasPrefix(name, "test_") || name == "setUp" || name == "tearDown" {
			return true
		}
	case "javascript", "typescript":
		if name == "describe" || name == "it" || name == "test" || name == "beforeEach" || name == "afterEach" {
			return true
		}
	}
	return false
}
