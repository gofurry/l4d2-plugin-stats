package service

import (
	"strings"
	"testing"
)

func TestDerivedRankingDefinitionsFreezeDirectionAndSampleGates(t *testing.T) {
	tests := []struct {
		name             string
		higher           bool
		hardActive       int64
		belowSample      map[string]int64
		eligibleSample   map[string]int64
		expectedEligible bool
	}{
		{name: "pve:headshot_kills", higher: true},
		{name: "pve:rescues_per_hour", higher: true, hardActive: 7200},
		{name: "pve:incaps_per_hour", higher: false, hardActive: 7200},
		{name: "pve:deaths_per_hour", higher: false, hardActive: 7200},
		{name: "pve:friendly_fire_per_hour", higher: false, hardActive: 7200},
		{name: "versus_survivor:rescues_per_hour", higher: true, hardActive: 7200},
		{name: "versus_survivor:incaps_per_hour", higher: false, hardActive: 7200},
		{name: "pve:tank_participation_rate", higher: true, belowSample: map[string]int64{"tank_encounters": 4}, eligibleSample: map[string]int64{"tank_encounters": 5}, expectedEligible: true},
		{name: "pve:witch_participation_rate", higher: true, belowSample: map[string]int64{"witch_encounters": 4}, eligibleSample: map[string]int64{"witch_encounters": 5}, expectedEligible: true},
		{name: "versus_infected:incaps_per_spawn", higher: true, belowSample: map[string]int64{"spawn_count": 19}, eligibleSample: map[string]int64{"spawn_count": 20}, expectedEligible: true},
		{name: "versus_infected:controls_per_spawn", higher: true, belowSample: map[string]int64{"spawn_count": 19}, eligibleSample: map[string]int64{"spawn_count": 20}, expectedEligible: true},
		{name: "versus_infected:kills_per_spawn", higher: true, belowSample: map[string]int64{"spawn_count": 19}, eligibleSample: map[string]int64{"spawn_count": 20}, expectedEligible: true},
		{name: "pve:teammate_protections", higher: true},
		{name: "pve:hunter_skeets", higher: true},
		{name: "pve:charger_levels", higher: true},
		{name: "versus_survivor:teammate_protections", higher: true},
		{name: "versus_survivor:hunter_skeets", higher: true},
		{name: "versus_survivor:charger_levels", higher: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition, ok := rankingDefinitions[test.name]
			if !ok {
				t.Fatal("definition is missing")
			}
			if definition.higherIsBetter != test.higher {
				t.Fatalf("higherIsBetter = %v, want %v", definition.higherIsBetter, test.higher)
			}
			if definition.hardMinimumActive != test.hardActive {
				t.Fatalf("hardMinimumActive = %d, want %d", definition.hardMinimumActive, test.hardActive)
			}
			if strings.Contains(test.name, "teammate_protections") || strings.Contains(test.name, "hunter_skeets") || strings.Contains(test.name, "charger_levels") {
				if definition.rawMetric == "" {
					t.Fatal("v1.3.5 telemetry ranking must read nullable raw core rows")
				}
			}
			if test.expectedEligible {
				if definition.minimumSample == nil || definition.minimumSample(test.belowSample) || !definition.minimumSample(test.eligibleSample) {
					t.Fatal("sample gate does not enforce the frozen threshold")
				}
			}
		})
	}
}
