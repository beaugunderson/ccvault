# Documentation Audit Report
Generated: 2026-02-14 | Commit: 1c9c531

## Executive Summary

| Metric | Count |
|--------|-------|
| Documents scanned | 2 (README.md, CLAUDE.md) |
| Claims verified | ~55 |
| Verified TRUE | ~44 (80%) |
| **Verified FALSE** | **7 (13%)** |
| Gaps (undocumented features) | 4 (7%) |

## False Claims Requiring Fixes

### README.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 24 | "Requires Go 1.21 or later" | `go.mod` specifies `go 1.25.5` | Update to "Go 1.25 or later" |
| 91 | `has:tool` search filter — "Sessions with any tool usage" | Code only supports `has:error` and `has:subagent`/`has:agent`. No `has:tool`. | Remove `has:tool` or implement it; document `has:error` and `has:subagent` instead |
| 107 | `ccvault serve` starts MCP server | Actual CLI command is `ccvault mcp` (`mcpCmd` with `Use: "mcp"`) | Change to `ccvault mcp` |
| 126 | `"args": ["serve"]` in Claude Desktop config | Should be `["mcp"]` | Change to `"args": ["mcp"]` |
| 110-114 | MCP tools: lists only 4 tools (`search_conversations`, `get_session`, `list_projects`, `get_analytics`) | Server actually exposes **8 tools**: `search_conversations`, `get_session_summary`, `get_turns`, `get_session`, `list_sessions`, `list_projects`, `get_stats`, `get_analytics` | Add missing 4 tools to docs |
| 64-79 | Commands table lists 11 commands | Missing `version` and `export` commands which are registered in `init()` | Add `version` and `export` to table |
| 148-161 | Architecture directory tree | Missing `internal/export/` directory (markdown export functionality) | Add `export/` to the tree |

### CLAUDE.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| — | Key Files section lists 6 paths | Missing `internal/config/`, `internal/sync/`, `internal/search/`, `internal/export/` | Add missing directories |

## Verified TRUE Claims

### README.md
- **File references**: All architecture directories exist (`cmd/ccvault/`, `pkg/models/`, `pkg/parser/`, `internal/config/`, `internal/db/`, `internal/sync/`, `internal/search/`, `internal/tui/`, `internal/analytics/`, `internal/mcp/`)
- **Environment variables**: `CCVAULT_CLAUDE_HOME` defaults to `~/.claude`, `CCVAULT_DATA_DIR` defaults to `~/.ccvault` (verified in `config.go`)
- **Data storage**: `~/.ccvault/ccvault.db` confirmed in `db.go:32`
- **Analytics cache**: `~/.ccvault/analytics/sessions.parquet` path confirmed
- **Search syntax**: `project:`, `model:`, `tool:`, `before:`, `after:`, `"exact phrase"` all verified in `query.go`
- **Module path**: `github.com/2389-research/ccvault` matches `go.mod`
- **All 11 listed commands**: Verified in `main.go` `init()` function
- **Dependencies**: Bubble Tea, DuckDB, Parquet, Cobra, Viper all present in `go.mod`

### CLAUDE.md
- **Architecture pipeline**: `~/.claude/projects/<encoded-path>/<uuid>.jsonl → SQLite + FTS5 + Parquet` confirmed
- **Components**: Parser, SQLite+FTS5, DuckDB+Parquet, Bubble Tea, MCP Server all verified
- **Makefile targets**: `make build`, `make test`, `make sync`, `make tui`, `make cache` all exist
- **Key files listed**: All 6 paths exist

## Gaps (Undocumented Features)

| Feature | Location | Recommendation |
|---------|----------|----------------|
| `export` command | `main.go:700-786` | Document in README Commands table |
| `version` command | `main.go:39-45` | Document in README Commands table |
| `file:` search filter | `query.go:43` | Document in README Search Syntax section |
| `has:error`, `has:subagent` filters | `query.go:50-54` | Document in README Search Syntax section |
| MCP prompts (6 prompts) | `server.go:387-433` | Already documented in `mcp` command help text but not in README MCP section |
| `--no-thinking`, `--no-tool-results` export flags | `main.go:899-901` | Document with export command |

## Pattern Summary

| Pattern | Count | Root Cause |
|---------|-------|------------|
| Wrong command name (`serve` vs `mcp`) | 2 | Likely renamed during development, docs not updated |
| Incomplete feature lists | 3 | Features added after initial docs written |
| Stale version requirement | 1 | Go version upgraded, README not updated |

## Human Review Queue

- [ ] Homebrew tap `brew install 2389-research/tap/ccvault` — cannot verify if tap actually exists
- [ ] `go install` path — works syntactically but may need testing
- [ ] FTS5 triggers claim in CLAUDE.md — referenced but not verified in db.go audit scope
