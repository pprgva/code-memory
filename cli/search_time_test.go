package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "minutes",
			input: "30m",
			want:  30 * time.Minute,
		},
		{
			name:  "hours",
			input: "2h",
			want:  2 * time.Hour,
		},
		{
			name:  "days",
			input: "7d",
			want:  7 * 24 * time.Hour,
		},
		{
			name:  "weeks",
			input: "2w",
			want:  2 * 7 * 24 * time.Hour,
		},
		{
			name:    "invalid unit",
			input:   "10x",
			wantErr: true,
		},
		{
			name:    "invalid number",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "d",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRecentlyModifiedFiles(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create test files with different modification times
	oldFile := filepath.Join(tmpDir, "old.txt")
	recentFile := filepath.Join(tmpDir, "recent.txt")

	// Create old file and set mtime to 2 days ago
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create recent file (current time)
	if err := os.WriteFile(recentFile, []byte("recent"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test: find files modified in last 24 hours
	files, err := getRecentlyModifiedFiles(tmpDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("getRecentlyModifiedFiles() error = %v", err)
	}

	// Should find only the recent file
	if !files[recentFile] {
		t.Errorf("Expected to find recent file %s", recentFile)
	}
	if files[oldFile] {
		t.Errorf("Did not expect to find old file %s", oldFile)
	}

	// Test: find files modified in last 3 days
	files, err = getRecentlyModifiedFiles(tmpDir, 72*time.Hour)
	if err != nil {
		t.Fatalf("getRecentlyModifiedFiles() error = %v", err)
	}

	// Should find both files
	if !files[recentFile] {
		t.Errorf("Expected to find recent file %s", recentFile)
	}
	if !files[oldFile] {
		t.Errorf("Expected to find old file %s", oldFile)
	}
}

func TestGetGitModifiedFiles(t *testing.T) {
	// This test requires a git repository to work
	// Skip if not in a git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	// Test with HEAD (should return empty as there are no changes from HEAD to HEAD)
	files, err := getGitModifiedFiles(".", "HEAD")
	if err != nil {
		t.Fatalf("getGitModifiedFiles() error = %v", err)
	}

	// Should return a map (might be empty if no changes)
	if files == nil {
		t.Error("Expected non-nil map")
	}
}
