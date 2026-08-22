package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ChatAuditService struct {
	dashboard store.DashboardAuditStore
	stats     store.StatsChatAuditStore
	audit     store.ChatAuditStore
	logger    *zap.Logger

	planMu sync.Mutex
	plans  map[string]store.ChatRetentionPlan
}

func NewChatAuditService(dashboard store.DashboardAuditStore, stats store.StatsChatAuditStore, audit store.ChatAuditStore, logger *zap.Logger) *ChatAuditService {
	return &ChatAuditService{dashboard: dashboard, stats: stats, audit: audit, logger: logger, plans: make(map[string]store.ChatRetentionPlan)}
}

func (s *ChatAuditService) Run(ctx context.Context) {
	ingest := time.NewTicker(5 * time.Second)
	cleanup := time.NewTicker(time.Hour)
	defer ingest.Stop()
	defer cleanup.Stop()
	s.ingestWithTimeout(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ingest.C:
			s.ingestWithTimeout(ctx)
		case <-cleanup.C:
			s.cleanupWithTimeout(ctx)
		}
	}
}

func (s *ChatAuditService) ingestWithTimeout(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	if err := s.IngestOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		s.logger.Warn("chat audit ingestion unavailable", zap.String("error_code", "chat_ingest_failed"))
	}
}

func (s *ChatAuditService) cleanupWithTimeout(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	settings, err := s.dashboard.ChatAuditSettings(ctx)
	if err != nil || settings.RetentionDays == 0 {
		return
	}
	if _, err := s.audit.ApplyRetention(ctx, settings.RetentionDays, time.Now().Unix()); err != nil {
		s.logger.Warn("chat audit cleanup unavailable", zap.String("error_code", "chat_cleanup_failed"))
		return
	}
	_ = s.dashboard.MarkChatAuditCleanup(ctx, time.Now().Unix())
}

func (s *ChatAuditService) IngestOnce(ctx context.Context) error {
	settings, err := s.dashboard.ChatAuditSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}
	states, err := s.stats.ListChatCaptureStates(ctx)
	if err != nil {
		return fmt.Errorf("list chat capture states: %w", err)
	}
	var firstErr error
	for _, state := range states {
		if err := s.ingestBoot(ctx, state); err != nil && firstErr == nil {
			// Continue other boot sources: one corrupt/unavailable source must not
			// starve otherwise healthy chat transport streams.
			firstErr = fmt.Errorf("ingest chat boot: %w", err)
		}
	}
	return firstErr
}

func (s *ChatAuditService) ingestBoot(ctx context.Context, state store.ChatCaptureState) error {
	cursor, err := s.audit.Cursor(ctx, state.BootID)
	if err != nil {
		return err
	}
	oldest, err := s.stats.OldestChatOutboxSeq(ctx, state.BootID)
	if err != nil {
		return err
	}
	if oldest > 0 && cursor.LastChatSeq+1 < oldest {
		if err := s.audit.Ingest(ctx, state, nil, cursor.LastChatSeq+1, oldest-1); err != nil {
			return err
		}
		cursor.LastChatSeq = oldest - 1
	}
	// Bound each source per pass so one busy server cannot starve other boots.
	for batch := 0; batch < 8; batch++ {
		messages, err := s.stats.ListChatOutbox(ctx, state.BootID, cursor.LastChatSeq, 128)
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			// When the last durable capture state accounts for every observed
			// sequence, an empty outbox proves any remaining suffix was dropped
			// or expired. Do not make this claim while rows may still be queued.
			settled := state.PersistedCount+state.DroppedCount >= state.ObservedCount
			if settled && state.LastChatSeq > cursor.LastChatSeq {
				return s.audit.Ingest(ctx, state, nil, cursor.LastChatSeq+1, state.LastChatSeq)
			}
			return nil
		}
		if err := s.ingestSequenceBatch(ctx, state, &cursor, messages); err != nil {
			return err
		}
		if len(messages) < 128 {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

func (s *ChatAuditService) ingestSequenceBatch(ctx context.Context, state store.ChatCaptureState, cursor *store.ChatIngestCursor, messages []store.ChatMessage) error {
	start := 0
	for index, message := range messages {
		expected := cursor.LastChatSeq + int64(index-start) + 1
		if message.ChatSeq <= expected {
			continue
		}
		if index > start {
			chunk := messages[start:index]
			if err := s.audit.Ingest(ctx, state, chunk, 0, 0); err != nil {
				return err
			}
			cursor.LastChatSeq = chunk[len(chunk)-1].ChatSeq
		}
		if err := s.audit.Ingest(ctx, state, nil, cursor.LastChatSeq+1, message.ChatSeq-1); err != nil {
			return err
		}
		cursor.LastChatSeq = message.ChatSeq - 1
		start = index
	}
	if start < len(messages) {
		chunk := messages[start:]
		if err := s.audit.Ingest(ctx, state, chunk, 0, 0); err != nil {
			return err
		}
		cursor.LastChatSeq = chunk[len(chunk)-1].ChatSeq
	}
	return nil
}

func (s *ChatAuditService) Search(ctx context.Context, filter store.ChatSearchFilter) (store.ChatSearchPage, error) {
	return s.audit.Search(ctx, filter)
}

func (s *ChatAuditService) Settings(ctx context.Context) (store.ChatAuditSettings, error) {
	return s.dashboard.ChatAuditSettings(ctx)
}

func (s *ChatAuditService) UpdateSettings(ctx context.Context, settings store.ChatAuditSettings) (store.ChatRetentionPlan, error) {
	current, err := s.dashboard.ChatAuditSettings(ctx)
	if err != nil {
		return store.ChatRetentionPlan{}, err
	}
	if settings.RetentionDays != 0 && settings.RetentionDays != 7 && settings.RetentionDays != 14 && settings.RetentionDays != 30 && settings.RetentionDays != 60 && settings.RetentionDays != 90 && settings.RetentionDays != 180 && settings.RetentionDays != 365 {
		return store.ChatRetentionPlan{}, errors.New("unsupported chat retention")
	}
	if retentionShortened(current.RetentionDays, settings.RetentionDays) {
		plan, err := s.audit.RetentionPlan(ctx, settings.RetentionDays, time.Now().Unix())
		if err != nil {
			return store.ChatRetentionPlan{}, err
		}
		if plan.DeleteCount > 0 {
			plan.PlanID = uuid.NewString()
			s.planMu.Lock()
			if len(s.plans) >= 128 {
				// Preview state is intentionally in-memory and short-lived. Bound it
				// rather than allowing abandoned confirmations to grow forever.
				s.plans = make(map[string]store.ChatRetentionPlan)
			}
			s.plans[plan.PlanID] = plan
			s.planMu.Unlock()
			return plan, nil
		}
	}
	return store.ChatRetentionPlan{}, s.dashboard.UpdateChatAuditSettings(ctx, settings)
}

func retentionShortened(oldDays, newDays int64) bool {
	return newDays != 0 && (oldDays == 0 || newDays < oldDays)
}

func (s *ChatAuditService) ConfirmSettings(ctx context.Context, planID string, settings store.ChatAuditSettings) (store.ChatRetentionConfirmation, error) {
	s.planMu.Lock()
	plan, ok := s.plans[planID]
	delete(s.plans, planID)
	s.planMu.Unlock()
	if !ok || plan.RetentionDays != settings.RetentionDays || plan.Cutoff <= 0 {
		return store.ChatRetentionConfirmation{}, errors.New("chat retention preview is missing or stale")
	}
	if err := s.dashboard.UpdateChatAuditSettings(ctx, settings); err != nil {
		return store.ChatRetentionConfirmation{}, err
	}
	result := store.ChatRetentionConfirmation{Settings: settings, CleanupStatus: "pending"}
	deleted, err := s.audit.ApplyRetention(ctx, settings.RetentionDays, time.Now().Unix())
	result.Deleted = deleted
	if err != nil {
		s.logger.Warn("chat retention policy saved with cleanup pending", zap.String("error_code", "chat_cleanup_incomplete"))
		return result, nil
	}
	_ = s.dashboard.MarkChatAuditCleanup(ctx, time.Now().Unix())
	result.CleanupStatus = "complete"
	return result, nil
}

func (s *ChatAuditService) Status(ctx context.Context) (store.ChatAuditStatus, error) {
	settings, err := s.dashboard.ChatAuditSettings(ctx)
	if err != nil {
		return store.ChatAuditStatus{}, err
	}
	states, err := s.stats.ListChatCaptureStates(ctx)
	if err != nil {
		return store.ChatAuditStatus{}, err
	}
	var lag, dropped int64
	for _, state := range states {
		cursor, cursorErr := s.audit.Cursor(ctx, state.BootID)
		if cursorErr != nil {
			return store.ChatAuditStatus{}, cursorErr
		}
		if state.LastChatSeq > cursor.LastChatSeq {
			lag += state.LastChatSeq - cursor.LastChatSeq
		}
		dropped += state.DroppedCount
	}
	return s.audit.Status(ctx, settings, lag, dropped)
}

func (s *ChatAuditService) RecordExport(ctx context.Context, admin, format string, filter store.ChatSearchFilter, rowCount *int64, completed bool) error {
	filter.Keyword = ""
	summary, err := json.Marshal(filter)
	if err != nil {
		return err
	}
	return s.dashboard.RecordChatExport(ctx, store.ChatExportAuditEntry{
		ExportID: uuid.NewString(), ExportedAt: time.Now().Unix(), AdminIdentity: admin,
		OutputFormat: format, FilterSummary: string(summary), RowCount: rowCount, Completed: completed,
	})
}
