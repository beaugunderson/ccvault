# Orient: Session-Start Context Gathering

Quickly understand what's been happening in the current project before starting work. This is the "where did we leave off?" workflow. Run this when you start a session in a project you haven't touched recently, or any time you need to rebuild context.

## Workflow

### Step 1: Get archive scope

```
get_stats
```

Confirms ccvault is populated and gives you a sense of the archive size: total sessions, projects, token usage, and date range. If the archive is empty, stop here -- there's no history to orient from.

### Step 2: Find recent sessions in this project

```
list_sessions project:<current_project_name> limit:5
```

Use the current working directory name or project identifier. The `project` filter does case-insensitive partial matching on both path and display name, so `project:myapp` matches `/home/user/projects/myapp/backend`. Returns sessions sorted by most recent first.

### Step 3: Summarize top sessions

```
get_session_summary session_id:<id_1>
get_session_summary session_id:<id_2>
get_session_summary session_id:<id_3>
```

Call in parallel for the 2-3 most recent sessions. Each summary returns:
- **First/last user messages** -- what was asked and where work left off (500 chars each)
- **Tools used** -- what kind of work was happening (top 10 tools ranked by usage)
- **Turn count and model** -- how deep the session went
- **Git branch** -- what branch was active

### Step 4: Synthesize

Combine the summaries into a brief orientation. Look for:
- What was recently worked on (from first user messages)
- Which tools dominated (lots of Bash = infrastructure/ops, lots of Edit = refactoring, lots of Grep/Read = investigation)
- Where work left off (last user message of most recent session)
- Patterns across sessions (high turn count = deep work, branch changes = context switching)

## Output Format

Present the orientation in this structure:

```
**Project Orient: <project_name>**
- Last session: <date> -- <first_user_msg summary> (<turn_count> turns, <model>)
- Previous: <date> -- <first_user_msg summary>
- Recent tools: <top 3 tools across sessions>
- Left off: <last_user_msg of most recent session>
```

Keep it concise. The goal is a 4-6 line briefing, not a report.

## Tips

- **No results?** Try a broader project name. Partial match is forgiving: `list_sessions project:api` matches `/path/to/my-api-server`.
- **Too many sessions?** Focus on the last 3. Don't try to summarize a 50-session backlog.
- **First user message** is the best indicator of session purpose -- it's what the human asked for when they started.
- **Last user message** shows where work was interrupted or left off -- use this to pick up the thread.
- **Branch divergence**: if `git_branch` differs between recent sessions, note which branches were active. This often signals parallel workstreams or a feature branch that was merged.
- **Use `list_projects`** if you're unsure of the exact project name. Sort by activity to find the most relevant match.
