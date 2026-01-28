package search

import (
	"testing"
)

func TestNewFederatedSearcher(t *testing.T) {
	// Test that NewFederatedSearcher creates a valid searcher with a nil embedder
	// (nil embedder would panic on actual search, but constructor should work)
	fs := NewFederatedSearcher(nil)
	if fs == nil {
		t.Error("NewFederatedSearcher() returned nil")
	}
}

func TestFederatedResult_Fields(t *testing.T) {
	// Test that FederatedResult has all expected fields
	result := FederatedResult{
		ProjectName: "myproject",
		ProjectPath: "~/projects/myproject",
	}

	if result.ProjectName != "myproject" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "myproject")
	}

	if result.ProjectPath != "~/projects/myproject" {
		t.Errorf("ProjectPath = %q, want %q", result.ProjectPath, "~/projects/myproject")
	}
}
