// ABOUTME: Main entry point for ccvault CLI application
// ABOUTME: Initializes and executes the root command via Cobra

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/2389-research/ccvault/internal/config"
	"github.com/2389-research/ccvault/internal/db"
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
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")
		limit, _ := cmd.Flags().GetInt("limit")
		fmt.Printf("Searching for: %s (json=%v, limit=%d)\n", query, jsonOutput, limit)
		// TODO: Implement search
		fmt.Println("Search not yet implemented")
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show archive statistics",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement stats
		fmt.Println("Stats not yet implemented")
	},
}

var listProjectsCmd = &cobra.Command{
	Use:   "list-projects",
	Short: "List all indexed projects",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement list-projects
		fmt.Println("List projects not yet implemented")
	},
}

var listSessionsCmd = &cobra.Command{
	Use:   "list-sessions",
	Short: "List sessions",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement list-sessions
		fmt.Println("List sessions not yet implemented")
	},
}

var showCmd = &cobra.Command{
	Use:   "show [session-id]",
	Short: "Show a specific session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sessionID := args[0]
		fmt.Printf("Showing session: %s\n", sessionID)
		// TODO: Implement show
		fmt.Println("Show not yet implemented")
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
	listSessionsCmd.Flags().Bool("json", false, "Output as JSON")
	listSessionsCmd.Flags().String("project", "", "Filter by project")

	// Serve flags
	serveCmd.Flags().Int("port", 8765, "Port for MCP server")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
