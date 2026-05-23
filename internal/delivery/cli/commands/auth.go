package commands

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/thatsbass/veil-cli/internal/adapter/api"
	"github.com/thatsbass/veil-cli/internal/adapter/config"
)

// NewAuthCmd returns the `veil auth` command tree. The repo parameter is
// injected from the composition root in cmd/main.go.
func NewAuthCmd(repo config.Repository) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Gérer l'authentification Veil",
	}
	cmd.AddCommand(newLoginCmd(repo), newLogoutCmd(repo), newAuthStatusCmd(repo))
	return cmd
}

func newLoginCmd(repo config.Repository) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Connecter le CLI à votre compte Veil",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(repo)
		},
	}
}

func newLogoutCmd(repo config.Repository) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Déconnecter le CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := repo.Load()
			if err != nil || cfg.APIKey == "" {
				fmt.Println("not logged in")
				return nil
			}
			cfg.APIKey = ""
			if err := repo.Save(cfg); err != nil {
				return err
			}
			fmt.Println("logged out")
			return nil
		},
	}
}

func newAuthStatusCmd(repo config.Repository) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Afficher l'état de connexion",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := repo.Load()
			if err != nil || cfg.APIKey == "" {
				fmt.Println("not logged in — run: veil auth login")
				return nil
			}
			fmt.Printf("logged in  api_url: %s\n", cfg.APIURL)
			return nil
		},
	}
}

func runLogin(repo config.Repository) error {
	cfg, err := repo.Load()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.APIURL, "")
	ctx := context.Background()

	deviceResp, err := client.InitiateDeviceAuth(ctx)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	fmt.Printf("\n  Your activation code:  %s\n\n", codeStyle.Render(deviceResp.UserCode))
	fmt.Printf("  Opening: %s\n\n", deviceResp.VerificationURL)

	openBrowser(deviceResp.VerificationURL)

	p := tea.NewProgram(newLoginModel(client, deviceResp.DeviceCode))
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	result := m.(loginModel)
	if result.apiKey == "" {
		if result.err != nil {
			return result.err
		}
		return fmt.Errorf("login cancelled")
	}

	cfg.APIKey = result.apiKey
	if err := repo.Save(cfg); err != nil {
		return err
	}

	fmt.Println("\n  Logged in. API key saved to ~/.veil/config.json")
	return nil
}

// ── Device auth polling model (BubbleTea) ──

type loginModel struct {
	spinner spinner.Model
	client  *api.Client
	code    string
	apiKey  string
	err     error
	done    bool
}

type pollMsg struct {
	apiKey string
	err    error
}

func newLoginModel(client *api.Client, deviceCode string) loginModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	return loginModel{spinner: s, client: client, code: deviceCode}
}

func (m loginModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.doPoll())
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.done = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case pollMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}
		if msg.apiKey != "" {
			m.apiKey = msg.apiKey
			m.done = true
			return m, tea.Quit
		}
		// still pending — wait then poll again
		return m, tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
			return m.poll()
		})
	}

	return m, nil
}

func (m loginModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("  %s Waiting for browser confirmation… (Ctrl+C to cancel)\n", m.spinner.View())
}

func (m loginModel) doPoll() tea.Cmd {
	return func() tea.Msg { return m.poll() }
}

func (m loginModel) poll() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := m.client.PollDeviceToken(ctx, m.code)
	if err != nil {
		return pollMsg{err: err}
	}
	switch resp.Status {
	case "authorized":
		return pollMsg{apiKey: resp.APIKey}
	case "expired":
		return pollMsg{err: fmt.Errorf("session expired — run: veil auth login")}
	default:
		return pollMsg{} // authorization_pending
	}
}

// openBrowser opens url in the default browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
