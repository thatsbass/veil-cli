package repl

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thatsbass/veil-cli/internal/api"
	"github.com/thatsbass/veil-cli/internal/config"
)

const version = "v1.0.0"

// model is the BubbleTea REPL state.
type model struct {
	input    textinput.Model
	commands []SlashCommand
	filtered []SlashCommand
	cursor   int
	showMenu bool
	output   string
	quitting bool
	cfg      *config.Config
	client   *api.Client
	ctx      context.Context
	s        styles
}

type styles struct {
	logo     lipgloss.Style
	status   lipgloss.Style
	hint     lipgloss.Style
	menuBox  lipgloss.Style
	item     lipgloss.Style
	selected lipgloss.Style
	cmdName  lipgloss.Style
	cmdDesc  lipgloss.Style
	output   lipgloss.Style
	prompt   lipgloss.Style
}

// func newStyles() styles {
// 	return styles{
// 		logo: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#7C3AED")).
// 			Bold(true),
// 		status: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#9CA3AF")),
// 		hint: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#6B7280")).
// 			Italic(true),
// 		menuBox: lipgloss.NewStyle().
// 			Border(lipgloss.RoundedBorder()).
// 			BorderForeground(lipgloss.Color("#3F3F46")).
// 			Padding(0, 1).
// 			MarginLeft(2),
// 		item: lipgloss.NewStyle().
// 			PaddingLeft(1),
// 		selected: lipgloss.NewStyle().
// 			Background(lipgloss.Color("#1E1B4B")).
// 			Foreground(lipgloss.Color("#E0E7FF")).
// 			PaddingLeft(1),
// 		cmdName: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#A78BFA")).
// 			Width(12),
// 		cmdDesc: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#D1D5DB")),
// 		output: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#9CA3AF")).
// 			MarginLeft(2).
// 			MarginTop(1),
// 		prompt: lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#7C3AED")).
// 			Bold(true),
// 	}
// }


func newStyles() styles {
	return styles{
		logo: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9F8F6")).
			Bold(true),

		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B8B8B8")),

		hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A1A1A1")).
			Italic(true),

		menuBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F5F5F5")).
			Padding(0, 1).
			MarginLeft(2),

		item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EAEAEA")).
			PaddingLeft(1),

		selected: lipgloss.NewStyle().
			Background(lipgloss.Color("#F5F5F5")).
			Foreground(lipgloss.Color("#111111")).
			Bold(true).
			PaddingLeft(1),

		cmdName: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9F8F6")).
			Bold(true).
			Width(12),

		cmdDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CFCFCF")),

		output: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B8B8B8")).
			MarginLeft(2).
			MarginTop(1),

		prompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9F8F6")).
			Bold(true),
	}
}
// New creates an initialised REPL model.
func New(cfg *config.Config) *model {
	ti := textinput.New()
	ti.Placeholder = "Type / for commands"
	ti.Focus()
	ti.CharLimit = 128

	cmds := allCommands()
	return &model{
		input:    ti,
		commands: cmds,
		filtered: cmds,
		cfg:      cfg,
		client:   api.NewClient(cfg.APIURL, cfg.APIKey),
		ctx:      context.Background(),
		s:        newStyles(),
	}
}

// Start launches the BubbleTea program.
func Start(cfg *config.Config) error {
	p := tea.NewProgram(New(cfg))
	_, err := p.Run()
	return err
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case outputMsg:
		m.output = string(msg)
		m.input.SetValue("")
		m.showMenu = false
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.syncMenu()
	return m, cmd
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEsc:
		if m.showMenu {
			m.showMenu = false
			return m, nil
		}

	case tea.KeyUp:
		if m.showMenu && m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.showMenu && m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil

	case tea.KeyTab:
		if m.showMenu && len(m.filtered) > 0 {
			m.cursor = (m.cursor + 1) % len(m.filtered)
		}
		return m, nil

	case tea.KeyEnter:
		if m.showMenu && len(m.filtered) > 0 {
			return m, m.filtered[m.cursor].Execute(m)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.syncMenu()
	return m, cmd
}

func (m *model) syncMenu() {
	val := m.input.Value()
	if !strings.HasPrefix(val, "/") {
		m.showMenu = false
		return
	}
	m.showMenu = true
	m.filtered = filterCommands(m.commands, val)
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

func filterCommands(cmds []SlashCommand, prefix string) []SlashCommand {
	if prefix == "/" {
		return cmds
	}
	var out []SlashCommand
	for _, c := range cmds {
		if strings.HasPrefix(c.Name, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.s.logo.Render(logo))
	b.WriteString("\n")
	b.WriteString(m.s.status.Render(statusLine(m.cfg)))
	b.WriteString("\n")
	b.WriteString(m.s.hint.Render("  Type / for commands · Ctrl+C to exit"))
	b.WriteString("\n\n")

	if m.output != "" {
		b.WriteString(m.s.output.Render(m.output))
		b.WriteString("\n\n")
	}

	b.WriteString(m.s.prompt.Render("❯ "))
	b.WriteString(m.input.View())
	b.WriteString("\n")

	if m.showMenu && len(m.filtered) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderMenu())
	}

	return b.String()
}

func (m *model) renderMenu() string {
	var rows []string
	for i, cmd := range m.filtered {
		name := m.s.cmdName.Render(cmd.Name)
		desc := m.s.cmdDesc.Render(cmd.Description)
		row := fmt.Sprintf("%s  %s", name, desc)
		if i == m.cursor {
			row = m.s.selected.Render(fmt.Sprintf("%s  %s", cmd.Name, cmd.Description))
		} else {
			row = m.s.item.Render(row)
		}
		rows = append(rows, row)
	}
	return m.s.menuBox.Render(strings.Join(rows, "\n"))
}

func statusLine(cfg *config.Config) string {
	if cfg.APIKey == "" {
		return fmt.Sprintf("  %s  ●  not configured — run: veil init", version)
	}
	return fmt.Sprintf("  %s  ●  %s", version, cfg.APIURL)
}

func formatStats(s *api.Stats) string {
	used := float64(s.UsedTokens) / 1_000_000
	quota := float64(s.QuotaTokens) / 1_000_000
	return fmt.Sprintf("%.1fM / %.1fM tokens used (%d%%)  ·  resets %s", used, quota, s.Percent, s.ResetsAt)
}

const logo = `
  ██╗   ██╗███████╗██╗██╗
  ██║   ██║██╔════╝██║██║
  ██║   ██║█████╗  ██║██║
  ╚██╗ ██╔╝██╔══╝  ██║██║
   ╚████╔╝ ███████╗██║███████╗
    ╚═══╝  ╚══════╝╚═╝╚══════╝`
