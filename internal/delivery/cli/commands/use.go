package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/thatsbass/veil-cli/internal/adapter/config"
	"github.com/thatsbass/veil-cli/internal/adapter/configurator"
)

// NewUseCmd returns the `veil use [tool]` command. The repo parameter is
// injected from the composition root in cmd/main.go.
func NewUseCmd(repo config.Repository) *cobra.Command {
	return &cobra.Command{
		Use:   "use [claude|codex|cursor|aider]",
		Short: "Configure a tool to use Veil",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUse(repo, args)
		},
	}
}

func runUse(repo config.Repository, args []string) error {
	cfg, err := repo.Load()
	if err != nil {
		return err
	}
	if cfg.APIKey == "" {
		fmt.Println("  not authenticated — run: veil auth login")
		fmt.Println("  you can still configure tools now and add the key later")
		fmt.Println()
	}

	tools := configurator.All()

	var target configurator.ToolConfigurator
	if len(args) == 1 {
		slug := strings.ToLower(args[0])
		for _, t := range tools {
			if t.Slug() == slug {
				target = t
				break
			}
		}
		if target == nil {
			return fmt.Errorf("unknown tool %q — valid: claude, codex, cursor, aider", args[0])
		}
	} else {
		selected, err := runSelector(tools)
		if err != nil {
			return err
		}
		if selected == nil {
			return nil // user quit
		}
		target = selected
	}

	return runConfigure(target, cfg.APIURL, cfg.APIKey)
}

// runConfigure executes the interactive configuration flow for a single tool.
func runConfigure(tool configurator.ToolConfigurator, apiURL, apiKey string) error {
	fmt.Println()

	detected := tool.Detect()
	if detected {
		fmt.Printf("  %s   detected\n", tool.Name())
	} else {
		fmt.Printf("  %s   not detected (binary not found, no config)\n", tool.Name())
		if !confirm("  Configure anyway?") {
			fmt.Println("  aborted")
			return nil
		}
	}

	if existing := tool.ExistingKey(); existing != "" {
		masked := maskKey(existing)
		fmt.Printf("  existing key found: %s\n", masked)
		if !confirm("  Overwrite?") {
			fmt.Println("  aborted — existing key preserved")
			return nil
		}
	}

	fmt.Println()

	if err := tool.Configure(apiURL, apiKey); err != nil {
		return fmt.Errorf("configure %s: %w", tool.Name(), err)
	}

	printSuccess(tool, apiURL, apiKey)
	return nil
}

func printSuccess(tool configurator.ToolConfigurator, apiURL, apiKey string) {
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Width(18)

	path, err := tool.ConfigPath()
	if err != nil {
		path = "(unknown)"
	}

	fmt.Printf("  %s configured\n\n", tool.Name())
	fmt.Printf("  %s\n", pathStyle.Render(path+".veil.bak  (backup)"))
	fmt.Printf("  %s\n\n", pathStyle.Render(path))
	fmt.Printf("  %s%s\n", labelStyle.Render("api_url"), keyStyle.Render(apiURL))
	fmt.Printf("  %s%s\n\n", labelStyle.Render("api_key"), keyStyle.Render(maskKey(apiKey)))
	fmt.Printf("  test: %s\n\n", tool.TestCmd())
}

// ── Interactive tool selector (BubbleTea) ──

type selectorModel struct {
	tools    []configurator.ToolConfigurator
	cursor   int
	selected configurator.ToolConfigurator
	quit     bool
	s        selectorStyles
}

type selectorStyles struct {
	title    lipgloss.Style
	cursor   lipgloss.Style
	name     lipgloss.Style
	detected lipgloss.Style
	missing  lipgloss.Style
	hint     lipgloss.Style
}

func newSelectorStyles() selectorStyles {
	return selectorStyles{
		title:    lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).MarginBottom(1),
		cursor:   lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true),
		name:     lipgloss.NewStyle().Width(16),
		detected: lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")),
		missing:  lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		hint:     lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Italic(true).MarginTop(1),
	}
}

func runSelector(tools []configurator.ToolConfigurator) (configurator.ToolConfigurator, error) {
	m := selectorModel{tools: tools, s: newSelectorStyles()}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(selectorModel)
	if final.quit || final.selected == nil {
		return nil, nil
	}
	return final.selected, nil
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.quit = true
		return m, tea.Quit
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.tools)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		m.selected = m.tools[m.cursor]
		return m, tea.Quit
	}
	return m, nil
}

func (m selectorModel) View() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.s.title.Render("Select a tool to configure:"))
	b.WriteString("\n\n")

	for i, t := range m.tools {
		arrow := "  "
		if i == m.cursor {
			arrow = m.s.cursor.Render("▶ ")
		}

		name := m.s.name.Render(t.Name())

		var status string
		if t.Detect() {
			status = m.s.detected.Render("installed")
		} else {
			status = m.s.missing.Render("not found")
		}

		fmt.Fprintf(&b, "  %s%s  %s\n", arrow, name, status)
	}

	b.WriteString("\n")
	b.WriteString(m.s.hint.Render("  ↑↓ navigate   Enter select   q/Esc quit"))
	b.WriteString("\n\n")
	return b.String()
}

// --- helpers ---

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line)) == "y"
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:8] + "…"
}
