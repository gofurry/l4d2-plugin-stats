package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/a2s"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/logging"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/server"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/systemd"
	webassets "github.com/gofurry/l4d2-plugin-stats/dashboard/web"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var Version = "0.8.1-dev"

type rootOptions struct{ configPath string }

func Execute() error {
	options := &rootOptions{}
	root := &cobra.Command{Use: "l4d2-stats", Short: "L4D2 player statistics dashboard", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&options.configPath, "config", "./config.yaml", "path to config.yaml")
	root.AddCommand(serveCommand(options), doctorCommand(options), versionCommand(), installCommand(options), uninstallCommand(), migrateCommand(options), bootstrapCommand(options))
	return root.Execute()
}

func serveCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "serve", Short: "start the dashboard server", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		logger, err := logging.New(cfg.Logging)
		if err != nil {
			return err
		}
		defer logger.Sync()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		dashboard, err := store.OpenDashboard(ctx, cfg.DashboardDatabase.Path)
		if err != nil {
			return err
		}
		defer dashboard.Close()
		if applied, err := dashboard.Bootstrap(ctx, cfg.Bootstrap, false); err != nil {
			return err
		} else if applied {
			logger.Info("dashboard bootstrap applied")
		}
		stats, err := store.OpenStats(ctx, cfg.StatsDatabase)
		if err != nil {
			return err
		}
		defer stats.Close()
		version, err := stats.SchemaVersion(ctx)
		if err != nil {
			return fmt.Errorf("read stats schema version: %w", err)
		}
		if version != 1 {
			return fmt.Errorf("unsupported stats schema version %d; expected 1", version)
		}
		assets, err := webassets.Dist()
		if err != nil {
			return fmt.Errorf("open embedded frontend: %w", err)
		}
		overview := service.NewOverviewService(stats, 60*time.Second)
		status := a2s.NewProvider(dashboard, a2s.SteamClient{})
		app := server.New(cfg, server.Dependencies{Dashboard: dashboard, Stats: stats, Overview: overview, Status: status, Logger: logger, Assets: assets})
		logger.Info("dashboard starting", zap.String("listen", cfg.Server.Listen), zap.String("config", cfg.Path))
		errCh := make(chan error, 1)
		go func() { errCh <- app.Listen(cfg.Server.Listen, fiber.ListenConfig{DisableStartupMessage: true}) }()
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		select {
		case sig := <-signals:
			logger.Info("shutdown signal received", zap.String("signal", sig.String()))
			return app.ShutdownWithTimeout(10 * time.Second)
		case err := <-errCh:
			return err
		}
	}}
}

func doctorCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "validate configuration and database connectivity", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		dashboard, err := store.OpenDashboard(ctx, cfg.DashboardDatabase.Path)
		if err != nil {
			return err
		}
		defer dashboard.Close()
		stats, err := store.OpenStats(ctx, cfg.StatsDatabase)
		if err != nil {
			return err
		}
		defer stats.Close()
		version, err := stats.SchemaVersion(ctx)
		if err != nil {
			return err
		}
		dashboardVersion, err := dashboard.MigrationVersion(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "config: ok\ndashboard database: ok (schema %d)\nstats database: ok (schema %d)\n", dashboardVersion, version)
		return nil
	}}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{Use: "version", Short: "print version", Run: func(cmd *cobra.Command, args []string) { fmt.Fprintln(cmd.OutOrStdout(), Version) }}
}

func migrateCommand(options *rootOptions) *cobra.Command {
	migrate := &cobra.Command{Use: "migrate", Short: "dashboard database migration commands"}
	migrate.AddCommand(&cobra.Command{Use: "status", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		dashboard, err := store.OpenDashboard(ctx, cfg.DashboardDatabase.Path)
		if err != nil {
			return err
		}
		defer dashboard.Close()
		version, err := dashboard.MigrationVersion(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "dashboard schema version: %d\n", version)
		return nil
	}})
	return migrate
}

func bootstrapCommand(options *rootOptions) *cobra.Command {
	bootstrap := &cobra.Command{Use: "bootstrap", Short: "dashboard bootstrap commands"}
	var replace bool
	apply := &cobra.Command{Use: "apply", Short: "apply site and server bootstrap configuration", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		dashboard, err := store.OpenDashboard(ctx, cfg.DashboardDatabase.Path)
		if err != nil {
			return err
		}
		defer dashboard.Close()
		applied, err := dashboard.Bootstrap(ctx, cfg.Bootstrap, replace)
		if err != nil {
			return err
		}
		if !applied {
			fmt.Fprintln(cmd.OutOrStdout(), "bootstrap skipped: dashboard data already exists")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "bootstrap applied")
		}
		return nil
	}}
	apply.Flags().BoolVar(&replace, "replace", false, "replace existing site and server settings")
	bootstrap.AddCommand(apply)
	return bootstrap
}

func installCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{Use: "install", Short: "install and start the systemd service", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		binary, err := os.Executable()
		if err != nil {
			return err
		}
		binary, err = filepath.EvalSymlinks(binary)
		if err != nil {
			return err
		}
		identity, err := systemd.ResolveIdentity()
		if err != nil {
			return err
		}
		if identity.Root {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning: service will run as root because root invoked install directly")
		}
		return systemd.Install(systemd.Options{BinaryPath: binary, Config: cfg, Identity: identity})
	}}
}

func uninstallCommand() *cobra.Command {
	return &cobra.Command{Use: "uninstall", Short: "stop and remove the systemd service without deleting data", RunE: func(cmd *cobra.Command, args []string) error { return systemd.Uninstall() }}
}

func embeddedExists(assets fs.FS) error {
	_, err := fs.Stat(assets, "index.html")
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("embedded frontend is missing index.html")
	}
	return err
}
