// ABOUTME: Main entry point for ccvault CLI application
// ABOUTME: Initializes and executes the root command via Cobra

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/2389-research/ccvault/internal/config"
	"github.com/2389-research/ccvault/internal/db"
	"github.com/2389-research/ccvault/internal/search"
	"github.com/2389-research/ccvault/internal/sync"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "ccvault",
	Short: "Archive and search Claude Code conversations",
	Long: `ccvault archives your Claude Code conversation history for offline
search, analytics, and AI integration.

Similar to msgvault for email, ccvault provides:
  - Full-text search across all conversations
  - Interactive TUI for drill-down analytics
  - MCP server for AI assistant integration`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ccvault %s\n", version)
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync conversations from Claude Code",
	Long:  `Scan ~/.claude and index new or updated sessions into the ccvault database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		full, _ := cmd.Flags().GetBool("full")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Ensure data directory exists
		if err := config.EnsureDataDir(cfg); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}

		// Open database
		database, err := db.Open(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer database.Close()

		// Create syncer
		syncer := sync.New(database, cfg.ClaudeHome,
			sync.WithFullSync(full),
			sync.WithVerbose(verbose),
			sync.WithProgressCallback(func(msg string) {
				fmt.Println(msg)
			}),
		)

		// Run sync
		stats, err := syncer.Run()
		if err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		// Print summary
		fmt.Println()
		fmt.Printf("Sync completed in %s\n", stats.Duration.Round(time.Millisecond))
		fmt.Printf("  Projects:  %d\n", stats.ProjectsFound)
		fmt.Printf("  Sessions:  %d indexed, %d skipped\n", stats.SessionsIndexed, stats.SessionsSkipped)
		fmt.Printf("  Turns:     %d\n", stats.TurnsIndexed)
		fmt.Printf("  Tool uses: %d\n", stats.ToolUsesIndexed)

		if len(stats.Errors) > 0 {
			fmt.Printf("  Errors:    %d\n", len(stats.Errors))
			if verbose {
				for _, e := range stats.Errors {
					fmt.Printf("    - %v\n", e)
				}
			}
		}

		return nil
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	Long:  `Open the interactive terminal UI for browsing and analyzing conversations.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement TUI
		fmt.Println("TUI not yet implemented")
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search conversations",
	Long: `Full-text search across all archived conversations.

Supports Gmail-like query syntax:
  project:name     Filter by project
  model:opus       Filter by model
  tool:Bash        Sessions using specific tool
  before:date      Date filters
  after:date
  "exact phrase"   Exact match`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		queryStr := strings.Join(args, " ")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		limit, _ := cmd.Flags().GetInt("limit")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		database, err := db.Open(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer database.Close()

		// Parse and execute search
		query := search.Parse(queryStr)
		searcher := search.New(database.DB)
		results, err := searcher.Search(query, limit)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		}

		// Pretty print results
		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		fmt.Printf("Found %d results:\n\n", len(results))
		for i, r := range results {
			fmt.Printf("%d. [%s] %s\n", i+1, r.Turn.Type, r.Turn.Timestamp.Format("2006-01-02 15:04"))
			fmt.Printf("   Project: %s\n", r.ProjectPath)
			if r.Model != "" {
				fmt.Printf("   Model: %s\n", r.Model)
			}
			fmt.Printf("   %s\n\n", r.Snippet)
		}

		return nil
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show archive statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		database, err := db.Open(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer database.Close()

		// Get statistics
		projectCount, projectTokens, err := database.GetProjectStats()
		if err != nil {
			return fmt.Errorf("get project stats: %w", err)
		}

		sessionCount, turnCount, sessionTokens, err := database.GetSessionStats()
		if err != nil {
			return fmt.Errorf("get session stats: %w", err)
		}

		first, last, err := database.GetFirstAndLastActivity()
		if err != nil {
			return fmt.Errorf("get activity range: %w", err)
		}

		toolStats, err := database.GetToolUsageStats(10)
		if err != nil {
			return fmt.Errorf("get tool stats: %w", err)
		}

		tokensByModel, err := database.GetTokensByModel()
		if err != nil {
			return fmt.Errorf("get tokens by model: %w", err)
		}

		// Print statistics
		fmt.Println("ccvault Archive Statistics")
		fmt.Println("==========================")
		fmt.Println()
		fmt.Printf("Projects:      %d\n", projectCount)
		fmt.Printf("Sessions:      %d\n", sessionCount)
		fmt.Printf("Turns:         %d\n", turnCount)
		fmt.Printf("Total Tokens:  %s\n", formatTokens(sessionTokens))
		fmt.Println()

		if !first.IsZero() && !last.IsZero() {
			fmt.Printf("Date Range:    %s to %s\n", first.Format("2006-01-02"), last.Format("2006-01-02"))
			fmt.Printf("Duration:      %d days\n", int(last.Sub(first).Hours()/24))
			fmt.Println()
		}

		if len(tokensByModel) > 0 {
			fmt.Println("Tokens by Model:")
			for model, tokens := range tokensByModel {
				shortModel := model
				if len(shortModel) > 30 {
					shortModel = shortModel[:30] + "..."
				}
				fmt.Printf("  %-35s %s\n", shortModel, formatTokens(tokens))
			}
			fmt.Println()
		}

		if len(toolStats) > 0 {
			fmt.Println("Top Tools:")
			for tool, count := range toolStats {
				fmt.Printf("  %-20s %d uses\n", tool, count)
			}
		}

		// Also print _ tokens from project stats if different (shouldn't be, but just in case)
		_ = projectTokens

		return nil
	},
}

var listProjectsCmd = &cobra.Command{
	Use:   "list-projects",
	Short: "List all indexed projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		sortBy, _ := cmd.Flags().GetString("sort")
		limit, _ := cmd.Flags().GetInt("limit")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		database, err := db.Open(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer database.Close()

		projects, err := database.GetProjects(sortBy, limit)
		if err != nil {
			return fmt.Errorf("get projects: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(projects)
		}

		if len(projects) == 0 {
			fmt.Println("No projects found. Run 'ccvault sync' first.")
			return nil
		}

		fmt.Printf("%-50s %8s %10s %12s\n", "PROJECT", "SESSIONS", "TOKENS", "LAST ACTIVE")
		fmt.Println(strings.Repeat("-", 85))
		for _, p := range projects {
			name := p.DisplayName
			if len(name) > 48 {
				name = "..." + name[len(name)-45:]
			}
			lastActive := p.LastActivityAt.Format("2006-01-02")
			fmt.Printf("%-50s %8d %10s %12s\n", name, p.SessionCount, formatTokens(p.TotalTokens), lastActive)
		}

		return nil
	},
}

var listSessionsCmd = &cobra.Command{
	Use:   "list-sessions",
	Short: "List sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		projectFilter, _ := cmd.Flags().GetString("project")
		limit, _ := cmd.Flags().GetInt("limit")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		database, err := db.Open(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer database.Close()

		// Get project ID if filter specified
		var projectID int64
		if projectFilter != "" {
			project, err := database.GetProjectByPath(projectFilter)
			if err != nil {
				return fmt.Errorf("get project: %w", err)
			}
			if project == nil {
				// Try partial match
				projects, err := database.GetProjects("activity", 0)
				if err != nil {
					return fmt.Errorf("get projects: %w", err)
				}
				for _, p := range projects {
					if strings.Contains(strings.ToLower(p.Path), strings.ToLower(projectFilter)) ||
						strings.Contains(strings.ToLower(p.DisplayName), strings.ToLower(projectFilter)) {
						projectID = p.ID
						break
					}
				}
				if projectID == 0 {
					return fmt.Errorf("project not found: %s", projectFilter)
				}
			} else {
				projectID = project.ID
			}
		}

		sessions, err := database.GetSessions(projectID, limit)
		if err != nil {
			return fmt.Errorf("get sessions: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(sessions)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		fmt.Printf("%-38s %16s %6s %10s %s\n", "SESSION ID", "STARTED", "TURNS", "TOKENS", "MODEL")
		fmt.Println(strings.Repeat("-", 100))
		for _, s := range sessions {
			model := s.Model
			if len(model) > 25 {
				model = model[:22] + "..."
			}
			tokens := s.InputTokens + s.OutputTokens
			fmt.Printf("%-38s %16s %6d %10s %s\n",
				s.ID,
				s.StartedAt.Format("2006-01-02 15:04"),
				s.TurnCount,
				formatTokens(tokens),
				model,
			)
		}

		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show [session-id]",
	Short: "Show a specific session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		database, err := db.Open(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer database.Close()

		session, err := database.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session not found: %s", sessionID)
		}

		turns, err := database.GetTurns(sessionID)
		if err != nil {
			return fmt.Errorf("get turns: %w", err)
		}

		if jsonOutput {
			result := map[string]interface{}{
				"session": session,
				"turns":   turns,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		// Pretty print conversation
		fmt.Printf("Session: %s\n", session.ID)
		fmt.Printf("Model: %s\n", session.Model)
		fmt.Printf("Started: %s\n", session.StartedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Turns: %d\n", len(turns))
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		for _, t := range turns {
			switch t.Type {
			case "user":
				fmt.Printf("[USER] %s\n", t.Timestamp.Format("15:04:05"))
				fmt.Println(t.Content)
				fmt.Println()
			case "assistant":
				fmt.Printf("[ASSISTANT] %s\n", t.Timestamp.Format("15:04:05"))
				content := t.Content
				if len(content) > 500 {
					content = content[:500] + "\n... (truncated)"
				}
				fmt.Println(content)
				fmt.Println()
			}
		}

		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server",
	Long:  `Start the MCP server for AI assistant integration.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		fmt.Printf("Starting MCP server on port %d...\n", port)
		// TODO: Implement MCP server
		fmt.Println("MCP server not yet implemented")
	},
}

var buildCacheCmd = &cobra.Command{
	Use:   "build-cache",
	Short: "Rebuild analytics cache",
	Long:  `Rebuild the Parquet analytics cache for fast DuckDB queries.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement build-cache
		fmt.Println("Build cache not yet implemented")
	},
}

func init() {
	// Add commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(listProjectsCmd)
	rootCmd.AddCommand(listSessionsCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(buildCacheCmd)

	// Sync flags
	syncCmd.Flags().Bool("full", false, "Force full rescan instead of incremental")
	syncCmd.Flags().BoolP("verbose", "v", false, "Show verbose output")

	// Search flags
	searchCmd.Flags().Bool("json", false, "Output results as JSON")
	searchCmd.Flags().Int("limit", 20, "Maximum number of results")

	// List flags
	listProjectsCmd.Flags().Bool("json", false, "Output as JSON")
	listProjectsCmd.Flags().String("sort", "activity", "Sort by: name, activity, tokens, sessions")
	listProjectsCmd.Flags().Int("limit", 50, "Maximum number of results")
	listSessionsCmd.Flags().Bool("json", false, "Output as JSON")
	listSessionsCmd.Flags().String("project", "", "Filter by project")
	listSessionsCmd.Flags().Int("limit", 50, "Maximum number of results")

	// Show flags
	showCmd.Flags().Bool("json", false, "Output as JSON")

	// Serve flags
	serveCmd.Flags().Int("port", 8765, "Port for MCP server")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// formatTokens formats a token count for display
func formatTokens(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
