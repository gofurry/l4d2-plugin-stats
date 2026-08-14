package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDashboardAchievementPersistence(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()

	engine, err := dashboard.AchievementEngineState(ctx)
	if err != nil || engine.AchievementContractVersion != AchievementContractVersion || engine.BackfillComplete {
		t.Fatalf("engine=%#v err=%v", engine, err)
	}
	engine.BackfillCursor, engine.LastSuccessAt, engine.UpdatedAt = "765", 100, 100
	if err := dashboard.UpdateAchievementEngineState(ctx, engine); err != nil {
		t.Fatal(err)
	}

	unlock := AchievementUnlock{SteamID: "765", AchievementKey: "career.veteran.1", AchievementContractVersion: 1, UnlockedAt: 100, GrantKind: "backfill", ValueAtUnlock: 36000}
	inserted, err := dashboard.InsertAchievementUnlocks(ctx, []AchievementUnlock{unlock})
	if err != nil || len(inserted) != 1 {
		t.Fatalf("inserted=%#v err=%v", inserted, err)
	}
	inserted, err = dashboard.InsertAchievementUnlocks(ctx, []AchievementUnlock{unlock})
	if err != nil || len(inserted) != 0 {
		t.Fatalf("idempotent inserted=%#v err=%v", inserted, err)
	}
	if err := dashboard.UpsertAchievementEvaluationState(ctx, AchievementEvaluationState{SteamID: "765", AchievementContractVersion: 1, SourceWatermark: 55, EvaluatedAt: 101}); err != nil {
		t.Fatal(err)
	}
	evaluation, err := dashboard.AchievementEvaluationState(ctx, "765")
	if err != nil || evaluation.SourceWatermark != 55 {
		t.Fatalf("evaluation=%#v err=%v", evaluation, err)
	}

	if err := dashboard.ReplaceBadgeShowcase(ctx, "765", []BadgeShowcaseSlot{{Slot: 1, AchievementKey: unlock.AchievementKey}}, 102); err != nil {
		t.Fatal(err)
	}
	badges, err := dashboard.BadgeShowcase(ctx, "765")
	if err != nil || len(badges) != 1 || badges[0].AchievementKey != unlock.AchievementKey {
		t.Fatalf("badges=%#v err=%v", badges, err)
	}
	if err := dashboard.ReplaceBadgeShowcase(ctx, "765", []BadgeShowcaseSlot{{Slot: 1, AchievementKey: "secret.crashed"}}, 103); err == nil {
		t.Fatal("locked badge selection was accepted")
	}
	if err := dashboard.MarkAchievementUnlocksSeen(ctx, "765", 104); err != nil {
		t.Fatal(err)
	}
	unlocks, err := dashboard.ListAchievementUnlocks(ctx, "765")
	if err != nil || len(unlocks) != 1 || unlocks[0].SeenAt != 104 || unlocks[0].EvidenceSteamID != "" {
		t.Fatalf("unlocks=%#v err=%v", unlocks, err)
	}
}
