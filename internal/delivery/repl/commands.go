package repl

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thatsbass/veil-cli/internal/adapter/api"
	"github.com/thatsbass/veil-cli/internal/adapter/configurator"
	"github.com/thatsbass/veil-cli/internal/usecase"
)

// SlashCommand represents a REPL command triggered by typing "/name".
type SlashCommand struct {
	Name        string
	Description string
	Execute     func(m *model) tea.Cmd
}

// allCommands returns every registered slash command in menu order.
func allCommands() []SlashCommand {
	return []SlashCommand{
		{Name: "/status", Description: "Server status", Execute: cmdStatus},
		{Name: "/stats", Description: "Monthly usage and savings", Execute: cmdStats},
		{Name: "/billing", Description: "Current plan and quota", Execute: cmdBilling},
		{Name: "/logs", Description: "Live log stream (Esc to stop)", Execute: cmdLogs},
		{Name: "/config", Description: "Edit local configuration", Execute: cmdConfig},
		{Name: "/use", Description: "Tool status — run: veil use", Execute: cmdUse},
		{Name: "/provider", Description: "Change LLM provider", Execute: cmdProvider},
		{Name: "/doctor", Description: "System diagnostic", Execute: cmdDoctor},
		{Name: "/login", Description: "Log in", Execute: cmdLogin},
		{Name: "/logout", Description: "Log out", Execute: cmdLogout},
		{Name: "/help", Description: "Full help", Execute: cmdHelp},
		{Name: "/exit", Description: "Quit", Execute: cmdExit},
	}
}

// ── tea.Msg types for async commands ──

type outputMsg string

// logEventMsg carries one raw SSE log line to the REPL model.
type logEventMsg string

// logDoneMsg signals that the SSE stream has closed, optionally with an error.
type logDoneMsg struct{ err error }

func cmdStatus(m *model) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 8*time.Second)
		defer cancel()
		result, err := usecase.CheckHealth(ctx, m.client, m.cfg.APIURL)
		if err != nil {
			return outputMsg("server unreachable — check your connection or run /doctor")
		}
		return outputMsg("server : " + result.Status + "  [" + result.APIURL + "]")
	}
}

func cmdStats(m *model) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 8*time.Second)
		defer cancel()
		stats, err := m.client.GetStats(ctx)
		if err != nil {
			return outputMsg("usage unavailable — " + err.Error())
		}
		return outputMsg(formatStats(stats))
	}
}

func cmdBilling(m *model) tea.Cmd {
	return func() tea.Msg {
		if m.cfg.APIKey == "" {
			return outputMsg("not authenticated — run: veil auth login")
		}

		ctx, cancel := context.WithTimeout(m.ctx, 8*time.Second)
		defer cancel()

		result, err := usecase.GetBillingOverview(ctx, m.client)
		if err != nil {
			return outputMsg("billing unavailable — " + err.Error())
		}
		return outputMsg(renderBillingCard(result.PlanID, result.PriceUSD, result.TokenQuota, result.Stats))
	}
}

func cmdLogs(m *model) tea.Cmd {
	if m.isStreaming() {
		m.stopStreaming()
		return func() tea.Msg { return outputMsg("Log streaming stopped.") }
	}
	if m.cfg.APIKey == "" {
		return func() tea.Msg { return outputMsg("not authenticated — run: veil auth login") }
	}

	ctx, cancel := context.WithCancel(m.ctx)
	ch := make(chan string, 64)
	m.startStreaming(ch, cancel)

	go func() {
		_ = m.client.GetLogs(ctx, ch)
		close(ch)
	}()

	return readNextLog(ch)
}

// readNextLog returns a tea.Cmd that reads the next SSE event from ch.
func readNextLog(ch chan string) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logEventMsg(event)
	}
}

func cmdConfig(m *model) tea.Cmd {
	m.openConfigEditor()
	return nil
}

func cmdDoctor(m *model) tea.Cmd {
	return func() tea.Msg {
		if m.cfg.APIKey == "" {
			return outputMsg("no API key — run: veil auth login")
		}
		ctx, cancel := context.WithTimeout(m.ctx, 8*time.Second)
		defer cancel()
		if _, err := usecase.CheckHealth(ctx, m.client, m.cfg.APIURL); err != nil {
			return outputMsg("API unreachable — check your connection and api_url in /config")
		}
		return outputMsg("all checks passed")
	}
}

func cmdUse(m *model) tea.Cmd {
	return func() tea.Msg {
		tools := configurator.All()
		var sb strings.Builder
		sb.WriteString("Tool status:\n\n")
		for _, t := range tools {
			status := "not found"
			if t.Detect() {
				if k := t.ExistingKey(); k != "" {
					n := 8
					if len(k) < n {
						n = len(k)
					}
					status = fmt.Sprintf("installed  key: %s…", k[:n])
				} else {
					status = "installed  no key configured"
				}
			}
			sb.WriteString(fmt.Sprintf("  %-14s  %s\n", t.Name(), status))
		}
		sb.WriteString("\nRun: veil use [claude|codex|cursor|aider]")
		return outputMsg(sb.String())
	}
}

func cmdProvider(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg(
			"Provider selection coming in v1.1.0.\n\n" +
				"Current provider : auto (smart routing to cheapest available)\n" +
				"Change manually  : /config → api_url",
		)
	}
}

func cmdLogin(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Run: veil auth login")
	}
}

func cmdLogout(m *model) tea.Cmd {
	return func() tea.Msg {
		m.cfg.APIKey = ""
		if err := m.repo.Save(m.cfg); err != nil {
			return outputMsg("error: " + err.Error())
		}
		return outputMsg("Logged out. Run 'veil auth login' to reconnect.")
	}
}

func cmdHelp(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg(
			"Type / to open the command menu.\n" +
				"↑↓ navigate  Tab cycle  Enter execute  Esc close\n\n" +
				"Key commands:\n" +
				"  /status    check server\n" +
				"  /stats     token usage\n" +
				"  /billing   plan + quota\n" +
				"  /logs      live log stream\n" +
				"  /config    edit api_url\n" +
				"  /use       configure tools\n" +
				"  /login     connect account\n",
		)
	}
}

func cmdExit(_ *model) tea.Cmd {
	return tea.Quit
}

// ── Billing card renderer ──

func renderBillingCard(planID string, priceUSD float64, tokenQuota int64, stats *api.Stats) string {
	used := float64(stats.UsedTokens) / 1_000_000
	quota := float64(tokenQuota) / 1_000_000
	bar := progressBar(stats.Percent, 10)
	daysLeft := daysUntil(stats.ResetsAt)

	planName := strings.ToUpper(planID[:1]) + planID[1:]
	priceStr := fmt.Sprintf("$%.0f/month", priceUSD)
	if priceUSD == 0 {
		priceStr = "free"
	}

	lines := []string{
		fmt.Sprintf("  Current Plan : %s", planName),
		fmt.Sprintf("  Tokens used  : %.1fM / %.1fM  %s  %d%%", used, quota, bar, stats.Percent),
		fmt.Sprintf("  Resets in    : %d days", daysLeft),
		fmt.Sprintf("  Price        : %s", priceStr),
		"",
		"  Upgrade : veil.dev/billing",
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3F3F46")).
		Padding(0, 1).
		MarginLeft(2).
		Render(strings.Join(lines, "\n"))
}

func progressBar(percent, width int) string {
	filled := percent * width / 100
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func daysUntil(s string) int {
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			d := int(time.Until(t).Hours() / 24)
			if d < 0 {
				return 0
			}
			return d
		}
	}
	return 0
}
