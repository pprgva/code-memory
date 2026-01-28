package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pprgva/code-memory/config"
	"github.com/spf13/cobra"
)

// ClaudeInitResponse is returned by `grepai init` for Claude to understand what to do
type ClaudeInitResponse struct {
	Status       string            `json:"status"`                  // "ready", "needs_setup", "needs_machine_setup"
	Message      string            `json:"message"`                 // Human-readable message
	Project      *ClaudeProjectInfo `json:"project,omitempty"`      // Project info if ready
	NextStep     string            `json:"next_step,omitempty"`     // Command to run if not ready
	Commands     map[string]string `json:"commands,omitempty"`      // Available commands if ready
	Instructions string            `json:"instructions,omitempty"`  // Usage instructions
}

type ClaudeProjectInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Backend   string   `json:"backend"`
	Languages []string `json:"languages,omitempty"`
}

var initClaudeCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize grepai and return instructions for Claude",
	Long: `Check project status and return JSON instructions for Claude.

This command is designed for AI agents. It returns:
- Project status (ready, needs_setup, needs_machine_setup)
- Next step to run if not ready
- Available commands if ready
- Usage instructions

Example:
  grepai init   # Returns JSON with status and instructions`,
	RunE: runInitClaude,
}

func init() {
	rootCmd.AddCommand(initClaudeCmd)
}

func runInitClaude(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	response := ClaudeInitResponse{}

	// Check machine config first
	machineCfg, machineErr := config.LoadMachineConfig()
	hasMachineConfig := machineErr == nil && machineCfg.Database.DSN != ""

	if !hasMachineConfig {
		response.Status = "needs_machine_setup"
		response.Message = "Machine configuration not found. Ask the user to run 'grepai machine setup' first."
		response.NextStep = "grepai machine setup"
		response.Instructions = "The user needs to configure the database connection once per machine. This is a one-time setup."
		return outputInitJSON(response)
	}

	// Check if project is initialized
	if !config.Exists(cwd) {
		response.Status = "needs_setup"
		response.Message = "Project not initialized. Run setup to index this project."
		response.NextStep = "grepai setup --auto --json"
		response.Instructions = "Run the next_step command to initialize and index this project. It will auto-detect languages and create the index."
		return outputInitJSON(response)
	}

	// Project is ready
	cfg, err := config.Load(cwd)
	if err != nil {
		response.Status = "error"
		response.Message = fmt.Sprintf("Failed to load config: %v", err)
		return outputInitJSON(response)
	}

	response.Status = "ready"
	response.Message = "grepai is ready. Use the commands below to search and explore code."
	response.Project = &ClaudeProjectInfo{
		ID:      cfg.ProjectID,
		Name:    cfg.ProjectName,
		Path:    cwd,
		Backend: cfg.Store.Backend,
	}
	response.Commands = map[string]string{
		"search":  "grepai search \"your query\" --json --compact",
		"trace":   "grepai trace callers \"FunctionName\" --json",
		"refresh": "grepai refresh",
		"status":  "grepai doctor",
	}
	response.Instructions = `Use 'search' for semantic code search (describe what you're looking for in natural language).
Use 'trace' to find function callers/callees.
Always use --json --compact for search results.
Use English queries for best results.`

	return outputInitJSON(response)
}

func outputInitJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
