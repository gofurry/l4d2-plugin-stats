package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"gopkg.in/yaml.v3"
)

const maximumManifestBytes = 1 << 20

type RestoreResult struct {
	ArchivePath    string
	RollbackCopies []string
	Message        string
}

func (s *ArchiveService) RestoreBackup(ctx context.Context, archivePath string) (RestoreResult, error) {
	absArchive, err := filepath.Abs(archivePath)
	if err != nil {
		return RestoreResult{}, err
	}
	info, err := os.Stat(absArchive)
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = fmt.Errorf("not a regular file")
		}
		return RestoreResult{}, fmt.Errorf("open backup archive: %w", err)
	}
	if err := ensureDashboardStopped(s.cfg.Server.Listen); err != nil {
		return RestoreResult{}, err
	}
	temp, err := os.MkdirTemp("", "l4d2-stats-restore-")
	if err != nil {
		return RestoreResult{}, err
	}
	defer os.RemoveAll(temp)
	if err := os.Chmod(temp, 0o700); err != nil && !os.IsPermission(err) {
		return RestoreResult{}, err
	}
	manifest, err := extractAndValidateBackup(absArchive, temp, normalizeStatsDriver(s.cfg.StatsDatabase.Driver))
	if err != nil {
		return RestoreResult{}, err
	}
	if err := validateDashboardSnapshot(ctx, filepath.Join(temp, "dashboard", "dashboard.db")); err != nil {
		return RestoreResult{}, fmt.Errorf("validate restored Dashboard database: %w", err)
	}
	if manifest.StatsBackupMode == "sqlite_online" {
		if err := validateStatsSnapshot(ctx, filepath.Join(temp, "stats", "stats.db")); err != nil {
			return RestoreResult{}, fmt.Errorf("validate restored Stats database: %w", err)
		}
	}
	configBytes, err := os.ReadFile(filepath.Join(temp, "config.yaml"))
	if err != nil {
		return RestoreResult{}, err
	}
	rewrittenConfig, err := rewriteRestoredConfig(configBytes, s.cfg)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("rewrite restored configuration: %w", err)
	}

	targets := []restoreTarget{
		{target: s.cfg.DashboardDatabase.Path, source: filepath.Join(temp, "dashboard", "dashboard.db"), database: true},
		{target: s.cfg.Path, contents: rewrittenConfig},
	}
	if manifest.StatsBackupMode == "sqlite_online" {
		statsPath, err := sqlitePathFromDSN(s.cfg.StatsDatabase.DSN)
		if err != nil {
			return RestoreResult{}, fmt.Errorf("resolve current Stats SQLite path: %w", err)
		}
		targets = append(targets, restoreTarget{target: statsPath, source: filepath.Join(temp, "stats", "stats.db"), database: true})
	}
	if err := validateRestoreTargets(targets); err != nil {
		return RestoreResult{}, err
	}
	stamp := s.now().UTC().Format("20060102T150405Z")
	for i := range targets {
		if err := targets[i].prepare(stamp); err != nil {
			cleanupRestoreStages(targets)
			return RestoreResult{}, fmt.Errorf("prepare restore staging file: %w", err)
		}
	}
	if _, err := config.Load(targets[1].stage); err != nil {
		cleanupRestoreStages(targets)
		return RestoreResult{}, fmt.Errorf("validate restored configuration: %w", err)
	}
	rollbackCopies, err := s.replaceRestoreTargets(targets, stamp)
	if err != nil {
		cleanupRestoreStages(targets)
		return RestoreResult{}, err
	}
	message := "Dashboard database, Stats SQLite database, and configuration restored"
	if manifest.StatsBackupMode == "external_required" {
		message = "Dashboard database and configuration restored; restore the external Stats database separately"
	}
	return RestoreResult{ArchivePath: absArchive, RollbackCopies: rollbackCopies, Message: message}, nil
}

func ensureDashboardStopped(listen string) error {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return fmt.Errorf("validate Dashboard listen address: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	connection, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 500*time.Millisecond)
	if err == nil {
		connection.Close()
		return fmt.Errorf("Dashboard service appears to be running on %s; stop it before restore", listen)
	}
	return nil
}

func extractAndValidateBackup(archivePath, destination, currentDriver string) (BackupManifest, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return BackupManifest{}, fmt.Errorf("open backup zip: %w", err)
	}
	defer reader.Close()
	allowed := map[string]bool{"manifest.json": true, "config.yaml": true, "dashboard/dashboard.db": true, "stats/stats.db": true}
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		name := file.Name
		if strings.Contains(name, "\\") || path.IsAbs(name) || path.Clean(name) != name || strings.HasPrefix(name, "../") || !allowed[name] {
			return BackupManifest{}, fmt.Errorf("backup contains unsafe or unknown member %q", name)
		}
		if _, exists := files[name]; exists {
			return BackupManifest{}, fmt.Errorf("backup contains duplicate member %q", name)
		}
		files[name] = file
	}
	manifestFile := files["manifest.json"]
	if manifestFile == nil || manifestFile.UncompressedSize64 > maximumManifestBytes {
		return BackupManifest{}, fmt.Errorf("backup manifest is missing or too large")
	}
	manifestBytes, err := readZipFile(manifestFile, maximumManifestBytes)
	if err != nil {
		return BackupManifest{}, err
	}
	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return BackupManifest{}, fmt.Errorf("decode backup manifest: %w", err)
	}
	if manifest.FormatVersion != archiveFormatVersion || manifest.Application != archiveApplication {
		return BackupManifest{}, fmt.Errorf("unsupported backup format or application")
	}
	if manifest.DashboardSchema != store.DashboardSchemaVersion || manifest.StatsSchema != store.StatsSchemaVersion || manifest.AggregateVersion != store.AggregateContractVersion {
		return BackupManifest{}, fmt.Errorf("backup schema or aggregate contract is incompatible")
	}
	if normalizeStatsDriver(manifest.StatsDriver) != currentDriver {
		return BackupManifest{}, fmt.Errorf("backup Stats driver %q does not match current driver %q", manifest.StatsDriver, currentDriver)
	}
	if manifest.StatsBackupMode != "sqlite_online" && manifest.StatsBackupMode != "external_required" {
		return BackupManifest{}, fmt.Errorf("unsupported Stats backup mode %q", manifest.StatsBackupMode)
	}
	if currentDriver == "sqlite" && manifest.StatsBackupMode != "sqlite_online" {
		return BackupManifest{}, fmt.Errorf("SQLite restore requires an included Stats database")
	}
	if currentDriver != "sqlite" && manifest.StatsBackupMode != "external_required" {
		return BackupManifest{}, fmt.Errorf("external Stats driver cannot restore an embedded SQLite database")
	}
	expected := map[string]ArchiveMember{}
	for _, member := range manifest.Members {
		if member.Path == "manifest.json" || !allowed[member.Path] {
			return BackupManifest{}, fmt.Errorf("manifest contains unknown member %q", member.Path)
		}
		if _, duplicate := expected[member.Path]; duplicate {
			return BackupManifest{}, fmt.Errorf("manifest contains duplicate member %q", member.Path)
		}
		expected[member.Path] = member
	}
	required := []string{"config.yaml", "dashboard/dashboard.db"}
	if manifest.StatsBackupMode == "sqlite_online" {
		required = append(required, "stats/stats.db")
	}
	if len(expected) != len(required) || len(files) != len(required)+1 {
		return BackupManifest{}, fmt.Errorf("backup member set does not match its manifest")
	}
	for _, name := range required {
		member, ok := expected[name]
		file := files[name]
		if !ok || file == nil || int64(file.UncompressedSize64) != member.Size {
			return BackupManifest{}, fmt.Errorf("backup member %q is missing or has the wrong size", name)
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return BackupManifest{}, err
		}
		if err := extractZipMember(file, target, member); err != nil {
			return BackupManifest{}, err
		}
	}
	return manifest, nil
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, limit+1))
}

func extractZipMember(file *zip.File, target string, member ArchiveMember) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, hash), reader)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != member.Size || hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(member.SHA256) {
		return fmt.Errorf("backup member %q failed size or SHA-256 validation", member.Path)
	}
	return nil
}

func rewriteRestoredConfig(contents []byte, current *config.Config) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(contents, &document); err != nil {
		return nil, err
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("configuration root is not a mapping")
	}
	root := document.Content[0]
	if err := setYAMLScalar(root, []string{"dashboard_database", "path"}, current.DashboardDatabase.Path); err != nil {
		return nil, err
	}
	if err := setYAMLScalar(root, []string{"stats_database", "driver"}, current.StatsDatabase.Driver); err != nil {
		return nil, err
	}
	if err := setYAMLScalar(root, []string{"stats_database", "dsn"}, current.StatsDatabase.DSN); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func setYAMLScalar(mapping *yaml.Node, keys []string, value string) error {
	current := mapping
	for index, key := range keys {
		if current.Kind != yaml.MappingNode {
			return fmt.Errorf("configuration path %s is not a mapping", strings.Join(keys[:index], "."))
		}
		var next *yaml.Node
		for i := 0; i+1 < len(current.Content); i += 2 {
			if current.Content[i].Value == key {
				next = current.Content[i+1]
				break
			}
		}
		if next == nil {
			return fmt.Errorf("configuration is missing %s", strings.Join(keys[:index+1], "."))
		}
		if index == len(keys)-1 {
			next.Kind = yaml.ScalarNode
			next.Tag = "!!str"
			next.Value = value
			return nil
		}
		current = next
	}
	return nil
}

func sqlitePathFromDSN(dsn string) (string, error) {
	value := strings.TrimSpace(dsn)
	if strings.Contains(strings.ToLower(value), "mode=memory") {
		return "", fmt.Errorf("Stats SQLite DSN does not identify a restorable file")
	}
	if strings.HasPrefix(value, "file:") {
		value = strings.TrimPrefix(value, "file:")
		if index := strings.IndexByte(value, '?'); index >= 0 {
			value = value[:index]
		}
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return "", err
		}
		value = decoded
		if len(value) >= 3 && value[0] == '/' && value[2] == ':' {
			value = value[1:]
		}
	}
	if value == "" || value == ":memory:" {
		return "", fmt.Errorf("Stats SQLite DSN does not identify a restorable file")
	}
	return filepath.Abs(filepath.FromSlash(value))
}

type restoreTarget struct {
	target   string
	source   string
	contents []byte
	database bool
	stage    string
}

func (t *restoreTarget) prepare(stamp string) error {
	abs, err := filepath.Abs(t.target)
	if err != nil {
		return err
	}
	t.target = abs
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return err
	}
	t.stage = abs + ".restore-stage-" + stamp
	if t.source != "" {
		return copyFileExclusive(t.source, t.stage, 0o600)
	}
	file, err := os.OpenFile(t.stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(t.contents); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func copyFileExclusive(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func validateRestoreTargets(targets []restoreTarget) error {
	seen := make(map[string]bool, len(targets))
	for _, target := range targets {
		abs, err := filepath.Abs(target.target)
		if err != nil {
			return err
		}
		key := strings.ToLower(filepath.Clean(abs))
		if seen[key] {
			return fmt.Errorf("restore targets resolve to the same path %q", abs)
		}
		seen[key] = true
	}
	return nil
}

type movedRestoreFile struct{ original, backup string }

func (s *ArchiveService) replaceRestoreTargets(targets []restoreTarget, stamp string) ([]string, error) {
	var moved []movedRestoreFile
	var installed []string
	rollback := func(cause error) error {
		var rollbackErrors []error
		for i := len(installed) - 1; i >= 0; i-- {
			if err := os.Remove(installed[i]); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		for i := len(moved) - 1; i >= 0; i-- {
			if err := s.rename(moved[i].backup, moved[i].original); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		return errors.Join(append([]error{cause}, rollbackErrors...)...)
	}
	for _, target := range targets {
		candidates := []string{target.target}
		if target.database {
			candidates = append(candidates, target.target+"-wal", target.target+"-shm")
		}
		for _, original := range candidates {
			if _, err := os.Stat(original); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, rollback(err)
			}
			backup := original + ".pre-restore-" + stamp
			if _, err := os.Stat(backup); err == nil {
				return nil, rollback(fmt.Errorf("rollback copy already exists: %s", backup))
			} else if !os.IsNotExist(err) {
				return nil, rollback(err)
			}
			if err := s.rename(original, backup); err != nil {
				return nil, rollback(err)
			}
			moved = append(moved, movedRestoreFile{original: original, backup: backup})
		}
	}
	for _, target := range targets {
		if err := s.rename(target.stage, target.target); err != nil {
			return nil, rollback(fmt.Errorf("install restored file %s: %w", target.target, err))
		}
		installed = append(installed, target.target)
	}
	rollbackCopies := make([]string, 0, len(moved))
	for _, file := range moved {
		rollbackCopies = append(rollbackCopies, file.backup)
	}
	return rollbackCopies, nil
}

func cleanupRestoreStages(targets []restoreTarget) {
	for _, target := range targets {
		if target.stage != "" {
			_ = os.Remove(target.stage)
		}
	}
}
