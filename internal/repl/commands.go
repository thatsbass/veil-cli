package repl

import tea "github.com/charmbracelet/bubbletea"

// SlashCommand is a REPL command triggered by typing "/name".
type SlashCommand struct {
	Name        string
	Description string
	Execute     func(m *model) tea.Cmd
}

// allCommands returns the full ordered list of slash commands.
func allCommands() []SlashCommand {
	return []SlashCommand{
		{
			Name:        "/status",
			Description: "Dashboard temps réel",
			Execute:     cmdStatus,
		},
		{
			Name:        "/provider",
			Description: "Changer de provider",
			Execute:     cmdProvider,
		},
		{
			Name:        "/stats",
			Description: "Usage et économies",
			Execute:     cmdStats,
		},
		{
			Name:        "/logs",
			Description: "Logs en streaming",
			Execute:     cmdLogs,
		},
		{
			Name:        "/config",
			Description: "Modifier la configuration",
			Execute:     cmdConfig,
		},
		{
			Name:        "/doctor",
			Description: "Diagnostic système",
			Execute:     cmdDoctor,
		},
		{
			Name:        "/use",
			Description: "Configurer un outil (Codex, Cursor...)",
			Execute:     cmdUse,
		},
		{
			Name:        "/billing",
			Description: "Gérer l'abonnement",
			Execute:     cmdBilling,
		},
		{
			Name:        "/login",
			Description: "Connecter le CLI à votre compte Veil",
			Execute:     cmdLogin,
		},
		{
			Name:        "/logout",
			Description: "Déconnecter le CLI",
			Execute:     cmdLogout,
		},
		{
			Name:        "/help",
			Description: "Aide complète",
			Execute:     cmdHelp,
		},
		{
			Name:        "/exit",
			Description: "Quitter",
			Execute:     cmdExit,
		},
	}
}

// --- command implementations ---

type outputMsg string

func cmdStatus(m *model) tea.Cmd {
	return func() tea.Msg {
		status, err := m.client.GetStatus(m.ctx)
		if err != nil {
			return outputMsg("error: " + err.Error())
		}
		return outputMsg("server: " + status.Status)
	}
}

func cmdStats(m *model) tea.Cmd {
	return func() tea.Msg {
		stats, err := m.client.GetStats(m.ctx)
		if err != nil {
			return outputMsg("error: " + err.Error())
		}
		return outputMsg(formatStats(stats))
	}
}

func cmdDoctor(m *model) tea.Cmd {
	return func() tea.Msg {
		_, err := m.client.GetStatus(m.ctx)
		if err != nil {
			return outputMsg("API unreachable — check your connection and api_url in ~/.veil/config.json")
		}
		if m.cfg.APIKey == "" {
			return outputMsg("No API key configured. Run: veil init")
		}
		return outputMsg("All checks passed.")
	}
}

func cmdHelp(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Type / to see available commands. Tab or Enter to execute.")
	}
}

func cmdExit(_ *model) tea.Cmd {
	return tea.Quit
}

func cmdProvider(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Provider selection — coming in next release.")
	}
}

func cmdLogs(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Log streaming — coming in next release.")
	}
}

func cmdConfig(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Config editor — coming in next release.")
	}
}

func cmdUse(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Tool setup — coming in next release.")
	}
}

func cmdBilling(_ *model) tea.Cmd {
	return func() tea.Msg {
		return outputMsg("Billing management — coming in next release.")
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
		if err := m.cfg.Save(); err != nil {
			return outputMsg("error: " + err.Error())
		}
		return outputMsg("Logged out.")
	}
}
