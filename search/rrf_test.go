package search

import (
	"testing"

	"github.com/pprgva/code-memory/store"
)

func TestMergeResultsRRF(t *testing.T) {
	tests := []struct {
		name       string
		resultSets [][]FederatedResult
		k          float32
		limit      int
		wantLen    int
	}{
		{
			name:       "empty input",
			resultSets: [][]FederatedResult{},
			k:          60,
			limit:      10,
			wantLen:    0,
		},
		{
			name: "single result set",
			resultSets: [][]FederatedResult{
				{
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "1", FilePath: "a.go", StartLine: 1}, Score: 0.9}, ProjectName: "proj1", ProjectPath: "~/proj1"},
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "2", FilePath: "b.go", StartLine: 1}, Score: 0.8}, ProjectName: "proj1", ProjectPath: "~/proj1"},
				},
			},
			k:       60,
			limit:   10,
			wantLen: 2,
		},
		{
			name: "multiple result sets with overlap",
			resultSets: [][]FederatedResult{
				{
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "1", FilePath: "a.go", StartLine: 1}, Score: 0.9}, ProjectName: "proj1", ProjectPath: "~/proj1"},
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "2", FilePath: "b.go", StartLine: 1}, Score: 0.8}, ProjectName: "proj1", ProjectPath: "~/proj1"},
				},
				{
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "3", FilePath: "c.go", StartLine: 1}, Score: 0.95}, ProjectName: "proj2", ProjectPath: "~/proj2"},
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "1", FilePath: "a.go", StartLine: 1}, Score: 0.85}, ProjectName: "proj1", ProjectPath: "~/proj1"}, // Duplicate
				},
			},
			k:       60,
			limit:   10,
			wantLen: 3, // 3 unique results
		},
		{
			name: "limit applied",
			resultSets: [][]FederatedResult{
				{
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "1", FilePath: "a.go", StartLine: 1}, Score: 0.9}, ProjectName: "proj1", ProjectPath: "~/proj1"},
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "2", FilePath: "b.go", StartLine: 1}, Score: 0.8}, ProjectName: "proj1", ProjectPath: "~/proj1"},
					{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "3", FilePath: "c.go", StartLine: 1}, Score: 0.7}, ProjectName: "proj1", ProjectPath: "~/proj1"},
				},
			},
			k:       60,
			limit:   2,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeResultsRRF(tt.resultSets, tt.k, tt.limit)
			if len(got) != tt.wantLen {
				t.Errorf("MergeResultsRRF() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestMergeResultsRRF_ScoreOrder(t *testing.T) {
	// Test that results appearing in multiple lists get boosted
	resultSets := [][]FederatedResult{
		{
			{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "common", FilePath: "common.go", StartLine: 1}, Score: 0.8}, ProjectName: "proj1", ProjectPath: "~/proj1"},
			{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "unique1", FilePath: "unique1.go", StartLine: 1}, Score: 0.9}, ProjectName: "proj1", ProjectPath: "~/proj1"},
		},
		{
			{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "common", FilePath: "common.go", StartLine: 1}, Score: 0.85}, ProjectName: "proj1", ProjectPath: "~/proj1"},
			{SearchResult: store.SearchResult{Chunk: store.Chunk{ID: "unique2", FilePath: "unique2.go", StartLine: 1}, Score: 0.95}, ProjectName: "proj2", ProjectPath: "~/proj2"},
		},
	}

	got := MergeResultsRRF(resultSets, 60, 10)

	if len(got) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(got))
	}

	// "common" appears in both lists so should have highest RRF score
	if got[0].Chunk.ID != "common" {
		t.Errorf("Expected 'common' to be ranked first due to RRF boosting, got %s", got[0].Chunk.ID)
	}
}

func TestMergeSearchResultsRRF(t *testing.T) {
	tests := []struct {
		name       string
		resultSets [][]store.SearchResult
		k          float32
		limit      int
		wantLen    int
	}{
		{
			name:       "empty input",
			resultSets: [][]store.SearchResult{},
			k:          60,
			limit:      10,
			wantLen:    0,
		},
		{
			name: "single result set",
			resultSets: [][]store.SearchResult{
				{
					{Chunk: store.Chunk{ID: "1"}, Score: 0.9},
					{Chunk: store.Chunk{ID: "2"}, Score: 0.8},
				},
			},
			k:       60,
			limit:   10,
			wantLen: 2,
		},
		{
			name: "multiple result sets",
			resultSets: [][]store.SearchResult{
				{
					{Chunk: store.Chunk{ID: "1"}, Score: 0.9},
					{Chunk: store.Chunk{ID: "2"}, Score: 0.8},
				},
				{
					{Chunk: store.Chunk{ID: "3"}, Score: 0.95},
					{Chunk: store.Chunk{ID: "1"}, Score: 0.85}, // Duplicate
				},
			},
			k:       60,
			limit:   10,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeSearchResultsRRF(tt.resultSets, tt.k, tt.limit)
			if len(got) != tt.wantLen {
				t.Errorf("MergeSearchResultsRRF() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}
