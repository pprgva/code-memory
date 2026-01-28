package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/pprgva/code-memory/config"
	"github.com/spf13/cobra"
)

var (
	projectAddName string
	projectInfoJSON bool
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage registered projects",
	Long: `Manage registered projects in the global grepai registry.

Projects are registered in ~/.grepai/projects.yaml for easy access
from any directory. Use this to quickly switch between projects.`,
}

var projectAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Register a project",
	Long: `Register a project in the global registry.

If no path is provided, uses the current directory.
The project name is derived from the directory name unless --name is specified.

Example:
  grepai project add                    # Register current directory
  grepai project add ~/projects/myapp   # Register specific path
  grepai project add . --name myapp     # Register with custom name`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectAdd,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	Long: `List all registered projects with their paths and status.

Status can be:
  - indexed:     Project has been indexed (.grepai/ exists)
  - watching:    Project watcher is running
  - not indexed: Project not yet initialized with grepai`,
	RunE: runProjectList,
}

var projectRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Unregister a project",
	Long: `Remove a project from the global registry.

This only removes the project from the registry.
It does NOT delete the .grepai/ directory or any index data.

Example:
  grepai project remove myapp`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectRemove,
}

var projectInfoCmd = &cobra.Command{
	Use:   "info [name]",
	Short: "Show project details",
	Long: `Show detailed information about a project.

If no name is provided and the current directory is a registered project,
shows info for that project.

Example:
  grepai project info myapp
  grepai project info --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectInfo,
}

func init() {
	projectAddCmd.Flags().StringVarP(&projectAddName, "name", "n", "", "Override auto-detected project name")
	projectInfoCmd.Flags().BoolVar(&projectInfoJSON, "json", false, "Output as JSON")

	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectInfoCmd)
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
	// Determine path
	var projectPath string
	if len(args) > 0 {
		projectPath = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectPath = cwd
	}

	// Make path absolute
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Determine project name
	projectName := projectAddName
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}

	// Load projects config
	cfg, err := config.LoadProjectsConfig()
	if err != nil {
		return fmt.Errorf("failed to load projects config: %w", err)
	}

	// Add project
	if err := cfg.AddProject(projectName, absPath); err != nil {
		return err
	}

	// Detect languages and framework
	languages := detectLanguages(absPath)
	if len(languages) > 0 {
		_ = cfg.UpdateProjectLanguages(projectName, languages)
	}

	framework := detectFramework(absPath)
	if framework != "" {
		_ = cfg.UpdateProjectFramework(projectName, framework)
	}

	// Save config
	if err := config.SaveProjectsConfig(cfg); err != nil {
		return fmt.Errorf("failed to save projects config: %w", err)
	}

	fmt.Printf("Registered project %q at %s\n", projectName, config.ContractPath(absPath))
	if len(languages) > 0 {
		fmt.Printf("  Languages: %s\n", strings.Join(languages, ", "))
	}
	if framework != "" {
		fmt.Printf("  Framework: %s\n", framework)
	}

	// Check if project is initialized
	status := config.GetProjectStatus(absPath)
	if status == config.ProjectStatusNotIndexed {
		fmt.Printf("\nTo index this project:\n")
		fmt.Printf("  cd %s && grepai init\n", absPath)
	}

	return nil
}

func runProjectList(cmd *cobra.Command, args []string) error {
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
	fmt.Fprintln(w, "NAME\tPATH\tSTATUS")

	for _, name := range names {
		project, _ := cfg.GetProject(name)
		absPath := config.ExpandPath(project.Path)
		status := config.GetProjectStatus(absPath)
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, project.Path, status)
	}

	w.Flush()
	return nil
}

func runProjectRemove(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	cfg, err := config.LoadProjectsConfig()
	if err != nil {
		return fmt.Errorf("failed to load projects config: %w", err)
	}

	// Get project info before removing (for display)
	project, err := cfg.GetProject(projectName)
	if err != nil {
		return err
	}

	if err := cfg.RemoveProject(projectName); err != nil {
		return err
	}

	if err := config.SaveProjectsConfig(cfg); err != nil {
		return fmt.Errorf("failed to save projects config: %w", err)
	}

	fmt.Printf("Unregistered project %q (%s)\n", projectName, project.Path)
	fmt.Println("Note: This does not delete the .grepai/ directory or index data.")
	return nil
}

func runProjectInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadProjectsConfig()
	if err != nil {
		return fmt.Errorf("failed to load projects config: %w", err)
	}

	// Determine which project to show
	var projectName string
	var project *config.RegisteredProject

	if len(args) > 0 {
		projectName = args[0]
		project, err = cfg.GetProject(projectName)
		if err != nil {
			return err
		}
	} else {
		// Try to find project by current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName, project, err = cfg.FindProjectByPath(cwd)
		if err != nil {
			return fmt.Errorf("current directory is not a registered project; specify a project name")
		}
	}

	absPath := config.ExpandPath(project.Path)
	status := config.GetProjectStatus(absPath)

	if projectInfoJSON {
		return outputProjectInfoJSON(projectName, project, absPath, status)
	}

	return outputProjectInfoText(projectName, project, absPath, status)
}

type projectInfoOutput struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	AbsPath   string   `json:"abs_path"`
	Languages []string `json:"languages,omitempty"`
	Framework string   `json:"framework,omitempty"`
	Status    string   `json:"status"`
	AddedAt   string   `json:"added_at,omitempty"`
	HasConfig bool     `json:"has_config"`
	HasIndex  bool     `json:"has_index"`
}

func outputProjectInfoJSON(name string, project *config.RegisteredProject, absPath string, status config.ProjectStatus) error {
	// Check for config and index files
	grepaiDir := filepath.Join(absPath, config.ConfigDir)
	hasConfig := false
	hasIndex := false

	if _, err := os.Stat(filepath.Join(grepaiDir, config.ConfigFileName)); err == nil {
		hasConfig = true
	}
	if _, err := os.Stat(filepath.Join(grepaiDir, config.IndexFileName)); err == nil {
		hasIndex = true
	}

	output := projectInfoOutput{
		Name:      name,
		Path:      project.Path,
		AbsPath:   absPath,
		Languages: project.Languages,
		Framework: project.Framework,
		Status:    string(status),
		AddedAt:   project.AddedAt,
		HasConfig: hasConfig,
		HasIndex:  hasIndex,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputProjectInfoText(name string, project *config.RegisteredProject, absPath string, status config.ProjectStatus) error {
	fmt.Printf("Project: %s\n\n", name)
	fmt.Printf("Path:      %s\n", project.Path)
	fmt.Printf("Absolute:  %s\n", absPath)
	fmt.Printf("Status:    %s\n", status)

	if len(project.Languages) > 0 {
		fmt.Printf("Languages: %s\n", strings.Join(project.Languages, ", "))
	}
	if project.Framework != "" {
		fmt.Printf("Framework: %s\n", project.Framework)
	}
	if project.AddedAt != "" {
		fmt.Printf("Added:     %s\n", project.AddedAt)
	}

	// Show grepai config info if exists
	grepaiDir := filepath.Join(absPath, config.ConfigDir)
	if _, err := os.Stat(grepaiDir); err == nil {
		fmt.Println("\ngrepai config:")

		// Try to load project config
		if projectCfg, err := config.Load(absPath); err == nil {
			fmt.Printf("  Backend:  %s\n", projectCfg.Store.Backend)
			fmt.Printf("  Chunking: size=%d, overlap=%d\n", projectCfg.Chunking.Size, projectCfg.Chunking.Overlap)
			if !projectCfg.Watch.LastIndexTime.IsZero() {
				fmt.Printf("  Last indexed: %s\n", projectCfg.Watch.LastIndexTime.Format("2006-01-02 15:04:05"))
			}
		}
	}

	return nil
}

// detectLanguages detects programming languages used in a project
func detectLanguages(absPath string) []string {
	extensions := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".jsx":   "javascript",
		".rs":    "rust",
		".java":  "java",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
	}

	found := make(map[string]bool)

	// Walk only top-level and one level deep
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check one level deep
			subPath := filepath.Join(absPath, entry.Name())
			subEntries, err := os.ReadDir(subPath)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() {
					ext := strings.ToLower(filepath.Ext(subEntry.Name()))
					if lang, ok := extensions[ext]; ok {
						found[lang] = true
					}
				}
			}
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if lang, ok := extensions[ext]; ok {
				found[lang] = true
			}
		}
	}

	languages := make([]string, 0, len(found))
	for lang := range found {
		languages = append(languages, lang)
	}
	return languages
}

// detectFramework detects the framework used in a project
func detectFramework(absPath string) string {
	// Check for common framework indicators
	indicators := map[string]string{
		"go.mod":           "go",      // Go module
		"package.json":     "",        // Need to check contents
		"requirements.txt": "python",
		"Cargo.toml":       "rust",
		"pom.xml":          "maven",
		"build.gradle":     "gradle",
		"Gemfile":          "ruby",
		"composer.json":    "php",
		"Package.swift":    "swift",
	}

	for file, framework := range indicators {
		if _, err := os.Stat(filepath.Join(absPath, file)); err == nil {
			if framework != "" {
				return framework
			}

			// Check package.json for specific frameworks
			if file == "package.json" {
				return detectJSFramework(filepath.Join(absPath, file))
			}
		}
	}

	// Check for Django
	if _, err := os.Stat(filepath.Join(absPath, "manage.py")); err == nil {
		return "django"
	}

	// Check for Rails
	if _, err := os.Stat(filepath.Join(absPath, "config", "routes.rb")); err == nil {
		return "rails"
	}

	return ""
}

// detectJSFramework detects JavaScript/TypeScript framework from package.json
func detectJSFramework(packageJSONPath string) string {
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return "node"
	}

	content := string(data)

	// Check for common frameworks
	frameworks := []struct {
		indicator string
		name      string
	}{
		{"next", "next.js"},
		{"react", "react"},
		{"vue", "vue"},
		{"angular", "angular"},
		{"svelte", "svelte"},
		{"nuxt", "nuxt"},
		{"express", "express"},
		{"fastify", "fastify"},
		{"nest", "nest.js"},
	}

	for _, f := range frameworks {
		if strings.Contains(content, fmt.Sprintf(`"%s"`, f.indicator)) {
			return f.name
		}
	}

	return "node"
}
