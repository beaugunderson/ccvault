// ABOUTME: Integration tests for sync logic
// ABOUTME: Validates CWD-preferred project path resolution and projectsSeen tracking

package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/2389-research/ccvault/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ccvault-sync-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	database, err := db.Open(tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("open db: %v", err)
	}

	cleanup := func() {
		_ = database.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

// createFakeClaudeHome creates a temporary ~/.claude structure with session files.
// encodedDir is the dash-encoded directory name (e.g. "-Users-harper-canvas-plugins").
// cwd is the actual working directory to embed in the JSONL (e.g. "/Users/harper/canvas-plugins").
// If cwd is empty, no cwd field is written in the JSONL.
func createFakeClaudeHome(t *testing.T, encodedDir, cwd string) (claudeHome string, cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ccvault-claude-home-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "projects", encodedDir)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("create project dir: %v", err)
	}

	sessionID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	sessionFile := filepath.Join(projectDir, sessionID+".jsonl")

	var cwdField string
	if cwd != "" {
		cwdField = fmt.Sprintf(`,"cwd":"%s"`, cwd)
	}

	jsonl := fmt.Sprintf(
		`{"uuid":"turn1-aaa-bbbb-cccc-dddddddddddd","sessionId":"%s","type":"user","timestamp":"2026-02-02T20:00:00.000Z"%s,"message":{"role":"user","content":"Hello"}}
{"uuid":"turn2-aaa-bbbb-cccc-dddddddddddd","sessionId":"%s","type":"assistant","timestamp":"2026-02-02T20:01:00.000Z"%s,"message":{"model":"claude-opus-4-5-20251101","role":"assistant","content":[{"type":"text","text":"Hi!"}],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		sessionID, cwdField, sessionID, cwdField,
	)

	if err := os.WriteFile(sessionFile, []byte(jsonl), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("write session file: %v", err)
	}

	return tmpDir, func() { _ = os.RemoveAll(tmpDir) }
}

func TestSync_CWDPreferredOverScannerPath(t *testing.T) {
	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// Encoded dir: "-Users-harper-canvas-plugins"
	// Scanner would decode this as "/Users/harper/canvas/plugins" (WRONG)
	// CWD in JSONL says "/Users/harper/canvas-plugins" (CORRECT)
	claudeHome, homeCleanup := createFakeClaudeHome(t,
		"-Users-harper-canvas-plugins",
		"/Users/harper/canvas-plugins",
	)
	defer homeCleanup()

	syncer := New(database, claudeHome, WithFullSync(true))
	stats, err := syncer.Run()
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if stats.SessionsIndexed != 1 {
		t.Fatalf("Expected 1 session indexed, got %d", stats.SessionsIndexed)
	}

	// Verify project was created with the CWD path, not the lossy decode
	project, err := database.GetProjectByPath("/Users/harper/canvas-plugins")
	if err != nil {
		t.Fatalf("GetProjectByPath failed: %v", err)
	}
	if project == nil {
		t.Fatal("Expected project with path '/Users/harper/canvas-plugins', got nil")
	}

	// The lossy-decoded path should NOT exist as a project
	wrongProject, err := database.GetProjectByPath("/Users/harper/canvas/plugins")
	if err != nil {
		t.Fatalf("GetProjectByPath failed: %v", err)
	}
	if wrongProject != nil {
		t.Error("Project with lossy-decoded path '/Users/harper/canvas/plugins' should not exist")
	}
}

func TestSync_FallsBackToScannerPathWhenNoCWD(t *testing.T) {
	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// No CWD in JSONL — should fall back to scanner's lossy decode
	claudeHome, homeCleanup := createFakeClaudeHome(t,
		"-Users-harper-simple-project",
		"", // no cwd
	)
	defer homeCleanup()

	syncer := New(database, claudeHome, WithFullSync(true))
	stats, err := syncer.Run()
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if stats.SessionsIndexed != 1 {
		t.Fatalf("Expected 1 session indexed, got %d", stats.SessionsIndexed)
	}

	// Falls back to scanner decode: "-Users-harper-simple-project" → "/Users/harper/simple/project"
	project, err := database.GetProjectByPath("/Users/harper/simple/project")
	if err != nil {
		t.Fatalf("GetProjectByPath failed: %v", err)
	}
	if project == nil {
		t.Fatal("Expected project with scanner-decoded path '/Users/harper/simple/project', got nil")
	}
}

func TestSync_ProjectsSeenTracksCWDPath(t *testing.T) {
	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	claudeHome, homeCleanup := createFakeClaudeHome(t,
		"-Users-harper-buddy-web",
		"/Users/harper/buddy-web",
	)
	defer homeCleanup()

	syncer := New(database, claudeHome, WithFullSync(true))
	stats, err := syncer.Run()
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// ProjectsFound should count 1 project (the CWD path)
	if stats.ProjectsFound != 1 {
		t.Errorf("Expected ProjectsFound=1, got %d", stats.ProjectsFound)
	}
}

func TestSync_IncrementalSkipUsesProjectsSeen(t *testing.T) {
	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	claudeHome, homeCleanup := createFakeClaudeHome(t,
		"-Users-harper-my-app",
		"/Users/harper/my-app",
	)
	defer homeCleanup()

	// First sync: full
	syncer := New(database, claudeHome, WithFullSync(true))
	stats, err := syncer.Run()
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if stats.SessionsIndexed != 1 {
		t.Fatalf("Expected 1 session indexed on first sync, got %d", stats.SessionsIndexed)
	}

	// Second sync: incremental — file should be skipped (same mtime)
	// Touch the file to ensure mtime is not after stored mtime
	syncer2 := New(database, claudeHome, WithFullSync(false))
	stats2, err := syncer2.Run()
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if stats2.SessionsSkipped != 1 {
		t.Errorf("Expected 1 session skipped on incremental sync, got %d", stats2.SessionsSkipped)
	}
	if stats2.SessionsIndexed != 0 {
		t.Errorf("Expected 0 sessions indexed on incremental sync, got %d", stats2.SessionsIndexed)
	}

	// ProjectsFound should still be 1 even though session was skipped
	if stats2.ProjectsFound != 1 {
		t.Errorf("Expected ProjectsFound=1 on incremental sync, got %d", stats2.ProjectsFound)
	}
}

func TestSync_IncrementalResyncsModifiedFile(t *testing.T) {
	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	claudeHome, homeCleanup := createFakeClaudeHome(t,
		"-Users-harper-test-proj",
		"/Users/harper/test-proj",
	)
	defer homeCleanup()

	// First sync: full
	syncer := New(database, claudeHome, WithFullSync(true))
	_, err := syncer.Run()
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	// Touch the file to bump mtime into the future
	sessionFile := filepath.Join(claudeHome, "projects", "-Users-harper-test-proj", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl")
	futureTime := time.Now().Add(1 * time.Hour)
	if err := os.Chtimes(sessionFile, futureTime, futureTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// Second sync: incremental — file should be re-processed
	syncer2 := New(database, claudeHome, WithFullSync(false))
	stats2, err := syncer2.Run()
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if stats2.SessionsIndexed != 1 {
		t.Errorf("Expected 1 session re-indexed after mtime bump, got %d", stats2.SessionsIndexed)
	}

	// Verify project still has CWD path after re-sync
	project, err := database.GetProjectByPath("/Users/harper/test-proj")
	if err != nil {
		t.Fatalf("GetProjectByPath failed: %v", err)
	}
	if project == nil {
		t.Fatal("Expected project with CWD path after re-sync, got nil")
	}
}
