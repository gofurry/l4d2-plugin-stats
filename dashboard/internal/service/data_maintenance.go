package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

var allowedAggregateIntervals = map[int64]struct{}{15: {}, 30: {}, 60: {}, 180: {}, 300: {}, 720: {}, 1440: {}}

type DataGrowthStatus struct {
	Aggregate     store.AggregateStatus         `json:"aggregate"`
	Settings      store.DataMaintenanceSettings `json:"settings"`
	StatsDatabase store.DatabaseUsage           `json:"stats_database"`
	DashboardDB   store.DatabaseUsage           `json:"dashboard_database"`
	LogBytes      int64                         `json:"log_bytes"`
	RetentionRuns int64                         `json:"retention_runs"`
	RetentionPlan store.RetentionPlan           `json:"retention_plan"`
	Analysis      store.AnalysisStatus          `json:"analysis"`
	IncidentPlan  store.IncidentRetentionPlan   `json:"incident_retention_plan"`
	ChatAudit     *store.ChatAuditStatus        `json:"chat_audit,omitempty"`
	GeoIP         *store.GeoIPSettings          `json:"geoip,omitempty"`
}

type chatAuditStatusSource interface {
	Status(context.Context) (store.ChatAuditStatus, error)
}

type geoIPStatusSource interface {
	Settings(context.Context) (store.GeoIPSettings, error)
}

type DataMaintenanceService struct {
	dashboard  store.DashboardAggregateStore
	stats      store.StatsAggregateStore
	aggregates *AggregateService
	statsCfg   config.StatsDatabaseConfig
	logFile    string
	logger     *zap.Logger
	chatAudit  chatAuditStatusSource
	geoIP      geoIPStatusSource
}

func (s *DataMaintenanceService) SetAuditSources(chatAudit chatAuditStatusSource, geoIP geoIPStatusSource) {
	s.chatAudit = chatAudit
	s.geoIP = geoIP
}

func NewDataMaintenanceService(dashboard store.DashboardAggregateStore, stats store.StatsAggregateStore, aggregates *AggregateService, statsCfg config.StatsDatabaseConfig, logFile string, logger *zap.Logger) *DataMaintenanceService {
	return &DataMaintenanceService{dashboard: dashboard, stats: stats, aggregates: aggregates, statsCfg: statsCfg, logFile: logFile, logger: logger}
}

func (s *DataMaintenanceService) Status(ctx context.Context) (DataGrowthStatus, error) {
	settings, err := s.dashboard.DataMaintenanceSettings(ctx)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	aggregate, err := s.dashboard.AggregateStatus(ctx)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	statsUsage, err := s.stats.DatabaseUsage(ctx)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	dashboardUsage, err := s.dashboard.DatabaseUsage(ctx)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	runs, err := s.dashboard.RetentionRunCount(ctx)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	plan, err := s.plan(ctx, settings, aggregate)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	analysisStore, ok := s.stats.(store.StatsAnalysisMaintenanceStore)
	if !ok {
		return DataGrowthStatus{}, fmt.Errorf("analysis maintenance is unavailable")
	}
	analysis, err := analysisStore.AnalysisStatus(ctx, settings.IncidentRetentionDays)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	incidentRuns, err := s.dashboard.IncidentRetentionRunCount(ctx)
	if err != nil {
		return DataGrowthStatus{}, err
	}
	analysis.CleanupRuns = incidentRuns
	incidentPlan, err := analysisStore.IncidentRetentionPlan(ctx, time.Now().UTC().AddDate(0, 0, -int(settings.IncidentRetentionDays)).Unix())
	if err != nil {
		return DataGrowthStatus{}, err
	}
	incidentPlan.PlanID = incidentRetentionPlanID(incidentPlan)
	result := DataGrowthStatus{Aggregate: aggregate, Settings: settings, StatsDatabase: statsUsage, DashboardDB: dashboardUsage, LogBytes: logDirectoryBytes(s.logFile), RetentionRuns: runs, RetentionPlan: plan, Analysis: analysis, IncidentPlan: incidentPlan}
	if s.chatAudit != nil {
		if status, statusErr := s.chatAudit.Status(ctx); statusErr == nil {
			result.ChatAudit = &status
		}
	}
	if s.geoIP != nil {
		if status, statusErr := s.geoIP.Settings(ctx); statusErr == nil {
			result.GeoIP = &status
		}
	}
	return result, nil
}

func (s *DataMaintenanceService) Settings(ctx context.Context) (store.DataMaintenanceSettings, error) {
	return s.dashboard.DataMaintenanceSettings(ctx)
}

func (s *DataMaintenanceService) UpdateSettings(ctx context.Context, value store.DataMaintenanceSettings) error {
	if _, ok := allowedAggregateIntervals[value.AggregateIntervalMinutes]; !ok {
		return fmt.Errorf("unsupported aggregate interval")
	}
	if value.DetailRetentionDays < 30 || value.DetailRetentionDays > 3650 || value.SessionRetentionDays < 30 || value.SessionRetentionDays > 3650 || value.ResultRetentionDays < 30 || value.ResultRetentionDays > 3650 || value.IncidentRetentionDays < 30 || value.IncidentRetentionDays > 3650 {
		return fmt.Errorf("retention days must be between 30 and 3650")
	}
	if err := s.dashboard.UpdateDataMaintenanceSettings(ctx, value); err != nil {
		return err
	}
	s.aggregates.Reschedule()
	return nil
}

func (s *DataMaintenanceService) ApplyIncidentRetention(ctx context.Context, planID string) (store.IncidentRetentionResult, error) {
	settings, err := s.dashboard.DataMaintenanceSettings(ctx)
	if err != nil {
		return store.IncidentRetentionResult{}, err
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -int(settings.IncidentRetentionDays)).Unix()
	analysisStore, ok := s.stats.(store.StatsAnalysisMaintenanceStore)
	if !ok {
		return store.IncidentRetentionResult{}, fmt.Errorf("analysis maintenance is unavailable")
	}
	plan, err := analysisStore.IncidentRetentionPlan(ctx, cutoff)
	if err != nil {
		return store.IncidentRetentionResult{}, err
	}
	plan.PlanID = incidentRetentionPlanID(plan)
	if planID == "" || planID != plan.PlanID {
		return store.IncidentRetentionResult{}, fmt.Errorf("incident cleanup preview has changed; refresh and confirm again")
	}
	if !plan.DeletionEnabled {
		return store.IncidentRetentionResult{}, fmt.Errorf("unknown incident version in deletion candidates")
	}
	maintenance, err := store.OpenStatsMaintenance(ctx, s.statsCfg)
	if err != nil {
		return store.IncidentRetentionResult{}, err
	}
	defer maintenance.Close()
	latest, err := maintenance.IncidentRetentionPlan(ctx, cutoff)
	if err != nil {
		return store.IncidentRetentionResult{}, err
	}
	latest.PlanID = incidentRetentionPlanID(latest)
	if latest.PlanID != plan.PlanID {
		return store.IncidentRetentionResult{}, fmt.Errorf("incident candidates changed; refresh cleanup preview")
	}
	result, err := maintenance.ApplyIncidentRetention(ctx, latest)
	if err != nil {
		return store.IncidentRetentionResult{}, err
	}
	if err := s.dashboard.RecordIncidentRetentionRun(ctx, latest, result); err != nil {
		return store.IncidentRetentionResult{}, fmt.Errorf("record incident cleanup audit: %w", err)
	}
	if s.logger != nil {
		s.logger.Info("incident analysis data cleaned", zap.String("retention_run_id", result.RunID), zap.Int64("incident_rows", result.IncidentRows))
	}
	return result, nil
}

func (s *DataMaintenanceService) AggregateNow(ctx context.Context) error {
	return s.aggregates.Sync(ctx)
}

func (s *DataMaintenanceService) RetentionPlan(ctx context.Context) (store.RetentionPlan, error) {
	settings, err := s.dashboard.DataMaintenanceSettings(ctx)
	if err != nil {
		return store.RetentionPlan{}, err
	}
	status, err := s.dashboard.AggregateStatus(ctx)
	if err != nil {
		return store.RetentionPlan{}, err
	}
	return s.plan(ctx, settings, status)
}

func (s *DataMaintenanceService) plan(ctx context.Context, settings store.DataMaintenanceSettings, status store.AggregateStatus) (store.RetentionPlan, error) {
	now := time.Now().UTC()
	plan, err := s.stats.RetentionPlan(ctx,
		now.AddDate(0, 0, -int(settings.DetailRetentionDays)).Unix(),
		now.AddDate(0, 0, -int(settings.SessionRetentionDays)).Unix(),
		now.AddDate(0, 0, -int(settings.ResultRetentionDays)).Unix(),
	)
	if err != nil {
		return store.RetentionPlan{}, err
	}
	plan.AggregateCoverageReady = status.State == "ready" &&
		status.AggregateVersion == plan.AggregateVersion &&
		status.SourceWatermark >= plan.SourceWatermark
	plan.DeletionEnabled = plan.AggregateCoverageReady
	plan.PlanID = retentionPlanID(plan)
	return plan, nil
}

func (s *DataMaintenanceService) ApplyRetention(ctx context.Context, planID string) (store.RetentionResult, error) {
	if err := s.aggregates.Sync(ctx); err != nil {
		return store.RetentionResult{}, fmt.Errorf("refresh aggregates before cleanup: %w", err)
	}
	plan, err := s.RetentionPlan(ctx)
	if err != nil {
		return store.RetentionResult{}, err
	}
	if planID == "" || planID != plan.PlanID {
		return store.RetentionResult{}, fmt.Errorf("cleanup preview has changed; refresh and confirm again")
	}
	if !plan.DeletionEnabled {
		return store.RetentionResult{}, fmt.Errorf("aggregate coverage is not ready")
	}
	maintenance, err := store.OpenStatsMaintenance(ctx, s.statsCfg)
	if err != nil {
		return store.RetentionResult{}, err
	}
	defer maintenance.Close()
	latest, err := maintenance.RetentionPlan(ctx, plan.DetailCutoff, plan.SessionCutoff, plan.ResultCutoff)
	if err != nil {
		return store.RetentionResult{}, err
	}
	if latest.AggregateVersion != plan.AggregateVersion {
		return store.RetentionResult{}, fmt.Errorf("aggregate contract version changed; cleanup is blocked")
	}
	latest.AggregateCoverageReady = plan.AggregateCoverageReady
	latest.DeletionEnabled = plan.DeletionEnabled
	latest.PlanID = retentionPlanID(latest)
	if latest.PlanID != plan.PlanID {
		return store.RetentionResult{}, fmt.Errorf("source data changed; refresh cleanup preview")
	}
	result, err := maintenance.ApplyRetention(ctx, latest)
	if err != nil {
		return store.RetentionResult{}, err
	}
	if err := s.dashboard.RecordRetentionRun(ctx, latest, result); err != nil {
		return store.RetentionResult{}, fmt.Errorf("record cleanup audit: %w", err)
	}
	if s.logger != nil {
		s.logger.Info("statistics raw data cleaned", zap.String("retention_run_id", result.RunID), zap.Int64("equipment_rows", result.EquipmentRows), zap.Int64("class_rows", result.VersusClassRows), zap.Int64("session_rows", result.SessionRows), zap.Int64("round_result_rows", result.VersusRoundResultRows), zap.Int64("run_result_rows", result.VersusRunResultRows))
	}
	return result, nil
}

func retentionPlanID(plan store.RetentionPlan) string {
	value := struct {
		Version, Detail, Session, Result, Equipment, Classes, Sessions, Rounds, Runs, Watermark int64
	}{plan.AggregateVersion, plan.DetailCutoff, plan.SessionCutoff, plan.ResultCutoff, plan.EquipmentRowsEligible, plan.VersusClassRowsEligible, plan.SessionRowsEligible, plan.VersusRoundResultsEligible, plan.VersusRunResultsEligible, plan.SourceWatermark}
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:16])
}

func incidentRetentionPlanID(plan store.IncidentRetentionPlan) string {
	value := struct{ Version, Cutoff, Rows, Unknown, Watermark int64 }{plan.IncidentVersion, plan.Cutoff, plan.IncidentRowsEligible, plan.UnknownVersionRows, plan.CandidateWatermark}
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:16])
}

func logDirectoryBytes(logFile string) int64 {
	directory := filepath.Dir(logFile)
	prefix := filepath.Base(logFile)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0
	}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if info, err := entry.Info(); err == nil {
			total += info.Size()
		}
	}
	return total
}
