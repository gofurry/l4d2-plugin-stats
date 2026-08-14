package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

type achievementStatsStub struct {
	metrics        map[string]store.PlayerAchievementMetrics
	metricFailures map[string]int
	metricCalls    map[string]int
	backfill       []store.AchievementSourcePlayer
	dirty          []store.AchievementSourcePlayer
	eligible       int64
}

func (s *achievementStatsStub) PlayerAchievementMetrics(_ context.Context, steamID string) (store.PlayerAchievementMetrics, error) {
	s.metricCalls[steamID]++
	if s.metricFailures[steamID] > 0 {
		s.metricFailures[steamID]--
		return store.PlayerAchievementMetrics{}, errors.New("temporary metric failure")
	}
	return s.metrics[steamID], nil
}

func (s *achievementStatsStub) PlayerAchievementWatermark(_ context.Context, steamID string) (int64, error) {
	return s.metrics[steamID].Watermark, nil
}

func (s *achievementStatsStub) AchievementDirtyPlayers(_ context.Context, afterWatermark int64, afterSteamID string, limit int32) ([]store.AchievementSourcePlayer, error) {
	return filterAchievementPlayers(s.dirty, func(player store.AchievementSourcePlayer) bool {
		return player.Watermark > afterWatermark || player.Watermark == afterWatermark && player.SteamID > afterSteamID
	}, limit), nil
}

func (s *achievementStatsStub) AchievementBackfillPlayers(_ context.Context, afterSteamID string, limit int32) ([]store.AchievementSourcePlayer, error) {
	return filterAchievementPlayers(s.backfill, func(player store.AchievementSourcePlayer) bool {
		return player.SteamID > afterSteamID
	}, limit), nil
}

func (s *achievementStatsStub) AchievementEligiblePlayerCount(context.Context) (int64, error) {
	return s.eligible, nil
}

func filterAchievementPlayers(players []store.AchievementSourcePlayer, keep func(store.AchievementSourcePlayer) bool, limit int32) []store.AchievementSourcePlayer {
	result := make([]store.AchievementSourcePlayer, 0, limit)
	for _, player := range players {
		if keep(player) {
			result = append(result, player)
			if len(result) == int(limit) {
				break
			}
		}
	}
	return result
}

func newAchievementTestService(t *testing.T, stats *achievementStatsStub) (*AchievementService, store.DashboardDatabase) {
	t.Helper()
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dashboard.Close() })
	if err := dashboard.ApplyAggregateChanges(ctx, store.AggregateChangeSet{Full: true, SourceWatermark: 1000}); err != nil {
		t.Fatal(err)
	}
	return NewAchievementService(dashboard, stats, zap.NewNop()), dashboard
}

func TestAchievementVisibilityCompletionAndShowcase(t *testing.T) {
	ctx := context.Background()
	steamID := "76561198000000001"
	stats := &achievementStatsStub{
		metrics: map[string]store.PlayerAchievementMetrics{steamID: {
			SteamID: steamID, Watermark: 1, Values: map[string]store.AchievementMetricValue{
				"career.active_play_seconds": {Available: true, Value: 10*3600 - 1},
				"tank_rocks_destroyed":       {Available: true, Value: 0},
				"survivor_fall_deaths":       {Available: true, Value: 0},
			},
		}},
		metricFailures: map[string]int{}, metricCalls: map[string]int{}, eligible: 1,
	}
	service, _ := newAchievementTestService(t, stats)
	result, err := service.Player(ctx, steamID, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Overview.Unlocked != 0 || result.Overview.Total != 58 || result.Overview.EasterEggs != 0 {
		t.Fatalf("locked overview=%#v", result.Overview)
	}
	mysteryKeys := make(map[string]bool)
	for _, card := range result.Items {
		if strings.HasPrefix(card.AchievementKey, "secret.") {
			t.Fatalf("locked secret leaked: %#v", card)
		}
		if card.Visibility == "mystery" {
			if card.Title != "???" || card.Description != "条件尚未发现" || card.Threshold != 0 || card.MetricID != "" || card.ArtworkKey != "" {
				t.Fatalf("mystery leaked frozen data: %#v", card)
			}
			mysteryKeys[card.GroupKey] = true
		}
	}
	if len(mysteryKeys) != 4 {
		t.Fatalf("mystery placeholders=%v", mysteryKeys)
	}

	stats.metrics[steamID] = store.PlayerAchievementMetrics{SteamID: steamID, Watermark: 2, Values: map[string]store.AchievementMetricValue{
		"career.active_play_seconds": {Available: true, Value: 10 * 3600},
		"tank_rocks_destroyed":       {Available: true, Value: 5},
		"survivor_fall_deaths":       {Available: true, Value: 5},
	}}
	result, err = service.Player(ctx, steamID, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Overview.Unlocked != 2 || result.Overview.EasterEggs != 1 || result.Overview.Total != 58 {
		t.Fatalf("unlocked overview=%#v", result.Overview)
	}
	for _, badge := range result.Overview.Badges {
		if strings.HasPrefix(badge.AchievementKey, "secret.") {
			t.Fatalf("secret entered automatic showcase: %#v", badge)
		}
	}
	badges, err := service.SetBadges(ctx, steamID, []store.BadgeShowcaseSlot{{Slot: 1, AchievementKey: "secret.crashed"}})
	if err != nil || len(badges.Items) != 1 || badges.Items[0].AchievementKey != "secret.crashed" {
		t.Fatalf("manual secret showcase=%#v err=%v", badges, err)
	}
}

func TestAchievementBackfillResumeRestartAndSourceWatermark(t *testing.T) {
	ctx := context.Background()
	first, second, live := "76561198000000001", "76561198000000002", "76561198000000003"
	metrics := func(steamID string, watermark, seconds int64) store.PlayerAchievementMetrics {
		return store.PlayerAchievementMetrics{SteamID: steamID, Watermark: watermark, Values: map[string]store.AchievementMetricValue{
			"career.active_play_seconds": {Available: true, Value: seconds},
		}}
	}
	stats := &achievementStatsStub{
		metrics: map[string]store.PlayerAchievementMetrics{
			first: metrics(first, 10, 10*3600), second: metrics(second, 20, 10*3600), live: metrics(live, 30, 10*3600),
		},
		metricFailures: map[string]int{second: 1}, metricCalls: map[string]int{}, eligible: 3,
		backfill: []store.AchievementSourcePlayer{{SteamID: first, Watermark: 10}, {SteamID: second, Watermark: 20}},
	}
	service, dashboard := newAchievementTestService(t, stats)
	if err := service.RunOnce(ctx); err == nil {
		t.Fatal("backfill failure was not returned")
	}
	state, err := dashboard.AchievementEngineState(ctx)
	if err != nil || state.BackfillCursor != first || state.BackfillComplete {
		t.Fatalf("resumable state=%#v err=%v", state, err)
	}
	restarted := NewAchievementService(dashboard, stats, zap.NewNop())
	if err := restarted.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	state, err = dashboard.AchievementEngineState(ctx)
	if err != nil || !state.BackfillComplete || state.BackfillCursor != second || stats.metricCalls[first] != 1 || stats.metricCalls[second] != 2 {
		t.Fatalf("resumed state=%#v calls=%v err=%v", state, stats.metricCalls, err)
	}
	for _, steamID := range []string{first, second} {
		unlocks, err := dashboard.ListAchievementUnlocks(ctx, steamID)
		if err != nil || len(unlocks) != 1 || unlocks[0].GrantKind != "backfill" {
			t.Fatalf("backfill unlock %s=%#v err=%v", steamID, unlocks, err)
		}
	}

	if _, err := restarted.EnsurePlayer(ctx, live); err != nil {
		t.Fatal(err)
	}
	stats.metrics[live] = metrics(live, 30, 50*3600)
	if _, err := restarted.EnsurePlayer(ctx, live); err != nil {
		t.Fatal(err)
	}
	unlocks, err := dashboard.ListAchievementUnlocks(ctx, live)
	if err != nil || len(unlocks) != 1 {
		t.Fatalf("unchanged watermark unlocks=%#v err=%v", unlocks, err)
	}
	stats.metrics[live] = metrics(live, 31, 50*3600)
	inserted, err := restarted.EnsurePlayer(ctx, live)
	if err != nil || len(inserted) != 1 || inserted[0].AchievementKey != "career.veteran.2" || inserted[0].GrantKind != "live" {
		t.Fatalf("changed watermark insert=%#v err=%v", inserted, err)
	}
}
