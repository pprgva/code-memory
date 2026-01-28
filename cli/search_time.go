package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// parseDuration parses duration strings like "7d", "2h", "30m", "2w" into time.Duration
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	unit := s[len(s)-1]
	num, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}
	switch unit {
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'm':
		return time.Duration(num) * time.Minute, nil
	case 'w':
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c (use d, h, m, w)", unit)
	}
}

// getGitModifiedFiles returns files changed since git ref (e.g., "main", "HEAD~10")
func getGitModifiedFiles(projectRoot, ref string) (map[string]bool, error) {
	cmd := exec.Command("git", "diff", "--name-only", ref+"...HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}
	files := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files[filepath.Join(projectRoot, line)] = true
		}
	}
	return files, nil
}

// getRecentlyModifiedFiles returns files modified within duration using file mtime
func getRecentlyModifiedFiles(projectRoot string, since time.Duration) (map[string]bool, error) {
	cutoff := time.Now().Add(-since)
	files := make(map[string]bool)
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().After(cutoff) {
			files[path] = true
		}
		return nil
	})
	return files, err
}
