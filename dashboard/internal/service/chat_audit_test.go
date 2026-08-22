package service

import (
	"context"
	"path/filepath"
	"strconv"
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
