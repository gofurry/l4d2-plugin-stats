package cli

import (
	"context"
	"encoding/json"
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
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
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

var Version = "1.2.1"

type rootOptions struct{ configPath string }

func Execute() error {
	options := &rootOptions{}
	root := &cobra.Command{Use: "l4d2-stats", Short: "L4D2 player statistics dashboard", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&options.configPath, "config", "./config.yaml", "path to config.yaml")
	root.AddCommand(serveCommand(options), doctorCommand(options), versionCommand(), installCommand(options), uninstallCommand(), migrateCommand(options), aggregateCommand(options), retentionCommand(options), backupCommand(options), diagnosticsCommand(options))
	return root.Execute()
}

func backupCommand(options *rootOptions) *cobra.Command {
	backup := &cobra.Command{Use: "backup", Short: "create and restore Dashboard backups"}
	backup.AddCommand(
		&cobra.Command{Use: "create", Short: "create a verified backup archive", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(options.configPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			result, err := service.NewArchiveService(cfg, Version).CreateBackup(ctx, ".")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), result.Path)
			if result.Message != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), result.Message)
			}
			return nil
		}},
		&cobra.Command{Use: "restore <file>", Short: "restore a verified backup while preserving rollback copies", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(options.configPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			result, err := service.NewArchiveService(cfg, Version).RestoreBackup(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), result.Message)
			for _, path := range result.RollbackCopies {
				fmt.Fprintf(cmd.OutOrStdout(), "rollback copy: %s\n", path)
			}
			return nil
		}},
	)
	return backup
}

func diagnosticsCommand(options *rootOptions) *cobra.Command {
	diagnostics := &cobra.Command{Use: "diagnostics", Short: "export redacted diagnostic data"}
	diagnostics.AddCommand(&cobra.Command{Use: "export", Short: "create a redacted diagnostics archive", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := service.NewArchiveService(cfg, Version).ExportDiagnostics(ctx, ".")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), result.Path)
		if result.Message != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), result.Message)
		}
		return nil
	}})
	return diagnostics
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
		authService, setupToken, err := auth.New(ctx, dashboard)
		if err != nil {
			return fmt.Errorf("initialize administrator authentication: %w", err)
		}
		if setupToken != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nAdministrator setup is required.\nOpen /admin/setup and enter this one-time token (valid for 30 minutes):\n%s\n\n", setupToken)
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
		if version != store.StatsSchemaVersion {
			return fmt.Errorf("unsupported stats schema version %d; expected %d", version, store.StatsSchemaVersion)
		}
		assets, err := webassets.Dist()
		if err != nil {
			return fmt.Errorf("open embedded frontend: %w", err)
		}
		overview := service.NewOverviewService(stats, 60*time.Second, dashboard)
		players := service.NewPlayerService(stats, dashboard)
		var analysis *service.AnalysisService
		if analysisStore, ok := stats.(store.StatsAnalysisStore); ok {
			analysis = service.NewAnalysisService(analysisStore)
		}
		aggregates := service.NewAggregateService(dashboard, stats, logger)
		dataMaintenance := service.NewDataMaintenanceService(dashboard, stats, aggregates, cfg.StatsDatabase, cfg.Logging.File, logger)
		rankings := service.NewRankingService(dashboard, stats)
		runCtx, stopBackground := context.WithCancel(context.Background())
		defer stopBackground()
		aggregates.Start(runCtx)
		a2sClient := a2s.SteamClient{}
		status := a2s.NewProvider(dashboard, a2sClient, stats)
		app := server.New(cfg, server.Dependencies{Dashboard: dashboard, Stats: stats, Overview: overview, Status: status, Players: players, Analysis: analysis, Rankings: rankings, Data: dataMaintenance, Auth: authService, Logger: logger, Assets: assets})
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

func aggregateCommand(options *rootOptions) *cobra.Command {
	aggregate := &cobra.Command{Use: "aggregate", Short: "manage the Dashboard aggregate read model"}
	aggregate.AddCommand(
		&cobra.Command{Use: "status", Short: "show aggregate status", RunE: func(cmd *cobra.Command, args []string) error {
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
			status, err := dashboard.AggregateStatus(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd, status)
		}},
		&cobra.Command{Use: "rebuild", Short: "rebuild all daily aggregates from the read-only Stats DB", RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(options.configPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
			service := service.NewAggregateService(dashboard, stats, zap.NewNop())
			if err := service.Rebuild(ctx); err != nil {
				return err
			}
			status, err := dashboard.AggregateStatus(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd, status)
		}},
	)
	return aggregate
}

func retentionCommand(options *rootOptions) *cobra.Command {
	retention := &cobra.Command{Use: "retention", Short: "inspect the non-destructive raw-data retention plan"}
	retention.AddCommand(&cobra.Command{Use: "plan", Short: "show rows that would be eligible; never deletes data", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(options.configPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
		now := time.Now()
		plan, err := stats.RetentionPlan(ctx, now.AddDate(0, 0, -180).Unix(), now.AddDate(-1, 0, 0).Unix(), now.AddDate(-1, 0, 0).Unix())
		if err != nil {
			return err
		}
		status, err := dashboard.AggregateStatus(ctx)
		if err != nil {
			return err
		}
		plan.AggregateCoverageReady = status.State == "ready" &&
			status.AggregateVersion == plan.AggregateVersion && status.LastFinishedAt > 0
		plan.DeletionEnabled = false
		return writeJSON(cmd, plan)
	}})
	return retention
}

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func doctorCommand(options *rootOptions) *cobra.Command {
	var deep bool
	command := &cobra.Command{Use: "doctor", Short: "validate configuration and database connectivity", RunE: func(cmd *cobra.Command, args []string) error {
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
		adminConfigured, err := dashboard.AdminConfigured(ctx)
		if err != nil {
			return err
		}
		site, err := dashboard.Site(ctx)
		if err != nil {
			return err
		}
		servers, err := dashboard.ListServers(ctx)
		if err != nil {
			return err
		}
		enabledServers := 0
		for _, server := range servers {
			if server.Enabled {
				enabledServers++
			}
		}
		settings, err := dashboard.SiteSettings(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "config: ok\ndashboard database: ok (schema %d)\nstats database: ok (schema %d)\nadministrator: %t\nsite configured: %t\nenabled servers: %d\nSteam OpenID ready: %t\nruntime monitor: %t\n", dashboardVersion, version, adminConfigured, site.Configured, enabledServers, settings.SteamOpenIDEnabled && settings.PublicOrigin != "", cfg.Monitor.Enabled)
		cancel()
		if !deep {
			return nil
		}
		deepCtx, deepCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer deepCancel()
		report := service.NewDoctorService(dashboard, stats).Deep(deepCtx)
		for _, check := range report.Checks {
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", check.Status, check.Name, check.Message)
		}
		if report.HasErrors() {
			return errors.New("deep doctor found data quality errors")
		}
		return nil
	}}
	command.Flags().BoolVar(&deep, "deep", false, "run read-only data quality and aggregate checks")
	return command
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
