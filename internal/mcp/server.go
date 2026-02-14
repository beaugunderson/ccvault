// ABOUTME: MCP server for ccvault AI integration
// ABOUTME: Implements JSON-RPC 2.0 over stdio for Model Context Protocol

package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/2389-research/ccvault/internal/analytics"
	"github.com/2389-research/ccvault/internal/config"
	"github.com/2389-research/ccvault/internal/db"
	"github.com/2389-research/ccvault/internal/export"
	"github.com/2389-research/ccvault/internal/search"
	"github.com/2389-research/ccvault/pkg/models"
)

// Server handles MCP protocol communication
type Server struct {
	db       *db.DB
	cfg      *config.Config
	analyzer *analytics.Analyzer
	debug    bool
}

// NewServer creates a new MCP server
func NewServer(database *db.DB, cfg *config.Config) (*Server, error) {
	cacheDir := filepath.Join(cfg.DataDir, "analytics")
	analyzer, err := analytics.NewAnalyzer(cacheDir)
	if err != nil {
		// Analytics not available, continue without it
		analyzer = nil
	}

	return &Server{
		db:       database,
		cfg:      cfg,
		analyzer: analyzer,
		debug:    os.Getenv("CCVAULT_MCP_DEBUG") == "1",
	}, nil
}

// Close cleans up server resources
func (s *Server) Close() error {
	if s.analyzer != nil {
		return s.analyzer.Close()
	}
	return nil
}

func (s *Server) log(format string, args ...interface{}) {
	if s.debug {
		fmt.Fprintf(os.Stderr, "[ccvault-mcp] "+format+"\n", args...)
	}
}

// JSON-RPC 2.0 types
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP types
type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type capabilities struct {
	Tools   *toolsCapability   `json:"tools,omitempty"`
	Prompts *promptsCapability `json:"prompts,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type promptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

type tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type toolsListResult struct {
	Tools []tool `json:"tools"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type toolResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Prompt types
type prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []promptArgument `json:"arguments,omitempty"`
}

type promptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type promptsListResult struct {
	Prompts []prompt `json:"prompts"`
}

type promptGetParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type promptMessage struct {
	Role    string      `json:"role"`
	Content contentItem `json:"content"`
}

type promptGetResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []promptMessage `json:"messages"`
}

// Run starts the MCP server on stdio
func (s *Server) Run() error {
	s.log("Starting MCP server")
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				s.log("EOF received, shutting down")
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		s.log("Received: %s", strings.TrimSpace(string(line)))

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.log("Parse error: %v", err)
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *Server) handleRequest(req *jsonRPCRequest) {
	s.log("Handling method: %s", req.Method)

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
		s.log("Client initialized")
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "prompts/list":
		s.handlePromptsList(req)
	case "prompts/get":
		s.handlePromptsGet(req)
	default:
		s.log("Unknown method: %s", req.Method)
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) handleInitialize(req *jsonRPCRequest) {
	result := initializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: capabilities{
			Tools:   &toolsCapability{},
			Prompts: &promptsCapability{},
		},
		ServerInfo: serverInfo{
			Name:    "ccvault",
			Version: "0.1.0",
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req *jsonRPCRequest) {
	tools := []tool{
		{
			Name:        "search_conversations",
			Description: "Search through Claude Code conversation history. Returns matching turns with context snippets.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"query": {
						Type:        "string",
						Description: "Search query. Supports: project:name, model:opus, tool:Bash, before:2024-01-01, after:2024-01-01, \"exact phrase\"",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum results (default: 20, max: 100)",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "get_session",
			Description: "Retrieve a complete conversation session with all turns, tool usage, and metadata.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"session_id": {
						Type:        "string",
						Description: "Session UUID",
					},
					"format": {
						Type:        "string",
						Description: "Output format",
						Enum:        []string{"json", "markdown"},
					},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "list_sessions",
			Description: "List recent sessions, optionally filtered by project.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"project": {
						Type:        "string",
						Description: "Filter by project path or name (partial match)",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum results (default: 20, max: 100)",
					},
				},
			},
		},
		{
			Name:        "list_projects",
			Description: "List all indexed projects with session counts and token usage.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"sort": {
						Type:        "string",
						Description: "Sort order",
						Enum:        []string{"name", "activity", "tokens", "sessions"},
					},
					"limit": {
						Type:        "number",
						Description: "Maximum results (default: 50)",
					},
				},
			},
		},
		{
			Name:        "get_stats",
			Description: "Get archive statistics: projects, sessions, tokens, models used, date range.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
		{
			Name:        "get_analytics",
			Description: "Get detailed analytics: token usage by day, model breakdown, top projects.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"days": {
						Type:        "number",
						Description: "Days of history (default: 30)",
					},
				},
			},
		},
	}

	s.sendResult(req.ID, toolsListResult{Tools: tools})
}

func (s *Server) handlePromptsList(req *jsonRPCRequest) {
	prompts := []prompt{
		{
			Name:        "summarize_recent",
			Description: "Summarize recent Claude Code activity across all projects",
			Arguments: []promptArgument{
				{Name: "days", Description: "Number of days to summarize (default: 7)", Required: false},
			},
		},
		{
			Name:        "analyze_project",
			Description: "Analyze Claude Code usage patterns for a specific project",
			Arguments: []promptArgument{
				{Name: "project", Description: "Project path or name to analyze", Required: true},
			},
		},
		{
			Name:        "find_solutions",
			Description: "Find past solutions and approaches for a given problem or topic",
			Arguments: []promptArgument{
				{Name: "topic", Description: "Problem or topic to search for", Required: true},
			},
		},
		{
			Name:        "review_session",
			Description: "Review and summarize a specific conversation session",
			Arguments: []promptArgument{
				{Name: "session_id", Description: "Session UUID to review", Required: true},
			},
		},
		{
			Name:        "compare_approaches",
			Description: "Find different approaches used for similar problems across sessions",
			Arguments: []promptArgument{
				{Name: "topic", Description: "Topic or problem type to compare", Required: true},
			},
		},
		{
			Name:        "tool_usage_report",
			Description: "Generate a report on which tools are used most and how",
			Arguments: []promptArgument{
				{Name: "tool", Description: "Specific tool to analyze (optional)", Required: false},
			},
		},
	}

	s.sendResult(req.ID, promptsListResult{Prompts: prompts})
}

func (s *Server) handlePromptsGet(req *jsonRPCRequest) {
	var params promptGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var result promptGetResult
	var err error

	switch params.Name {
	case "summarize_recent":
		result, err = s.promptSummarizeRecent(params.Arguments)
	case "analyze_project":
		result, err = s.promptAnalyzeProject(params.Arguments)
	case "find_solutions":
		result, err = s.promptFindSolutions(params.Arguments)
	case "review_session":
		result, err = s.promptReviewSession(params.Arguments)
	case "compare_approaches":
		result, err = s.promptCompareApproaches(params.Arguments)
	case "tool_usage_report":
		result, err = s.promptToolUsageReport(params.Arguments)
	default:
		s.sendError(req.ID, -32602, "Unknown prompt", params.Name)
		return
	}

	if err != nil {
		s.sendError(req.ID, -32603, "Prompt error", err.Error())
		return
	}

	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsCall(req *jsonRPCRequest) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	s.log("Tool call: %s with args: %v", params.Name, params.Arguments)

	var result interface{}
	var err error

	switch params.Name {
	case "search_conversations":
		result, err = s.searchConversations(params.Arguments)
	case "get_session":
		result, err = s.getSession(params.Arguments)
	case "list_sessions":
		result, err = s.listSessions(params.Arguments)
	case "list_projects":
		result, err = s.listProjects(params.Arguments)
	case "get_stats":
		result, err = s.getStats(params.Arguments)
	case "get_analytics":
		result, err = s.getAnalytics(params.Arguments)
	default:
		s.sendError(req.ID, -32602, "Unknown tool", params.Name)
		return
	}

	if err != nil {
		s.log("Tool error: %v", err)
		s.sendResult(req.ID, toolResult{
			Content: []contentItem{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		})
		return
	}

	// Marshal result to JSON text
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		s.sendResult(req.ID, toolResult{
			Content: []contentItem{{Type: "text", Text: fmt.Sprintf("Error marshaling result: %v", err)}},
			IsError: true,
		})
		return
	}

	s.sendResult(req.ID, toolResult{
		Content: []contentItem{{Type: "text", Text: string(jsonBytes)}},
	})
}

// Tool implementations

func (s *Server) searchConversations(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	parsed := search.Parse(query)
	searcher := search.New(s.db.DB)
	results, err := searcher.Search(parsed, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return map[string]interface{}{
		"count":   len(results),
		"results": results,
	}, nil
}

func (s *Server) getSession(args map[string]interface{}) (interface{}, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	format, _ := args["format"].(string)
	if format == "" {
		format = "json"
	}

	session, err := s.db.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	turns, err := s.db.GetTurns(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get turns: %w", err)
	}

	// Get project info
	var projectPath string
	if session.ProjectID > 0 {
		project, err := s.db.GetProject(session.ProjectID)
		if err == nil && project != nil {
			projectPath = project.Path
		}
	}

	if format == "markdown" {
		var buf strings.Builder
		exporter := export.NewMarkdownExporter()
		if err := exporter.Export(&buf, session, turns, projectPath); err != nil {
			return nil, fmt.Errorf("export failed: %w", err)
		}
		return map[string]interface{}{
			"format":  "markdown",
			"content": buf.String(),
		}, nil
	}

	return map[string]interface{}{
		"session":      session,
		"project_path": projectPath,
		"turns":        turns,
		"turn_count":   len(turns),
	}, nil
}

func (s *Server) listSessions(args map[string]interface{}) (interface{}, error) {
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	var projectID int64
	if projectFilter, ok := args["project"].(string); ok && projectFilter != "" {
		// Find matching project
		projects, err := s.db.GetProjects("activity", 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get projects: %w", err)
		}
		for _, p := range projects {
			if strings.Contains(strings.ToLower(p.Path), strings.ToLower(projectFilter)) ||
				strings.Contains(strings.ToLower(p.DisplayName), strings.ToLower(projectFilter)) {
				projectID = p.ID
				break
			}
		}
		if projectID == 0 {
			return nil, fmt.Errorf("project not found: %s", projectFilter)
		}
	}

	sessions, err := s.db.GetSessions(projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	return map[string]interface{}{
		"count":    len(sessions),
		"sessions": sessions,
	}, nil
}

func (s *Server) listProjects(args map[string]interface{}) (interface{}, error) {
	sortBy := "activity"
	if sb, ok := args["sort"].(string); ok && sb != "" {
		sortBy = sb
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	projects, err := s.db.GetProjects(sortBy, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	return map[string]interface{}{
		"count":    len(projects),
		"projects": projects,
	}, nil
}

func (s *Server) getStats(args map[string]interface{}) (interface{}, error) {
	projectCount, _, err := s.db.GetProjectStats()
	if err != nil {
		return nil, fmt.Errorf("get project stats: %w", err)
	}

	sessionCount, turnCount, totalTokens, err := s.db.GetSessionStats()
	if err != nil {
		return nil, fmt.Errorf("get session stats: %w", err)
	}

	tokensByModel, err := s.db.GetTokensByModel()
	if err != nil {
		return nil, fmt.Errorf("get tokens by model: %w", err)
	}

	firstActivity, lastActivity, _ := s.db.GetFirstAndLastActivity()

	toolStats, _ := s.db.GetToolUsageStats(10)

	result := map[string]interface{}{
		"projects":     projectCount,
		"sessions":     sessionCount,
		"turns":        turnCount,
		"total_tokens": totalTokens,
		"models":       tokensByModel,
		"top_tools":    toolStats,
	}

	if !firstActivity.IsZero() {
		result["first_activity"] = firstActivity.Format(time.RFC3339)
	}
	if !lastActivity.IsZero() {
		result["last_activity"] = lastActivity.Format(time.RFC3339)
	}
	if !firstActivity.IsZero() && !lastActivity.IsZero() {
		result["days_span"] = int(lastActivity.Sub(firstActivity).Hours() / 24)
	}

	return result, nil
}

func (s *Server) getAnalytics(args map[string]interface{}) (interface{}, error) {
	days := 30
	if d, ok := args["days"].(float64); ok {
		days = int(d)
	}

	result := make(map[string]interface{})

	// Get basic stats
	stats, err := s.getStats(nil)
	if err != nil {
		return nil, err
	}
	result["summary"] = stats

	// Try to get DuckDB analytics if available
	if s.analyzer != nil {
		dailyTokens, err := s.analyzer.GetTokensByDay(days)
		if err == nil {
			result["tokens_by_day"] = dailyTokens
		}

		topProjects, err := s.analyzer.GetTopProjects(10)
		if err == nil {
			result["top_projects"] = topProjects
		}

		modelStats, err := s.analyzer.GetTokensByModel()
		if err == nil {
			result["model_breakdown"] = modelStats
		}
	}

	return result, nil
}

// Prompt implementations

func (s *Server) promptSummarizeRecent(args map[string]interface{}) (promptGetResult, error) {
	days := 7
	if d, ok := args["days"].(float64); ok {
		days = int(d)
	}

	// Get recent stats
	stats, err := s.getStats(nil)
	if err != nil {
		return promptGetResult{}, err
	}

	// Get recent sessions
	sessions, err := s.db.GetSessions(0, 20)
	if err != nil {
		return promptGetResult{}, err
	}

	// Build context
	var context strings.Builder
	context.WriteString(fmt.Sprintf("## Claude Code Activity Summary (Last %d Days)\n\n", days))
	context.WriteString("### Archive Statistics\n")
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	context.WriteString("```json\n")
	context.WriteString(string(statsJSON))
	context.WriteString("\n```\n\n")

	context.WriteString("### Recent Sessions\n")
	for _, sess := range sessions {
		context.WriteString(fmt.Sprintf("- **%s** (%s): %d turns, %s model\n",
			sess.ID[:8],
			sess.StartedAt.Format("Jan 2 15:04"),
			sess.TurnCount,
			shortenModel(sess.Model)))
	}

	return promptGetResult{
		Description: fmt.Sprintf("Summary of Claude Code activity over the last %d days", days),
		Messages: []promptMessage{
			{
				Role: "user",
				Content: contentItem{
					Type: "text",
					Text: fmt.Sprintf("Please analyze and summarize my Claude Code usage over the last %d days. Here's the data:\n\n%s\n\nProvide insights on:\n1. Overall usage patterns\n2. Most active projects\n3. Model preferences\n4. Notable trends", days, context.String()),
				},
			},
		},
	}, nil
}

func (s *Server) promptAnalyzeProject(args map[string]interface{}) (promptGetResult, error) {
	projectName, _ := args["project"].(string)
	if projectName == "" {
		return promptGetResult{}, fmt.Errorf("project argument is required")
	}

	// Find project
	projects, err := s.db.GetProjects("activity", 0)
	if err != nil {
		return promptGetResult{}, err
	}

	var project *models.Project
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Path), strings.ToLower(projectName)) ||
			strings.Contains(strings.ToLower(p.DisplayName), strings.ToLower(projectName)) {
			project = &p
			break
		}
	}
	if project == nil {
		return promptGetResult{}, fmt.Errorf("project not found: %s", projectName)
	}

	// Get sessions for this project
	sessions, err := s.db.GetSessions(project.ID, 50)
	if err != nil {
		return promptGetResult{}, err
	}

	var context strings.Builder
	context.WriteString(fmt.Sprintf("## Project Analysis: %s\n\n", project.DisplayName))
	context.WriteString(fmt.Sprintf("- **Path**: %s\n", project.Path))
	context.WriteString(fmt.Sprintf("- **Sessions**: %d\n", project.SessionCount))
	context.WriteString(fmt.Sprintf("- **Total Tokens**: %d\n", project.TotalTokens))
	context.WriteString(fmt.Sprintf("- **First Seen**: %s\n", project.FirstSeenAt.Format("Jan 2, 2006")))
	context.WriteString(fmt.Sprintf("- **Last Activity**: %s\n\n", project.LastActivityAt.Format("Jan 2, 2006")))

	context.WriteString("### Session History\n")
	for _, sess := range sessions {
		context.WriteString(fmt.Sprintf("- %s: %d turns, %s\n",
			sess.StartedAt.Format("Jan 2 15:04"),
			sess.TurnCount,
			shortenModel(sess.Model)))
	}

	return promptGetResult{
		Description: fmt.Sprintf("Analysis of Claude Code usage for project: %s", project.DisplayName),
		Messages: []promptMessage{
			{
				Role: "user",
				Content: contentItem{
					Type: "text",
					Text: fmt.Sprintf("Analyze my Claude Code usage for this project:\n\n%s\n\nProvide insights on:\n1. How I've been using Claude in this project\n2. Common tasks and patterns\n3. Suggestions for more effective usage", context.String()),
				},
			},
		},
	}, nil
}

func (s *Server) promptFindSolutions(args map[string]interface{}) (promptGetResult, error) {
	topic, _ := args["topic"].(string)
	if topic == "" {
		return promptGetResult{}, fmt.Errorf("topic argument is required")
	}

	// Search for relevant conversations
	parsed := search.Parse(topic)
	searcher := search.New(s.db.DB)
	results, err := searcher.Search(parsed, 10)
	if err != nil {
		return promptGetResult{}, err
	}

	var context strings.Builder
	context.WriteString(fmt.Sprintf("## Search Results for: %s\n\n", topic))
	context.WriteString(fmt.Sprintf("Found %d relevant conversations:\n\n", len(results)))

	for i, r := range results {
		context.WriteString(fmt.Sprintf("### Result %d\n", i+1))
		context.WriteString(fmt.Sprintf("- **Session**: %s\n", r.SessionID[:8]))
		context.WriteString(fmt.Sprintf("- **Date**: %s\n", r.Turn.Timestamp.Format("Jan 2, 2006 15:04")))
		context.WriteString(fmt.Sprintf("- **Type**: %s\n", r.Turn.Type))
		context.WriteString(fmt.Sprintf("- **Snippet**: %s\n\n", r.Snippet))
	}

	return promptGetResult{
		Description: fmt.Sprintf("Past solutions and approaches for: %s", topic),
		Messages: []promptMessage{
			{
				Role: "user",
				Content: contentItem{
					Type: "text",
					Text: fmt.Sprintf("I'm looking for past solutions related to: %s\n\nHere are relevant conversations from my Claude Code history:\n\n%s\n\nPlease:\n1. Summarize the approaches used\n2. Identify common patterns\n3. Suggest which sessions might be most helpful to review in detail", topic, context.String()),
				},
			},
		},
	}, nil
}

func (s *Server) promptReviewSession(args map[string]interface{}) (promptGetResult, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return promptGetResult{}, fmt.Errorf("session_id argument is required")
	}

	session, err := s.db.GetSession(sessionID)
	if err != nil || session == nil {
		return promptGetResult{}, fmt.Errorf("session not found: %s", sessionID)
	}

	turns, err := s.db.GetTurns(sessionID)
	if err != nil {
		return promptGetResult{}, err
	}

	// Get project info
	var projectPath string
	if session.ProjectID > 0 {
		project, _ := s.db.GetProject(session.ProjectID)
		if project != nil {
			projectPath = project.Path
		}
	}

	// Export to markdown for easier reading
	var buf strings.Builder
	exporter := export.NewMarkdownExporter(
		export.WithThinking(false), // Skip thinking for summary
	)
	_ = exporter.Export(&buf, session, turns, projectPath)

	return promptGetResult{
		Description: fmt.Sprintf("Review of session %s", sessionID[:8]),
		Messages: []promptMessage{
			{
				Role: "user",
				Content: contentItem{
					Type: "text",
					Text: fmt.Sprintf("Please review and summarize this Claude Code session:\n\n%s\n\nProvide:\n1. A brief summary of what was accomplished\n2. Key decisions made\n3. Any notable tools or techniques used\n4. Lessons learned or improvements for next time", buf.String()),
				},
			},
		},
	}, nil
}

func (s *Server) promptCompareApproaches(args map[string]interface{}) (promptGetResult, error) {
	topic, _ := args["topic"].(string)
	if topic == "" {
		return promptGetResult{}, fmt.Errorf("topic argument is required")
	}

	// Search for relevant conversations
	parsed := search.Parse(topic)
	searcher := search.New(s.db.DB)
	results, err := searcher.Search(parsed, 20)
	if err != nil {
		return promptGetResult{}, err
	}

	// Group by session
	sessionSnippets := make(map[string][]string)
	for _, r := range results {
		sessionSnippets[r.SessionID] = append(sessionSnippets[r.SessionID], r.Snippet)
	}

	var context strings.Builder
	context.WriteString(fmt.Sprintf("## Comparing Approaches for: %s\n\n", topic))
	context.WriteString(fmt.Sprintf("Found %d sessions with relevant content:\n\n", len(sessionSnippets)))

	i := 0
	for sessionID, snippets := range sessionSnippets {
		if i >= 5 {
			break
		}
		session, _ := s.db.GetSession(sessionID)
		context.WriteString(fmt.Sprintf("### Session %d (%s)\n", i+1, sessionID[:8]))
		if session != nil {
			context.WriteString(fmt.Sprintf("Date: %s, Model: %s\n", session.StartedAt.Format("Jan 2"), shortenModel(session.Model)))
		}
		for _, snippet := range snippets {
			context.WriteString(fmt.Sprintf("- %s\n", snippet))
		}
		context.WriteString("\n")
		i++
	}

	return promptGetResult{
		Description: fmt.Sprintf("Comparison of approaches for: %s", topic),
		Messages: []promptMessage{
			{
				Role: "user",
				Content: contentItem{
					Type: "text",
					Text: fmt.Sprintf("Compare the different approaches I've used for: %s\n\n%s\n\nAnalyze:\n1. Different strategies attempted\n2. What worked well vs. what didn't\n3. Evolution of approach over time\n4. Recommended best practices based on past experience", topic, context.String()),
				},
			},
		},
	}, nil
}

func (s *Server) promptToolUsageReport(args map[string]interface{}) (promptGetResult, error) {
	specificTool, _ := args["tool"].(string)

	toolStats, err := s.db.GetToolUsageStats(20)
	if err != nil {
		return promptGetResult{}, err
	}

	var context strings.Builder
	context.WriteString("## Tool Usage Report\n\n")
	context.WriteString("### Tool Frequency\n")
	for tool, count := range toolStats {
		context.WriteString(fmt.Sprintf("- **%s**: %d uses\n", tool, count))
	}

	if specificTool != "" {
		context.WriteString(fmt.Sprintf("\n### Focus: %s\n", specificTool))
		// Could add more detailed analysis for specific tool
	}

	return promptGetResult{
		Description: "Analysis of tool usage patterns",
		Messages: []promptMessage{
			{
				Role: "user",
				Content: contentItem{
					Type: "text",
					Text: fmt.Sprintf("Analyze my Claude Code tool usage:\n\n%s\n\nProvide insights on:\n1. Most relied-upon tools\n2. Tool usage patterns\n3. Suggestions for tools I might be underutilizing\n4. Workflow optimization opportunities", context.String()),
				},
			},
		},
	}, nil
}

// Helper functions

func shortenModel(model string) string {
	if len(model) <= 20 {
		return model
	}
	parts := strings.Split(model, "-")
	if len(parts) >= 2 {
		return parts[1]
	}
	return model[:17] + "..."
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.send(resp)
}

func (s *Server) send(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		s.log("Marshal error: %v", err)
		return
	}
	s.log("Sending: %s", string(data))
	fmt.Println(string(data))
}
