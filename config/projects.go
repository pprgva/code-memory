package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ProjectsConfigFileName = "projects.yaml"
)

// ProjectsConfig holds global projects registry.
// Stored at ~/.grepai/projects.yaml
type ProjectsConfig struct {
	Version  int                        `yaml:"version"`
	Projects map[string]RegisteredProject `yaml:"projects"`
	Groups   map[string][]string        `yaml:"groups,omitempty"`
}

// RegisteredProject represents a registered project in the global registry.
type RegisteredProject struct {
	Path      string   `yaml:"path"`                // Uses ~ for portability
	Languages []string `yaml:"languages,omitempty"` // Detected languages (e.g., go, python, typescript)
	Framework string   `yaml:"framework,omitempty"` // Detected framework (e.g., react, django, gin)
	AddedAt   string   `yaml:"added_at,omitempty"`  // RFC3339 timestamp
}

// GetProjectsConfigPath returns the path to the projects config file.
// ~/.grepai/projects.yaml
func GetProjectsConfigPath() string {
	return filepath.Join(GetHomeDir(), ProjectsConfigFileName)
}

// LoadProjectsConfig loads the projects configuration from ~/.grepai/projects.yaml.
// Returns an empty config (not nil) if the file doesn't exist.
func LoadProjectsConfig() (*ProjectsConfig, error) {
	configPath := GetProjectsConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultProjectsConfig(), nil
		}
		return nil, fmt.Errorf("failed to read projects config: %w", err)
	}

	var cfg ProjectsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse projects config: %w", err)
	}

	// Ensure maps are initialized
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]RegisteredProject)
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}

	return &cfg, nil
}

// SaveProjectsConfig saves the projects configuration to ~/.grepai/projects.yaml.
func SaveProjectsConfig(cfg *ProjectsConfig) error {
	globalDir := GetHomeDir()

	// Ensure directory exists
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return fmt.Errorf("failed to create global config directory: %w", err)
	}

	configPath := GetProjectsConfigPath()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal projects config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write projects config: %w", err)
	}

	return nil
}

// DefaultProjectsConfig returns an empty projects configuration.
func DefaultProjectsConfig() *ProjectsConfig {
	return &ProjectsConfig{
		Version:  1,
		Projects: make(map[string]RegisteredProject),
		Groups:   make(map[string][]string),
	}
}

// AddProject adds a new project to the registry.
// The path is stored in portable format (with ~).
func (c *ProjectsConfig) AddProject(name, absPath string) error {
	if c.Projects == nil {
		c.Projects = make(map[string]RegisteredProject)
	}

	// Check if name already exists
	if _, exists := c.Projects[name]; exists {
		return fmt.Errorf("project %q already exists", name)
	}

	// Check if path already registered under another name
	portablePath := ContractPath(absPath)
	for existingName, p := range c.Projects {
		if p.Path == portablePath {
			return fmt.Errorf("path already registered as %q", existingName)
		}
	}

	c.Projects[name] = RegisteredProject{
		Path:    portablePath,
		AddedAt: time.Now().Format(time.RFC3339),
	}

	return nil
}

// RemoveProject removes a project from the registry.
func (c *ProjectsConfig) RemoveProject(name string) error {
	if c.Projects == nil {
		return fmt.Errorf("project %q not found", name)
	}

	if _, exists := c.Projects[name]; !exists {
		return fmt.Errorf("project %q not found", name)
	}

	delete(c.Projects, name)

	// Also remove from any groups
	for groupName, projects := range c.Groups {
		filtered := make([]string, 0, len(projects))
		for _, p := range projects {
			if p != name {
				filtered = append(filtered, p)
			}
		}
		c.Groups[groupName] = filtered
	}

	return nil
}

// GetProject returns a project by name.
func (c *ProjectsConfig) GetProject(name string) (*RegisteredProject, error) {
	if c.Projects == nil {
		return nil, fmt.Errorf("project %q not found", name)
	}

	project, exists := c.Projects[name]
	if !exists {
		return nil, fmt.Errorf("project %q not found", name)
	}

	return &project, nil
}

// ListProjects returns a sorted list of all project names.
func (c *ProjectsConfig) ListProjects() []string {
	if c.Projects == nil {
		return nil
	}

	names := make([]string, 0, len(c.Projects))
	for name := range c.Projects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FindProjectByPath finds a project by its path (accepts both portable and absolute).
func (c *ProjectsConfig) FindProjectByPath(path string) (string, *RegisteredProject, error) {
	if c.Projects == nil {
		return "", nil, fmt.Errorf("no projects registered")
	}

	// Normalize the search path
	portablePath := ContractPath(path)

	for name, project := range c.Projects {
		if project.Path == portablePath {
			return name, &project, nil
		}
	}

	return "", nil, fmt.Errorf("no project found at path %q", path)
}

// UpdateProjectLanguages updates the detected languages for a project.
func (c *ProjectsConfig) UpdateProjectLanguages(name string, languages []string) error {
	project, err := c.GetProject(name)
	if err != nil {
		return err
	}

	project.Languages = languages
	c.Projects[name] = *project
	return nil
}

// UpdateProjectFramework updates the detected framework for a project.
func (c *ProjectsConfig) UpdateProjectFramework(name string, framework string) error {
	project, err := c.GetProject(name)
	if err != nil {
		return err
	}

	project.Framework = framework
	c.Projects[name] = *project
	return nil
}

// Note: ContractPath and ExpandPath are defined in machine.go

// ProjectStatus represents the indexing status of a project.
type ProjectStatus string

const (
	ProjectStatusIndexed    ProjectStatus = "indexed"
	ProjectStatusWatching   ProjectStatus = "watching"
	ProjectStatusNotIndexed ProjectStatus = "not indexed"
)

// GetProjectStatus returns the status of a project based on its .grepai directory.
func GetProjectStatus(absPath string) ProjectStatus {
	// Check if .grepai directory exists
	grepaiDir := filepath.Join(absPath, ConfigDir)
	if _, err := os.Stat(grepaiDir); os.IsNotExist(err) {
		return ProjectStatusNotIndexed
	}

	// Check for PID file (watching)
	pidFile := filepath.Join(grepaiDir, "daemon.pid")
	if _, err := os.Stat(pidFile); err == nil {
		// Verify PID is still running
		pidData, err := os.ReadFile(pidFile)
		if err == nil {
			var pid int
			if _, err := fmt.Sscanf(string(pidData), "%d", &pid); err == nil {
				process, err := os.FindProcess(pid)
				if err == nil {
					// On Unix, FindProcess always succeeds, need to send signal 0 to check
					if err := process.Signal(os.Signal(nil)); err == nil {
						return ProjectStatusWatching
					}
				}
			}
		}
	}

	// Check for index file (indexed)
	indexFile := filepath.Join(grepaiDir, IndexFileName)
	if _, err := os.Stat(indexFile); err == nil {
		return ProjectStatusIndexed
	}

	// Also check if config exists (minimally initialized)
	configFile := filepath.Join(grepaiDir, ConfigFileName)
	if _, err := os.Stat(configFile); err == nil {
		return ProjectStatusIndexed
	}

	return ProjectStatusNotIndexed
}
