package service

import (
	"reflect"
	"sort"
	"testing"
)

func TestAchievementCatalogV1Frozen(t *testing.T) {
	catalog := AchievementCatalog()
	if len(catalog) != 63 {
		t.Fatalf("catalog has %d achievements, want 63", len(catalog))
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
	if completion != 58 || secrets != 5 || len(artwork) != 26 {
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
