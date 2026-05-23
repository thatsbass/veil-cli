package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/thatsbass/veil-cli/internal/adapter/api"
	"github.com/thatsbass/veil-cli/internal/adapter/config"
	"github.com/thatsbass/veil-cli/internal/delivery/cli/commands"
	"github.com/thatsbass/veil-cli/internal/delivery/repl"
	"github.com/thatsbass/veil-cli/internal/usecase"
)

// deps is the dependency injection container wired once at startup.
type deps struct {
	Repo   config.Repository
	Client *api.Client
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	// Wire all dependencies once at startup (composition root).
	repo := config.NewJSONRepository()
	cfg, err := repo.Load()
	if err != nil {
		// Corrupt config file; fall back to defaults so the CLI can boot.
		cfg = &config.Config{}
	}
	client := api.NewClient(cfg.APIURL, cfg.APIKey)

	d := &deps{Repo: repo, Client: client}

	root := &cobra.Command{
		Use:          "veil",
		Short:        "Veil — LLM gateway CLI",
		Version:      repl.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return repl.Start(cfg, repo)
		},
	}

	root.AddCommand(
		commands.NewAuthCmd(d.Repo),
		commands.NewUseCmd(d.Repo),
		newUpCmd(d),
		newDownCmd(d),
		newStatusCmd(d),
		newDoctorCmd(d),
		newLogsCmd(d),
		newStatsCmd(d),
		newUpdateCmd(),
		newVersionCmd(),
	)
	return root
}

func newUpCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Vérifie la connexion et affiche le statut du compte",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := d.Repo.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("not configured — run: veil auth login")
				return nil
			}

			ctx := context.Background()
			client := api.NewClient(cfg.APIURL, cfg.APIKey)

			result, err := usecase.GetDashboard(ctx, client, cfg.APIURL)
			if err != nil {
				fmt.Printf("server  : unreachable (%s)\n", cfg.APIURL)
				fmt.Println("check your connection or run: veil doctor")
				return nil
			}
			fmt.Printf("server  : %s  [%s]\n", result.APIURL, result.Status)
			if result.QuotaTokens > 0 {
				used := float64(result.UsedTokens) / 1_000_000
				quota := float64(result.QuotaTokens) / 1_000_000
				fmt.Printf("tokens  : %.1fM / %.1fM used (%d%%)\n", used, quota, result.Percent)
				fmt.Printf("resets  : %s\n", result.ResetsAt)
			} else {
				fmt.Println("usage   : unavailable")
			}
			return nil
		},
	}
}

func newDownCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Supprime la session locale",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Repo.Delete(); err != nil {
				return err
			}
			fmt.Println("Session cleared. Run 'veil auth login' to reconnect.")
			return nil
		},
	}
}

func newStatusCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Affiche la configuration locale",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := d.Repo.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("not configured — run: veil auth login")
				return nil
			}
			n := 16
			if len(cfg.APIKey) < n {
				n = len(cfg.APIKey)
			}
			fmt.Printf("api url : %s\n", cfg.APIURL)
			fmt.Printf("api key : %s…\n", cfg.APIKey[:n])
			return nil
		},
	}
}

func newDoctorCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostic de la configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := d.Repo.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("no api key — run: veil auth login")
				return nil
			}
			client := api.NewClient(cfg.APIURL, cfg.APIKey)
			if _, err := usecase.CheckHealth(context.Background(), client, cfg.APIURL); err != nil {
				fmt.Printf("api unreachable : %s\n", cfg.APIURL)
				return nil
			}
			fmt.Println("all checks passed")
			return nil
		},
	}
}

func newLogsCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Streaming des logs serveur (live)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := d.Repo.Load()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				fmt.Println("not configured — run: veil auth login")
				return nil
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			client := api.NewClient(cfg.APIURL, cfg.APIKey)
			events := make(chan string, 64)

			errCh := make(chan error, 1)
			go func() {
				errCh <- client.GetLogs(ctx, events)
				close(events)
			}()

			fmt.Println("streaming logs  Ctrl+C to stop")
			fmt.Println()
			for event := range events {
				fmt.Println(api.FormatLogEvent(event))
			}

			if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("logs: %w", err)
			}
			return nil
		},
	}
}

func newStatsCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Usage et économies du mois",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := d.Repo.Load()
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

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Affiche la version de veil-cli",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(repl.Version)
		},
	}
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Mise à jour de veil-cli",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Current version : %s\n\n", repl.Version)
			fmt.Println("Check for updates at:")
			fmt.Println("  github.com/thatsbass/veil-cli/releases")
			fmt.Println()
			fmt.Println("Auto-update coming in v1.1.0")
			return nil
		},
	}
}
