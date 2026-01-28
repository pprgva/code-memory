package search

import (
	"testing"
	"time"

	"github.com/pprgva/code-memory/store"
)

func TestFilterByType(t *testing.T) {
	results := []store.SearchResult{
		{Chunk: store.Chunk{FilePath: "main.go"}, Score: 0.9},
		{Chunk: store.Chunk{FilePath: "test.ts"}, Score: 0.8},
		{Chunk: store.Chunk{FilePath: "component.vue"}, Score: 0.7},
		{Chunk: store.Chunk{FilePath: "utils.py"}, Score: 0.6},
	}

	t.Run("filter by single type", func(t *testing.T) {
		filtered := FilterByType(results, []string{"go"})
		if len(filtered) != 1 {
			t.Errorf("expected 1 result, got %d", len(filtered))
		}
		if filtered[0].Chunk.FilePath != "main.go" {
			t.Errorf("expected main.go, got %s", filtered[0].Chunk.FilePath)
		}
	})

	t.Run("filter by multiple types", func(t *testing.T) {
		filtered := FilterByType(results, []string{"ts", "vue"})
		if len(filtered) != 2 {
			t.Errorf("expected 2 results, got %d", len(filtered))
		}
	})

	t.Run("filter with dot prefix", func(t *testing.T) {
		filtered := FilterByType(results, []string{".py"})
		if len(filtered) != 1 {
			t.Errorf("expected 1 result, got %d", len(filtered))
		}
		if filtered[0].Chunk.FilePath != "utils.py" {
			t.Errorf("expected utils.py, got %s", filtered[0].Chunk.FilePath)
		}
	})

	t.Run("no types specified", func(t *testing.T) {
		filtered := FilterByType(results, []string{})
		if len(filtered) != len(results) {
			t.Errorf("expected %d results, got %d", len(results), len(filtered))
		}
	})

	t.Run("no matching types", func(t *testing.T) {
		filtered := FilterByType(results, []string{"js", "cpp"})
		if len(filtered) != 0 {
			t.Errorf("expected 0 results, got %d", len(filtered))
		}
	})
}

func TestFilterByGlob(t *testing.T) {
	results := []store.SearchResult{
		{Chunk: store.Chunk{FilePath: "src/main.go"}, Score: 0.9},
		{Chunk: store.Chunk{FilePath: "tests/test.go"}, Score: 0.8},
		{Chunk: store.Chunk{FilePath: "pkg/utils.go"}, Score: 0.7},
		{Chunk: store.Chunk{FilePath: "README.md"}, Score: 0.6},
	}

	t.Run("filter by basename pattern", func(t *testing.T) {
		filtered := FilterByGlob(results, []string{"*.md"})
		if len(filtered) != 1 {
			t.Errorf("expected 1 result, got %d", len(filtered))
		}
		if filtered[0].Chunk.FilePath != "README.md" {
			t.Errorf("expected README.md, got %s", filtered[0].Chunk.FilePath)
		}
	})

	t.Run("filter by path pattern", func(t *testing.T) {
		filtered := FilterByGlob(results, []string{"src/*"})
		if len(filtered) != 1 {
			t.Errorf("expected 1 result, got %d", len(filtered))
		}
		if filtered[0].Chunk.FilePath != "src/main.go" {
			t.Errorf("expected src/main.go, got %s", filtered[0].Chunk.FilePath)
		}
	})

	t.Run("filter by multiple patterns", func(t *testing.T) {
		filtered := FilterByGlob(results, []string{"src/*", "pkg/*"})
		if len(filtered) != 2 {
			t.Errorf("expected 2 results, got %d", len(filtered))
		}
	})

	t.Run("no globs specified", func(t *testing.T) {
		filtered := FilterByGlob(results, []string{})
		if len(filtered) != len(results) {
			t.Errorf("expected %d results, got %d", len(results), len(filtered))
		}
	})

	t.Run("no matching globs", func(t *testing.T) {
		filtered := FilterByGlob(results, []string{"lib/*"})
		if len(filtered) != 0 {
			t.Errorf("expected 0 results, got %d", len(filtered))
		}
	})
}

// Helper function to create a chunk for testing
func makeChunk(filePath string) store.Chunk {
	return store.Chunk{
		ID:        "test-id",
		FilePath:  filePath,
		StartLine: 1,
		EndLine:   10,
		Content:   "test content",
		Vector:    []float32{0.1, 0.2, 0.3},
		Hash:      "test-hash",
		UpdatedAt: time.Now(),
	}
}
