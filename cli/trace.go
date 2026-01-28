package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/pprgva/code-memory/config"
	"github.com/pprgva/code-memory/trace"
)

var (
	traceDepth int
	traceJSON  bool
)

var traceCmd = &cobra.Command{
	Use:   "trace <subcommand> <symbol>",
	Short: "Trace symbol callers and callees",
	Long: `Trace command helps you understand code dependencies by finding:
- callers: functions that call the specified symbol
- callees: functions that the specified symbol calls
- graph: full call graph visualization

Examples:
  grepai trace callers "Login"
  grepai trace callees "HandleRequest"
  grepai trace graph "ProcessOrder" --depth 3 --json`,
}

var traceCallersCmd = &cobra.Command{
	Use:   "callers <symbol>",
	Short: "Find all functions that call the specified symbol",
	Long: `Find all functions that call the specified symbol.

Examples:
  grepai trace callers "Login"
  grepai trace callers "HandleRequest" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTraceCallers,
}

var traceCalleesCmd = &cobra.Command{
	Use:   "callees <symbol>",
	Short: "Find all functions called by the specified symbol",
	Long: `Find all functions called by the specified symbol.

Examples:
  grepai trace callees "Login"
  grepai trace callees "HandleRequest" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTraceCallees,
}

var traceGraphCmd = &cobra.Command{
	Use:   "graph <symbol>",
	Short: "Build a call graph around the specified symbol",
	Long: `Build a call graph showing callers and callees around a symbol.

Examples:
  grepai trace graph "Login" --depth 2
  grepai trace graph "HandleRequest" --depth 3 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTraceGraph,
}

var traceImpactCmd = &cobra.Command{
	Use:   "impact <symbol>",
	Short: "Analyze blast radius of changing a symbol",
	Long: `Show all functions affected by modifying the target symbol.

Performs upward call graph traversal to find:
- Direct callers (depth 1)
- Transitive callers (depth 2+)
- All affected files

Examples:
  grepai trace impact "HandleLogin"
  grepai trace impact "UserService" --depth 3 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTraceImpact,
}

func init() {
	// Add flags to all trace subcommands
	for _, cmd := range []*cobra.Command{traceCallersCmd, traceCalleesCmd, traceGraphCmd, traceImpactCmd} {
		cmd.Flags().BoolVar(&traceJSON, "json", false, "Output results in JSON format")
	}
	traceGraphCmd.Flags().IntVarP(&traceDepth, "depth", "d", 2, "Maximum depth for graph traversal")
	traceImpactCmd.Flags().IntVarP(&traceDepth, "depth", "d", 3, "Maximum depth for impact analysis")

	traceCmd.AddCommand(traceCallersCmd)
	traceCmd.AddCommand(traceCalleesCmd)
	traceCmd.AddCommand(traceGraphCmd)
	traceCmd.AddCommand(traceImpactCmd)

	rootCmd.AddCommand(traceCmd)
}

func runTraceCallers(cmd *cobra.Command, args []string) error {
	symbolName := args[0]
	ctx := context.Background()

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Initialize symbol store
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return fmt.Errorf("failed to load symbol index: %w", err)
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return fmt.Errorf("symbol index is empty. Run 'grepai watch' first to build the index")
	}

	// Lookup symbol definition (peut ne pas exister si c'est une API externe)
	symbols, err := symbolStore.LookupSymbol(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("failed to lookup symbol: %w", err)
	}

	// Find callers — même si le symbole n'est pas défini localement
	refs, err := symbolStore.LookupCallers(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("failed to lookup callers: %w", err)
	}

	if len(symbols) == 0 && len(refs) == 0 {
		if traceJSON {
			return outputJSON(trace.TraceResult{Query: symbolName, Mode: "fast"})
		}
		fmt.Printf("No symbol or callers found for: %s\n", symbolName)
		return nil
	}

	result := trace.TraceResult{
		Query: symbolName,
		Mode:  "fast",
	}
	if len(symbols) > 0 {
		result.Symbol = &symbols[0]
	}

	// Convert refs to CallerInfo
	for _, ref := range refs {
		callerSyms, _ := symbolStore.LookupSymbol(ctx, ref.CallerName)
		var callerSym trace.Symbol
		if len(callerSyms) > 0 {
			callerSym = callerSyms[0]
		} else {
			callerSym = trace.Symbol{Name: ref.CallerName, File: ref.CallerFile, Line: ref.CallerLine}
		}
		result.Callers = append(result.Callers, trace.CallerInfo{
			Symbol: callerSym,
			CallSite: trace.CallSite{
				File:    ref.File,
				Line:    ref.Line,
				Context: ref.Context,
			},
		})
	}

	if traceJSON {
		return outputJSON(result)
	}

	return displayCallersResult(result)
}

func runTraceCallees(cmd *cobra.Command, args []string) error {
	symbolName := args[0]
	ctx := context.Background()

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return fmt.Errorf("failed to load symbol index: %w", err)
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return fmt.Errorf("symbol index is empty. Run 'grepai watch' first to build the index")
	}

	// Lookup symbol definition
	symbols, err := symbolStore.LookupSymbol(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("failed to lookup symbol: %w", err)
	}

	if len(symbols) == 0 {
		if traceJSON {
			return outputJSON(trace.TraceResult{Query: symbolName, Mode: "fast"})
		}
		fmt.Printf("No symbol found: %s\n", symbolName)
		return nil
	}

	// Find callees
	refs, err := symbolStore.LookupCallees(ctx, symbolName, symbols[0].File)
	if err != nil {
		return fmt.Errorf("failed to lookup callees: %w", err)
	}

	result := trace.TraceResult{
		Query:  symbolName,
		Mode:   "fast",
		Symbol: &symbols[0],
	}

	for _, ref := range refs {
		calleeSyms, _ := symbolStore.LookupSymbol(ctx, ref.SymbolName)
		var calleeSym trace.Symbol
		if len(calleeSyms) > 0 {
			calleeSym = calleeSyms[0]
		} else {
			calleeSym = trace.Symbol{Name: ref.SymbolName}
		}
		result.Callees = append(result.Callees, trace.CalleeInfo{
			Symbol: calleeSym,
			CallSite: trace.CallSite{
				File:    ref.File,
				Line:    ref.Line,
				Context: ref.Context,
			},
		})
	}

	if traceJSON {
		return outputJSON(result)
	}

	return displayCalleesResult(result)
}

func runTraceGraph(cmd *cobra.Command, args []string) error {
	symbolName := args[0]
	ctx := context.Background()

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return fmt.Errorf("failed to load symbol index: %w", err)
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return fmt.Errorf("symbol index is empty. Run 'grepai watch' first to build the index")
	}

	graph, err := symbolStore.GetCallGraph(ctx, symbolName, traceDepth)
	if err != nil {
		return fmt.Errorf("failed to build call graph: %w", err)
	}

	result := trace.TraceResult{
		Query: symbolName,
		Mode:  "fast",
		Graph: graph,
	}

	if traceJSON {
		return outputJSON(result)
	}

	return displayGraphResult(result)
}

func outputJSON(result trace.TraceResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func displayCallersResult(result trace.TraceResult) error {
	if result.Symbol != nil {
		fmt.Printf("Symbol: %s (%s)\n", result.Symbol.Name, result.Symbol.Kind)
		fmt.Printf("File: %s:%d\n", result.Symbol.File, result.Symbol.Line)
	} else {
		fmt.Printf("Symbol: %s (external/unresolved)\n", result.Query)
	}
	fmt.Printf("\nCallers (%d):\n", len(result.Callers))
	fmt.Println(strings.Repeat("-", 60))

	if len(result.Callers) == 0 {
		fmt.Println("No callers found.")
		return nil
	}

	for i, caller := range result.Callers {
		fmt.Printf("\n%d. %s\n", i+1, caller.Symbol.Name)
		if caller.Symbol.File != "" {
			fmt.Printf("   Defined: %s:%d\n", caller.Symbol.File, caller.Symbol.Line)
		}
		fmt.Printf("   Calls at: %s:%d\n", caller.CallSite.File, caller.CallSite.Line)
		if caller.CallSite.Context != "" {
			fmt.Printf("   Context: %s\n", truncate(caller.CallSite.Context, 80))
		}
	}

	return nil
}

func displayCalleesResult(result trace.TraceResult) error {
	fmt.Printf("Symbol: %s (%s)\n", result.Symbol.Name, result.Symbol.Kind)
	fmt.Printf("File: %s:%d\n", result.Symbol.File, result.Symbol.Line)
	fmt.Printf("\nCallees (%d):\n", len(result.Callees))
	fmt.Println(strings.Repeat("-", 60))

	if len(result.Callees) == 0 {
		fmt.Println("No callees found.")
		return nil
	}

	for i, callee := range result.Callees {
		fmt.Printf("\n%d. %s\n", i+1, callee.Symbol.Name)
		if callee.Symbol.File != "" {
			fmt.Printf("   Defined: %s:%d\n", callee.Symbol.File, callee.Symbol.Line)
		}
		fmt.Printf("   Called at: %s:%d\n", callee.CallSite.File, callee.CallSite.Line)
	}

	return nil
}

func displayGraphResult(result trace.TraceResult) error {
	fmt.Printf("Call Graph for: %s (depth: %d)\n", result.Query, result.Graph.Depth)
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\nNodes (%d):\n", len(result.Graph.Nodes))
	for name, sym := range result.Graph.Nodes {
		fmt.Printf("  - %s (%s) @ %s:%d\n", name, sym.Kind, sym.File, sym.Line)
	}

	fmt.Printf("\nEdges (%d):\n", len(result.Graph.Edges))
	for _, edge := range result.Graph.Edges {
		fmt.Printf("  %s -> %s [%s:%d]\n", edge.Caller, edge.Callee, edge.File, edge.Line)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func runTraceImpact(cmd *cobra.Command, args []string) error {
	symbolName := args[0]
	ctx := context.Background()

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		return fmt.Errorf("failed to load symbol index: %w", err)
	}
	defer symbolStore.Close()

	// Check if index exists
	stats, err := symbolStore.GetStats(ctx)
	if err != nil || stats.TotalSymbols == 0 {
		return fmt.Errorf("symbol index is empty. Run 'grepai watch' first to build the index")
	}

	// Build impact analysis using BFS upward
	type queueItem struct {
		name  string
		depth int
	}

	visited := make(map[string]bool)
	affectedFiles := make(map[string]bool)
	var impactedFuncs []trace.CallerInfo

	queue := []queueItem{{symbolName, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.name] || current.depth >= traceDepth {
			continue
		}
		visited[current.name] = true

		callers, _ := symbolStore.LookupCallers(ctx, current.name)
		for _, ref := range callers {
			callerSyms, _ := symbolStore.LookupSymbol(ctx, ref.CallerName)
			var callerSym trace.Symbol
			if len(callerSyms) > 0 {
				callerSym = callerSyms[0]
			} else {
				callerSym = trace.Symbol{Name: ref.CallerName, File: ref.CallerFile, Line: ref.CallerLine}
			}

			if !visited[ref.CallerName] {
				impactedFuncs = append(impactedFuncs, trace.CallerInfo{
					Symbol: callerSym,
					CallSite: trace.CallSite{
						File:    ref.File,
						Line:    ref.Line,
						Context: ref.Context,
					},
				})
				affectedFiles[callerSym.File] = true
				queue = append(queue, queueItem{ref.CallerName, current.depth + 1})
			}
		}
	}

	// Get target symbol
	symbols, _ := symbolStore.LookupSymbol(ctx, symbolName)

	if traceJSON {
		result := map[string]interface{}{
			"symbol":         symbolName,
			"depth":          traceDepth,
			"impacted_funcs": len(impactedFuncs),
			"affected_files": len(affectedFiles),
			"callers":        impactedFuncs,
		}
		if len(symbols) > 0 {
			result["definition"] = symbols[0]
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Text output
	fmt.Printf("Impact Analysis: %s (depth %d)\n", symbolName, traceDepth)
	fmt.Println(strings.Repeat("=", 50))

	if len(symbols) > 0 {
		fmt.Printf("\nTarget: %s (%s) @ %s:%d\n", symbols[0].Name, symbols[0].Kind, symbols[0].File, symbols[0].Line)
	}

	fmt.Printf("\nImpacted functions: %d\n", len(impactedFuncs))
	fmt.Printf("Affected files: %d\n\n", len(affectedFiles))

	for i, caller := range impactedFuncs {
		fmt.Printf("%d. %s\n", i+1, caller.Symbol.Name)
		if caller.Symbol.File != "" {
			fmt.Printf("   Defined: %s:%d\n", caller.Symbol.File, caller.Symbol.Line)
		}
		fmt.Printf("   Calls at: %s:%d\n", caller.CallSite.File, caller.CallSite.Line)
	}

	fmt.Printf("\n%s\n", strings.Repeat("-", 50))
	fmt.Printf("SUMMARY: Changing %s may affect %d functions in %d files\n",
		symbolName, len(impactedFuncs), len(affectedFiles))

	return nil
}
