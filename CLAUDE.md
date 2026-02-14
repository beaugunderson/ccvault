# ccvault - Claude Code Conversation Vault

## Names
- **AI**: VaultMaster 3000
- **Human**: The Harp-Dawg Supreme

## Project Overview

ccvault is a Go application that archives Claude Code conversations from `~/.claude` for offline search, analytics, and MCP integration. Based on [msgvault](https://github.com/wesm/msgvault).

## Architecture

```
~/.claude/projects/<encoded-path>/<uuid>.jsonl  →  SQLite + FTS5 + Parquet
```

Key components:
- **Parser**: Reads JSONL session files from Claude Code
- **SQLite + FTS5**: Full-text search with triggers
- **DuckDB + Parquet**: Fast analytics queries
- **Bubble Tea**: Terminal UI
- **MCP Server**: AI integration via JSON-RPC

## Development

```bash
# Build
make build

# Run tests
make test

# Sync and explore
make sync
make tui

# Build analytics cache
make cache
```

## Key Files

- `cmd/ccvault/main.go` - CLI entry point
- `pkg/parser/` - JSONL parsing
- `internal/db/` - SQLite + FTS5 layer
- `internal/tui/` - Terminal UI
- `internal/mcp/` - MCP server
- `internal/analytics/` - DuckDB/Parquet

## Important Notes

- Session files are JSONL with nested message structures
- Project paths in Claude are URL-encoded directory names
- FTS5 uses triggers to auto-sync with turns table
- Incremental sync tracks file modification times
