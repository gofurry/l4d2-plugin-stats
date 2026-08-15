package server

import (
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestPlayerPreviewBadgesIncludeEveryShowcaseSlotWithoutMutatingCache(t *testing.T) {
	legacy := store.PlayerPreviewBadge{Slot: 1, AchievementKey: "old", Title: "old"}
	source := &store.PlayerPreview{
		SteamID:    "76561198000000001",
		Companions: []store.PlayerCompanion{{PlayerName: "companion", SharedRounds: 3}},
		Badges:     []store.PlayerPreviewBadge{legacy},
		MainBadge:  &legacy,
	}
	result := clonePlayerPreview(source)
	attachPlayerPreviewBadges(result, []service.AchievementBadge{
		{Slot: 1, AchievementKey: "career.veteran.1", Title: "Veteran I", ArtworkKey: "career.veteran", Tier: 1},
		{Slot: 2, AchievementKey: "support.field_medic.2", Title: "Medic II", ArtworkKey: "support.field_medic", Tier: 2},
		{Slot: 3, AchievementKey: "secret.dispatch", Title: "Dispatch", ArtworkKey: "secret.dispatch"},
	})

	if len(result.Badges) != 3 {
		t.Fatalf("badges=%#v", result.Badges)
	}
	if result.MainBadge == nil || result.MainBadge.AchievementKey != result.Badges[0].AchievementKey {
		t.Fatalf("main badge=%#v first=%#v", result.MainBadge, result.Badges[0])
	}
	if len(source.Badges) != 1 || source.Badges[0].AchievementKey != "old" || source.MainBadge == nil || source.MainBadge.AchievementKey != "old" {
		t.Fatalf("cached source was mutated: %#v", source)
	}
	result.Companions[0].PlayerName = "changed"
	if source.Companions[0].PlayerName != "companion" {
		t.Fatalf("cached companions were mutated: %#v", source.Companions)
	}
}
