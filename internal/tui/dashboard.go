// ABOUTME: Dashboard view for ccvault TUI
// ABOUTME: Shows summary statistics and navigation options

package tui

import (
	"fmt"
	"strings"

	"github.com/2389-research/ccvault/internal/db"
	"github.com/2389-research/ccvault/pkg/models"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DashboardModel holds dashboard state
type DashboardModel struct {
	db           *db.DB
	width        int
	height       int
	stats        *Stats
	topProjects  []models.Project
	selectedItem int
	loading      bool
}

// Stats holds dashboard statistics
type Stats struct {
	Projects    int
	Sessions    int
	Turns       int
	TotalTokens int64
}

// Menu items
var dashboardItems = []string{
	"Browse Projects",
	"Recent Sessions",
	"Search",
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(database *db.DB) *DashboardModel {
	return &DashboardModel{
		db:      database,
		loading: true,
	}
}

// Init loads dashboard data
func (m *DashboardModel) Init() tea.Cmd {
	return m.loadStats
}

// loadStats fetches statistics from the database
func (m *DashboardModel) loadStats() tea.Msg {
	projectCount, _, err := m.db.GetProjectStats()
	if err != nil {
		return ErrorMsg{Err: err}
	}

	sessionCount, turnCount, totalTokens, err := m.db.GetSessionStats()
	if err != nil {
		return ErrorMsg{Err: err}
	}

	projects, err := m.db.GetProjects("activity", 5)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	return statsLoadedMsg{
		stats: &Stats{
			Projects:    projectCount,
			Sessions:    sessionCount,
			Turns:       turnCount,
			TotalTokens: totalTokens,
		},
		topProjects: projects,
	}
}

type statsLoadedMsg struct {
	stats       *Stats
	topProjects []models.Project
}

// Update handles dashboard events
func (m *DashboardModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case statsLoadedMsg:
		m.stats = msg.stats
		m.topProjects = msg.topProjects
		m.loading = false
		return nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.selectedItem > 0 {
				m.selectedItem--
			}
		case key.Matches(msg, keys.Down):
			if m.selectedItem < len(dashboardItems)-1 {
				m.selectedItem++
			}
		case key.Matches(msg, keys.Enter):
			return m.selectItem()
		case key.Matches(msg, keys.Refresh):
			m.loading = true
			return m.loadStats
		}
	}
	return nil
}

// selectItem handles menu selection
func (m *DashboardModel) selectItem() tea.Cmd {
	switch m.selectedItem {
	case 0: // Browse Projects
		return func() tea.Msg {
			return NavigateMsg{View: ProjectsView}
		}
	case 1: // Recent Sessions
		return func() tea.Msg {
			return NavigateMsg{View: SessionsView}
		}
	case 2: // Search
		return func() tea.Msg {
			return NavigateMsg{View: SearchView}
		}
	}
	return nil
}

// SetSize sets the viewport size
func (m *DashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the dashboard
func (m *DashboardModel) View() string {
	if m.loading {
		return titleStyle.Render("Loading...")
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("ccvault"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Claude Code Conversation Archive"))
	b.WriteString("\n\n")

	// Stats box
	if m.stats != nil {
		statsBox := m.renderStats()
		b.WriteString(statsBox)
		b.WriteString("\n\n")
	}

	// Menu
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Navigation"))
	b.WriteString("\n")
	for i, item := range dashboardItems {
		if i == m.selectedItem {
			b.WriteString(selectedStyle.Render("▶ " + item))
		} else {
			b.WriteString(normalStyle.Render("  " + item))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Top projects
	if len(m.topProjects) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Projects"))
		b.WriteString("\n")
		for _, p := range m.topProjects {
			name := p.DisplayName
			if len(name) > 40 {
				name = "..." + name[len(name)-37:]
			}
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %-42s %3d sessions", name, p.SessionCount)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • r: refresh • q: quit"))

	return b.String()
}

// renderStats creates the statistics display
func (m *DashboardModel) renderStats() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("%s %s",
		statLabelStyle.Render("Projects:"),
		statValueStyle.Render(fmt.Sprintf("%d", m.stats.Projects))))

	lines = append(lines, fmt.Sprintf("%s %s",
		statLabelStyle.Render("Sessions:"),
		statValueStyle.Render(fmt.Sprintf("%d", m.stats.Sessions))))

	lines = append(lines, fmt.Sprintf("%s %s",
		statLabelStyle.Render("Turns:"),
		statValueStyle.Render(fmt.Sprintf("%d", m.stats.Turns))))

	lines = append(lines, fmt.Sprintf("%s %s",
		statLabelStyle.Render("Total Tokens:"),
		formatTokens(m.stats.TotalTokens)))

	return boxStyle.Render(strings.Join(lines, "\n"))
}
