package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type DoctorCheck struct {
	Status  string `json:"status"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type DoctorReport struct {
	Checks []DoctorCheck `json:"checks"`
}

func (r DoctorReport) HasErrors() bool {
	for _, check := range r.Checks {
		if check.Status == "error" {
			return true
		}
	}
	return false
}

type doctorDashboard interface {
	MigrationVersion(context.Context) (int64, error)
	AggregateStatus(context.Context) (store.AggregateStatus, error)
	DataMaintenanceSettings(context.Context) (store.DataMaintenanceSettings, error)
}

type doctorStats interface {
	SchemaVersion(context.Context) (int64, error)
	DeepDataQuality(context.Context, int64) (store.StatsDataQuality, error)
}

type DoctorService struct {
	dashboard doctorDashboard
	stats     doctorStats
	chatAudit store.ChatAuditStore
	audit     store.DashboardAuditStore
	now       func() time.Time
}

func NewDoctorService(dashboard doctorDashboard, stats doctorStats) *DoctorService {
	return &DoctorService{dashboard: dashboard, stats: stats, now: time.Now}
}

func (s *DoctorService) WithAudit(chatAudit store.ChatAuditStore, dashboard store.DashboardAuditStore) *DoctorService {
	s.chatAudit = chatAudit
	s.audit = dashboard
	return s
}

func (s *DoctorService) Deep(ctx context.Context) DoctorReport {
	report := DoctorReport{Checks: make([]DoctorCheck, 0, 12)}
	statsVersion, statsVersionErr := s.stats.SchemaVersion(ctx)
	report.Checks = append(report.Checks, versionCheck("stats_schema", statsVersion, store.StatsSchemaVersion, statsVersionErr))
	dashboardVersion, dashboardVersionErr := s.dashboard.MigrationVersion(ctx)
	report.Checks = append(report.Checks, versionCheck("dashboard_schema", dashboardVersion, store.DashboardSchemaVersion, dashboardVersionErr))

	now := s.now().UTC()
	quality, qualityErr := s.stats.DeepDataQuality(ctx, now.Add(-15*time.Minute).Unix())
	if qualityErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{Status: "error", Name: "stats_data_quality", Message: qualityErr.Error()})
	} else {
		report.Checks = append(report.Checks,
			findingCheck("stats_version", quality.UnknownStatsVersion, "error"),
			findingCheck("lifecycle_links", quality.LifecycleLinks, "error"),
			findingCheck("mode_side_contract", quality.ModeSideMismatch, "error"),
			findingCheck("pve_totals", quality.PVETotalMismatch, "error"),
			findingCheck("round_context_contract", quality.ContextContract, "error"),
			findingCheck("incident_contract", quality.IncidentContract, "error"),
			findingCheck("incident_completeness", quality.IncidentCompleteness, "error"),
			findingCheck("relationship_contract", quality.RelationshipContract, "error"),
			findingCheck("pve_assist_contract", quality.PVEAssistContract, "error"),
			findingCheck("versus_assist_contract", quality.VersusAssistContract, "error"),
			findingCheck("fall_death_contract", quality.FallDeathContract, "error"),
			findingCheck("telemetry_contract", quality.TelemetryContract, "error"),
			findingCheck("chat_capture_contract", quality.ChatCaptureContract, "error"),
			findingCheck("active_boot_heartbeat", quality.StaleActiveBoots, "warning"),
		)
	}
	if s.chatAudit != nil {
		version, err := s.chatAudit.SchemaVersion(ctx)
		report.Checks = append(report.Checks, versionCheck("chat_audit_schema", version, store.ChatAuditSchemaVersion, err))
		if err == nil {
			settings, settingsErr := s.audit.ChatAuditSettings(ctx)
			if settingsErr != nil {
				report.Checks = append(report.Checks, DoctorCheck{Status: "error", Name: "chat_audit_health", Message: settingsErr.Error()})
			} else {
				status, statusErr := s.chatAudit.Status(ctx, settings, 0, 0)
				if statusErr != nil {
					report.Checks = append(report.Checks, DoctorCheck{Status: "error", Name: "chat_audit_health", Message: statusErr.Error()})
				} else {
					report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "chat_audit_health", Message: fmt.Sprintf("%d messages; %d known gaps", status.MessageCount, status.KnownGapCount)})
				}
			}
		}
	}
	if s.audit != nil {
		config, err := s.audit.GeoIPRuntimeConfig(ctx)
		if err != nil {
			report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "geoip_health", Message: err.Error()})
		} else if config.APIKey == "" {
			report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "geoip_health", Message: "disabled"})
		} else if config.LastErrorAt > config.LastSuccessAt {
			report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "geoip_health", Message: "provider status: " + config.LastErrorCode})
		} else {
			report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "geoip_health", Message: fmt.Sprintf("IPv4=%s IPv6=%s QPS=%d", config.IPv4Status, config.IPv6Status, config.QPSLimit)})
		}
	}

	status, statusErr := s.dashboard.AggregateStatus(ctx)
	if statusErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{Status: "error", Name: "aggregate_contract", Message: statusErr.Error()})
		return report
	}
	report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "aggregate_contract", Message: fmt.Sprintf("version %d", status.AggregateVersion)})
	if status.State != "ready" {
		report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "aggregate_state", Message: fmt.Sprintf("state is %q, expected ready", status.State)})
	} else {
		report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "aggregate_state", Message: "ready"})
	}
	if status.LastFinishedAt == 0 {
		report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "aggregate_last_success", Message: "aggregation has never completed successfully"})
	} else {
		report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "aggregate_last_success", Message: time.Unix(status.LastFinishedAt, 0).UTC().Format(time.RFC3339)})
	}
	if qualityErr == nil {
		if status.SourceWatermark < quality.SourceWatermark {
			report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "aggregate_watermark", Message: fmt.Sprintf("aggregate watermark %d is behind source watermark %d", status.SourceWatermark, quality.SourceWatermark)})
		} else {
			report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "aggregate_watermark", Message: fmt.Sprintf("covers source watermark %d", quality.SourceWatermark)})
		}
	}
	settings, settingsErr := s.dashboard.DataMaintenanceSettings(ctx)
	if settingsErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{Status: "error", Name: "aggregate_schedule", Message: settingsErr.Error()})
	} else if status.LastFinishedAt == 0 {
		report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "aggregate_schedule", Message: "freshness cannot be evaluated before the first successful aggregation"})
	} else {
		maximumAge := time.Duration(settings.AggregateIntervalMinutes*2) * time.Minute
		age := now.Sub(time.Unix(status.LastFinishedAt, 0))
		if age > maximumAge {
			report.Checks = append(report.Checks, DoctorCheck{Status: "warning", Name: "aggregate_schedule", Message: fmt.Sprintf("last success is %s old; maximum expected age is %s", age.Round(time.Second), maximumAge)})
		} else {
			report.Checks = append(report.Checks, DoctorCheck{Status: "ok", Name: "aggregate_schedule", Message: fmt.Sprintf("last success is %s old", age.Round(time.Second))})
		}
	}
	return report
}

func versionCheck(name string, actual, expected int64, err error) DoctorCheck {
	if err != nil {
		return DoctorCheck{Status: "error", Name: name, Message: err.Error()}
	}
	if actual != expected {
		return DoctorCheck{Status: "error", Name: name, Message: fmt.Sprintf("version %d, expected %d", actual, expected)}
	}
	return DoctorCheck{Status: "ok", Name: name, Message: fmt.Sprintf("version %d", actual)}
}

func findingCheck(name string, finding store.DataQualityFinding, severity string) DoctorCheck {
	if finding.Count == 0 {
		return DoctorCheck{Status: "ok", Name: name, Message: "no anomalies"}
	}
	message := fmt.Sprintf("%d anomalies", finding.Count)
	if len(finding.IDs) > 0 {
		message += " (sample IDs: " + strings.Join(finding.IDs, ", ") + ")"
	}
	return DoctorCheck{Status: severity, Name: name, Message: message}
}
