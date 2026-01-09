package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jayteealao/otterstack/internal/compose"
	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
)

// View represents the current view.
type View int

const (
	ViewList View = iota
	ViewDetail
)

// ProjectInfo holds project information for display.
type ProjectInfo struct {
	Project    *state.Project
	Deployment *state.Deployment
	Services   []compose.ServiceStatus
	Error      error
}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	ctx           context.Context
	cancel        context.CancelFunc
	store         *state.Store
	projects      []ProjectInfo
	table         table.Model
	currentView   View
	selectedIndex int
	width         int
	height        int
	refreshTicker time.Duration
	lastRefresh   time.Time
	err           error
	quitting      bool
}

// KeyMap defines the keybindings.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Help    key.Binding
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "details"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// Messages
type tickMsg time.Time
type refreshMsg []ProjectInfo
type errMsg struct{ err error }

// NewModel creates a new TUI model.
func NewModel(ctx context.Context, store *state.Store, refreshInterval time.Duration) Model {
	ctx, cancel := context.WithCancel(ctx)

	columns := []table.Column{
		{Title: "PROJECT", Width: 20},
		{Title: "TYPE", Width: 10},
		{Title: "REF", Width: 15},
		{Title: "STATUS", Width: 15},
		{Title: "SERVICES", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorMuted).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(ColorPrimary).
		Bold(false)
	t.SetStyles(s)

	return Model{
		ctx:           ctx,
		cancel:        cancel,
		store:         store,
		table:         t,
		currentView:   ViewList,
		refreshTicker: refreshInterval,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadProjects(),
		m.tick(),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.cancel() // Cancel ongoing operations
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Refresh):
			return m, m.loadProjects()

		case key.Matches(msg, keys.Enter):
			if m.currentView == ViewList && len(m.projects) > 0 {
				m.selectedIndex = m.table.Cursor()
				m.currentView = ViewDetail
			}
			return m, nil

		case key.Matches(msg, keys.Back):
			if m.currentView == ViewDetail {
				m.currentView = ViewList
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 10)

	case tickMsg:
		return m, tea.Batch(m.loadProjects(), m.tick())

	case refreshMsg:
		m.projects = msg
		m.lastRefresh = time.Now()
		m.updateTable()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	if m.currentView == ViewList {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.currentView {
	case ViewDetail:
		return m.detailView()
	default:
		return m.listView()
	}
}

func (m *Model) listView() string {
	var s string

	// Title
	title := TitleStyle.Render("OtterStack Monitor")
	s += title + "\n\n"

	// Table
	s += m.table.View() + "\n\n"

	// Footer
	lastRefresh := m.lastRefresh.Format("15:04:05")
	footer := HelpStyle.Render(fmt.Sprintf(
		"[↑↓] Navigate  [Enter] Details  [r] Refresh  [q] Quit  |  Last refresh: %s",
		lastRefresh,
	))
	s += footer

	return s
}

func (m *Model) detailView() string {
	if m.selectedIndex >= len(m.projects) {
		return "No project selected"
	}

	info := m.projects[m.selectedIndex]
	project := info.Project

	var s string

	// Title
	title := TitleStyle.Render(fmt.Sprintf("Project: %s", project.Name))
	s += title + "\n\n"

	// Project details
	s += LabelStyle.Render("Type:") + ValueStyle.Render(project.RepoType) + "\n"
	s += LabelStyle.Render("Path:") + ValueStyle.Render(project.RepoPath) + "\n"
	if project.RepoURL != "" {
		s += LabelStyle.Render("URL:") + ValueStyle.Render(project.RepoURL) + "\n"
	}
	s += LabelStyle.Render("Compose:") + ValueStyle.Render(project.ComposeFile) + "\n"
	s += LabelStyle.Render("Status:") + GetStatusStyle(project.Status).Render(project.Status) + "\n"
	s += "\n"

	// Deployment details
	if info.Deployment != nil {
		d := info.Deployment
		s += LabelStyle.Render("Deployment:") + "\n"
		s += "  " + LabelStyle.Render("Commit:") + ValueStyle.Render(git.ShortSHA(d.GitSHA)) + "\n"
		if d.GitRef != "" {
			s += "  " + LabelStyle.Render("Ref:") + ValueStyle.Render(d.GitRef) + "\n"
		}
		s += "  " + LabelStyle.Render("Status:") + GetStatusStyle(d.Status).Render(d.Status) + "\n"
		s += "  " + LabelStyle.Render("Started:") + ValueStyle.Render(d.StartedAt.Format("2006-01-02 15:04:05")) + "\n"
		s += "\n"
	}

	// Services
	if len(info.Services) > 0 {
		s += LabelStyle.Render("Services:") + "\n"
		for _, svc := range info.Services {
			icon := GetStatusIcon(svc.Status)
			style := GetStatusStyle(svc.Status)
			health := ""
			if svc.Health != "" {
				health = fmt.Sprintf(" (%s)", svc.Health)
			}
			s += fmt.Sprintf("  %s %s %s%s\n",
				style.Render(icon),
				ValueStyle.Render(svc.Name),
				style.Render(svc.Status),
				health,
			)
		}
		s += "\n"
	}

	// Footer
	footer := HelpStyle.Render("[Esc] Back  [r] Refresh  [q] Quit")
	s += footer

	return s
}

func (m *Model) updateTable() {
	rows := make([]table.Row, len(m.projects))
	for i, info := range m.projects {
		ref := "-"
		status := info.Project.Status
		services := "-"

		if info.Deployment != nil {
			if info.Deployment.GitRef != "" {
				ref = info.Deployment.GitRef
			} else {
				ref = git.ShortSHA(info.Deployment.GitSHA)
			}
			status = info.Deployment.Status
		}

		if len(info.Services) > 0 {
			running := 0
			for _, s := range info.Services {
				if compose.IsServiceRunning(s.Status) {
					running++
				}
			}
			services = fmt.Sprintf("%d/%d up", running, len(info.Services))
		}

		statusIcon := GetStatusIcon(status)

		rows[i] = table.Row{
			info.Project.Name,
			info.Project.RepoType,
			ref,
			statusIcon + " " + status,
			services,
		}
	}
	m.table.SetRows(rows)
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(m.refreshTicker, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		ctx := m.ctx

		projects, err := m.store.ListProjects(ctx)
		if err != nil {
			return errMsg{err}
		}

		infos := make([]ProjectInfo, len(projects))
		for i, p := range projects {
			info := ProjectInfo{Project: p}

			// Get active deployment
			deployment, err := m.store.GetActiveDeployment(ctx, p.ID)
			if err != nil && !errors.Is(err, apperrors.ErrNoActiveDeployment) {
				info.Error = err
			} else if deployment != nil {
				info.Deployment = deployment

				// Get service status
				projectName := compose.GenerateProjectName(p.Name, git.ShortSHA(deployment.GitSHA))
				services, _ := compose.GetProjectStatus(ctx, projectName)
				info.Services = services
			}

			infos[i] = info
		}

		return refreshMsg(infos)
	}
}
