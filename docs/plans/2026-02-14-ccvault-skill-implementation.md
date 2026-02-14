# ccvault Skill Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a multi-file skill that teaches Claude instances how to effectively mine conversation history via the ccvault MCP server.

**Architecture:** Four markdown files in `skills/ccvault/`: a core strategy guide (SKILL.md), two workflow prompts (orient-prompt.md, recall-prompt.md), and a tool reference card (reference.md). The skill auto-triggers for proactive research and provides explicit workflows for orientation and recall.

**Tech Stack:** Markdown skill files following superpowers skill format (YAML frontmatter + markdown body).

---

### Task 1: Create Skills Directory

**Files:**
- Create: `skills/ccvault/` directory

**Step 1: Create the directory structure**

Run: `mkdir -p /Users/harper/Public/src/2389/ccvault/skills/ccvault`

**Step 2: Verify directory exists**

Run: `ls -la /Users/harper/Public/src/2389/ccvault/skills/ccvault/`
Expected: Empty directory listing

**Step 3: Commit**

```bash
cd /Users/harper/Public/src/2389/ccvault
git add skills/
git commit -m "feat: add skills directory for ccvault skill"
```

---

### Task 2: Write reference.md — Tool & Query Reference Card

**Files:**
- Create: `skills/ccvault/reference.md`

This is the foundation that SKILL.md and the prompts will reference. Write it first so the strategy guide can point to it.

**Step 1: Write reference.md**

The file should contain these sections in this order:

1. **Tools Quick Reference** — Table format with columns: Tool, Required Params, Optional Params, Returns, Notes. All 8 tools:
   - `search_conversations` — query (string, required), limit (number, default 10, max 50), offset (number). Returns paginated results with snippets (200 chars). Check `has_more`/`next_offset`.
   - `get_session_summary` — session_id (string, required). Returns metadata, turn counts, tool usage, first/last user messages. Most cost-effective entry point for any session.
   - `get_turns` — session_id (required), offset (number), limit (number, default 20, max 50), type (enum: user/assistant/tool_result). Returns paginated turns with content (truncated at 1000 chars).
   - `get_session` — session_id (required). Returns full markdown. WARNING: returns warning for 100+ turn sessions, truncates at 50K chars.
   - `list_sessions` — project (string, partial match), limit (number, default 20, max 100). Returns recent sessions sorted by date.
   - `list_projects` — sort (enum: name/activity/tokens/sessions, default activity), limit (number, default 50). Returns projects with session counts and token usage.
   - `get_stats` — no params. Returns archive-wide statistics.
   - `get_analytics` — days (number, default 30). Returns stats + daily token breakdown, top projects, model breakdown. Requires DuckDB analytics cache.

2. **Search Query Syntax** — Table of operators:
   - `project:name` — filter by project path/name
   - `model:name` — filter by model (e.g., `model:opus`, `model:sonnet`)
   - `tool:Name` — filter by tool used (e.g., `tool:Bash`, `tool:Edit`)
   - `file:path` — filter by file path mentioned
   - `before:DATE` — filter before date
   - `after:DATE` — filter after date
   - `has:error` — sessions containing errors
   - `has:subagent` or `has:agent` — sessions using subagents
   - `"exact phrase"` — quoted exact phrase matching
   - Free text — FTS5 full-text search on remaining terms

3. **Date Format Examples** — Quick list:
   - `2026-01-15`, `2026/01/15`, `01/15/2026`
   - `Jan 15, 2026`, `January 15, 2026`
   - Relative: `today`, `yesterday`, `week`/`thisweek`, `month`/`thismonth`

4. **MCP Prompts** — Table of 6 available prompts: summarize_recent, analyze_project, find_solutions, review_session, compare_approaches, tool_usage_report. Include required args for each.

5. **Common Query Recipes** — Copy-paste ready examples:
   - Find errors in a project: `project:myapp has:error`
   - Recent Bash usage: `tool:Bash after:thisweek`
   - Find discussions about a library: `"react-query" project:frontend`
   - All subagent usage: `has:agent after:month`
   - Find file-specific work: `file:auth.py project:backend`

6. **Staleness Note** — If a query or tool call fails unexpectedly, check against the actual MCP server tool descriptions which are the source of truth.

**Step 2: Verify the file renders correctly**

Run: `wc -l /Users/harper/Public/src/2389/ccvault/skills/ccvault/reference.md`
Expected: 80-120 lines

**Step 3: Commit**

```bash
cd /Users/harper/Public/src/2389/ccvault
git add skills/ccvault/reference.md
git commit -m "feat: add ccvault skill reference card with tool/query docs"
```

---

### Task 3: Write SKILL.md — Core Strategy Guide

**Files:**
- Create: `skills/ccvault/SKILL.md`

**Step 1: Write SKILL.md**

The file must start with this exact frontmatter:

```yaml
---
name: ccvault
description: Use when searching past conversations, recalling previous solutions, understanding project history, or needing context from earlier sessions - proactive conversation history mining via ccvault MCP server
---
```

Then these sections:

1. **Overview** (2-3 sentences) — ccvault indexes Claude Code conversation history into a searchable archive. This skill teaches effective patterns for mining that history — finding past solutions, understanding project context, and learning from previous sessions.

2. **When This Fires** — Bullet list of trigger conditions:
   - You need to understand what was previously done in a project
   - An error or problem might have been solved in a prior session
   - The user asks "what did I do", "how did we solve", "where did I leave off"
   - You're about to implement something and want to check for prior art
   - You need cross-project patterns or approaches

3. **Commands** — Two explicit workflows:
   - `Orient` — Session-start context gathering. See `./orient-prompt.md`
   - `Recall` — On-demand historical lookup. See `./recall-prompt.md`

4. **The Search Playbook** — This is the strategy-heavy core. Include these patterns as a decision flowchart (graphviz):

   ```dot
   digraph search_strategy {
     "Need past context" -> "Know the session?";
     "Know the session?" -> "get_session_summary" [label="yes"];
     "Know the session?" -> "Know the project?" [label="no"];
     "Know the project?" -> "list_sessions project:X" [label="yes"];
     "Know the project?" -> "search_conversations" [label="no"];
     "list_sessions project:X" -> "get_session_summary (top 2-3)";
     "search_conversations" -> "Promising hits?";
     "Promising hits?" -> "get_session_summary" [label="yes"];
     "Promising hits?" -> "Broaden search" [label="no"];
     "Broaden search" -> "search_conversations" [label="remove a filter"];
     "get_session_summary" -> "Need details?";
     "Need details?" -> "get_turns with type filter" [label="yes"];
     "Need details?" -> "Done — synthesize" [label="no"];
   }
   ```

5. **Five Search Patterns** — Table format:

   | Pattern | When | Steps |
   |---------|------|-------|
   | Summary before deep dive | Any session lookup | `get_session_summary` → only `get_turns` if needed |
   | Narrow first, broaden later | Searching for specifics | Start with all filters, remove one at a time |
   | Project scan | Orienting in a project | `list_sessions project:X` → summarize top 2-3 |
   | Solution mining | Problem you've seen before | `search_conversations "error text"` → verify with summary |
   | Cross-project learning | Looking for patterns | Search without project filter, group by project |

6. **Efficiency Rules** — Numbered list:
   1. `get_session_summary` is always your first stop for any session — never go to `get_turns` or `get_session` first
   2. `search_conversations` returns 200-char snippets — enough to judge relevance without burning context
   3. Use `get_turns` with `type: "user"` to see just what the human asked, or `type: "assistant"` for just responses
   4. Always check `has_more` / `next_offset` — don't stop at page 1
   5. Never use `get_session` for sessions with 100+ turns — it returns a warning instead of content
   6. Combine operators to narrow before free-text: `project:X tool:Bash "deploy"` beats just `"deploy"`

7. **Anti-Patterns** — What NOT to do:
   - Fetching full sessions when summaries suffice (wastes context window)
   - Broad free-text searches without operators (returns noise)
   - Ignoring pagination (missing the best results on page 2+)
   - Using `get_session` for large sessions (100+ turns — server returns warning, not content)
   - Searching for MCP tool names like `mcp__ccvault__search_conversations` — the database stores tool names as they appear in Claude Code logs (e.g., `Bash`, `Read`, `Edit`)

8. **Quick Reference** — Point to `./reference.md` for full tool params and query syntax.

**Step 2: Verify structure and length**

Run: `wc -l /Users/harper/Public/src/2389/ccvault/skills/ccvault/SKILL.md`
Expected: 120-200 lines

**Step 3: Commit**

```bash
cd /Users/harper/Public/src/2389/ccvault
git add skills/ccvault/SKILL.md
git commit -m "feat: add ccvault skill core strategy guide"
```

---

### Task 4: Write orient-prompt.md — Session Start Orientation

**Files:**
- Create: `skills/ccvault/orient-prompt.md`

**Step 1: Write orient-prompt.md**

This is the workflow prompt for session-start orientation. Structure:

1. **Purpose** — Quickly understand what's been happening in the current project before starting work.

2. **Workflow** — Numbered steps with exact tool calls:

   Step 1: Get archive scope
   ```
   get_stats
   ```
   → Provides total sessions, projects, token usage. Gives you a sense of the archive size.

   Step 2: Find recent sessions in this project
   ```
   list_sessions project:<current_project_name> limit:5
   ```
   → Use the current working directory name or project identifier. Returns most recent sessions.

   Step 3: Summarize top sessions
   ```
   get_session_summary session_id:<id_1>
   get_session_summary session_id:<id_2>
   ```
   → Call in parallel for the 2-3 most recent sessions. Each returns: what was worked on (first/last user message), tools used, turn count, model, git branch.

   Step 4: Synthesize
   → Combine the summaries into a brief orientation:
   - What was recently worked on
   - Which tools were used most
   - What the last user message was (indicates where work left off)
   - Any in-progress patterns (high turn count in recent session = likely deep work)

3. **Output Format** — Example of what the orientation should look like:
   ```
   **Project Orient: <project_name>**
   - Last session: <date> — <first_user_msg summary> (<turn_count> turns, <model>)
   - Previous: <date> — <first_user_msg summary>
   - Recent tools: <top 3 tools>
   - Left off: <last_user_msg of most recent session>
   ```

4. **Tips**:
   - If no sessions found for the project, try a broader project name (partial match works)
   - For projects with many sessions (20+), focus on the last 3 — don't try to summarize everything
   - The `first_user_msg` field is truncated to 500 chars — it's the best indicator of session purpose

**Step 2: Verify**

Run: `wc -l /Users/harper/Public/src/2389/ccvault/skills/ccvault/orient-prompt.md`
Expected: 50-80 lines

**Step 3: Commit**

```bash
cd /Users/harper/Public/src/2389/ccvault
git add skills/ccvault/orient-prompt.md
git commit -m "feat: add ccvault orient workflow for session-start context"
```

---

### Task 5: Write recall-prompt.md — On-Demand Recall

**Files:**
- Create: `skills/ccvault/recall-prompt.md`

**Step 1: Write recall-prompt.md**

This is the workflow for finding past solutions and decisions. Structure:

1. **Purpose** — Find specific past solutions, decisions, or patterns from conversation history when the user (or agent) needs them.

2. **Trigger Examples** — When to use this:
   - "How did I solve the CORS issue last time?"
   - "Find that database migration pattern we used"
   - "What approach did we take for authentication?"
   - Agent needs prior art before implementing something similar

3. **Workflow** — Numbered steps:

   Step 1: Parse intent
   → Extract from the request:
   - Keywords (error messages, library names, patterns)
   - Project hints (if mentioned or inferrable from context)
   - Time hints ("last week", "in December", "recently")

   Step 2: Construct search query
   → Build query with operators:
   ```
   search_conversations query:"<keywords>" [project:<project>] [after:<date>] [tool:<tool>]
   ```
   → Start narrow (more operators), broaden if no results.

   Step 3: Evaluate results
   → For each promising hit (based on snippet relevance):
   ```
   get_session_summary session_id:<hit_session_id>
   ```
   → Verify the session is actually about the topic, not a false positive.

   Step 4: Extract the solution
   → For confirmed matches:
   ```
   get_turns session_id:<session_id> type:assistant offset:<relevant_area>
   ```
   → Or if the session is small (<100 turns):
   ```
   get_session session_id:<session_id>
   ```

   Step 5: Present findings
   → Always include:
   - Session ID (for future reference)
   - Date and project context
   - The actual solution/decision extracted
   - Confidence level (exact match vs. related topic)

4. **Search Narrowing Strategy** — Waterfall:
   ```
   Round 1: project:X tool:Y "exact error message"
   Round 2: project:X "error message"          (drop tool filter)
   Round 3: "error message"                     (drop project filter)
   Round 4: "key phrase from error"             (simplify search terms)
   ```

5. **When Multiple Results Found** — Group by session, present most recent first. Note if the same problem was solved differently across sessions (evolution of approach).

6. **When No Results Found** — Report honestly. Suggest:
   - Different search terms
   - Checking if ccvault has been synced recently (`get_stats` — check `last_activity`)
   - The solution might predate the ccvault archive

**Step 2: Verify**

Run: `wc -l /Users/harper/Public/src/2389/ccvault/skills/ccvault/recall-prompt.md`
Expected: 60-100 lines

**Step 3: Commit**

```bash
cd /Users/harper/Public/src/2389/ccvault
git add skills/ccvault/recall-prompt.md
git commit -m "feat: add ccvault recall workflow for on-demand history lookup"
```

---

### Task 6: Final Commit & Verify

**Files:**
- Verify: all 4 files in `skills/ccvault/`

**Step 1: Verify all files exist and have content**

Run: `ls -la /Users/harper/Public/src/2389/ccvault/skills/ccvault/ && wc -l /Users/harper/Public/src/2389/ccvault/skills/ccvault/*.md`

Expected: 4 files (SKILL.md, reference.md, orient-prompt.md, recall-prompt.md), total ~350-500 lines.

**Step 2: Verify SKILL.md frontmatter is valid**

Run: `head -5 /Users/harper/Public/src/2389/ccvault/skills/ccvault/SKILL.md`
Expected: Lines 1-3 should be `---`, `name: ccvault`, `description: Use when...`, `---`

**Step 3: Verify cross-references work**

Run: `grep -n '\./.*\.md' /Users/harper/Public/src/2389/ccvault/skills/ccvault/SKILL.md`
Expected: References to `./orient-prompt.md`, `./recall-prompt.md`, `./reference.md`

**Step 4: Final commit if any unstaged changes remain**

```bash
cd /Users/harper/Public/src/2389/ccvault
git add skills/ccvault/
git commit -m "feat: complete ccvault skill with strategy guide, workflows, and reference"
```
