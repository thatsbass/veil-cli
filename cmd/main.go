package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thatsbass/veil-cli/internal/api"
	"github.com/thatsbass/veil-cli/internal/commands"
	"github.com/thatsbass/veil-cli/internal/config"
	"github.com/thatsbass/veil-cli/internal/repl"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "veil",
		Short:        "Veil — LLM gateway CLI",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return repl.Start(cfg)
		},
	}

	root.AddCommand(
		commands.NewAuthCmd(),
		newInitCmd(),
		newUpCmd(),
		newDownCmd(),
		newStatusCmd(),
		newDoctorCmd(),
		newLogsCmd(),
		newStatsCmd(),
		newUpdateCmd(),
	)
	return root
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Setup interactif — configure Veil pour la première fois",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

// newUpCmd checks that api.veil.dev is reachable and shows account status.
func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Vérifie la connexion et affiche le statut du compte",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("not configured — run: veil auth login")
				return nil
			}

			client := api.NewClient(cfg.APIURL, cfg.APIKey)
			ctx := context.Background()

			status, err := client.GetStatus(ctx)
			if err != nil {
				fmt.Printf("server  : unreachable (%s)\n", cfg.APIURL)
				fmt.Println("check your connection or run: veil doctor")
				return nil
			}
			fmt.Printf("server  : %s  [%s]\n", cfg.APIURL, status.Status)

			stats, err := client.GetStats(ctx)
			if err != nil {
				fmt.Println("usage   : unavailable")
				return nil
			}
			used := float64(stats.UsedTokens) / 1_000_000
			quota := float64(stats.QuotaTokens) / 1_000_000
			fmt.Printf("tokens  : %.1fM / %.1fM used (%d%%)\n", used, quota, stats.Percent)
			fmt.Printf("resets  : %s\n", stats.ResetsAt)
			return nil
		},
	}
}

// newDownCmd clears the local session (equivalent to logout).
func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Supprime la session locale",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Delete(); err != nil {
				return err
			}
			fmt.Println("Session cleared. Run 'veil auth login' to reconnect.")
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Affiche la configuration locale",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("not configured — run: veil auth login")
				return nil
			}
			fmt.Printf("api url : %s\n", cfg.APIURL)
			fmt.Printf("api key : %s…\n", cfg.APIKey[:min(16, len(cfg.APIKey))])
			return nil
		},
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostic de la configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("no api key — run: veil auth login")
				return nil
			}

			client := api.NewClient(cfg.APIURL, cfg.APIKey)
			_, err = client.GetStatus(context.Background())
			if err != nil {
				fmt.Printf("api unreachable : %s\n", cfg.APIURL)
				return nil
			}
			fmt.Println("all checks passed")
			return nil
		},
	}
}

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Streaming des logs serveur",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("log streaming — coming in next release")
			return nil
		},
	}
}

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Usage et économies du mois",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("not configured — run: veil auth login")
				return nil
			}

			stats, err := api.NewClient(cfg.APIURL, cfg.APIKey).GetStats(context.Background())
			if err != nil {
				return fmt.Errorf("stats: %w", err)
			}
			used := float64(stats.UsedTokens) / 1_000_000
			quota := float64(stats.QuotaTokens) / 1_000_000
			fmt.Printf("tokens  : %.1fM / %.1fM used (%d%%)\n", used, quota, stats.Percent)
			fmt.Printf("resets  : %s\n", stats.ResetsAt)
			return nil
		},
	}
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Mise à jour de veil-cli",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("curl -fsSL https://veil.dev/install | bash")
			return nil
		},
	}
}

func runInit() error {
	r := bufio.NewReader(os.Stdin)

	fmt.Print("API key (vl_live_xxx): ")
	apiKey, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	fmt.Printf("API URL [https://api.veil.dev]: ")
	apiURL, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = "https://api.veil.dev"
	}

	cfg := &config.Config{
		APIKey: apiKey,
		APIURL: apiURL,
	}
	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Println("config saved to ~/.veil/config.json")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
