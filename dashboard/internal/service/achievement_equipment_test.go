package service

import (
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestAchievementEquipmentFamiliesAreFrozenAndDisjoint(t *testing.T) {
	want := map[string][]int64{
		"weapon.single_shotgun_kills":   {8, 9},
		"weapon.chainsaw_kills":         {22},
		"weapon.machine_gun_kills":      {21, 23, 24},
		"weapon.smg_kills":              {5, 6, 7},
		"weapon.bolt_sniper_kills":      {18, 19},
		"weapon.heavy_primary_kills":    {10, 11, 12, 13, 14, 15, 16, 17},
		"weapon.grenade_launcher_kills": {20},
		"weapon.melee_kills":            {25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37},
	}
	seen := make(map[int64]string)
	for metricID, wantIDs := range want {
		got := append([]int64(nil), achievementEquipmentFamilies[metricID]...)
		sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
		if !reflect.DeepEqual(got, wantIDs) {
			t.Fatalf("%s equipment IDs = %v, want %v", metricID, got, wantIDs)
		}
		for _, equipmentID := range got {
			if other, exists := seen[equipmentID]; exists {
				t.Fatalf("equipment %d belongs to both %s and %s", equipmentID, other, metricID)
			}
			seen[equipmentID] = metricID
		}
	}
	for _, excluded := range []int64{equipmentOtherFirearm, equipmentMolotov, equipmentPipeBomb, equipmentVomitJar} {
		if family, exists := seen[excluded]; exists {
			t.Fatalf("excluded equipment %d entered %s", excluded, family)
		}
	}
}

func TestLifetimeEquipmentAchievementMetrics(t *testing.T) {
	row := func(id int64, actions, common, special, tank, witch, headshots int64) store.AggregateRow {
		return store.AggregateRow{Dimension: strconv.FormatInt(id, 10), Metrics: map[string]int64{
			"actions": actions, "common_kills": common, "special_kills": special,
			"tank_kills": tank, "witch_kills": witch, "headshot_kills": headshots,
		}}
	}
	metrics := lifetimeEquipmentAchievementMetrics([]store.AggregateRow{
		row(equipmentPumpShotgun, 3, 10, 2, 1, 1, 4),
		row(equipmentM60, 2, 20, 3, 2, 1, 5),
		row(equipmentScout, 1, 30, 4, 3, 1, 6),
		row(equipmentHuntingRifle, 1, 40, 5, 4, 1, 7),
		row(equipmentChainsaw, 8, 50, 6, 5, 1, 8),
		row(equipmentKatana, 9, 60, 7, 6, 1, 9),
		row(equipmentOtherFirearm, 10, 100, 100, 100, 100, 10),
		row(equipmentMolotov, 11, 1, 1, 1, 1, 0),
		row(equipmentPipeBomb, 12, 1, 1, 1, 1, 0),
		row(equipmentVomitJar, 13, 1, 1, 1, 1, 0),
	})
	want := map[string]int64{
		"weapon.single_shotgun_kills":   14,
		"weapon.machine_gun_kills":      26,
		"weapon.bolt_sniper_kills":      38,
		"weapon.heavy_primary_kills":    50,
		"weapon.chainsaw_kills":         62,
		"weapon.melee_kills":            74,
		"weapon.smg_kills":              0,
		"weapon.grenade_launcher_kills": 0,
		"pve.throwables_used":           36,
		"pve.firearm_headshot_kills":    49,
	}
	for metricID, wantValue := range want {
		if got := metrics[metricID]; got != wantValue {
			t.Errorf("%s = %d, want %d", metricID, got, wantValue)
		}
	}
}
