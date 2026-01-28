package store

import (
	"testing"
	"time"
)

// TestProject_Struct verifies the Project struct fields.
func TestProject_Struct(t *testing.T) {
	now := time.Now()
	lastIndexed := now.Add(-time.Hour)

	p := &Project{
		ID:            "test-uuid",
		Name:          "my-project",
		LocalPath:     "/path/to/project",
		Languages:     []string{"go", "python"},
		Framework:     "gin",
		FileCount:     100,
		ChunkCount:    500,
		SymbolCount:   250,
		IndexStatus:   IndexStatusReady,
		LastError:     "",
		CreatedAt:     now,
		UpdatedAt:     now,
		LastIndexedAt: &lastIndexed,
	}

	if p.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %s", p.Name)
	}
	if p.IndexStatus != IndexStatusReady {
		t.Errorf("expected status 'ready', got %s", p.IndexStatus)
	}
	if len(p.Languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(p.Languages))
	}
	if p.LastIndexedAt == nil {
		t.Error("expected LastIndexedAt to be set")
	}
}

// TestFile_Struct verifies the File struct fields.
func TestFile_Struct(t *testing.T) {
	now := time.Now()

	f := &File{
		ID:           "file-uuid",
		ProjectID:    "project-uuid",
		RelativePath: "internal/server/handler.go",
		Language:     "go",
		ContentHash:  "abc123",
		LineCount:    150,
		ModTime:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if f.RelativePath != "internal/server/handler.go" {
		t.Errorf("expected path 'internal/server/handler.go', got %s", f.RelativePath)
	}
	if f.Language != "go" {
		t.Errorf("expected language 'go', got %s", f.Language)
	}
}

// TestIndexStatus_Constants verifies index status constants.
func TestIndexStatus_Constants(t *testing.T) {
	tests := []struct {
		status   IndexStatus
		expected string
	}{
		{IndexStatusPending, "pending"},
		{IndexStatusIndexing, "indexing"},
		{IndexStatusReady, "ready"},
		{IndexStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected status %s, got %s", tt.expected, tt.status)
		}
	}
}

// TestNullableString verifies nullable string helper functions.
func TestNullableString(t *testing.T) {
	// Test nullableString
	if result := nullableString(""); result != nil {
		t.Error("expected nil for empty string")
	}
	if result := nullableString("test"); result == nil || *result != "test" {
		t.Error("expected 'test' for non-empty string")
	}

	// Test derefString
	if result := derefString(nil); result != "" {
		t.Error("expected empty string for nil")
	}
	s := "test"
	if result := derefString(&s); result != "test" {
		t.Error("expected 'test' for non-nil pointer")
	}
}

// TestPostgresProjectStore_Interface verifies interface compliance.
func TestPostgresProjectStore_Interface(t *testing.T) {
	// Verify that PostgresProjectStore implements ProjectStore
	var _ ProjectStore = (*PostgresProjectStore)(nil)

	// Verify that PostgresProjectStore implements FileStore
	var _ FileStore = (*PostgresProjectStore)(nil)
}
