# ccvault Implementation Plan

**Date:** 2026-02-13
**Design:** 2026-02-13-ccvault-design.md
**Status:** Ready for execution

## Phase 1: Project Scaffolding

### Task 1.1: Initialize Go Module
- [ ] Run `go mod init github.com/2389-research/ccvault`
- [ ] Create basic directory structure per design
- [ ] Add `.gitignore` for Go projects
- **Verify:** `go mod tidy` succeeds

### Task 1.2: Add Core Dependencies
- [ ] Add cobra for CLI
- [ ] Add viper for config
- [ ] Add modernc.org/sqlite for SQLite
- [ ] Add charmbracelet/bubbletea for TUI
- [ ] Add charmbracelet/lipgloss for styling
- **Verify:** `go mod tidy && go build ./...` succeeds

### Task 1.3: Create Basic CLI Structure
- [ ] Create `cmd/ccvault/main.go` with root command
- [ ] Add version command
- [ ] Add placeholder commands: sync, tui, search, stats, serve
- **Verify:** `go run ./cmd/ccvault --help` shows all commands

## Phase 2: Data Models & Parser

### Task 2.1: Define Data Models
- [ ] Create `pkg/models/models.go` with Project, Session, Turn, ToolUse structs
- [ ] Add JSON tags matching Claude Code JSONL format
- [ ] Add utility methods for token aggregation
- **Verify:** Unit tests pass for model creation

### Task 2.2: JSONL Parser
- [ ] Create `pkg/parser/parser.go`
- [ ] Parse individual JSONL entries into Turn structs
- [ ] Handle all turn types: user, assistant, tool_use, tool_result, progress
- [ ] Extract tool names and file paths from tool calls
- **Verify:** Unit tests parse sample JSONL correctly

### Task 2.3: Session Scanner
- [ ] Create `pkg/parser/scanner.go`
- [ ] Scan `~/.claude/projects/` directory structure
- [ ] Decode URL-encoded project paths
- [ ] Find all `.jsonl` session files
- **Verify:** Unit tests find sessions in test fixtures

## Phase 3: Database Layer

### Task 3.1: SQLite Schema
- [ ] Create `internal/db/schema.sql` with all tables from design
- [ ] Create `internal/db/db.go` for connection management
- [ ] Add migration/init function
- **Verify:** Database creates successfully with all tables

### Task 3.2: CRUD Operations
- [ ] Create `internal/db/projects.go` - UpsertProject, GetProjects, GetProject
- [ ] Create `internal/db/sessions.go` - UpsertSession, GetSessions, GetSession
- [ ] Create `internal/db/turns.go` - InsertTurns, GetTurns, SearchTurns
- **Verify:** Integration tests for CRUD operations

### Task 3.3: FTS5 Search
- [ ] Add FTS5 virtual table setup
- [ ] Create `internal/search/search.go` for query execution
- [ ] Implement basic full-text search
- **Verify:** Search finds expected content in test data

## Phase 4: Sync Command

### Task 4.1: Sync Logic
- [ ] Create `internal/sync/sync.go`
- [ ] Implement incremental sync (track file mtimes)
- [ ] Parse sessions and populate database
- [ ] Create `sync_state.json` checkpoint
- **Verify:** Running twice only processes new files

### Task 4.2: Wire Up CLI
- [ ] Implement `ccvault sync` command
- [ ] Add `--full` flag for complete rescan
- [ ] Add progress output
- **Verify:** `ccvault sync` indexes real ~/.claude data

## Phase 5: Search Command

### Task 5.1: Query Parser
- [ ] Create `internal/search/query.go`
- [ ] Parse operators: project:, model:, tool:, before:, after:
- [ ] Handle quoted phrases and bare words
- **Verify:** Unit tests for query parsing

### Task 5.2: Search Execution
- [ ] Build SQL queries from parsed search
- [ ] Return ranked results with context
- [ ] Support `--json` output format
- **Verify:** `ccvault search "test"` returns results

## Phase 6: Stats & List Commands

### Task 6.1: Stats Command
- [ ] Implement `ccvault stats`
- [ ] Show total projects, sessions, turns, tokens
- [ ] Show date range of archive
- **Verify:** Output matches database counts

### Task 6.2: List Commands
- [ ] Implement `ccvault list-projects`
- [ ] Implement `ccvault list-sessions`
- [ ] Add filtering and `--json` output
- **Verify:** Lists show expected data

### Task 6.3: Show Command
- [ ] Implement `ccvault show <session-id>`
- [ ] Format conversation for terminal
- [ ] Syntax highlight code blocks
- **Verify:** Shows readable conversation

## Phase 7: TUI

### Task 7.1: TUI Framework
- [ ] Create `internal/tui/app.go` - main Bubble Tea program
- [ ] Create `internal/tui/styles.go` - Lip Gloss styles
- [ ] Implement view switching infrastructure
- **Verify:** TUI launches and handles quit

### Task 7.2: Dashboard View
- [ ] Create `internal/tui/dashboard.go`
- [ ] Show summary stats
- [ ] Navigation to projects list
- **Verify:** Dashboard displays data from DB

### Task 7.3: Projects View
- [ ] Create `internal/tui/projects.go`
- [ ] List projects with stats
- [ ] Sort and filter capabilities
- **Verify:** Can browse and select projects

### Task 7.4: Sessions View
- [ ] Create `internal/tui/sessions.go`
- [ ] List sessions for selected project
- [ ] Show metadata (date, model, tokens)
- **Verify:** Can browse sessions

### Task 7.5: Conversation View
- [ ] Create `internal/tui/conversation.go`
- [ ] Render threaded conversation
- [ ] Scrolling and navigation
- **Verify:** Can read full conversations

## Phase 8: Analytics (Optional Enhancement)

### Task 8.1: Parquet Export
- [ ] Create `internal/analytics/export.go`
- [ ] Export sessions to Parquet format
- [ ] Implement `ccvault build-cache`
- **Verify:** Parquet files created

### Task 8.2: DuckDB Queries
- [ ] Add go-duckdb dependency
- [ ] Implement analytics queries
- [ ] Add analytics views to TUI
- **Verify:** Analytics queries return expected data

## Phase 9: MCP Server

### Task 9.1: MCP Protocol
- [ ] Create `internal/mcp/server.go`
- [ ] Implement JSON-RPC over stdio
- [ ] Handle initialize/shutdown lifecycle
- **Verify:** MCP handshake works

### Task 9.2: MCP Tools
- [ ] Implement search_conversations tool
- [ ] Implement get_session tool
- [ ] Implement list_projects tool
- [ ] Implement get_analytics tool
- **Verify:** Tools callable via MCP protocol

### Task 9.3: Claude Desktop Integration
- [ ] Document claude_desktop_config.json setup
- [ ] Test with Claude Desktop
- **Verify:** Claude can search conversation history

## Phase 10: Polish

### Task 10.1: Configuration
- [ ] Create `internal/config/config.go`
- [ ] Load from `~/.ccvault/config.toml`
- [ ] Support environment variable overrides
- **Verify:** Config loading works

### Task 10.2: README & Docs
- [ ] Write comprehensive README.md
- [ ] Add installation instructions
- [ ] Document all commands
- **Verify:** README covers all features

### Task 10.3: Release Build
- [ ] Add Makefile or goreleaser config
- [ ] Build for darwin/linux amd64/arm64
- **Verify:** Binaries work on target platforms
