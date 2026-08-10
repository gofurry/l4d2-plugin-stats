package service

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	_ "modernc.org/sqlite"
)

func TestCreateAndRestoreSQLiteBackup(t *testing.T) {
	ctx := context.Background()
	source := newArchiveFixture(t, "source")
	insertMarker(t, source.DashboardDatabase.Path, "source-dashboard")
	insertMarker(t, source.StatsDatabase.DSN, "source-stats")
	wal, err := sql.Open("sqlite", "file:"+filepath.ToSlash(source.DashboardDatabase.Path)+"?_pragma=journal_mode(WAL)&_pragma=wal_autocheckpoint(0)")
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	if _, err := wal.Exec(`UPDATE archive_test_marker SET value='source-dashboard'`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source.DashboardDatabase.Path + "-wal"); err != nil {
		t.Fatalf("WAL was not active during backup: %v", err)
	}

	backupService := NewArchiveService(source, "1.2.0")
	backupService.now = func() time.Time { return time.Date(2026, 8, 10, 1, 2, 3, 0, time.UTC) }
	backup, err := backupService.CreateBackup(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if backup.StatsBackupMode != "sqlite_online" {
		t.Fatalf("backup mode = %q", backup.StatsBackupMode)
	}
	assertZipModeAndManifest(t, backup.Path)

	current := newArchiveFixture(t, "current")
	insertMarker(t, current.DashboardDatabase.Path, "current-dashboard")
	insertMarker(t, current.StatsDatabase.DSN, "current-stats")
	restoreService := NewArchiveService(current, "1.2.0")
	restoreService.now = func() time.Time { return time.Date(2026, 8, 10, 2, 3, 4, 0, time.UTC) }
	result, err := restoreService.RestoreBackup(ctx, backup.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RollbackCopies) < 3 {
		t.Fatalf("rollback copies = %#v", result.RollbackCopies)
	}
	assertMarker(t, current.DashboardDatabase.Path, "source-dashboard")
	assertMarker(t, current.StatsDatabase.DSN, "source-stats")
	reloaded, err := config.Load(current.Path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.DashboardDatabase.Path != current.DashboardDatabase.Path || reloaded.StatsDatabase.DSN != current.StatsDatabase.DSN {
		t.Fatalf("restored paths changed: dashboard=%q stats=%q", reloaded.DashboardDatabase.Path, reloaded.StatsDatabase.DSN)
	}
	for _, path := range result.RollbackCopies {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("rollback copy %q: %v", path, err)
		}
	}
}

func TestRestoreExternalDatabaseBackupLeavesStatsForNativeRestore(t *testing.T) {
	ctx := context.Background()
	source := newArchiveFixture(t, "external-source")
	insertMarker(t, source.DashboardDatabase.Path, "external-dashboard")
	backupService := NewArchiveService(source, "1.2.0")
	backupService.now = func() time.Time { return time.Date(2026, 8, 10, 2, 30, 0, 0, time.UTC) }
	backup, err := backupService.CreateBackup(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	externalArchive := filepath.Join(t.TempDir(), "external.zip")
	convertToExternalBackup(t, backup.Path, externalArchive)

	current := newArchiveFixture(t, "external-current")
	insertMarker(t, current.StatsDatabase.DSN, "current-stats-untouched")
	currentStatsPath := current.StatsDatabase.DSN
	current.StatsDatabase.Driver = "mysql"
	current.StatsDatabase.DSN = "stats:secret@tcp(127.0.0.1:3306)/l4d2_stats"
	restoreService := NewArchiveService(current, "1.2.0")
	restoreService.now = func() time.Time { return time.Date(2026, 8, 10, 2, 31, 0, 0, time.UTC) }
	result, err := restoreService.RestoreBackup(ctx, externalArchive)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Message, "external Stats database separately") {
		t.Fatalf("restore message = %q", result.Message)
	}
	assertMarker(t, current.DashboardDatabase.Path, "external-dashboard")
	assertMarker(t, currentStatsPath, "current-stats-untouched")
}

func TestBackupRejectsUnsafeAndCorruptArchives(t *testing.T) {
	ctx := context.Background()
	cfg := newArchiveFixture(t, "reject")
	service := NewArchiveService(cfg, "1.2.0")
	service.now = func() time.Time { return time.Date(2026, 8, 10, 3, 4, 5, 0, time.UTC) }
	backup, err := service.CreateBackup(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	unsafePath := filepath.Join(t.TempDir(), "unsafe.zip")
	writeZip(t, unsafePath, map[string][]byte{"../config.yaml": []byte("secret")})
	if _, err := service.RestoreBackup(ctx, unsafePath); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unsafe archive error = %v", err)
	}
	duplicatePath := filepath.Join(t.TempDir(), "duplicate.zip")
	writeDuplicateZip(t, duplicatePath, "config.yaml")
	if _, err := service.RestoreBackup(ctx, duplicatePath); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate archive error = %v", err)
	}

	corruptPath := filepath.Join(t.TempDir(), "corrupt.zip")
	copyZipReplacing(t, backup.Path, corruptPath, "config.yaml", []byte("same-size-corrupt-config"))
	if _, err := service.RestoreBackup(ctx, corruptPath); err == nil || !strings.Contains(err.Error(), "wrong size") && !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("corrupt archive error = %v", err)
	}
}

func TestRestoreReplacementRollsBackOnInstallFailure(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	second := filepath.Join(dir, "second")
	firstStage := first + ".stage"
	secondStage := second + ".stage"
	for path, contents := range map[string]string{first: "old-first", second: "old-second", firstStage: "new-first", secondStage: "new-second"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	svc := &ArchiveService{rename: os.Rename}
	installCalls := 0
	svc.rename = func(oldPath, newPath string) error {
		if strings.Contains(oldPath, ".stage") {
			installCalls++
			if installCalls == 2 {
				return fmt.Errorf("injected install failure")
			}
		}
		return os.Rename(oldPath, newPath)
	}
	_, err := svc.replaceRestoreTargets([]restoreTarget{{target: first, stage: firstStage}, {target: second, stage: secondStage}}, "20260810T040506Z")
	if err == nil {
		t.Fatal("replacement unexpectedly succeeded")
	}
	for path, expected := range map[string]string{first: "old-first", second: "old-second"} {
		contents, readErr := os.ReadFile(path)
		if readErr != nil || string(contents) != expected {
			t.Fatalf("rollback %s = %q, %v", path, contents, readErr)
		}
	}
}

func TestDiagnosticsAreRedactedAndExcludeDatabases(t *testing.T) {
	cfg := newArchiveFixture(t, "diagnostics")
	secret := "diagnostic-password"
	if err := os.WriteFile(cfg.Logging.File, []byte("Authorization: Bearer "+secret+"\nCookie: sid="+secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configBytes, err := os.ReadFile(cfg.Path)
	if err != nil {
		t.Fatal(err)
	}
	configBytes = append(configBytes, []byte("# postgres://user:"+secret+"@localhost/stats\n")...)
	if err := os.WriteFile(cfg.Path, configBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewArchiveService(cfg, "1.2.0")
	svc.now = func() time.Time { return time.Date(2026, 8, 10, 5, 6, 7, 0, time.UTC) }
	result, err := svc.ExportDiagnostics(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	foundDeep := false
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".db") {
			t.Fatalf("diagnostics contains database %q", file.Name)
		}
		contents := readZipEntry(t, file)
		if strings.Contains(string(contents), secret) {
			t.Fatalf("diagnostics member %q leaked secret", file.Name)
		}
		if file.Name == "doctor/deep.json" {
			foundDeep = true
		}
	}
	if !foundDeep {
		t.Fatal("deep doctor report missing")
	}
}

func TestRedactSensitiveCredentials(t *testing.T) {
	input := []byte("dsn: postgres://user:secret@host/db?password=querysecret&openid.claimed_id=openidsecret\nAuthorization: Bearer authsecret\nCookie: sid=cookiesecret\npassword_hash: hashsecret\nsetup_token=setupsecret\neyJabc.def.ghi\n")
	redacted := string(RedactSensitive(input))
	for _, secret := range []string{"secret@", "querysecret", "openidsecret", "authsecret", "cookiesecret", "hashsecret", "setupsecret", "eyJabc.def.ghi"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redaction leaked %q in %q", secret, redacted)
		}
	}
	if !strings.Contains(redacted, "[REDACTED]") {
		t.Fatalf("redaction marker missing: %q", redacted)
	}
}

func newArchiveFixture(t *testing.T, name string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	dashboardPath := filepath.Join(dir, name+"-dashboard.db")
	statsPath := filepath.Join(dir, name+"-stats.db")
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, dashboardPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := dashboard.Close(); err != nil {
		t.Fatal(err)
	}
	createStatsFixture(t, statsPath)
	listen := unusedListenAddress(t)
	configPath := filepath.Join(dir, "config.yaml")
	logPath := filepath.Join(dir, "dashboard.log")
	contents := fmt.Sprintf("server:\n  listen: %q\ndashboard_database:\n  path: %q\nstats_database:\n  driver: sqlite\n  dsn: %q\n  query_timeout: 5s\n  max_open_conns: 2\n  max_idle_conns: 1\n  conn_max_lifetime: 1m\nlogging:\n  level: info\n  format: json\n  file: %q\n  max_size_mb: 10\n  max_backups: 1\n  max_age_days: 1\n  compress: false\n  also_console: false\nmonitor:\n  enabled: false\n  refresh: 2s\n  disk_paths: []\n", listen, dashboardPath, statsPath, logPath)
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func createStatsFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, current, _, _ := runtime.Caller(0)
	migration := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "..", "database", "migrations", "sqlite", "0001_initial.sql"))
	contents, err := os.ReadFile(migration)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(contents), "-- statement-breakpoint") {
		if statement = strings.TrimSpace(statement); statement != "" {
			if _, err := db.Exec(statement); err != nil {
				t.Fatalf("apply Stats migration: %v", err)
			}
		}
	}
	if _, err := db.Exec(`INSERT INTO lps_schema_migrations VALUES (1,'initial',1)`); err != nil {
		t.Fatal(err)
	}
}

func insertMarker(t *testing.T, databasePath, value string) {
	t.Helper()
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS archive_test_marker (value TEXT NOT NULL); DELETE FROM archive_test_marker; INSERT INTO archive_test_marker VALUES (?)`, value); err != nil {
		t.Fatal(err)
	}
}

func assertMarker(t *testing.T, databasePath, expected string) {
	t.Helper()
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var actual string
	if err := db.QueryRow(`SELECT value FROM archive_test_marker`).Scan(&actual); err != nil || actual != expected {
		t.Fatalf("marker = %q, %v; expected %q", actual, err, expected)
	}
}

func unusedListenAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return address
}

func assertZipModeAndManifest(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("archive permissions = %o", info.Mode().Perm())
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != "manifest.json" {
			continue
		}
		var manifest BackupManifest
		if err := json.Unmarshal(readZipEntry(t, file), &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.DashboardSchema != 9 || manifest.StatsSchema != 1 || manifest.AggregateVersion != 1 || len(manifest.Members) != 3 {
			t.Fatalf("manifest = %#v", manifest)
		}
		return
	}
	t.Fatal("manifest missing")
}

func readZipEntry(t *testing.T, file *zip.File) []byte {
	t.Helper()
	reader, err := file.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	contents, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func writeZip(t *testing.T, target string, members map[string][]byte) {
	t.Helper()
	file, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, contents := range members {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeDuplicateZip(t *testing.T, target, name string) {
	t.Helper()
	file, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for range 2 {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("duplicate")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func copyZipReplacing(t *testing.T, source, target, replacementName string, replacement []byte) {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	members := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		members[file.Name] = readZipEntry(t, file)
	}
	members[replacementName] = replacement
	writeZip(t, target, members)
}

func convertToExternalBackup(t *testing.T, source, target string) {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	members := make(map[string][]byte, len(reader.File)-1)
	var manifest BackupManifest
	for _, file := range reader.File {
		contents := readZipEntry(t, file)
		if file.Name == "manifest.json" {
			if err := json.Unmarshal(contents, &manifest); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if file.Name != "stats/stats.db" {
			members[file.Name] = contents
		}
	}
	reader.Close()
	manifest.StatsDriver = "mysql"
	manifest.StatsBackupMode = "external_required"
	filtered := manifest.Members[:0]
	for _, member := range manifest.Members {
		if member.Path != "stats/stats.db" {
			filtered = append(filtered, member)
		}
	}
	manifest.Members = filtered
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	members["manifest.json"] = append(contents, '\n')
	writeZip(t, target, members)
}
