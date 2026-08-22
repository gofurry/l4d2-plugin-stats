package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func (s *ArchiveService) CreateBackup(ctx context.Context, outputDirectory string) (ArchiveResult, error) {
	now := s.now().UTC()
	outputPath, err := s.archivePath(outputDirectory, "backup", now)
	if err != nil {
		return ArchiveResult{}, err
	}
	temp, err := os.MkdirTemp("", "l4d2-stats-backup-")
	if err != nil {
		return ArchiveResult{}, err
	}
	defer os.RemoveAll(temp)
	if err := os.Chmod(temp, 0o700); err != nil && !os.IsPermission(err) {
		return ArchiveResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(temp, "dashboard"), 0o700); err != nil {
		return ArchiveResult{}, err
	}
	dashboardSnapshot := filepath.Join(temp, "dashboard", "dashboard.db")
	if err := sqliteOnlineBackup(ctx, s.cfg.DashboardDatabase.Path, dashboardSnapshot); err != nil {
		return ArchiveResult{}, fmt.Errorf("backup Dashboard database: %w", err)
	}
	if err := sanitizeDashboardSnapshot(ctx, dashboardSnapshot); err != nil {
		return ArchiveResult{}, fmt.Errorf("sanitize Dashboard GeoIP data: %w", err)
	}
	if err := os.Chmod(dashboardSnapshot, 0o600); err != nil && !os.IsPermission(err) {
		return ArchiveResult{}, err
	}
	if err := validateDashboardSnapshot(ctx, dashboardSnapshot); err != nil {
		return ArchiveResult{}, fmt.Errorf("validate Dashboard snapshot: %w", err)
	}
	configBytes, err := os.ReadFile(s.cfg.Path)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("read configuration: %w", err)
	}
	if err := os.WriteFile(filepath.Join(temp, "config.yaml"), configBytes, 0o600); err != nil {
		return ArchiveResult{}, err
	}

	driver := normalizeStatsDriver(s.cfg.StatsDatabase.Driver)
	mode := "external_required"
	names := []string{"config.yaml", "dashboard/dashboard.db"}
	if driver == "sqlite" {
		mode = "sqlite_online"
		if err := os.MkdirAll(filepath.Join(temp, "stats"), 0o700); err != nil {
			return ArchiveResult{}, err
		}
		statsSnapshot := filepath.Join(temp, "stats", "stats.db")
		if err := sqliteOnlineBackup(ctx, s.cfg.StatsDatabase.DSN, statsSnapshot); err != nil {
			return ArchiveResult{}, fmt.Errorf("backup Stats database: %w", err)
		}
		if err := sanitizeStatsSnapshot(ctx, statsSnapshot); err != nil {
			return ArchiveResult{}, fmt.Errorf("sanitize Stats private data: %w", err)
		}
		if err := os.Chmod(statsSnapshot, 0o600); err != nil && !os.IsPermission(err) {
			return ArchiveResult{}, err
		}
		if err := validateStatsSnapshot(ctx, statsSnapshot); err != nil {
			return ArchiveResult{}, fmt.Errorf("validate Stats snapshot: %w", err)
		}
		names = append(names, "stats/stats.db")
	} else {
		stats, err := store.OpenStats(ctx, s.cfg.StatsDatabase)
		if err != nil {
			return ArchiveResult{}, fmt.Errorf("open external Stats database: %w", err)
		}
		version, versionErr := stats.SchemaVersion(ctx)
		closeErr := stats.Close()
		if versionErr != nil {
			return ArchiveResult{}, fmt.Errorf("read external Stats schema: %w", versionErr)
		}
		if closeErr != nil {
			return ArchiveResult{}, closeErr
		}
		if version != store.StatsSchemaVersion {
			return ArchiveResult{}, fmt.Errorf("Stats schema %d, expected %d", version, store.StatsSchemaVersion)
		}
	}
	members := make([]ArchiveMember, 0, len(names))
	for _, name := range names {
		member, err := archiveMember(temp, name)
		if err != nil {
			return ArchiveResult{}, err
		}
		members = append(members, member)
	}
	manifest := BackupManifest{
		FormatVersion: archiveFormatVersion, Application: archiveApplication, ApplicationVersion: s.version,
		CreatedAt: now.Format("2006-01-02T15:04:05Z"), DashboardSchema: store.DashboardSchemaVersion, StatsSchema: store.StatsSchemaVersion,
		AggregateVersion: store.AggregateContractVersion, StatsDriver: driver, StatsBackupMode: mode, Members: members,
	}
	if err := writeJSONFile(filepath.Join(temp, "manifest.json"), manifest); err != nil {
		return ArchiveResult{}, err
	}
	zipNames := append([]string{"manifest.json"}, names...)
	if err := createZip(outputPath, temp, zipNames); err != nil {
		return ArchiveResult{}, fmt.Errorf("create backup archive: %w", err)
	}
	message := "Dashboard and Stats SQLite databases are included"
	if mode == "external_required" {
		message = "Dashboard database is included; the external Stats database requires a separate native backup"
	}
	return ArchiveResult{Path: outputPath, StatsBackupMode: mode, Message: message}, nil
}

func sanitizeStatsSnapshot(ctx context.Context, path string) error {
	return sanitizeSQLiteSnapshot(ctx, path,
		"DELETE FROM lps_chat_outbox",
		"UPDATE lps_sessions SET ip_address='' WHERE ip_address<>''",
	)
}

func sanitizeDashboardSnapshot(ctx context.Context, path string) error {
	return sanitizeSQLiteSnapshot(ctx, path,
		"UPDATE geoip_settings SET api_key='', cache_secret='' WHERE singleton_id=1",
		"DELETE FROM geoip_cache",
	)
}

func sanitizeSQLiteSnapshot(ctx context.Context, path string, statements ...string) error {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "PRAGMA secure_delete=ON"); err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		return err
	}
	return nil
}
