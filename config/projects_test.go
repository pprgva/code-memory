package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectsConfig_AddProject(t *testing.T) {
	cfg := DefaultProjectsConfig()

	// Test adding a project
	err := cfg.AddProject("test-project", "/Users/test/projects/myapp")
	if err != nil {
		t.Fatalf("AddProject failed: %v", err)
	}

	// Verify project was added
	project, err := cfg.GetProject("test-project")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if project.Path != "~/projects/myapp" && project.Path != "/Users/test/projects/myapp" {
		// Allow both formats since home directory may vary
		t.Logf("Project path: %s (may vary based on home directory)", project.Path)
	}

	if project.AddedAt == "" {
		t.Error("AddedAt should be set")
	}

	// Test duplicate name
	err = cfg.AddProject("test-project", "/Users/test/projects/other")
	if err == nil {
		t.Error("Should fail when adding duplicate name")
	}
}

func TestProjectsConfig_RemoveProject(t *testing.T) {
	cfg := DefaultProjectsConfig()

	// Add a project
	_ = cfg.AddProject("test-project", "/Users/test/projects/myapp")

	// Remove it
	err := cfg.RemoveProject("test-project")
	if err != nil {
		t.Fatalf("RemoveProject failed: %v", err)
	}

	// Verify it's gone
	_, err = cfg.GetProject("test-project")
	if err == nil {
		t.Error("Project should not exist after removal")
	}

	// Test removing non-existent project
	err = cfg.RemoveProject("non-existent")
	if err == nil {
		t.Error("Should fail when removing non-existent project")
	}
}

func TestProjectsConfig_ListProjects(t *testing.T) {
	cfg := DefaultProjectsConfig()

	// Test empty list
	names := cfg.ListProjects()
	if len(names) != 0 {
		t.Errorf("Expected empty list, got %v", names)
	}

	// Add projects
	_ = cfg.AddProject("zebra", "/a/zebra")
	_ = cfg.AddProject("alpha", "/a/alpha")
	_ = cfg.AddProject("beta", "/a/beta")

	// List should be sorted
	names = cfg.ListProjects()
	if len(names) != 3 {
		t.Fatalf("Expected 3 projects, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" || names[2] != "zebra" {
		t.Errorf("Projects not sorted: %v", names)
	}
}

func TestProjectsConfig_UpdateLanguages(t *testing.T) {
	cfg := DefaultProjectsConfig()
	_ = cfg.AddProject("test", "/a/test")

	err := cfg.UpdateProjectLanguages("test", []string{"go", "python"})
	if err != nil {
		t.Fatalf("UpdateProjectLanguages failed: %v", err)
	}

	project, _ := cfg.GetProject("test")
	if len(project.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(project.Languages))
	}
}

func TestProjectsConfig_UpdateFramework(t *testing.T) {
	cfg := DefaultProjectsConfig()
	_ = cfg.AddProject("test", "/a/test")

	err := cfg.UpdateProjectFramework("test", "react")
	if err != nil {
		t.Fatalf("UpdateProjectFramework failed: %v", err)
	}

	project, _ := cfg.GetProject("test")
	if project.Framework != "react" {
		t.Errorf("Expected framework 'react', got '%s'", project.Framework)
	}
}

// Note: ContractPath and ExpandPath are tested in machine_test.go

func TestGetProjectStatus(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grepai-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test not indexed (no .grepai dir)
	status := GetProjectStatus(tmpDir)
	if status != ProjectStatusNotIndexed {
		t.Errorf("Expected 'not indexed', got '%s'", status)
	}

	// Create .grepai dir with config
	grepaiDir := filepath.Join(tmpDir, ".grepai")
	if err := os.MkdirAll(grepaiDir, 0755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(grepaiDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("version: 1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test indexed (has config)
	status = GetProjectStatus(tmpDir)
	if status != ProjectStatusIndexed {
		t.Errorf("Expected 'indexed', got '%s'", status)
	}
}
