package cli

import (
	"testing"
)

func TestMatchScore(t *testing.T) {
	tests := []struct {
		question string
		topic    string
		wantMin  int
	}{
		// Exact match
		{"json output", "json output", 100},

		// Topic contained in question
		{"how to get json output", "json output", 80},

		// Word-by-word matching
		{"search across projects", "search multiple projects", 20},

		// Keyword boosts
		{"cross project search", "cross-project", 25},
		{"how to use daemon", "daemon", 15},
		{"trace who calls this function", "trace callers", 30},

		// No match
		{"xyz abc", "json output", 0},
	}

	for _, tt := range tests {
		t.Run(tt.question+"->"+tt.topic, func(t *testing.T) {
			got := matchScore(tt.question, tt.topic)
			if got < tt.wantMin {
				t.Errorf("matchScore(%q, %q) = %d, want at least %d", tt.question, tt.topic, got, tt.wantMin)
			}
		})
	}
}

func TestHelpTopicsExist(t *testing.T) {
	expectedTopics := []string{
		"basic",
		"filtering",
		"trace",
		"workspace",
		"llm",
		"daemon",
		"config",
	}

	for _, topic := range expectedTopics {
		if _, ok := helpTopics[topic]; !ok {
			t.Errorf("missing help topic: %s", topic)
		}
	}
}

func TestExplainTopicsExist(t *testing.T) {
	expectedTopics := []string{
		"search multiple projects",
		"json output",
		"compact",
		"daemon",
		"trace callers",
		"trace callees",
		"trace graph",
		"impact analysis",
		"filter type",
		"filter glob",
		"filter time",
		"setup",
		"best practices",
	}

	for _, topic := range expectedTopics {
		if _, ok := explainTopics[topic]; !ok {
			t.Errorf("missing explain topic: %s", topic)
		}
	}
}

func TestSuggestTopics(t *testing.T) {
	suggestions := suggestTopics()
	if suggestions == "" {
		t.Error("suggestTopics() returned empty string")
	}
	// Should contain some known topics
	if !contains(suggestions, "json output") {
		t.Error("suggestTopics() missing 'json output'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
