package cli

import (
	"testing"
	"time"
)

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "just now",
			duration: 30 * time.Second,
			expected: "just now",
		},
		{
			name:     "1 min ago",
			duration: 90 * time.Second,
			expected: "1 min ago",
		},
		{
			name:     "5 min ago",
			duration: 5 * time.Minute,
			expected: "5 min ago",
		},
		{
			name:     "1 hour ago",
			duration: 90 * time.Minute,
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			duration: 3 * time.Hour,
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			duration: 36 * time.Hour,
			expected: "1 day ago",
		},
		{
			name:     "5 days ago",
			duration: 5 * 24 * time.Hour,
			expected: "5 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Now().Add(-tt.duration)
			result := formatTimeAgo(testTime)
			if result != tt.expected {
				t.Errorf("formatTimeAgo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo_OldDate(t *testing.T) {
	// Test dates older than 30 days
	oldDate := time.Now().AddDate(0, -2, 0) // 2 months ago
	result := formatTimeAgo(oldDate)

	// Should be in YYYY-MM-DD format
	if len(result) != 10 || result[4] != '-' || result[7] != '-' {
		t.Errorf("formatTimeAgo() for old date = %v, expected YYYY-MM-DD format", result)
	}
}
