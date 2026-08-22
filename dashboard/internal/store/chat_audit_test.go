package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestChatAuditIngestionCursorSearchAndRetention(t *testing.T) {
	ctx := context.Background()
	audit, err := OpenChatAudit(ctx, filepath.Join(t.TempDir(), "chat-audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer audit.Close()
	if version, err := audit.SchemaVersion(ctx); err != nil || version != ChatAuditSchemaVersion {
		t.Fatalf("schema=%d err=%v", version, err)
	}
	now := time.Now().Unix()
	state := ChatCaptureState{BootID: "boot-1", ServerKey: "server-1", CaptureVersion: 1, LastChatSeq: 4}
	messages := []ChatMessage{
		{MessageID: "boot-1:chat:2", BootID: "boot-1", ServerKey: "server-1", ChatSeq: 2, SteamID: "76561198000000001", SourceUserID: 3, PlayerName: "Alice", OccurredAt: now - 100, MapName: "c1m1_hotel", GameMode: "coop", Team: "survivor", Channel: "global", Alive: true, Content: "hello"},
		{MessageID: "boot-1:chat:3", BootID: "boot-1", ServerKey: "server-1", ChatSeq: 3, SteamID: "76561198000000002", SourceUserID: 4, PlayerName: "Bob", OccurredAt: now - 50, MapName: "c1m1_hotel", GameMode: "coop", Team: "survivor", Channel: "team", CommandLike: true, Content: "!help"},
	}
	if err := audit.Ingest(ctx, state, messages, 1, 1); err != nil {
		t.Fatal(err)
	}
	// Retry is idempotent and must not duplicate messages.
	if err := audit.Ingest(ctx, state, messages, 0, 0); err != nil {
		t.Fatal(err)
	}
	cursor, err := audit.Cursor(ctx, state.BootID)
	if err != nil || cursor.LastChatSeq != 3 || cursor.GapCount != 1 || cursor.LastGapFrom != 1 || cursor.LastGapTo != 1 {
		t.Fatalf("cursor=%+v err=%v", cursor, err)
	}
	page, err := audit.Search(ctx, ChatSearchFilter{Channel: "team", MessageKind: "command", Keyword: "help", Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].PlayerName != "Bob" {
		t.Fatalf("search=%+v err=%v", page, err)
	}
	all, err := audit.Search(ctx, ChatSearchFilter{Limit: 1})
	if err != nil || len(all.Items) != 1 || all.NextCursorID == "" {
		t.Fatalf("first page=%+v err=%v", all, err)
	}
	next, err := audit.Search(ctx, ChatSearchFilter{Limit: 1, CursorAt: all.NextCursorAt, CursorID: all.NextCursorID})
	if err != nil || len(next.Items) != 1 || next.Items[0].MessageID == all.Items[0].MessageID {
		t.Fatalf("next page=%+v err=%v", next, err)
	}
	plan, err := audit.RetentionPlan(ctx, 0, now)
	if err != nil || plan.DeleteCount != 0 {
		t.Fatalf("permanent plan=%+v err=%v", plan, err)
	}
	deleted, err := audit.ApplyRetention(ctx, 1, now+2*86400)
	if err != nil || deleted != 2 {
		t.Fatalf("deleted=%d err=%v", deleted, err)
	}
}

func TestChatAuditFailedTransactionDoesNotAdvanceCursor(t *testing.T) {
	ctx := context.Background()
	audit, err := OpenChatAudit(ctx, filepath.Join(t.TempDir(), "chat-audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer audit.Close()
	state := ChatCaptureState{BootID: "boot-bad", ServerKey: "server", CaptureVersion: 1}
	bad := ChatMessage{MessageID: "boot-bad:chat:1", BootID: "boot-bad", ServerKey: "server", ChatSeq: 1, SourceUserID: 1, PlayerName: "Bad", OccurredAt: time.Now().Unix(), Team: "survivor", Channel: "invalid", Content: "body"}
	if err := audit.Ingest(ctx, state, []ChatMessage{bad}, 0, 0); err == nil {
		t.Fatal("invalid channel unexpectedly succeeded")
	}
	cursor, err := audit.Cursor(ctx, state.BootID)
	if err != nil || cursor.LastChatSeq != 0 {
		t.Fatalf("cursor advanced after rollback: %+v err=%v", cursor, err)
	}
}
