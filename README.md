# ccvault

Archive and search your Claude Code conversation history.

Inspired by [msgvault](https://github.com/wesm/msgvault), ccvault provides offline search, analytics, and AI integration for Claude Code sessions stored in `~/.claude`.

## Features

- **Full-text search** across all conversations with Gmail-like query syntax
- **Interactive TUI** for browsing and analyzing sessions
- **DuckDB analytics** for fast aggregate queries over Parquet exports
- **MCP server** for AI assistant integration

## Installation

### Homebrew (macOS/Linux)

```bash
brew install 2389-research/tap/ccvault
```

### Using `go install`

Requires Go 1.21 or later:

```bash
go install github.com/2389-research/ccvault/cmd/ccvault@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `PATH`.

### Build from source

```bash
git clone https://github.com/2389-research/ccvault.git
cd ccvault
go build -o ccvault ./cmd/ccvault
sudo mv ccvault /usr/local/bin/
```

### Verify installation

```bash
ccvault version
```

## Quick Start

```bash
# Sync conversations from ~/.claude
ccvault sync

# Launch interactive TUI
ccvault tui

# Search conversations
ccvault search "debugging async"
ccvault search "project:myapp model:opus"

# View statistics
ccvault stats
```

## Commands

| Command | Description |
|---------|-------------|
| `sync` | Sync conversations from Claude Code |
| `tui` | Launch interactive terminal UI |
| `search [query]` | Full-text search across conversations |
| `stats` | Show archive statistics |
| `list-projects` | List all indexed projects |
| `list-sessions` | List sessions (optionally filtered by project) |
| `show [session-id]` | Display a specific session |
| `build-cache` | Build Parquet analytics cache |
| `serve` | Start MCP server for AI integration |

## Search Syntax

ccvault supports Gmail-like query syntax:

```
project:name     Filter by project path/name
model:opus       Filter by model (partial match)
tool:Bash        Sessions using specific tool
before:date      Sessions before date (YYYY-MM-DD)
after:date       Sessions after date
has:tool         Sessions with any tool usage
"exact phrase"   Exact phrase match
```

Examples:
```bash
ccvault search "debugging the API endpoint"
ccvault search "project:myapp after:2024-01-01"
ccvault search "tool:Edit model:opus"
ccvault search '"error handling" project:backend'
```

## MCP Server

ccvault includes an MCP (Model Context Protocol) server for AI assistant integration.

```bash
ccvault serve
```

Available tools:
- `search_conversations` - Full-text search across conversations
- `get_session` - Retrieve a specific session with all turns
- `list_projects` - List all indexed projects
- `get_analytics` - Get usage analytics summary

### Claude Desktop Configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "ccvault": {
      "command": "ccvault",
      "args": ["serve"]
    }
  }
}
```

## Configuration

ccvault uses sensible defaults but can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `CCVAULT_CLAUDE_HOME` | `~/.claude` | Claude Code data directory |
| `CCVAULT_DATA_DIR` | `~/.ccvault` | ccvault data directory |

## Data Storage

- **SQLite database**: `~/.ccvault/ccvault.db` - Session data with FTS5 full-text search
- **Analytics cache**: `~/.ccvault/analytics/sessions.parquet` - Parquet export for DuckDB queries

## Architecture

```
ccvault/
├── cmd/ccvault/     # CLI entry point
├── pkg/
│   ├── models/      # Data structures
│   └── parser/      # JSONL session parser
└── internal/
    ├── config/      # Configuration
    ├── db/          # SQLite layer with FTS5
    ├── sync/        # Incremental sync logic
    ├── search/      # Query parsing and execution
    ├── tui/         # Bubble Tea terminal UI
    ├── analytics/   # DuckDB/Parquet export
    └── mcp/         # MCP server
```

## License

MIT
