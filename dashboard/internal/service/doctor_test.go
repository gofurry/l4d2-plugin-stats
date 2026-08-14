package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type doctorDashboardStub struct {
	version   int64
	status    store.AggregateStatus
	settings  store.DataMaintenanceSettings
	statusErr error
}

func (s doctorDashboardStub) MigrationVersion(context.Context) (int64, error) { return s.version, nil }
func (s doctorDashboardStub) AggregateStatus(context.Context) (store.AggregateStatus, error) {
	return s.status, s.statusErr
}
func (s doctorDashboardStub) DataMaintenanceSettings(context.Context) (store.DataMaintenanceSettings, error) {
	return s.settings, nil
}

type doctorStatsStub struct {
	version int64
	quality store.StatsDataQuality
}

func (s doctorStatsStub) SchemaVersion(context.Context) (int64, error) { return s.version, nil }
func (s doctorStatsStub) DeepDataQuality(context.Context, int64) (store.StatsDataQuality, error) {
	return s.quality, nil
}

func TestDeepDoctorHealthyAndWarningExitSemantics(t *testing.T) {
	now := time.Unix(1700000000, 0)
	healthy := NewDoctorService(
		doctorDashboardStub{version: store.DashboardSchemaVersion, status: store.AggregateStatus{AggregateVersion: 1, State: "ready", LastFinishedAt: now.Add(-10 * time.Minute).Unix(), SourceWatermark: 100}, settings: store.DataMaintenanceSettings{AggregateIntervalMinutes: 30}},
		doctorStatsStub{version: store.StatsSchemaVersion, quality: store.StatsDataQuality{SourceWatermark: 100}},
	)
	healthy.now = func() time.Time { return now }
	report := healthy.Deep(context.Background())
	if report.HasErrors() {
		t.Fatalf("healthy report has errors: %+v", report)
	}
	for _, check := range report.Checks {
		if check.Status != "ok" {
			t.Fatalf("healthy check=%+v", check)
		}
	}

	warnings := NewDoctorService(
		doctorDashboardStub{version: store.DashboardSchemaVersion, status: store.AggregateStatus{AggregateVersion: 1, State: "empty", SourceWatermark: 50}, settings: store.DataMaintenanceSettings{AggregateIntervalMinutes: 30}},
		doctorStatsStub{version: store.StatsSchemaVersion, quality: store.StatsDataQuality{SourceWatermark: 100, StaleActiveBoots: store.DataQualityFinding{Count: 1, IDs: []string{"boot:b"}}}},
	)
	warnings.now = func() time.Time { return now }
	report = warnings.Deep(context.Background())
	if report.HasErrors() {
		t.Fatalf("warning-only report should keep exit status successful: %+v", report)
	}
	if !hasDoctorStatus(report, "warning") {
		t.Fatalf("warning report has no warnings: %+v", report)
	}
}

func TestDeepDoctorReportsAllDataErrors(t *testing.T) {
	service := NewDoctorService(
		doctorDashboardStub{version: 8, statusErr: errors.New("unsupported aggregate contract version 2")},
		doctorStatsStub{version: store.StatsSchemaVersion + 1, quality: store.StatsDataQuality{
			UnknownStatsVersion: store.DataQualityFinding{Count: 1}, LifecycleLinks: store.DataQualityFinding{Count: 1},
			ModeSideMismatch: store.DataQualityFinding{Count: 1}, PVETotalMismatch: store.DataQualityFinding{Count: 1},
			RelationshipContract: store.DataQualityFinding{Count: 1}, PVEAssistContract: store.DataQualityFinding{Count: 1}, VersusAssistContract: store.DataQualityFinding{Count: 1},
		}},
	)
	report := service.Deep(context.Background())
	if !report.HasErrors() {
		t.Fatalf("error report did not request a failing exit status: %+v", report)
	}
	if countDoctorStatus(report, "error") < 7 {
		t.Fatalf("not all errors were reported: %+v", report)
	}
}

func hasDoctorStatus(report DoctorReport, status string) bool {
	return countDoctorStatus(report, status) > 0
}

func countDoctorStatus(report DoctorReport, status string) int {
	count := 0
	for _, check := range report.Checks {
		if check.Status == status {
			count++
		}
	}
	return count
}
