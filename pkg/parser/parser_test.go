// ABOUTME: Tests for JSONL parser functionality
// ABOUTME: Validates parsing of Claude Code conversation formats

package parser

import (
	"strings"
	"testing"
	"time"
)

func TestParseTurn_UserMessage(t *testing.T) {
	input := `{"parentUuid":"85e5053c-9b1a-4d3a-aebe-ac044bd6170e","isSidechain":false,"userType":"external","cwd":"/Users/harper/project","sessionId":"0684b40f-4463-4492-83c6-3baa18bfb9ad","version":"2.1.29","gitBranch":"main","type":"user","message":{"role":"user","content":"Hello, can you help me?"},"uuid":"abc12345-1234-1234-1234-123456789abc","timestamp":"2026-02-02T20:48:10.345Z"}`

	turn, err := ParseTurn([]byte(input))
	if err != nil {
		t.Fatalf("ParseTurn failed: %v", err)
	}

	if turn == nil {
		t.Fatal("Expected turn, got nil")
	}

	if turn.ID != "abc12345-1234-1234-1234-123456789abc" {
		t.Errorf("Expected ID abc12345-1234-1234-1234-123456789abc, got %s", turn.ID)
	}

	if turn.Type != "user" {
		t.Errorf("Expected type user, got %s", turn.Type)
	}

	if turn.SessionID != "0684b40f-4463-4492-83c6-3baa18bfb9ad" {
		t.Errorf("Expected session ID 0684b40f-4463-4492-83c6-3baa18bfb9ad, got %s", turn.SessionID)
	}

	if turn.Content != "Hello, can you help me?" {
		t.Errorf("Expected content 'Hello, can you help me?', got %s", turn.Content)
	}

	if turn.ParentID != "85e5053c-9b1a-4d3a-aebe-ac044bd6170e" {
		t.Errorf("Expected parent ID 85e5053c-9b1a-4d3a-aebe-ac044bd6170e, got %s", turn.ParentID)
	}
}

func TestParseTurn_AssistantMessage(t *testing.T) {
	input := `{"type":"assistant","uuid":"def67890-1234-1234-1234-123456789abc","sessionId":"session123","timestamp":"2026-02-02T20:49:00.000Z","message":{"model":"claude-opus-4-5-20251101","id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"I can help you with that!"}],"usage":{"input_tokens":100,"output_tokens":50}}}`

	turn, err := ParseTurn([]byte(input))
	if err != nil {
		t.Fatalf("ParseTurn failed: %v", err)
	}

	if turn.Type != "assistant" {
		t.Errorf("Expected type assistant, got %s", turn.Type)
	}

	if turn.Content != "I can help you with that!" {
		t.Errorf("Expected content 'I can help you with that!', got %s", turn.Content)
	}

	if turn.InputTokens != 100 {
		t.Errorf("Expected 100 input tokens, got %d", turn.InputTokens)
	}

	if turn.OutputTokens != 50 {
		t.Errorf("Expected 50 output tokens, got %d", turn.OutputTokens)
	}
}

func TestParseTurn_SkipsNonTurns(t *testing.T) {
	// file-history-snapshot entries should be skipped
	input := `{"type":"file-history-snapshot","messageId":"123","snapshot":{}}`

	turn, err := ParseTurn([]byte(input))
	if err != nil {
		t.Fatalf("ParseTurn failed: %v", err)
	}

	if turn != nil {
		t.Error("Expected nil for file-history-snapshot, got turn")
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			input:    "2026-02-02T20:48:10.345Z",
			expected: time.Date(2026, 2, 2, 20, 48, 10, 345000000, time.UTC),
			wantErr:  false,
		},
		{
			input:    "2026-02-02T20:48:10Z",
			expected: time.Date(2026, 2, 2, 20, 48, 10, 0, time.UTC),
			wantErr:  false,
		},
		{
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseTimestamp(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !result.Equal(tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestParseSessionReader(t *testing.T) {
	input := `{"type":"file-history-snapshot","messageId":"msg1","snapshot":{}}
{"uuid":"turn1","sessionId":"session1","type":"user","timestamp":"2026-02-02T20:00:00.000Z","message":{"role":"user","content":"Hello"}}
{"uuid":"turn2","sessionId":"session1","type":"assistant","timestamp":"2026-02-02T20:01:00.000Z","message":{"model":"claude-opus-4-5-20251101","role":"assistant","content":[{"type":"text","text":"Hi there!"}],"usage":{"input_tokens":10,"output_tokens":5}}}`

	turns, session, err := ParseSessionReader(strings.NewReader(input), "/test/session.jsonl")
	if err != nil {
		t.Fatalf("ParseSessionReader failed: %v", err)
	}

	if len(turns) != 2 {
		t.Errorf("Expected 2 turns, got %d", len(turns))
	}

	if session.ID != "session1" {
		t.Errorf("Expected session ID session1, got %s", session.ID)
	}

	if session.TurnCount != 2 {
		t.Errorf("Expected turn count 2, got %d", session.TurnCount)
	}

	if session.Model != "claude-opus-4-5-20251101" {
		t.Errorf("Expected model claude-opus-4-5-20251101, got %s", session.Model)
	}

	if session.InputTokens != 10 {
		t.Errorf("Expected 10 input tokens, got %d", session.InputTokens)
	}

	if session.SourceFile != "/test/session.jsonl" {
		t.Errorf("Expected source file /test/session.jsonl, got %s", session.SourceFile)
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"0684b40f-4463-4492-83c6-3baa18bfb9ad", true},
		{"ABC12345-1234-1234-1234-123456789ABC", true},
		{"not-a-uuid", false},
		{"0684b40f44634492-83c6-3baa18bfb9ad", false}, // Wrong dash positions
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := isValidUUID(tc.input)
			if result != tc.valid {
				t.Errorf("isValidUUID(%s) = %v, want %v", tc.input, result, tc.valid)
			}
		})
	}
}

func TestDecodeProjectPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-Users-harper-project", "/Users/harper/project"},
		{"-Users-harper-Public-src-2389-ccvault", "/Users/harper/Public/src/2389/ccvault"},
		{"simple", "simple"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := decodeProjectPath(tc.input)
			if result != tc.expected {
				t.Errorf("decodeProjectPath(%s) = %s, want %s", tc.input, result, tc.expected)
			}
		})
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/harper/Public/src/2389/ccvault", "src/2389/ccvault"},
		{"/short/path", "/short/path"},
		{"/a/b/c/d/e", "c/d/e"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := GetDisplayName(tc.input)
			if result != tc.expected {
				t.Errorf("GetDisplayName(%s) = %s, want %s", tc.input, result, tc.expected)
			}
		})
	}
}
