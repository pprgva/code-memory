package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Help content organized by topic
var helpTopics = map[string]string{
	"basic": `BASIC SEARCH
  Search your codebase using natural language:

  grepai search "authentication flow"
  grepai search "error handling middleware"
  grepai search "database connection pooling"

  Limit results:
  grepai search "api validation" -n 5
  grepai search "user login" --limit 20`,

	"filtering": `FILTERING RESULTS
  Filter by file type:
  grepai search "api handlers" --type go
  grepai search "react components" --type tsx
  grepai search "data models" --type py

  Filter by glob pattern:
  grepai search "tests" --glob "**/*_test.go"
  grepai search "api" --glob "src/api/**"

  Filter by time:
  grepai search "recent changes" --since main
  grepai search "bug fixes" --modified 7d`,

	"trace": `CALL GRAPH TRACING
  Find all callers of a function:
  grepai trace callers "HandleRequest"
  grepai trace callers "ValidateToken" --json

  Find all functions called by a function:
  grepai trace callees "ProcessOrder"
  grepai trace callees "Login" --json

  Build complete call graph:
  grepai trace graph "main" --depth 3
  grepai trace graph "HandleRequest" --depth 2 --json

  Analyze impact of changes:
  grepai trace impact "validateToken" --depth 3
  grepai trace impact "UserService" --json`,

	"workspace": `MULTI-PROJECT SEARCH
  Create a workspace:
  grepai workspace create mywork

  Add projects to workspace:
  grepai workspace add mywork /path/to/project1
  grepai workspace add mywork /path/to/project2

  Search across projects:
  grepai search "authentication" --workspace mywork
  grepai search "api" --workspace mywork --project project1

  List workspaces:
  grepai workspace list
  grepai workspace show mywork`,

	"llm": `LLM/AI AGENT INTEGRATION
  JSON output for parsing:
  grepai search "auth flow" --json
  grepai trace callers "Login" --json

  Compact output (saves ~80%% tokens):
  grepai search "authentication" --json --compact

  Configure AI agents:
  grepai agent-setup
  grepai agent-setup --with-subagent

  Best practices for AI agents:
  - Always use --json flag for structured output
  - Use --compact to reduce token usage
  - Use English queries for better semantic matching
  - Describe intent, not implementation`,

	"daemon": `DAEMON MODE
  Start the background indexer:
  grepai watch

  Run in background:
  grepai watch &

  Check status:
  grepai status

  Benefits of daemon mode:
  - Real-time index updates on file changes
  - ~10ms search latency (vs ~500ms cold start)
  - Persistent embedder connection`,

	"config": `CONFIGURATION
  Initialize a new project:
  grepai init
  grepai init --yes  # Auto-accept defaults

  Check installation health:
  grepai doctor
  grepai doctor --json

  Configuration file location:
  .grepai/config.yaml

  Key settings:
  - embedder.model_path: Path to E5 model
  - store.backend: gob, postgres, or qdrant
  - chunking.size: Chunk size in tokens`,
}

var troubleshootingContent = `COMMON ISSUES AND SOLUTIONS

1. "no .grepai/config.yaml found"
   Solution: Run 'grepai init' in your project root

2. "symbol index is empty"
   Solution: Run 'grepai watch' to build the index
   Note: First run may take a few minutes for large codebases

3. "failed to initialize embedder"
   Causes:
   - Python venv not set up: Run 'grepai init --yes'
   - Model not downloaded: Run 'grepai init --yes'
   - Ollama not running (if using Ollama): Start with 'ollama serve'

4. "search is slow (~500ms+)"
   Solution: Run 'grepai watch &' to keep the daemon running
   With daemon: ~10ms latency

5. "no results found"
   Tips:
   - Use English queries (model is English-trained)
   - Be more specific: "JWT token validation" vs "token"
   - Check if index exists: grepai status
   - Try different phrasing

6. "workspace search fails"
   Checklist:
   - Backend must be postgres or qdrant (not gob)
   - All project paths must exist
   - Run: grepai workspace status

7. "trace shows no callers"
   Causes:
   - Symbol might be an entry point (main, handlers)
   - External API calls won't have local callers
   - Index might need refresh: grepai watch

8. "grepai doctor shows failures"
   Run each fix in order:
   - Config: grepai init
   - Venv/Deps: grepai init --yes
   - Index: grepai watch

DIAGNOSTICS
  grepai doctor           # Full health check
  grepai doctor --json    # Machine-readable output
  grepai status           # Quick index status
`

// explainTopics maps keywords to explanations
var explainTopics = map[string]string{
	"search multiple projects": `To search across multiple projects:

1. Create a workspace (requires postgres or qdrant backend):
   grepai workspace create myworkspace

2. Add projects to the workspace:
   grepai workspace add myworkspace /path/to/project1
   grepai workspace add myworkspace /path/to/project2

3. Index the workspace:
   grepai watch --workspace myworkspace

4. Search across all projects:
   grepai search "query" --workspace myworkspace

5. Or search specific projects:
   grepai search "query" --workspace myworkspace --project project1`,

	"cross-project": `Cross-project search requires a workspace with a shared backend.

Setup:
  grepai workspace create work
  grepai workspace add work /path/to/repo1
  grepai workspace add work /path/to/repo2

Search:
  grepai search "authentication" --workspace work

Note: GOB backend doesn't support workspaces. Use postgres or qdrant.`,

	"json output": `Use --json flag for machine-readable output:

  grepai search "query" --json
  grepai trace callers "Func" --json

For AI agents, add --compact to reduce token usage:

  grepai search "query" --json --compact

Compact mode omits the content field, returning only:
  - file_path
  - start_line, end_line
  - score`,

	"compact": `The --compact flag reduces JSON output by ~80%% by omitting code content.

With content (default):
  {"file_path": "...", "start_line": 1, "score": 0.9, "content": "...code..."}

With --compact:
  {"file_path": "...", "start_line": 1, "score": 0.9}

Use --compact when:
  - Integrating with LLMs (saves tokens)
  - You'll read the file separately
  - You only need file locations`,

	"daemon": `The grepai daemon provides:
  - Real-time file watching and index updates
  - Fast search (~10ms vs ~500ms cold start)
  - Persistent embedder connection

Start the daemon:
  grepai watch &

Check if running:
  grepai status

The daemon automatically:
  - Indexes new/modified files
  - Removes deleted files from index
  - Keeps the embedder loaded in memory`,

	"trace callers": `Find all functions that call a symbol:

  grepai trace callers "HandleRequest"
  grepai trace callers "validateToken" --json

This helps you understand:
  - Who depends on this function
  - Impact of changing the function
  - Entry points that lead to this code`,

	"trace callees": `Find all functions called by a symbol:

  grepai trace callees "ProcessOrder"
  grepai trace callees "main" --json

This helps you understand:
  - What dependencies a function has
  - The call tree from a starting point
  - Code flow through the application`,

	"trace graph": `Build a complete call graph around a symbol:

  grepai trace graph "ValidateToken" --depth 2
  grepai trace graph "HandleLogin" --depth 3 --json

The graph shows:
  - All callers (up to depth levels)
  - All callees (down to depth levels)
  - Call sites with file and line numbers`,

	"impact analysis": `Analyze the blast radius of changing a function:

  grepai trace impact "validateToken" --depth 3

This shows:
  - All functions affected by the change
  - Files that may need testing
  - Transitive callers (callers of callers)

Useful before refactoring to understand scope.`,

	"filter type": `Filter search results by file type:

  grepai search "handlers" --type go
  grepai search "components" --type tsx
  grepai search "models" --type py

Common types: go, ts, tsx, js, jsx, py, rs, java, rb`,

	"filter glob": `Filter search results by glob pattern:

  grepai search "api" --glob "src/**"
  grepai search "tests" --glob "**/*_test.go"
  grepai search "models" --glob "internal/db/**"

Useful for limiting search to specific directories.`,

	"filter time": `Filter by file modification time:

By git ref:
  grepai search "changes" --since main
  grepai search "recent" --since HEAD~10

By duration:
  grepai search "fixes" --modified 7d
  grepai search "today" --modified 24h`,

	"setup": `Set up grepai for a new project:

1. Initialize:
   grepai init --yes

2. Build the index:
   grepai watch

3. (Optional) Configure AI agents:
   grepai agent-setup

4. Search:
   grepai search "your query"`,

	"best practices": `Best practices for grepai:

1. Use English queries (embedding model is English-trained)

2. Describe intent, not implementation:
   Good: "handles user authentication"
   Bad:  "func Login"

3. Be specific:
   Good: "JWT token validation"
   Bad:  "token"

4. Run daemon for fast searches:
   grepai watch &

5. Use --json --compact for AI integrations

6. Run 'grepai doctor' if things aren't working`,
}

var explainJSON bool

var examplesCmd = &cobra.Command{
	Use:   "examples [topic]",
	Short: "Show usage examples",
	Long: `Show practical usage examples for grepai commands.

Available topics:
  basic       - Basic search examples
  filtering   - Filtering results by type, glob, time
  trace       - Call graph tracing examples
  workspace   - Multi-project search setup
  llm         - LLM/AI agent integration
  daemon      - Background daemon usage
  config      - Configuration and setup

Run without arguments to see all examples.`,
	ValidArgs: []string{"basic", "filtering", "trace", "workspace", "llm", "daemon", "config"},
	RunE:      runExamples,
}

var troubleshootingCmd = &cobra.Command{
	Use:     "troubleshooting",
	Aliases: []string{"troubleshoot", "debug-help"},
	Short:   "Common issues and solutions",
	Long:    "Display common issues and their solutions for grepai.",
	RunE:    runTroubleshooting,
}

var explainCmd = &cobra.Command{
	Use:   "explain <question>",
	Short: "Answer questions about grepai usage",
	Long: `Answer questions about grepai usage and features.

Examples:
  grepai explain "how to search multiple projects"
  grepai explain "json output"
  grepai explain "trace callers"
  grepai explain "best practices"

Use --json for machine-readable output.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runExplain,
}

func init() {
	explainCmd.Flags().BoolVar(&explainJSON, "json", false, "Output in JSON format")

	rootCmd.AddCommand(examplesCmd)
	rootCmd.AddCommand(troubleshootingCmd)
	rootCmd.AddCommand(explainCmd)
}

func runExamples(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show all topics
		fmt.Println("grepai examples")
		fmt.Println("===============")
		fmt.Println()

		// Define order for consistent output
		order := []string{"basic", "filtering", "trace", "workspace", "llm", "daemon", "config"}
		for _, topic := range order {
			if content, ok := helpTopics[topic]; ok {
				fmt.Println(content)
				fmt.Println()
			}
		}
		return nil
	}

	topic := strings.ToLower(args[0])
	content, ok := helpTopics[topic]
	if !ok {
		fmt.Printf("Unknown topic: %s\n\n", topic)
		fmt.Println("Available topics: basic, filtering, trace, workspace, llm, daemon, config")
		return nil
	}

	fmt.Println(content)
	return nil
}

func runTroubleshooting(cmd *cobra.Command, args []string) error {
	fmt.Println(troubleshootingContent)
	return nil
}

func runExplain(cmd *cobra.Command, args []string) error {
	question := strings.ToLower(strings.Join(args, " "))

	// Find the best matching topic
	var bestMatch string
	var bestScore int

	for key := range explainTopics {
		score := matchScore(question, key)
		if score > bestScore {
			bestScore = score
			bestMatch = key
		}
	}

	if bestScore == 0 {
		// No match found, provide suggestions
		if explainJSON {
			return outputExplainJSON("", "No matching topic found. Try: "+suggestTopics())
		}
		fmt.Println("No matching topic found.")
		fmt.Println()
		fmt.Println("Try one of these queries:")
		fmt.Println("  grepai explain \"search multiple projects\"")
		fmt.Println("  grepai explain \"json output\"")
		fmt.Println("  grepai explain \"trace callers\"")
		fmt.Println("  grepai explain \"daemon\"")
		fmt.Println("  grepai explain \"best practices\"")
		fmt.Println("  grepai explain \"setup\"")
		return nil
	}

	answer := explainTopics[bestMatch]

	if explainJSON {
		return outputExplainJSON(bestMatch, answer)
	}

	fmt.Println(answer)
	return nil
}

// matchScore returns a score for how well the question matches the topic
func matchScore(question, topic string) int {
	question = strings.ToLower(question)
	topic = strings.ToLower(topic)

	// Exact match
	if question == topic {
		return 100
	}

	// Topic contained in question
	if strings.Contains(question, topic) {
		return 80
	}

	// Word-by-word matching
	questionWords := strings.Fields(question)
	topicWords := strings.Fields(topic)

	score := 0
	for _, qw := range questionWords {
		for _, tw := range topicWords {
			if qw == tw {
				score += 20
			} else if strings.Contains(qw, tw) || strings.Contains(tw, qw) {
				score += 10
			}
		}
	}

	// Keyword boosts
	keywordBoosts := map[string][]string{
		"search multiple projects": {"multiple", "projects", "cross", "workspace"},
		"cross-project":            {"cross", "project", "multiple", "workspace"},
		"json output":              {"json", "output", "format", "api"},
		"compact":                  {"compact", "token", "llm", "ai"},
		"daemon":                   {"daemon", "watch", "background", "fast", "slow"},
		"trace callers":            {"trace", "callers", "who", "calls", "call"},
		"trace callees":            {"trace", "callees", "called", "dependencies"},
		"trace graph":              {"trace", "graph", "visualization", "call"},
		"impact analysis":          {"impact", "blast", "radius", "change", "refactor"},
		"filter type":              {"filter", "type", "extension", "language"},
		"filter glob":              {"filter", "glob", "pattern", "directory"},
		"filter time":              {"filter", "time", "since", "modified", "recent"},
		"setup":                    {"setup", "install", "init", "start", "begin", "new"},
		"best practices":           {"best", "practice", "tips", "advice", "how"},
	}

	for key, keywords := range keywordBoosts {
		if key == topic {
			for _, kw := range keywords {
				if strings.Contains(question, kw) {
					score += 15
				}
			}
		}
	}

	return score
}

func suggestTopics() string {
	topics := []string{
		"search multiple projects",
		"json output",
		"trace callers",
		"daemon",
		"best practices",
	}
	return strings.Join(topics, ", ")
}

func outputExplainJSON(topic, answer string) error {
	result := map[string]string{
		"topic":  topic,
		"answer": answer,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
