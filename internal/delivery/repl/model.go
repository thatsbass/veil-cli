package repl

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thatsbass/veil-cli/internal/adapter/api"
	"github.com/thatsbass/veil-cli/internal/adapter/config"
)

// Version is the current release version, referenced by the CLI help text.
const Version = "v1.0.0"

// cfgField represents a single editable row in the /config editor.
type cfgField struct {
	key      string
	label    string
	value    string
	editable bool
}

// model is the root BubbleTea state for the interactive REPL.
type model struct {
	input    textinput.Model
	commands []SlashCommand
	filtered []SlashCommand
	cursor   int
	showMenu bool
	output   string
	quitting bool
	cfg      *config.Config
	repo     config.Repository
	client   *api.Client
	ctx      context.Context
	s        styles

	// Log streaming state, protected by mu.
	mu         sync.Mutex
	streaming  bool
	logsCh     chan string
	logsCancel context.CancelFunc

	// Inline config editor state.
	showConfig bool
	cfgFields  []cfgField
	cfgCursor  int
	cfgEditing bool
	cfgInput   textinput.Model
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

// New builds a REPL model wired to the given config and repository.
func New(cfg *config.Config, repo config.Repository) *model {
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
		repo:     repo,
		client:   api.NewClient(cfg.APIURL, cfg.APIKey),
		ctx:      context.Background(),
		s:        newStyles(),
	}
}

// Start launches the BubbleTea event loop with the given config and storage.
func Start(cfg *config.Config, repo config.Repository) error {
	p := tea.NewProgram(New(cfg, repo))
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

	case logEventMsg:
		m.output = api.FormatLogEvent(string(msg))
		m.mu.Lock()
		still := m.streaming
		ch := m.logsCh
		m.mu.Unlock()
		if still {
			return m, readNextLog(ch)
		}
		return m, nil

	case logDoneMsg:
		m.stopStreaming()
		if msg.err == nil {
			m.output = "Log stream ended."
		} else {
			m.output = "Log stream error: " + msg.err.Error()
		}
		return m, nil
	}

	// config editor gets its own input updates
	if m.showConfig && m.cfgEditing {
		var cmd tea.Cmd
		m.cfgInput, cmd = m.cfgInput.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.syncMenu()
	return m, cmd
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showConfig {
		return m.handleConfigKey(msg)
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		m.mu.Lock()
		if m.streaming && m.logsCancel != nil {
			m.logsCancel()
		}
		m.mu.Unlock()
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEsc:
		m.mu.Lock()
		streaming := m.streaming
		m.mu.Unlock()
		if streaming {
			m.stopStreaming()
			m.output = "Log streaming stopped."
			return m, nil
		}
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

// ── Streaming helpers (concurrency-safe) ──

func (m *model) startStreaming(ch chan string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streaming = true
	m.logsCh = ch
	m.logsCancel = cancel
}

func (m *model) stopStreaming() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logsCancel != nil {
		m.logsCancel()
	}
	m.streaming = false
	m.logsCancel = nil
	m.logsCh = nil
}

func (m *model) isStreaming() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.streaming
}

// ── Inline config editor ──

func (m *model) openConfigEditor() {
	n := 12
	if len(m.cfg.APIKey) < n {
		n = len(m.cfg.APIKey)
	}
	maskedKey := ""
	if n > 0 {
		maskedKey = m.cfg.APIKey[:n] + "… (read-only)"
	} else {
		maskedKey = "(not set — run: veil auth login)"
	}

	m.cfgFields = []cfgField{
		{key: "api_url", label: "api_url", value: m.cfg.APIURL, editable: true},
		{key: "api_key", label: "api_key", value: maskedKey, editable: false},
	}
	m.cfgCursor = 0
	m.cfgEditing = false
	m.showConfig = true

	m.cfgInput = textinput.New()
	m.cfgInput.CharLimit = 256
}

func (m *model) handleConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.cfgEditing {
		return m.handleConfigEditKey(msg)
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.cfgCursor > 0 {
			m.cfgCursor--
		}
	case tea.KeyDown:
		if m.cfgCursor < len(m.cfgFields)-1 {
			m.cfgCursor++
		}
	case tea.KeyEnter:
		f := m.cfgFields[m.cfgCursor]
		if !f.editable {
			return m, nil
		}
		m.cfgEditing = true
		m.cfgInput.SetValue(f.value)
		m.cfgInput.Focus()
		var cmd tea.Cmd
		m.cfgInput, cmd = m.cfgInput.Update(textinput.Blink)
		return m, cmd
	case tea.KeyEsc:
		m.showConfig = false
		m.cfgEditing = false
		m.output = "Configuration closed (not saved)."
	case tea.KeyRunes:
		if strings.ToLower(string(msg.Runes)) == "s" {
			m.applyAndSaveConfig()
			m.showConfig = false
			m.output = "Configuration saved to ~/.veil/config.json"
		}
	}
	return m, nil
}

func (m *model) handleConfigEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.cfgFields[m.cfgCursor].value = m.cfgInput.Value()
		m.cfgEditing = false
		m.cfgInput.Blur()
		return m, nil
	case tea.KeyEsc:
		m.cfgEditing = false
		m.cfgInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.cfgInput, cmd = m.cfgInput.Update(msg)
	return m, cmd
}

func (m *model) applyAndSaveConfig() {
	for _, f := range m.cfgFields {
		if f.key == "api_url" {
			m.cfg.APIURL = f.value
		}
	}
	_ = m.repo.Save(m.cfg)
	// rebuild client with updated URL
	m.client = api.NewClient(m.cfg.APIURL, m.cfg.APIKey)
}

// --- views ---

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	if m.showConfig {
		return m.renderConfigEditor()
	}
	return m.renderMain()
}

func (m *model) renderMain() string {
	var b strings.Builder
	b.WriteString(m.s.logo.Render(logo))
	b.WriteString("\n")
	b.WriteString(m.s.status.Render(statusLine(m.cfg)))
	b.WriteString("\n")

	if m.isStreaming() {
		b.WriteString(m.s.hint.Render("  Streaming logs  Esc to stop"))
	} else {
		b.WriteString(m.s.hint.Render("  Type / for commands · Ctrl+C to exit"))
	}
	b.WriteString("\n\n")

	if m.output != "" {
		b.WriteString(m.s.output.Render(m.output))
		b.WriteString("\n\n")
	}

	if !m.isStreaming() {
		b.WriteString(m.s.prompt.Render("❯ "))
		b.WriteString(m.input.View())
		b.WriteString("\n")

		if m.showMenu && len(m.filtered) > 0 {
			b.WriteString("\n")
			b.WriteString(m.renderMenu())
		}
	}

	return b.String()
}

func (m *model) renderConfigEditor() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9F8F6")).
		Bold(true).
		Width(12)
	readOnlyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)
	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true)

	var b strings.Builder
	b.WriteString(m.s.logo.Render(logo))
	b.WriteString("\n")

	if m.cfgEditing {
		b.WriteString(m.s.hint.Render("  Enter confirm · Esc cancel"))
	} else {
		b.WriteString(m.s.hint.Render("  ↑↓ navigate · Enter edit · s save · Esc exit"))
	}
	b.WriteString("\n\n")

	for i, f := range m.cfgFields {
		arrow := "   "
		if i == m.cfgCursor {
			arrow = cursorStyle.Render("▶  ")
		}
		label := labelStyle.Render(f.label)

		var valueStr string
		switch {
		case m.cfgEditing && i == m.cfgCursor:
			valueStr = m.cfgInput.View()
		case !f.editable:
			valueStr = readOnlyStyle.Render(f.value)
		default:
			valueStr = f.value
		}

		b.WriteString(fmt.Sprintf("  %s%s  %s\n", arrow, label, valueStr))
	}

	b.WriteString("\n")
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

// --- helpers ---

func statusLine(cfg *config.Config) string {
	if cfg.APIKey == "" {
		return fmt.Sprintf("  %s  ●  not configured — run: veil auth login", Version)
	}
	return fmt.Sprintf("  %s  ●  %s", Version, cfg.APIURL)
}

func formatStats(s *api.Stats) string {
	used := float64(s.UsedTokens) / 1_000_000
	quota := float64(s.QuotaTokens) / 1_000_000
	return fmt.Sprintf("%.1fM / %.1fM tokens used (%d%%)  ·  resets %s",
		used, quota, s.Percent, s.ResetsAt)
}

const logo = `
  ██╗   ██╗███████╗██╗██╗
  ██║   ██║██╔════╝██║██║
  ██║   ██║█████╗  ██║██║
  ╚██╗ ██╔╝██╔══╝  ██║██║
   ╚████╔╝ ███████╗██║███████╗
    ╚═══╝  ╚══════╝╚═╝╚══════╝`
