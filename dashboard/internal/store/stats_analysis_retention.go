package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *statsStore) AnalysisStatus(ctx context.Context, retentionDays int64) (AnalysisStatus, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	status := AnalysisStatus{IncidentVersion: incidentContractVersion, RetentionDays: retentionDays}
	cutoff30 := time.Now().UTC().AddDate(0, 0, -30).Unix()
	statement := `SELECT COUNT(*),COALESCE(SUM(CASE WHEN occurred_at>=` + s.bind(1) + ` THEN 1 ELSE 0 END),0),COALESCE(MIN(occurred_at),0),COALESCE(MAX(occurred_at),0) FROM lps_incidents WHERE incident_version=1`
	if err := s.db.QueryRowContext(queryCtx, statement, cutoff30).Scan(&status.IncidentRows, &status.RowsLast30Days, &status.EarliestIncidentAt, &status.LatestIncidentAt); err != nil {
		return status, err
	}
	if err := s.db.QueryRowContext(queryCtx, `SELECT COALESCE(SUM(CASE WHEN incident_capture_enabled=1 THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN incident_capture_enabled=1 AND incident_capture_complete=1 AND incident_dropped_count=0 THEN 1 ELSE 0 END),0) FROM lps_round_contexts WHERE context_version=1`).Scan(&status.CaptureEnabledRounds, &status.CompleteRounds); err != nil {
		return status, err
	}
	if status.CaptureEnabledRounds > 0 {
		status.CompleteRatio = float64(status.CompleteRounds) / float64(status.CaptureEnabledRounds)
	}
	status.ProjectedRowsForRetention = status.RowsLast30Days * retentionDays / 30
	return status, nil
}

func (s *statsStore) IncidentRetentionPlan(ctx context.Context, cutoff int64) (IncidentRetentionPlan, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	plan := IncidentRetentionPlan{IncidentVersion: incidentContractVersion, GeneratedAt: time.Now().UTC().Unix(), Cutoff: cutoff}
	statement := `SELECT COALESCE(SUM(CASE WHEN i.incident_version=1 THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN i.incident_version<>1 THEN 1 ELSE 0 END),0),COALESCE(MAX(i.occurred_at),0)
FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id
WHERE i.occurred_at<` + s.bind(1) + ` AND r.status<>'active'`
	if err := s.db.QueryRowContext(queryCtx, statement, cutoff).Scan(&plan.IncidentRowsEligible, &plan.UnknownVersionRows, &plan.CandidateWatermark); err != nil {
		return plan, err
	}
	plan.DeletionEnabled = plan.UnknownVersionRows == 0
	return plan, nil
}

func (s *statsStore) ApplyIncidentRetention(ctx context.Context, plan IncidentRetentionPlan) (IncidentRetentionResult, error) {
	if plan.IncidentVersion != incidentContractVersion || !plan.DeletionEnabled {
		return IncidentRetentionResult{}, fmt.Errorf("incident retention contract is not ready")
	}
	queryCtx, cancel := context.WithTimeout(ctx, maxDuration(s.timeout, 2*time.Minute))
	defer cancel()
	deleted, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_incidents", columns: []string{"round_id", "incident_seq"},
		selectSQL: `SELECT i.round_id,i.incident_seq FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id WHERE i.incident_version=1 AND i.occurred_at < %s AND r.status<>'active' ORDER BY i.occurred_at,i.round_id,i.incident_seq LIMIT 500`,
	}, plan.Cutoff)
	if err != nil {
		return IncidentRetentionResult{}, err
	}
	return IncidentRetentionResult{RunID: uuid.NewString(), ExecutedAt: time.Now().UTC().Unix(), IncidentRows: deleted}, nil
}
