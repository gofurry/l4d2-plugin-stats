package service

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestAchievementCatalogV1Frozen(t *testing.T) {
	catalog := AchievementCatalog()
	if len(catalog) != 105 {
		t.Fatalf("catalog has %d achievements, want 105", len(catalog))
	}
	keys := make(map[string]bool, len(catalog))
	artwork := make(map[string]bool)
	completion, secrets := 0, 0
	for _, item := range catalog {
		if keys[item.AchievementKey] {
			t.Fatalf("duplicate achievement key %q", item.AchievementKey)
		}
		keys[item.AchievementKey] = true
		artwork[item.ArtworkKey] = true
		if item.CountsTowardCompletion {
			completion++
		}
		if item.Visibility == "secret" {
			secrets++
			if item.CountsTowardCompletion {
				t.Fatalf("secret %q counts toward completion", item.AchievementKey)
			}
		}
	}
	if completion != 100 || secrets != 5 || len(artwork) != 38 {
		t.Fatalf("completion=%d secrets=%d artwork=%d", completion, secrets, len(artwork))
	}
	assertThresholds := func(group string, want []int64) {
		t.Helper()
		got := make([]int64, 0)
		for _, item := range catalog {
			if item.GroupKey == group {
				got = append(got, item.Threshold)
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s thresholds=%v want %v", group, got, want)
		}
	}
	assertThresholds("career.veteran", []int64{36000, 180000, 720000, 1800000})
	assertThresholds("combat.marksman", []int64{5000, 25000, 100000, 250000})
	assertThresholds("bond.comrade", []int64{36000, 180000, 360000})
	assertThresholds("weapon.throwable_expert", []int64{50, 250, 1000, 2500})
	assertThresholds("career.objective_master", []int64{25, 100, 500})
	assertThresholds("career.temp_health_addict", []int64{100, 500, 2000, 5000})
	assertThresholds("support.firepower_upgrade", []int64{25, 100, 500})
	assertThresholds("weapon.single_shotgun", []int64{1000, 5000, 20000, 50000})
	assertThresholds("weapon.chainsaw", []int64{250, 1000, 5000})
	assertThresholds("weapon.machine_gun", []int64{500, 2500, 10000})
	assertThresholds("weapon.smg", []int64{1000, 5000, 20000, 50000})
	assertThresholds("weapon.bolt_sniper", []int64{250, 1000, 5000})
	assertThresholds("weapon.heavy_primary", []int64{1000, 5000, 20000, 50000})
	assertThresholds("weapon.grenade_launcher", []int64{500, 2500, 10000})
	assertThresholds("weapon.melee", []int64{1000, 5000, 20000, 50000})
	assertThresholds("secret.crashed", []int64{5})
	assertThresholds("secret.see_u_again", []int64{100})

	titles := make([]string, 0, 26)
	for _, item := range catalog {
		if item.Tier <= 1 {
			titles = append(titles, item.Title)
		}
	}
	sort.Strings(titles)
	for _, frozen := range []string{"老兵", "生还者", "清道夫", "神枪手", "生死之交", "已坠机", "See u Again", "黑枪王"} {
		index := sort.SearchStrings(titles, frozen)
		if index == len(titles) || titles[index] != frozen {
			t.Fatalf("missing frozen title %q", frozen)
		}
	}
}

func TestAchievementCatalogV1NamesThresholdsAndVisibility(t *testing.T) {
	type spec struct {
		group, title, visibility string
		counts                   bool
		thresholds               []int64
	}
	expected := []spec{
		{"career.veteran", "老兵", "public", true, []int64{36000, 180000, 720000, 1800000}},
		{"career.survivor", "生还者", "public", true, []int64{10, 50, 200, 500}},
		{"combat.scavenger", "清道夫", "public", true, []int64{10000, 50000, 200000, 500000}},
		{"combat.special_hunter", "特感猎手", "public", true, []int64{1000, 5000, 20000, 50000}},
		{"combat.marksman", "神枪手", "public", true, []int64{5000, 25000, 100000, 250000}},
		{"combat.team_hunt", "协同猎杀", "public", true, []int64{500, 2500, 10000, 25000}},
		{"support.steadfast", "坚毅不倒", "public", true, []int64{100, 500, 2000, 5000}},
		{"support.defuser", "拆火专家", "public", true, []int64{100, 500, 2000}},
		{"support.field_medic", "战地医生", "public", true, []int64{10000, 50000, 200000}},
		{"boss.tank_hunter", "Tank 猎手", "public", true, []int64{50, 200, 500}},
		{"boss.witch_hunter", "女巫猎人", "public", true, []int64{50, 200, 500}},
		{"boss.boss_nemesis", "Boss 克星", "public", true, []int64{100, 500, 2000}},
		{"versus.player_hunter", "玩家猎手", "public", true, []int64{500, 2500, 10000}},
		{"versus.infected_master", "感染大师", "public", true, []int64{50000, 250000, 1000000}},
		{"bond.comrade", "生死之交", "public", true, []int64{36000, 180000, 360000}},
		{"weapon.throwable_expert", "投掷专家", "public", true, []int64{50, 250, 1000, 2500}},
		{"career.objective_master", "机关大师", "public", true, []int64{25, 100, 500}},
		{"career.temp_health_addict", "药不能停", "public", true, []int64{100, 500, 2000, 5000}},
		{"support.firepower_upgrade", "火力升级", "public", true, []int64{25, 100, 500}},
		{"weapon.single_shotgun", "单喷足以", "public", true, []int64{1000, 5000, 20000, 50000}},
		{"weapon.chainsaw", "电锯狂魔", "public", true, []int64{250, 1000, 5000}},
		{"weapon.machine_gun", "枪林弹雨", "public", true, []int64{500, 2500, 10000}},
		{"weapon.smg", "冲锋陷阵", "public", true, []int64{1000, 5000, 20000, 50000}},
		{"weapon.bolt_sniper", "一发入魂", "public", true, []int64{250, 1000, 5000}},
		{"weapon.heavy_primary", "重火力", "public", true, []int64{1000, 5000, 20000, 50000}},
		{"weapon.grenade_launcher", "爆破专家", "public", true, []int64{500, 2500, 10000}},
		{"weapon.melee", "冷兵器大师", "public", true, []int64{1000, 5000, 20000, 50000}},
		{"special.rock_breaker", "碎石机", "mystery", true, []int64{5}},
		{"special.one_shot", "一击毙命", "mystery", true, []int64{5}},
		{"special.witch_nemesis", "女巫克星", "mystery", true, []int64{5}},
		{"special.tongue_cutter", "砍舌达人", "mystery", true, []int64{5}},
		{"special.defib_rescuer", "起死回生", "public", true, []int64{100}},
		{"special.miracle_healer", "妙手回春", "public", true, []int64{100}},
		{"secret.crashed", "已坠机", "secret", false, []int64{5}},
		{"secret.see_u_again", "See u Again", "secret", false, []int64{100}},
		{"secret.dispatch", "出警", "secret", false, []int64{100}},
		{"secret.ff_king", "黑枪王", "secret", false, []int64{10000}},
		{"secret.submissive", "已老实", "secret", false, []int64{1000}},
	}
	catalog := AchievementCatalog()
	for _, want := range expected {
		got := make([]AchievementDefinition, 0, len(want.thresholds))
		for _, item := range catalog {
			if item.GroupKey == want.group {
				got = append(got, item)
			}
		}
		if len(got) != len(want.thresholds) {
			t.Fatalf("%s has %d definitions, want %d", want.group, len(got), len(want.thresholds))
		}
		for index, item := range got {
			wantKey := want.group
			if len(want.thresholds) > 1 {
				wantKey = fmt.Sprintf("%s.%d", want.group, index+1)
			}
			if item.AchievementKey != wantKey || item.Title != want.title || item.Visibility != want.visibility || item.CountsTowardCompletion != want.counts || item.Threshold != want.thresholds[index] || item.ArtworkKey != want.group {
				t.Fatalf("frozen definition mismatch: got %#v, want key=%s title=%s visibility=%s counts=%t threshold=%d artwork=%s", item, wantKey, want.title, want.visibility, want.counts, want.thresholds[index], want.group)
			}
		}
	}
}

func TestAchievementThresholdBoundariesAndMultiTierUnlock(t *testing.T) {
	metric := store.PlayerAchievementMetrics{SteamID: "765", Values: map[string]store.AchievementMetricValue{
		"career.active_play_seconds": {Available: true, Value: 10*3600 - 1},
	}}
	if got := achievementUnlockCandidates("765", "live", 100, metric, nil); len(got) != 0 {
		t.Fatalf("threshold - 1 unlocked %d achievements", len(got))
	}
	metric.Values["career.active_play_seconds"] = store.AchievementMetricValue{Available: true, Value: 10 * 3600}
	got := achievementUnlockCandidates("765", "live", 101, metric, nil)
	if len(got) != 1 || got[0].AchievementKey != "career.veteran.1" || got[0].ValueAtUnlock != 10*3600 || got[0].GrantKind != "live" {
		t.Fatalf("threshold unlock=%#v", got)
	}
	metric.Values["career.active_play_seconds"] = store.AchievementMetricValue{Available: true, Value: 10*3600 + 1}
	got = achievementUnlockCandidates("765", "live", 101, metric, nil)
	if len(got) != 1 || got[0].AchievementKey != "career.veteran.1" || got[0].ValueAtUnlock != 10*3600+1 {
		t.Fatalf("threshold + 1 unlock=%#v", got)
	}
	metric.Values["career.active_play_seconds"] = store.AchievementMetricValue{Available: true, Value: 500 * 3600}
	got = achievementUnlockCandidates("765", "backfill", 102, metric, nil)
	if len(got) != 4 || got[3].AchievementKey != "career.veteran.4" || got[3].GrantKind != "backfill" {
		t.Fatalf("multi-tier unlock=%#v", got)
	}
	got = achievementUnlockCandidates("765", "live", 103, metric, []store.AchievementUnlock{{AchievementKey: "career.veteran.1"}})
	if len(got) != 3 || got[0].AchievementKey != "career.veteran.2" {
		t.Fatalf("existing unlock was not skipped: %#v", got)
	}
}

func TestV133AchievementThresholdBoundaries(t *testing.T) {
	tests := []struct {
		metric string
		key    string
		value  int64
	}{
		{"survivor.throwables_used", "weapon.throwable_expert.1", 50},
		{"survivor.objective_interactions", "career.objective_master.1", 25},
		{"survivor.temp_health_items_used", "career.temp_health_addict.1", 100},
		{"survivor.upgrade_packs_deployed", "support.firepower_upgrade.1", 25},
		{"weapon.single_shotgun_kills", "weapon.single_shotgun.1", 1000},
		{"weapon.chainsaw_kills", "weapon.chainsaw.1", 250},
		{"weapon.machine_gun_kills", "weapon.machine_gun.1", 500},
		{"weapon.smg_kills", "weapon.smg.1", 1000},
		{"weapon.bolt_sniper_kills", "weapon.bolt_sniper.1", 250},
		{"weapon.heavy_primary_kills", "weapon.heavy_primary.1", 1000},
		{"weapon.grenade_launcher_kills", "weapon.grenade_launcher.1", 500},
		{"weapon.melee_kills", "weapon.melee.1", 1000},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			metrics := store.PlayerAchievementMetrics{SteamID: "765", Values: map[string]store.AchievementMetricValue{
				test.metric: {Available: true, Value: test.value - 1},
			}}
			if got := achievementUnlockCandidates("765", "live", 1, metrics, nil); len(got) != 0 {
				t.Fatalf("threshold - 1 unlocked %#v", got)
			}
			for _, value := range []int64{test.value, test.value + 1} {
				metrics.Values[test.metric] = store.AchievementMetricValue{Available: true, Value: value}
				got := achievementUnlockCandidates("765", "live", 1, metrics, nil)
				if len(got) != 1 || got[0].AchievementKey != test.key || got[0].ValueAtUnlock != value {
					t.Fatalf("value %d unlocked %#v", value, got)
				}
			}
		})
	}
}

func TestV133AchievementCategories(t *testing.T) {
	want := map[string]string{
		"weapon.throwable_expert":   "weapon",
		"career.objective_master":   "career",
		"career.temp_health_addict": "career",
		"support.firepower_upgrade": "support",
		"weapon.single_shotgun":     "weapon",
		"weapon.chainsaw":           "weapon",
		"weapon.machine_gun":        "weapon",
		"weapon.smg":                "weapon",
		"weapon.bolt_sniper":        "weapon",
		"weapon.heavy_primary":      "weapon",
		"weapon.grenade_launcher":   "weapon",
		"weapon.melee":              "weapon",
	}
	seen := make(map[string]bool, len(want))
	for _, item := range AchievementCatalog() {
		category, ok := want[item.GroupKey]
		if !ok {
			continue
		}
		seen[item.GroupKey] = true
		if item.Category != category || item.Visibility != "public" || !item.CountsTowardCompletion {
			t.Errorf("%s classification = %#v", item.GroupKey, item)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("saw %d v1.3.3 groups, want %d", len(seen), len(want))
	}
}

func TestAchievementUnavailableMetricAndEvidence(t *testing.T) {
	metrics := store.PlayerAchievementMetrics{SteamID: "765", Values: map[string]store.AchievementMetricValue{
		"survivor_fall_deaths":                 {Available: false, Value: 100},
		"relationship.max_peer_shared_seconds": {Available: true, Value: 10 * 3600, EvidenceSteamID: "76561198000000002"},
	}}
	got := achievementUnlockCandidates("765", "backfill", 200, metrics, nil)
	if len(got) != 1 || got[0].AchievementKey != "bond.comrade.1" || got[0].EvidenceSteamID != "76561198000000002" {
		t.Fatalf("NULL/evidence semantics=%#v", got)
	}
	metrics.Values["relationship.max_peer_shared_seconds"] = store.AchievementMetricValue{Available: true, Value: 50 * 3600, EvidenceSteamID: "76561198000000003"}
	got = achievementUnlockCandidates("765", "live", 201, metrics, got)
	for _, item := range got {
		if item.AchievementKey == "bond.comrade.1" {
			t.Fatalf("existing evidence drifted to %q", item.EvidenceSteamID)
		}
	}
}
