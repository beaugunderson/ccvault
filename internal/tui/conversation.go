// ABOUTME: Conversation view for ccvault TUI
// ABOUTME: Shows a session's conversation with scrolling

package tui

import (
	"fmt"
	"strings"

	"github.com/2389-research/ccvault/internal/db"
	"github.com/2389-research/ccvault/pkg/models"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ConversationModel holds conversation view state
type ConversationModel struct {
	db        *db.DB
	width     int
	height    int
	sessionID string
	session   *models.Session
	turns     []models.Turn
	viewport  viewport.Model
	loading   bool
	ready     bool
}

// NewConversationModel creates a new conversation model
func NewConversationModel(database *db.DB) *ConversationModel {
	return &ConversationModel{
		db:      database,
		loading: true,
	}
}

// SetSession sets the session to display
func (m *ConversationModel) SetSession(sessionID string) {
	m.sessionID = sessionID
	m.ready = false
}

// Init loads conversation data
func (m *ConversationModel) Init() tea.Cmd {
	return m.loadConversation
}

func (m *ConversationModel) loadConversation() tea.Msg {
	session, err := m.db.GetSession(m.sessionID)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	turns, err := m.db.GetTurns(m.sessionID)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	return conversationLoadedMsg{session: session, turns: turns}
}

type conversationLoadedMsg struct {
	session *models.Session
	turns   []models.Turn
}

// Update handles conversation view events
func (m *ConversationModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case conversationLoadedMsg:
		m.session = msg.session
		m.turns = msg.turns
		m.loading = false
		m.initViewport()
		return nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			m.viewport.LineUp(1)
		case key.Matches(msg, keys.Down):
			m.viewport.LineDown(1)
		case key.Matches(msg, keys.PageUp):
			m.viewport.HalfViewUp()
		case key.Matches(msg, keys.PageDown):
			m.viewport.HalfViewDown()
		case key.Matches(msg, keys.Refresh):
			m.loading = true
			return m.loadConversation
		}
	}
	return nil
}

func (m *ConversationModel) initViewport() {
	headerHeight := 5
	footerHeight := 2
	viewportHeight := m.height - headerHeight - footerHeight
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	m.viewport = viewport.New(m.width, viewportHeight)
	m.viewport.SetContent(m.renderConversation())
	m.ready = true
}

// SetSize sets the viewport size
func (m *ConversationModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.ready {
		m.initViewport()
	}
}

// View renders the conversation
func (m *ConversationModel) View() string {
	if m.loading {
		return titleStyle.Render("Loading conversation...")
	}

	var b strings.Builder

	// Header
	if m.session != nil {
		b.WriteString(titleStyle.Render("Conversation"))
		b.WriteString("\n")
		meta := fmt.Sprintf("%s • %d turns • %s",
			m.session.StartedAt.Format("2006-01-02 15:04"),
			len(m.turns),
			m.session.Model)
		b.WriteString(subtitleStyle.Render(meta))
		b.WriteString("\n\n")
	}

	// Viewport
	if m.ready {
		b.WriteString(m.viewport.View())
	}

	// Footer
	b.WriteString("\n")
	scrollPercent := 0
	if m.ready {
		scrollPercent = int(m.viewport.ScrollPercent() * 100)
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("↑/↓: scroll • pgup/pgdn: page • %d%% • esc: back", scrollPercent)))

	return b.String()
}

// renderConversation creates the formatted conversation content
func (m *ConversationModel) renderConversation() string {
	var b strings.Builder

	for _, t := range m.turns {
		switch t.Type {
		case "user":
			b.WriteString(userStyle.Render("[USER]"))
			b.WriteString(" ")
			b.WriteString(timestampStyle.Render(t.Timestamp.Format("15:04:05")))
			b.WriteString("\n")
			content := wrapText(t.Content, m.width-4)
			b.WriteString(contentStyle.Render(content))
			b.WriteString("\n\n")

		case "assistant":
			b.WriteString(assistantStyle.Render("[ASSISTANT]"))
			b.WriteString(" ")
			b.WriteString(timestampStyle.Render(t.Timestamp.Format("15:04:05")))
			b.WriteString("\n")
			content := t.Content
			if len(content) > 2000 {
				content = content[:2000] + "\n... (truncated)"
			}
			content = wrapText(content, m.width-4)
			b.WriteString(contentStyle.Render(content))
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Simple word wrapping
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			}
		}
		result.WriteString(currentLine)
	}

	return result.String()
}
