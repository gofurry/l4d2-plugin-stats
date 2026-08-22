package service

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

type chatStatsFixture struct {
	messages []store.ChatMessage
}

func (f *chatStatsFixture) ListChatCaptureStates(context.Context) ([]store.ChatCaptureState, error) {
	return nil, nil
}

func (f *chatStatsFixture) ListChatOutbox(_ context.Context, bootID string, after int64, limit int) ([]store.ChatMessage, error) {
	result := make([]store.ChatMessage, 0, limit)
	for _, message := range f.messages {
		if message.BootID == bootID && message.ChatSeq > after {
			result = append(result, message)
			if len(result) == limit {
				break
			}
		}
	}
	return result, nil
}

func (f *chatStatsFixture) OldestChatOutboxSeq(_ context.Context, bootID string) (int64, error) {
	for _, message := range f.messages {
		if message.BootID == bootID {
			return message.ChatSeq, nil
		}
	}
	return 0, nil
}

func (f *chatStatsFixture) ConnectionAudit(context.Context, store.ConnectionAuditFilter) (store.ConnectionAuditPage, error) {
	return store.ConnectionAuditPage{}, nil
}

func TestChatAuditIngestionDetectsInternalAndTrailingSequenceGaps(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name     string
		state    store.ChatCaptureState
		messages []store.ChatMessage
		wantGap  int64
		wantRows int
	}{
		{
			name:     "internal gaps",
			state:    store.ChatCaptureState{BootID: "boot", ServerKey: "server", ObservedCount: 5, PersistedCount: 3, DroppedCount: 2, LastChatSeq: 5},
			messages: []store.ChatMessage{chatFixtureMessage(1), chatFixtureMessage(3), chatFixtureMessage(5)},
			wantGap:  2, wantRows: 3,
		},
		{
			name:     "trailing dropped suffix",
			state:    store.ChatCaptureState{BootID: "boot", ServerKey: "server", ObservedCount: 3, PersistedCount: 1, DroppedCount: 2, LastChatSeq: 3},
			messages: []store.ChatMessage{chatFixtureMessage(1)},
			wantGap:  2, wantRows: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			audit, err := store.OpenChatAudit(ctx, filepath.Join(t.TempDir(), "chat-audit.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer audit.Close()
			stats := &chatStatsFixture{messages: test.messages}
			service := NewChatAuditService(nil, stats, audit, zap.NewNop())
			if err := service.ingestBoot(ctx, test.state); err != nil {
				t.Fatal(err)
			}
			cursor, err := audit.Cursor(ctx, "boot")
			if err != nil || cursor.LastChatSeq != test.state.LastChatSeq || cursor.GapCount != test.wantGap {
				t.Fatalf("cursor=%+v err=%v", cursor, err)
			}
			page, err := audit.Search(ctx, store.ChatSearchFilter{Limit: 100})
			if err != nil || len(page.Items) != test.wantRows {
				t.Fatalf("page rows=%d err=%v", len(page.Items), err)
			}
		})
	}
}

func chatFixtureMessage(sequence int64) store.ChatMessage {
	return store.ChatMessage{
		MessageID: "boot:chat:" + strconv.FormatInt(sequence, 10), BootID: "boot", ServerKey: "server", ChatSeq: sequence,
		SourceUserID: 1, PlayerName: "player", OccurredAt: sequence, MapName: "map", GameMode: "coop",
		Team: "survivor", Channel: "global", Content: "message",
	}
}

type retentionDashboardFixture struct {
	store.DashboardAuditStore
	settings  store.ChatAuditSettings
	updateErr error
	markCalls int
	events    *[]string
}

func (f *retentionDashboardFixture) ChatAuditSettings(context.Context) (store.ChatAuditSettings, error) {
	return f.settings, nil
}

func (f *retentionDashboardFixture) UpdateChatAuditSettings(_ context.Context, settings store.ChatAuditSettings) error {
	*f.events = append(*f.events, "update-settings")
	if f.updateErr != nil {
		return f.updateErr
	}
	f.settings = settings
	return nil
}

func (f *retentionDashboardFixture) MarkChatAuditCleanup(context.Context, int64) error {
	*f.events = append(*f.events, "mark-cleanup")
	f.markCalls++
	return nil
}

type retentionApplyResult struct {
	deleted int64
	err     error
}

type retentionAuditFixture struct {
	store.ChatAuditStore
	plan         store.ChatRetentionPlan
	applyResults []retentionApplyResult
	applyCalls   int
	events       *[]string
}

func (f *retentionAuditFixture) RetentionPlan(_ context.Context, retentionDays, _ int64) (store.ChatRetentionPlan, error) {
	plan := f.plan
	plan.RetentionDays = retentionDays
	return plan, nil
}

func (f *retentionAuditFixture) ApplyRetention(context.Context, int64, int64) (int64, error) {
	*f.events = append(*f.events, "apply-retention")
	index := f.applyCalls
	f.applyCalls++
	if index >= len(f.applyResults) {
		index = len(f.applyResults) - 1
	}
	result := f.applyResults[index]
	return result.deleted, result.err
}

func TestChatRetentionConfirmationPersistsSettingsBeforeCleanup(t *testing.T) {
	ctx := context.Background()
	newSettings := store.ChatAuditSettings{Enabled: true, RetentionDays: 7}

	t.Run("settings failure prevents deletion", func(t *testing.T) {
		events := []string{}
		dashboard := &retentionDashboardFixture{
			settings:  store.ChatAuditSettings{Enabled: true, RetentionDays: 30},
			updateErr: errors.New("injected settings failure"), events: &events,
		}
		audit := &retentionAuditFixture{
			plan:         store.ChatRetentionPlan{Cutoff: 1, DeleteCount: 10},
			applyResults: []retentionApplyResult{{deleted: 10}}, events: &events,
		}
		service := NewChatAuditService(dashboard, nil, audit, zap.NewNop())
		plan, err := service.UpdateSettings(ctx, newSettings)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.ConfirmSettings(ctx, plan.PlanID, newSettings); err == nil {
			t.Fatal("settings failure was not returned")
		}
		if audit.applyCalls != 0 || dashboard.settings.RetentionDays != 30 || strings.Join(events, ",") != "update-settings" {
			t.Fatalf("apply_calls=%d settings=%+v events=%v", audit.applyCalls, dashboard.settings, events)
		}
	})

	t.Run("successful settings and cleanup complete in order", func(t *testing.T) {
		events := []string{}
		dashboard := &retentionDashboardFixture{settings: store.ChatAuditSettings{Enabled: true, RetentionDays: 30}, events: &events}
		audit := &retentionAuditFixture{
			plan:         store.ChatRetentionPlan{Cutoff: 1, DeleteCount: 10},
			applyResults: []retentionApplyResult{{deleted: 10}}, events: &events,
		}
		service := NewChatAuditService(dashboard, nil, audit, zap.NewNop())
		plan, err := service.UpdateSettings(ctx, newSettings)
		if err != nil {
			t.Fatal(err)
		}
		result, err := service.ConfirmSettings(ctx, plan.PlanID, newSettings)
		if err != nil || result.Deleted != 10 || result.CleanupStatus != "complete" || dashboard.settings.RetentionDays != 7 {
			t.Fatalf("result=%+v settings=%+v err=%v", result, dashboard.settings, err)
		}
		if strings.Join(events, ",") != "update-settings,apply-retention,mark-cleanup" {
			t.Fatalf("events=%v", events)
		}
	})

	t.Run("cleanup failure keeps policy and hourly cleanup resumes", func(t *testing.T) {
		events := []string{}
		dashboard := &retentionDashboardFixture{settings: store.ChatAuditSettings{Enabled: true, RetentionDays: 30}, events: &events}
		audit := &retentionAuditFixture{
			plan: store.ChatRetentionPlan{Cutoff: 1, DeleteCount: 1000},
			applyResults: []retentionApplyResult{
				{deleted: 500, err: errors.New("injected cleanup failure")},
				{deleted: 500},
			},
			events: &events,
		}
		service := NewChatAuditService(dashboard, nil, audit, zap.NewNop())
		disabledSettings := store.ChatAuditSettings{Enabled: false, RetentionDays: 7}
		plan, err := service.UpdateSettings(ctx, disabledSettings)
		if err != nil {
			t.Fatal(err)
		}
		result, err := service.ConfirmSettings(ctx, plan.PlanID, disabledSettings)
		if err != nil || result.Deleted != 500 || result.CleanupStatus != "pending" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		if dashboard.settings.RetentionDays != 7 || dashboard.settings.Enabled || dashboard.markCalls != 0 {
			t.Fatalf("saved settings=%+v mark_calls=%d", dashboard.settings, dashboard.markCalls)
		}
		service.cleanupWithTimeout(ctx)
		if audit.applyCalls != 2 || dashboard.markCalls != 1 {
			t.Fatalf("apply_calls=%d mark_calls=%d events=%v", audit.applyCalls, dashboard.markCalls, events)
		}
	})
}
