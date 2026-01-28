package search

import (
	"path/filepath"
	"strings"

	"github.com/pprgva/code-memory/store"
)

// FilterByType filters results by file extensions
func FilterByType(results []store.SearchResult, types []string) []store.SearchResult {
	if len(types) == 0 {
		return results
	}

	typeMap := make(map[string]bool)
	for _, t := range types {
		t = strings.TrimSpace(strings.ToLower(t))
		if !strings.HasPrefix(t, ".") {
			t = "." + t
		}
		typeMap[t] = true
	}

	var filtered []store.SearchResult
	for _, r := range results {
		ext := strings.ToLower(filepath.Ext(r.Chunk.FilePath))
		if typeMap[ext] {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// FilterByGlob filters results by glob patterns
func FilterByGlob(results []store.SearchResult, globs []string) []store.SearchResult {
	if len(globs) == 0 {
		return results
	}

	var filtered []store.SearchResult
	for _, r := range results {
		for _, pattern := range globs {
			// Try matching against basename first
			if match, _ := filepath.Match(pattern, filepath.Base(r.Chunk.FilePath)); match {
				filtered = append(filtered, r)
				break
			}
			// Try matching against full path
			if match, _ := filepath.Match(pattern, r.Chunk.FilePath); match {
				filtered = append(filtered, r)
				break
			}
		}
	}

	return filtered
}
