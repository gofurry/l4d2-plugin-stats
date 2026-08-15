package service

import (
	"strconv"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

// Equipment IDs are the stable values frozen by the Collector's EquipmentType
// enum. Achievement families live here so retention-safe lifetime aggregates
// have one auditable mapping instead of per-achievement conditionals.
const (
	equipmentOtherFirearm    int64 = 1
	equipmentSMG             int64 = 5
	equipmentSilencedSMG     int64 = 6
	equipmentMP5             int64 = 7
	equipmentPumpShotgun     int64 = 8
	equipmentChromeShotgun   int64 = 9
	equipmentAutoShotgun     int64 = 10
	equipmentSPAS            int64 = 11
	equipmentM16             int64 = 12
	equipmentAK47            int64 = 13
	equipmentSCAR            int64 = 14
	equipmentSG552           int64 = 15
	equipmentHuntingRifle    int64 = 16
	equipmentMilitarySniper  int64 = 17
	equipmentScout           int64 = 18
	equipmentAWP             int64 = 19
	equipmentGrenadeLauncher int64 = 20
	equipmentM60             int64 = 21
	equipmentChainsaw        int64 = 22
	equipmentMountedGun      int64 = 23
	equipmentMinigun         int64 = 24
	equipmentBaseballBat     int64 = 25
	equipmentCricketBat      int64 = 26
	equipmentCrowbar         int64 = 27
	equipmentElectricGuitar  int64 = 28
	equipmentFireAxe         int64 = 29
	equipmentFryingPan       int64 = 30
	equipmentGolfClub        int64 = 31
	equipmentKatana          int64 = 32
	equipmentKnife           int64 = 33
	equipmentMachete         int64 = 34
	equipmentPitchfork       int64 = 35
	equipmentShovel          int64 = 36
	equipmentTonfa           int64 = 37
	equipmentMolotov         int64 = 38
	equipmentPipeBomb        int64 = 39
	equipmentVomitJar        int64 = 40
)

var achievementEquipmentFamilies = map[string][]int64{
	"weapon.single_shotgun_kills": {equipmentPumpShotgun, equipmentChromeShotgun},
	"weapon.chainsaw_kills":       {equipmentChainsaw},
	"weapon.machine_gun_kills":    {equipmentM60, equipmentMountedGun, equipmentMinigun},
	"weapon.smg_kills":            {equipmentSMG, equipmentSilencedSMG, equipmentMP5},
	"weapon.bolt_sniper_kills":    {equipmentScout, equipmentAWP},
	"weapon.heavy_primary_kills": {
		equipmentAutoShotgun, equipmentSPAS, equipmentHuntingRifle, equipmentMilitarySniper,
		equipmentM16, equipmentAK47, equipmentSCAR, equipmentSG552,
	},
	"weapon.grenade_launcher_kills": {equipmentGrenadeLauncher},
	"weapon.melee_kills": {
		equipmentBaseballBat, equipmentCricketBat, equipmentCrowbar, equipmentElectricGuitar,
		equipmentFireAxe, equipmentFryingPan, equipmentGolfClub, equipmentKatana, equipmentKnife,
		equipmentMachete, equipmentPitchfork, equipmentShovel, equipmentTonfa,
	},
}

var throwableEquipmentIDs = map[int64]struct{}{
	equipmentMolotov: {}, equipmentPipeBomb: {}, equipmentVomitJar: {},
}

func lifetimeEquipmentAchievementMetrics(rows []store.AggregateRow) map[string]int64 {
	result := make(map[string]int64, len(achievementEquipmentFamilies)+2)
	familyByEquipment := make(map[int64]string)
	for metricID, equipmentIDs := range achievementEquipmentFamilies {
		result[metricID] = 0
		for _, equipmentID := range equipmentIDs {
			familyByEquipment[equipmentID] = metricID
		}
	}

	for _, row := range rows {
		equipmentID, err := strconv.ParseInt(row.Dimension, 10, 64)
		if err != nil {
			continue
		}
		result["pve.firearm_headshot_kills"] += row.Metrics["headshot_kills"]
		if metricID, ok := familyByEquipment[equipmentID]; ok {
			result[metricID] += row.Metrics["common_kills"] + row.Metrics["special_kills"] + row.Metrics["tank_kills"] + row.Metrics["witch_kills"]
		}
		if _, ok := throwableEquipmentIDs[equipmentID]; ok {
			result["pve.throwables_used"] += row.Metrics["actions"]
		}
	}
	return result
}
