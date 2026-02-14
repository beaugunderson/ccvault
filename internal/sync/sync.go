// ABOUTME: Sync logic for indexing Claude Code conversations
// ABOUTME: Scans ~/.claude/projects/ and populates the ccvault database

package sync

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/2389-research/ccvault/internal/db"
	"github.com/2389-research/ccvault/pkg/models"
	"github.com/2389-research/ccvault/pkg/parser"
)

// Stats tracks sync statistics
type Stats struct {
	SessionsScanned  int
	SessionsIndexed  int
	SessionsSkipped  int
	TurnsIndexed     int
	ToolUsesIndexed  int
	ProjectsFound    int
	Errors           []error
	Duration         time.Duration
}

// Syncer handles syncing Claude Code data to ccvault
type Syncer struct {
	db         *db.DB
	claudeHome string
	full       bool
	verbose    bool
	onProgress func(msg string)
}

// Option configures a Syncer
type Option func(*Syncer)

// WithFullSync forces a complete rescan
func WithFullSync(full bool) Option {
	return func(s *Syncer) {
		s.full = full
	}
}

// WithVerbose enables verbose output
func WithVerbose(verbose bool) Option {
	return func(s *Syncer) {
		s.verbose = verbose
	}
}

// WithProgressCallback sets a callback for progress updates
func WithProgressCallback(fn func(string)) Option {
	return func(s *Syncer) {
		s.onProgress = fn
	}
}

// New creates a new Syncer
func New(database *db.DB, claudeHome string, opts ...Option) *Syncer {
	s := &Syncer{
		db:         database,
		claudeHome: claudeHome,
		onProgress: func(string) {}, // no-op default
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run performs the sync operation
func (s *Syncer) Run() (*Stats, error) {
	start := time.Now()
	stats := &Stats{}

	s.progress("Scanning %s for sessions...", s.claudeHome)

	// Discover session files
	sessionFiles, err := parser.ScanClaudeHome(s.claudeHome)
	if err != nil {
		return nil, fmt.Errorf("scan claude home: %w", err)
	}

	stats.SessionsScanned = len(sessionFiles)
	s.progress("Found %d session files", len(sessionFiles))

	// Track unique projects
	projectsSeen := make(map[string]bool)

	// Process each session
	for i, sf := range sessionFiles {
		projectsSeen[sf.ProjectPath] = true

		if err := s.processSession(sf, stats); err != nil {
			stats.Errors = append(stats.Errors, fmt.Errorf("session %s: %w", sf.SessionID, err))
			if s.verbose {
				s.progress("Error processing %s: %v", sf.SessionID, err)
			}
			continue
		}

		if (i+1)%100 == 0 || i == len(sessionFiles)-1 {
			s.progress("Processed %d/%d sessions", i+1, len(sessionFiles))
		}
	}

	stats.ProjectsFound = len(projectsSeen)
	stats.Duration = time.Since(start)

	s.progress("Sync complete: %d sessions indexed, %d turns, %d tool uses",
		stats.SessionsIndexed, stats.TurnsIndexed, stats.ToolUsesIndexed)

	return stats, nil
}

// processSession handles a single session file
func (s *Syncer) processSession(sf parser.SessionFile, stats *Stats) error {
	// Check if we need to process this file
	if !s.full {
		needsSync, err := s.needsSync(sf)
		if err != nil {
			return err
		}
		if !needsSync {
			stats.SessionsSkipped++
			return nil
		}
	}

	// Parse the session
	turns, session, err := parser.ParseSession(sf.Path)
	if err != nil {
		return fmt.Errorf("parse session: %w", err)
	}

	if session.ID == "" {
		stats.SessionsSkipped++
		return nil // Empty or invalid session
	}

	// Set project path from scanner
	session.ProjectPath = sf.ProjectPath

	// Extract tool uses
	toolUses := parser.ExtractToolUses(turns)

	// Store everything in a transaction
	err = s.db.WithTx(func(tx *sql.Tx) error {
		// Upsert project
		project := &models.Project{
			Path:           sf.ProjectPath,
			DisplayName:    parser.GetDisplayName(sf.ProjectPath),
			FirstSeenAt:    session.StartedAt,
			LastActivityAt: session.EndedAt,
			SessionCount:   1,
			TotalTokens:    session.TotalTokens(),
		}
		if err := s.db.UpsertProjectTx(tx, project); err != nil {
			return fmt.Errorf("upsert project: %w", err)
		}

		// Set project ID on session
		session.ProjectID = project.ID

		// Delete existing turns for this session (for re-sync)
		if err := s.db.DeleteTurnsForSessionTx(tx, session.ID); err != nil {
			return fmt.Errorf("delete old turns: %w", err)
		}

		// Delete existing tool uses
		if err := s.db.DeleteToolUsesForSessionTx(tx, session.ID); err != nil {
			return fmt.Errorf("delete old tool uses: %w", err)
		}

		// Upsert session
		if err := s.db.UpsertSessionTx(tx, session); err != nil {
			return fmt.Errorf("upsert session: %w", err)
		}

		// Insert turns
		if err := s.db.InsertTurnsTx(tx, turns); err != nil {
			return fmt.Errorf("insert turns: %w", err)
		}

		// Insert tool uses
		if len(toolUses) > 0 {
			if err := s.db.InsertToolUsesTx(tx, toolUses); err != nil {
				return fmt.Errorf("insert tool uses: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	stats.SessionsIndexed++
	stats.TurnsIndexed += len(turns)
	stats.ToolUsesIndexed += len(toolUses)

	return nil
}

// needsSync checks if a session file needs to be synced
func (s *Syncer) needsSync(sf parser.SessionFile) (bool, error) {
	// Get file modification time
	info, err := os.Stat(sf.Path)
	if err != nil {
		return false, err
	}
	fileMtime := info.ModTime()

	// Get stored modification time
	storedMtime, err := s.db.GetSourceMtime(sf.Path)
	if err != nil {
		return true, nil // If we can't get stored time, assume needs sync
	}

	// If no stored time or file is newer, needs sync
	if storedMtime.IsZero() || fileMtime.After(storedMtime) {
		return true, nil
	}

	return false, nil
}

// progress logs a progress message
func (s *Syncer) progress(format string, args ...interface{}) {
	s.onProgress(fmt.Sprintf(format, args...))
}
