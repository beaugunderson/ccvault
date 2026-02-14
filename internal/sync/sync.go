// ABOUTME: Sync logic for indexing Claude Code conversations
// ABOUTME: Scans ~/.claude/projects/ and populates the ccvault database

package sync

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"

	"github.com/2389-research/ccvault/internal/db"
	"github.com/2389-research/ccvault/pkg/models"
	"github.com/2389-research/ccvault/pkg/parser"
)

// Stats tracks sync statistics
type Stats struct {
	SessionsScanned int
	SessionsIndexed int
	SessionsSkipped int
	TurnsIndexed    int
	ToolUsesIndexed int
	ProjectsFound   int
	Errors          []error
	Duration        time.Duration
}

// Syncer handles syncing Claude Code data to ccvault
type Syncer struct {
	db              *db.DB
	claudeHome      string
	full            bool
	verbose         bool
	onProgress      func(msg string)
	onCountProgress func(current, total int)
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

// WithCountProgressCallback sets a callback for numeric progress (current/total)
func WithCountProgressCallback(fn func(current, total int)) Option {
	return func(s *Syncer) {
		s.onCountProgress = fn
	}
}

// New creates a new Syncer
func New(database *db.DB, claudeHome string, opts ...Option) *Syncer {
	s := &Syncer{
		db:              database,
		claudeHome:      claudeHome,
		onProgress:      func(string) {},   // no-op default
		onCountProgress: func(int, int) {}, // no-op default
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

	// Batch-load all stored mtimes in one query for fast incremental checks
	var storedMtimes map[string]time.Time
	if !s.full {
		storedMtimes, err = s.db.GetAllSourceMtimes()
		if err != nil {
			// Non-fatal: fall back to syncing everything
			s.progress("Warning: could not load mtimes, will rescan all")
			storedMtimes = make(map[string]time.Time)
		}
		s.progress("Loaded %d stored mtimes", len(storedMtimes))
	}

	// Track unique projects
	projectsSeen := make(map[string]bool)

	// Process each session
	total := len(sessionFiles)
	for i, sf := range sessionFiles {
		projectsSeen[sf.ProjectPath] = true

		if err := s.processSession(sf, stats, storedMtimes); err != nil {
			stats.Errors = append(stats.Errors, fmt.Errorf("session %s: %w", sf.SessionID, err))
			if s.verbose {
				s.progress("Error processing %s: %v", sf.SessionID, err)
			}
		}

		s.onCountProgress(i+1, total)

		if (i+1)%100 == 0 || i == len(sessionFiles)-1 {
			s.progress("Processed %d/%d sessions", i+1, total)
		}
	}

	stats.ProjectsFound = len(projectsSeen)
	stats.Duration = time.Since(start)

	s.progress("Sync complete: %d sessions indexed, %d turns, %d tool uses",
		stats.SessionsIndexed, stats.TurnsIndexed, stats.ToolUsesIndexed)

	return stats, nil
}

// processSession handles a single session file
func (s *Syncer) processSession(sf parser.SessionFile, stats *Stats, storedMtimes map[string]time.Time) error {
	// Check if we need to process this file
	if !s.full {
		if !s.needsSync(sf, storedMtimes) {
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
		// Record mtime so we skip this empty file next time
		_ = s.db.UpsertSourceFileMtime(sf.Path, sf.ModTime)
		stats.SessionsSkipped++
		return nil // Empty or invalid session
	}

	// Set project path from scanner
	session.ProjectPath = sf.ProjectPath

	// Extract tool uses
	toolUses := parser.ExtractToolUses(turns)

	// Detect session flags for search filtering
	session.HasError = detectErrors(turns)
	session.HasSubagent = detectSubagents(toolUses)

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

		// Record source file mtime so incremental sync skips this file next time
		if err := s.db.UpsertSourceFileMtimeTx(tx, sf.Path, sf.ModTime); err != nil {
			return fmt.Errorf("upsert source mtime: %w", err)
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

// needsSync checks if a session file needs to be synced using the pre-loaded mtime map
func (s *Syncer) needsSync(sf parser.SessionFile, storedMtimes map[string]time.Time) bool {
	storedMtime, exists := storedMtimes[sf.Path]
	if !exists || storedMtime.IsZero() {
		return true // No stored time, needs sync
	}

	// Use the mtime from the directory scan (avoids a separate stat call per file)
	if sf.ModTime.IsZero() {
		return true // No mtime available, assume needs sync
	}

	return sf.ModTime.After(storedMtime)
}

// detectErrors checks if any turn in the session contains a tool error
func detectErrors(turns []models.Turn) bool {
	isErrorMarker := []byte(`"is_error":true`)
	isErrorMarkerSpaced := []byte(`"is_error": true`)
	for _, t := range turns {
		if len(t.RawJSON) > 0 {
			if bytes.Contains(t.RawJSON, isErrorMarker) || bytes.Contains(t.RawJSON, isErrorMarkerSpaced) {
				return true
			}
		}
	}
	return false
}

// detectSubagents checks if any tool use is a Task (subagent spawn)
func detectSubagents(toolUses []models.ToolUse) bool {
	for _, tu := range toolUses {
		if tu.ToolName == "Task" {
			return true
		}
	}
	return false
}

// progress logs a progress message
func (s *Syncer) progress(format string, args ...interface{}) {
	s.onProgress(fmt.Sprintf(format, args...))
}
