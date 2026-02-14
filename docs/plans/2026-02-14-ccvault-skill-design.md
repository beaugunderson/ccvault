# ccvault Skill Design

## Overview

A strategy-heavy skill that teaches Claude instances how to effectively mine conversation history via the ccvault MCP server. Ships with the ccvault repo so any ccvault user gets it automatically.

## Audience

General ccvault users — no user-specific assumptions.

## Behavior Model

- **Proactive research companion (automatic):** Fires whenever the agent needs past context — searching for prior solutions, understanding project history, debugging familiar errors, or orienting in a project.
- **Session-start orientation (explicit command):** `/ccvault-orient` — pulls recent session context for the current project.
- **On-demand recall (explicit command):** `/ccvault-recall` — finds past solutions, decisions, and patterns.

## File Structure

```
skills/
  ccvault/
    SKILL.md              # Core strategy guide + auto-trigger behavior
    orient-prompt.md      # Session-start orientation workflow
    recall-prompt.md      # On-demand "find past solutions" workflow
    reference.md          # Full tool/query syntax reference
```

## SKILL.md — Core Strategy Guide

### Frontmatter

```yaml
---
name: ccvault
description: Use when searching past conversations, recalling previous solutions, understanding project history, or needing context from earlier sessions - proactive conversation history mining via ccvault MCP server
---
```

### Trigger Conditions

The skill fires when the agent is:
- Looking for how something was done before
- Starting work on a project and needs context
- Debugging something that might have been solved previously
- Trying to understand patterns across the codebase's history
- The user asks anything about "what did I/we do", "how did we solve", "where did I leave off"

### Search Strategy Patterns

| Pattern | When | How |
|---------|------|-----|
| Summary before deep dive | Any session lookup | `get_session_summary` first, then `get_turns` only if needed |
| Narrow first, broaden later | Searching for something specific | Start with `project:X tool:Y "exact phrase"`, remove filters one at a time |
| Project scan | Starting work on a project | `list_sessions project:X` → summarize most recent 2-3 sessions |
| Solution mining | Solving a problem you've seen before | `search_conversations "error message"` or `search_conversations "library_name pattern"` |
| Cross-project learning | Wondering if a pattern exists elsewhere | Search without project filter, group results by project |

### Anti-patterns

- Fetching full sessions (`get_session`) when summaries would do — wastes context window
- Searching with broad terms when operators would narrow fast
- Not paginating — missing results because you stopped at the first page
- Reading old session transcripts line-by-line instead of using search
- Using `get_session` for large sessions (100+ turns) — use `get_session_summary` + `get_turns` with pagination

### Efficiency Guidelines

- `get_session_summary` is the most cost-effective entry point for any session
- `search_conversations` returns snippets (200 chars) — enough to decide relevance before diving deeper
- `get_turns` with type filter (e.g., `type: "user"`) dramatically reduces noise
- Always check `has_more` / `next_offset` in paginated results

## orient-prompt.md — Session Start Orientation

Workflow when invoked:

1. `get_stats` → understand archive scope
2. `list_sessions project:<current>` → find recent sessions in this project
3. `get_session_summary` for top 2-3 sessions → understand what was done recently
4. Synthesize into a brief "here's where things stand" summary

Output: A concise paragraph telling the user what was recently worked on, what tools were used, and any in-progress work detected.

## recall-prompt.md — On-Demand Recall

Workflow when the user asks "how did I solve X" or "find that thing I did with Y":

1. Parse intent → extract keywords, project hints, time hints
2. `search_conversations` with operators → narrow results
3. For promising hits: `get_session_summary` → verify relevance
4. For confirmed matches: `get_turns` with offset → extract the actual solution
5. Present findings with session ID and approximate timestamp

Key: always present session ID and timestamp so the user can go back to the full conversation.

## reference.md — Quick Reference

Compact reference covering:
- All 8 tools with params and return shapes
- Full query operator syntax table
- Date format examples
- The 6 MCP prompts
- Common query recipes (copy-paste ready)
- Staleness note: check against actual MCP server if something doesn't work

## Design Decisions

1. **Multi-file over single file** — cleaner separation of strategy vs. workflows vs. reference. Each file stays focused and maintainable.
2. **Strategy-heavy** — the value isn't in listing tools (the MCP server already describes them), it's in teaching effective search patterns and when to reach for which tool.
3. **Proactive by default** — the skill auto-fires during implementation work, not just when explicitly asked. This is the primary use case.
4. **Ships with ccvault** — users who install ccvault get the skill automatically. No separate plugin install.
5. **reference.md acknowledged as stale-prone** — includes self-documentation about checking against the actual server.
