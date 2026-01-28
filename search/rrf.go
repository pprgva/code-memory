package search

import (
	"sort"

	"github.com/pprgva/code-memory/store"
)

// MergeResultsRRF merges multiple result sets using Reciprocal Rank Fusion.
// RRF score = sum(1 / (k + rank)) where k is typically 60.
// Results are deduplicated by a unique key (project + file + start_line).
func MergeResultsRRF(resultSets [][]FederatedResult, k float32, limit int) []FederatedResult {
	if len(resultSets) == 0 {
		return []FederatedResult{}
	}

	// Track RRF scores and best result for each unique key
	scores := make(map[string]float32)
	resultMap := make(map[string]FederatedResult)

	for _, resultSet := range resultSets {
		for rank, result := range resultSet {
			// Create unique key: project + file + start_line
			key := result.ProjectName + ":" + result.Chunk.FilePath + ":" + string(rune(result.Chunk.StartLine))

			// Calculate RRF contribution (rank is 0-indexed, add 1 for standard RRF formula)
			rrfScore := 1.0 / (k + float32(rank) + 1)
			scores[key] += rrfScore

			// Keep the result with the highest original score for this key
			if existing, ok := resultMap[key]; !ok || result.Score > existing.Score {
				resultMap[key] = result
			}
		}
	}

	// Convert to slice and update scores
	merged := make([]FederatedResult, 0, len(resultMap))
	for key, result := range resultMap {
		result.Score = scores[key]
		merged = append(merged, result)
	}

	// Sort by RRF score descending
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	// Limit results
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}

// MergeSearchResultsRRF merges standard SearchResult slices using RRF.
// This is useful for combining results from different search methods within the same project.
func MergeSearchResultsRRF(resultSets [][]store.SearchResult, k float32, limit int) []store.SearchResult {
	if len(resultSets) == 0 {
		return []store.SearchResult{}
	}

	// Track RRF scores and best result for each chunk ID
	scores := make(map[string]float32)
	resultMap := make(map[string]store.SearchResult)

	for _, resultSet := range resultSets {
		for rank, result := range resultSet {
			key := result.Chunk.ID

			// Calculate RRF contribution
			rrfScore := 1.0 / (k + float32(rank) + 1)
			scores[key] += rrfScore

			// Keep the result with the highest original score
			if existing, ok := resultMap[key]; !ok || result.Score > existing.Score {
				resultMap[key] = result
			}
		}
	}

	// Convert to slice and update scores
	merged := make([]store.SearchResult, 0, len(resultMap))
	for key, result := range resultMap {
		result.Score = scores[key]
		merged = append(merged, result)
	}

	// Sort by RRF score descending
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	// Limit results
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}
