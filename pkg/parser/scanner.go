// ABOUTME: Scanner for discovering Claude Code session files
// ABOUTME: Walks ~/.claude/projects/ to find all session JSONL files

package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/2389-research/ccvault/pkg/models"
)

// SessionFile represents a discovered session file
type SessionFile struct {
	Path        string // Full path to .jsonl file
	SessionID   string // UUID extracted from filename
	ProjectPath string // Decoded project path
}

// ScanClaudeHome scans the Claude Code data directory for session files
func ScanClaudeHome(claudeHome string) ([]SessionFile, error) {
	projectsDir := filepath.Join(claudeHome, "projects")

	info, err := os.Stat(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("projects directory does not exist: %s", projectsDir)
		}
		return nil, fmt.Errorf("stat projects dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("projects path is not a directory: %s", projectsDir)
	}

	var sessions []SessionFile

	// Walk the projects directory
	err = filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip subagents directories - we'll handle them separately if needed
			if d.Name() == "subagents" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .jsonl files
		if !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}

		// Extract session ID from filename (UUID.jsonl)
		sessionID := strings.TrimSuffix(d.Name(), ".jsonl")
		if !isValidUUID(sessionID) {
			return nil // Skip non-session files
		}

		// Extract project path from parent directory name
		relPath, err := filepath.Rel(projectsDir, filepath.Dir(path))
		if err != nil {
			return nil
		}

		// Decode URL-encoded project path
		projectPath := decodeProjectPath(relPath)

		sessions = append(sessions, SessionFile{
			Path:        path,
			SessionID:   sessionID,
			ProjectPath: projectPath,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk projects dir: %w", err)
	}

	return sessions, nil
}

// decodeProjectPath converts URL-encoded directory name back to filesystem path
func decodeProjectPath(encoded string) string {
	// Replace leading dash with /
	if strings.HasPrefix(encoded, "-") {
		encoded = "/" + encoded[1:]
	}

	// Replace remaining dashes with /
	decoded := strings.ReplaceAll(encoded, "-", "/")

	// URL decode any remaining encoded characters
	if unescaped, err := url.PathUnescape(decoded); err == nil {
		decoded = unescaped
	}

	return decoded
}

// isValidUUID checks if a string looks like a UUID
func isValidUUID(s string) bool {
	// Simple check: 36 chars with dashes in right places
	if len(s) != 36 {
		return false
	}

	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}

	return true
}

// GetDisplayName creates a short display name from a project path
func GetDisplayName(projectPath string) string {
	// Remove home directory prefix
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(projectPath, home) {
		projectPath = "~" + projectPath[len(home):]
	}

	// Take last 2-3 path components for display
	parts := strings.Split(projectPath, "/")
	if len(parts) <= 3 {
		return projectPath
	}

	// Return last 3 components
	return strings.Join(parts[len(parts)-3:], "/")
}

// ScanHistory reads the global history.jsonl file
func ScanHistory(claudeHome string) ([]HistoryEntry, error) {
	historyPath := filepath.Join(claudeHome, "history.jsonl")

	f, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No history file is OK
		}
		return nil, fmt.Errorf("open history file: %w", err)
	}
	defer f.Close()

	return ParseHistoryReader(f)
}

// HistoryEntry represents an entry from history.jsonl
// Re-export from models for convenience
type HistoryEntry = models.HistoryEntry

// ParseHistoryReader parses history entries from a reader
func ParseHistoryReader(r io.Reader) ([]HistoryEntry, error) {
	var entries []HistoryEntry

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed entries
		}
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}
