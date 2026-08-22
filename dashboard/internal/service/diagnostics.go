package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

const diagnosticsLogLimit int64 = 1 << 20

type DiagnosticsManifest struct {
	FormatVersion      int             `json:"format_version"`
	Application        string          `json:"application"`
	ApplicationVersion string          `json:"application_version"`
	CreatedAt          string          `json:"created_at"`
	Members            []ArchiveMember `json:"members"`
}

type diagnosticValue struct {
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func (s *ArchiveService) ExportDiagnostics(ctx context.Context, outputDirectory string) (ArchiveResult, error) {
	now := s.now().UTC()
	outputPath, err := s.archivePath(outputDirectory, "diagnostics", now)
	if err != nil {
		return ArchiveResult{}, err
	}
	temp, err := os.MkdirTemp("", "l4d2-stats-diagnostics-")
	if err != nil {
		return ArchiveResult{}, err
	}
	defer os.RemoveAll(temp)
	if err := os.Chmod(temp, 0o700); err != nil && !os.IsPermission(err) {
		return ArchiveResult{}, err
	}

	names := make([]string, 0, 8)
	addJSON := func(name string, value any) error {
		target := filepath.Join(temp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		contents, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		contents = append(contents, '\n')
		if err := os.WriteFile(target, RedactSensitive(contents), 0o600); err != nil {
			return err
		}
		names = append(names, name)
		return nil
	}

	configBytes, err := os.ReadFile(s.cfg.Path)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("read configuration: %w", err)
	}
	if err := os.WriteFile(filepath.Join(temp, "config.redacted.yaml"), RedactSensitive(configBytes), 0o600); err != nil {
		return ArchiveResult{}, err
	}
	names = append(names, "config.redacted.yaml")

	dashboard, dashboardErr := store.OpenDashboard(ctx, s.cfg.DashboardDatabase.Path)
	if dashboardErr == nil {
		defer dashboard.Close()
	}
	stats, statsErr := store.OpenStats(ctx, s.cfg.StatsDatabase)
	if statsErr == nil {
		defer stats.Close()
	}
	chatAudit, chatAuditErr := store.OpenChatAudit(ctx, s.cfg.ChatAudit.DatabasePath)
	if chatAuditErr == nil {
		defer chatAudit.Close()
	}
	quick := map[string]diagnosticValue{
		"config":              {Value: "ok"},
		"dashboard_database":  diagnosticErrorValue(dashboardErr),
		"stats_database":      diagnosticErrorValue(statsErr),
		"chat_audit_database": diagnosticErrorValue(chatAuditErr),
	}
	if dashboardErr == nil {
		version, err := dashboard.MigrationVersion(ctx)
		quick["dashboard_schema"] = diagnosticResult(version, err)
		configured, err := dashboard.AdminConfigured(ctx)
		quick["administrator_configured"] = diagnosticResult(configured, err)
		site, err := dashboard.Site(ctx)
		quick["site_configured"] = diagnosticResult(site.Configured, err)
	}
	if statsErr == nil {
		version, err := stats.SchemaVersion(ctx)
		quick["stats_schema"] = diagnosticResult(version, err)
	}
	if err := addJSON("doctor/quick.json", quick); err != nil {
		return ArchiveResult{}, err
	}
	deep := DoctorReport{Checks: []DoctorCheck{{Status: "error", Name: "deep_doctor", Message: "database connection unavailable"}}}
	if dashboardErr == nil && statsErr == nil {
		doctor := NewDoctorService(dashboard, stats)
		if chatAuditErr == nil {
			doctor.WithAudit(chatAudit, dashboard)
		}
		deep = doctor.Deep(ctx)
	}
	if err := addJSON("doctor/deep.json", deep); err != nil {
		return ArchiveResult{}, err
	}

	aggregate := diagnosticValue{}
	databaseUsage := map[string]diagnosticValue{}
	servers := diagnosticValue{Value: []store.GameServer{}}
	if dashboardErr != nil {
		aggregate.Error = dashboardErr.Error()
		databaseUsage["dashboard"] = diagnosticErrorValue(dashboardErr)
		servers.Error = dashboardErr.Error()
	} else {
		value, err := dashboard.AggregateStatus(ctx)
		aggregate = diagnosticResult(value, err)
		usage, err := dashboard.DatabaseUsage(ctx)
		databaseUsage["dashboard"] = diagnosticResult(usage, err)
		valueServers, err := dashboard.ListServers(ctx)
		servers = diagnosticResult(valueServers, err)
	}
	if statsErr != nil {
		databaseUsage["stats"] = diagnosticErrorValue(statsErr)
	} else {
		usage, err := stats.DatabaseUsage(ctx)
		databaseUsage["stats"] = diagnosticResult(usage, err)
	}
	if chatAuditErr != nil {
		databaseUsage["chat_audit"] = diagnosticErrorValue(chatAuditErr)
	} else {
		usage, err := chatAudit.DatabaseUsage(ctx)
		databaseUsage["chat_audit"] = diagnosticResult(usage, err)
	}
	if dashboardErr == nil {
		geoIP, err := dashboard.GeoIPSettings(ctx, 0)
		if err := addJSON("geoip-status.json", diagnosticResult(geoIP, err)); err != nil {
			return ArchiveResult{}, err
		}
	}
	if err := addJSON("aggregate-status.json", aggregate); err != nil {
		return ArchiveResult{}, err
	}
	if err := addJSON("database-usage.json", databaseUsage); err != nil {
		return ArchiveResult{}, err
	}
	if err := addJSON("servers.json", servers); err != nil {
		return ArchiveResult{}, err
	}

	if logBytes, err := tailFile(s.cfg.Logging.File, diagnosticsLogLimit); err == nil && len(logBytes) > 0 {
		if err := os.MkdirAll(filepath.Join(temp, "logs"), 0o700); err != nil {
			return ArchiveResult{}, err
		}
		if err := os.WriteFile(filepath.Join(temp, "logs", "recent.log"), RedactSensitive(logBytes), 0o600); err != nil {
			return ArchiveResult{}, err
		}
		names = append(names, "logs/recent.log")
	}
	sort.Strings(names)
	members := make([]ArchiveMember, 0, len(names))
	for _, name := range names {
		member, err := archiveMember(temp, name)
		if err != nil {
			return ArchiveResult{}, err
		}
		members = append(members, member)
	}
	manifest := DiagnosticsManifest{FormatVersion: archiveFormatVersion, Application: archiveApplication, ApplicationVersion: s.version, CreatedAt: now.Format(time.RFC3339), Members: members}
	if err := writeJSONFile(filepath.Join(temp, "manifest.json"), manifest); err != nil {
		return ArchiveResult{}, err
	}
	if err := createZip(outputPath, temp, append([]string{"manifest.json"}, names...)); err != nil {
		return ArchiveResult{}, err
	}
	message := "diagnostics exported"
	if deep.HasErrors() {
		message = "diagnostics exported; deep doctor reported errors in the included report"
	}
	return ArchiveResult{Path: outputPath, Message: message}, nil
}

func diagnosticErrorValue(err error) diagnosticValue {
	if err != nil {
		return diagnosticValue{Error: err.Error()}
	}
	return diagnosticValue{Value: "ok"}
}

func diagnosticResult(value any, err error) diagnosticValue {
	if err != nil {
		return diagnosticValue{Error: err.Error()}
	}
	return diagnosticValue{Value: value}
}

func tailFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	start := info.Size() - limit
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	contents, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, err
	}
	if start > 0 {
		if index := bytesIndexByte(contents, '\n'); index >= 0 {
			contents = contents[index+1:]
		}
	}
	return contents, nil
}

func bytesIndexByte(value []byte, target byte) int {
	for index, current := range value {
		if current == target {
			return index
		}
	}
	return -1
}
