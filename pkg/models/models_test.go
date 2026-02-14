// ABOUTME: Tests for the models package
// ABOUTME: Verifies data model behavior and computed fields

package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSession_TotalTokens(t *testing.T) {
	tests := []struct {
		name     string
		session  Session
		expected int64
	}{
		{
			name:     "all zeros",
			session:  Session{},
			expected: 0,
		},
		{
			name: "only input tokens",
			session: Session{
				InputTokens: 1000,
			},
			expected: 1000,
		},
		{
			name: "input and output",
			session: Session{
				InputTokens:  1000,
				OutputTokens: 2000,
			},
			expected: 3000,
		},
		{
			name: "all token types",
			session: Session{
				InputTokens:      1000,
				OutputTokens:     2000,
				CacheReadTokens:  500,
				CacheWriteTokens: 300,
			},
			expected: 3800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.TotalTokens()
			if result != tt.expected {
				t.Errorf("TotalTokens() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestProject_JSONSerialization(t *testing.T) {
	project := Project{
		ID:             1,
		Path:           "/path/to/project",
		DisplayName:    "my-project",
		FirstSeenAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		LastActivityAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		SessionCount:   10,
		TotalTokens:    50000,
	}

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Failed to marshal project: %v", err)
	}

	var parsed Project
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal project: %v", err)
	}

	if parsed.ID != project.ID {
		t.Errorf("ID mismatch: %d vs %d", parsed.ID, project.ID)
	}
	if parsed.Path != project.Path {
		t.Errorf("Path mismatch: %s vs %s", parsed.Path, project.Path)
	}
	if parsed.DisplayName != project.DisplayName {
		t.Errorf("DisplayName mismatch: %s vs %s", parsed.DisplayName, project.DisplayName)
	}
	if parsed.SessionCount != project.SessionCount {
		t.Errorf("SessionCount mismatch: %d vs %d", parsed.SessionCount, project.SessionCount)
	}
	if parsed.TotalTokens != project.TotalTokens {
		t.Errorf("TotalTokens mismatch: %d vs %d", parsed.TotalTokens, project.TotalTokens)
	}
}

func TestSession_JSONSerialization(t *testing.T) {
	session := Session{
		ID:           "test-uuid",
		ProjectID:    1,
		StartedAt:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		EndedAt:      time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		Model:        "claude-opus-4",
		GitBranch:    "main",
		TurnCount:    25,
		InputTokens:  1000,
		OutputTokens: 2000,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	var parsed Session
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	if parsed.ID != session.ID {
		t.Errorf("ID mismatch: %s vs %s", parsed.ID, session.ID)
	}
	if parsed.Model != session.Model {
		t.Errorf("Model mismatch: %s vs %s", parsed.Model, session.Model)
	}
	if parsed.GitBranch != session.GitBranch {
		t.Errorf("GitBranch mismatch: %s vs %s", parsed.GitBranch, session.GitBranch)
	}
	if parsed.TurnCount != session.TurnCount {
		t.Errorf("TurnCount mismatch: %d vs %d", parsed.TurnCount, session.TurnCount)
	}
}

func TestTurn_JSONSerialization(t *testing.T) {
	turn := Turn{
		ID:        "turn-uuid",
		SessionID: "session-uuid",
		ParentID:  "parent-uuid",
		Type:      "assistant",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Content:   "Hello world",
	}

	data, err := json.Marshal(turn)
	if err != nil {
		t.Fatalf("Failed to marshal turn: %v", err)
	}

	var parsed Turn
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal turn: %v", err)
	}

	if parsed.ID != turn.ID {
		t.Errorf("ID mismatch: %s vs %s", parsed.ID, turn.ID)
	}
	if parsed.Type != turn.Type {
		t.Errorf("Type mismatch: %s vs %s", parsed.Type, turn.Type)
	}
	if parsed.Content != turn.Content {
		t.Errorf("Content mismatch: %s vs %s", parsed.Content, turn.Content)
	}
}

func TestTokenUsage_JSONSerialization(t *testing.T) {
	usage := TokenUsage{
		InputTokens:              1000,
		OutputTokens:             2000,
		CacheCreationInputTokens: 500,
		CacheReadInputTokens:     300,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal token usage: %v", err)
	}

	var parsed TokenUsage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal token usage: %v", err)
	}

	if parsed.InputTokens != usage.InputTokens {
		t.Errorf("InputTokens mismatch: %d vs %d", parsed.InputTokens, usage.InputTokens)
	}
	if parsed.OutputTokens != usage.OutputTokens {
		t.Errorf("OutputTokens mismatch: %d vs %d", parsed.OutputTokens, usage.OutputTokens)
	}
	if parsed.CacheCreationInputTokens != usage.CacheCreationInputTokens {
		t.Errorf("CacheCreationInputTokens mismatch: %d vs %d", parsed.CacheCreationInputTokens, usage.CacheCreationInputTokens)
	}
	if parsed.CacheReadInputTokens != usage.CacheReadInputTokens {
		t.Errorf("CacheReadInputTokens mismatch: %d vs %d", parsed.CacheReadInputTokens, usage.CacheReadInputTokens)
	}
}

func TestHistoryEntry_JSONSerialization(t *testing.T) {
	entry := HistoryEntry{
		Display:   "search query",
		Timestamp: 1705312800000, // Jan 15, 2024
		Project:   "/path/to/project",
		SessionID: "session-uuid",
		PastedContents: map[string]string{
			"file1.txt": "content1",
		},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal history entry: %v", err)
	}

	var parsed HistoryEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal history entry: %v", err)
	}

	if parsed.Display != entry.Display {
		t.Errorf("Display mismatch: %s vs %s", parsed.Display, entry.Display)
	}
	if parsed.Timestamp != entry.Timestamp {
		t.Errorf("Timestamp mismatch: %d vs %d", parsed.Timestamp, entry.Timestamp)
	}
	if parsed.Project != entry.Project {
		t.Errorf("Project mismatch: %s vs %s", parsed.Project, entry.Project)
	}
	if parsed.SessionID != entry.SessionID {
		t.Errorf("SessionID mismatch: %s vs %s", parsed.SessionID, entry.SessionID)
	}
}

func TestAssistantContent_Types(t *testing.T) {
	tests := []struct {
		name    string
		content AssistantContent
	}{
		{
			name: "text content",
			content: AssistantContent{
				Type: "text",
				Text: "Hello world",
			},
		},
		{
			name: "thinking content",
			content: AssistantContent{
				Type:     "thinking",
				Thinking: "Let me think about this...",
			},
		},
		{
			name: "tool_use content",
			content: AssistantContent{
				Type:  "tool_use",
				ID:    "tool-id",
				Name:  "Bash",
				Input: json.RawMessage(`{"command":"ls -la"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed AssistantContent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.Type != tt.content.Type {
				t.Errorf("Type mismatch: %s vs %s", parsed.Type, tt.content.Type)
			}
		})
	}
}
