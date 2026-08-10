package service

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	modernsqlite "modernc.org/sqlite"
)

const (
	archiveFormatVersion = 1
	archiveApplication   = "l4d2-player-stats"
)

type ArchiveMember struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type BackupManifest struct {
	FormatVersion      int             `json:"format_version"`
	Application        string          `json:"application"`
	ApplicationVersion string          `json:"application_version"`
	CreatedAt          string          `json:"created_at"`
	DashboardSchema    int64           `json:"dashboard_schema"`
	StatsSchema        int64           `json:"stats_schema"`
	AggregateVersion   int64           `json:"aggregate_version"`
	StatsDriver        string          `json:"stats_driver"`
	StatsBackupMode    string          `json:"stats_backup_mode"`
	Members            []ArchiveMember `json:"members"`
}

type ArchiveResult struct {
	Path            string
	StatsBackupMode string
	Message         string
}

type ArchiveService struct {
	cfg     *config.Config
	version string
	now     func() time.Time
	rename  func(string, string) error
}

func NewArchiveService(cfg *config.Config, version string) *ArchiveService {
	return &ArchiveService{cfg: cfg, version: version, now: time.Now, rename: os.Rename}
}

func normalizeStatsDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "pgsql", "postgresql", "postgres":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func (s *ArchiveService) archivePath(outputDirectory, prefix string, now time.Time) (string, error) {
	if outputDirectory == "" {
		outputDirectory = "."
	}
	abs, err := filepath.Abs(outputDirectory)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return "", err
	}
	name := prefix + "-" + now.UTC().Format("20060102T150405Z") + ".zip"
	return filepath.Join(abs, name), nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func archiveMember(root, name string) (ArchiveMember, error) {
	path := filepath.Join(root, filepath.FromSlash(name))
	f, err := os.Open(path)
	if err != nil {
		return ArchiveMember{}, err
	}
	defer f.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, f)
	if err != nil {
		return ArchiveMember{}, err
	}
	return ArchiveMember{Path: name, Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func createZip(path, root string, names []string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		_ = f.Close()
		if !succeeded {
			_ = os.Remove(path)
		}
	}()
	zw := zip.NewWriter(f)
	for _, name := range names {
		source, err := os.Open(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			_ = zw.Close()
			return err
		}
		info, err := source.Stat()
		if err != nil {
			source.Close()
			_ = zw.Close()
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			source.Close()
			_ = zw.Close()
			return err
		}
		header.Name = name
		header.Method = zip.Deflate
		header.SetMode(0o600)
		writer, err := zw.CreateHeader(header)
		if err == nil {
			_, err = io.Copy(writer, source)
		}
		source.Close()
		if err != nil {
			_ = zw.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsPermission(err) {
		return err
	}
	succeeded = true
	return nil
}

type sqliteOnlineBackuper interface {
	NewBackup(string) (*modernsqlite.Backup, error)
}

func sqliteOnlineBackup(ctx context.Context, sourceDSN, destination string) error {
	if !strings.HasPrefix(sourceDSN, "file:") {
		sourceDSN = "file:" + filepath.ToSlash(sourceDSN)
	}
	db, err := sql.Open("sqlite", sourceDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		backuper, ok := driverConn.(sqliteOnlineBackuper)
		if !ok {
			return fmt.Errorf("modernc SQLite connection does not support online backup")
		}
		backup, err := backuper.NewBackup(filepath.ToSlash(destination))
		if err != nil {
			return err
		}
		_, stepErr := backup.Step(-1)
		finishErr := backup.Finish()
		if stepErr != nil {
			return stepErr
		}
		return finishErr
	})
}

func validateSQLiteDatabase(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite integrity_check returned %q", result)
	}
	return nil
}

func validateDashboardSnapshot(ctx context.Context, path string) error {
	if err := validateSQLiteDatabase(ctx, path); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	var schema, unsupported int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version_id),0) FROM goose_db_version WHERE is_applied=1`).Scan(&schema); err != nil {
		return err
	}
	if schema != store.DashboardSchemaVersion {
		return fmt.Errorf("dashboard schema %d, expected %d", schema, store.DashboardSchemaVersion)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
SELECT aggregate_version FROM aggregate_rows
UNION ALL SELECT aggregate_version FROM aggregate_monthly_rows
UNION ALL SELECT aggregate_version FROM aggregate_lifetime_rows
UNION ALL SELECT aggregate_version FROM aggregate_state
UNION ALL SELECT aggregate_version FROM retention_runs
) versions WHERE aggregate_version <> ?`, store.AggregateContractVersion).Scan(&unsupported); err != nil {
		return err
	}
	if unsupported != 0 {
		return fmt.Errorf("found %d unsupported aggregate contract rows", unsupported)
	}
	return nil
}

func validateStatsSnapshot(ctx context.Context, path string) error {
	if err := validateSQLiteDatabase(ctx, path); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	var schema int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM lps_schema_migrations`).Scan(&schema); err != nil {
		return err
	}
	if schema != store.StatsSchemaVersion {
		return fmt.Errorf("stats schema %d, expected %d", schema, store.StatsSchemaVersion)
	}
	return nil
}
