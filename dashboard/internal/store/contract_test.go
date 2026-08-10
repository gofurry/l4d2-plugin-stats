package store

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestDatabaseContract(t *testing.T) {
	ctx := context.Background()
	stats := openDatabaseContractFixture(t)

	version, err := stats.SchemaVersion(ctx)
	contractEqual(t, "SchemaVersion", version, StatsSchemaVersion, err)

	overview, err := stats.Overview(ctx, time.Unix(contractBaseTime-1, 0))
	overview.Generated = time.Time{}
	contractEqual(t, "Overview", overview, Overview{
		Core:   CoreOverview{TotalPlayers: 2, ActivePlayers7Days: 2, TotalActivePlaySeconds: 300, CompletedPVERuns: 1, CompletedVersusRuns: 1},
		PVE:    PVEOverview{CommonKills: 100, SpecialKills: 12, TankKills: 2, WitchKills: 1, Rescues: 9},
		Versus: VersusOverview{CompletedMatches: 1, CompletedHalves: 1, HumanControlledKills: 9, HumanSurvivorControls: 11},
	}, err)

	summary, err := stats.PlayerSummary(ctx, "1")
	contractEqual(t, "PlayerSummary", summary, &PlayerSummary{
		SteamID: "1", LastName: "Alice", FirstSeenAt: contractBaseTime - 100, LastSeenAt: contractBaseTime + 400,
		SessionCount: 1, ConnectedSeconds: 300, ActiveSeconds: 240,
	}, err)

	activity, err := stats.PlayerActivity(ctx, "1", 0)
	contractEqual(t, "PlayerActivity", activity, PlayerActivity{
		Timeline: []PlayerActivityPoint{{Day: contractBaseTime / 86400, SessionCount: 1, ConnectedSeconds: 300, ActiveSeconds: 240}},
		Servers:  []PlayerServerActivity{{ServerKey: "one", SessionCount: 1, ActiveSeconds: 240}},
	}, err)

	pve, err := stats.PlayerPVE(ctx, "1", 0)
	contractEqual(t, "PlayerPVE", pve, expectedContractPVE(), err)

	versus, err := stats.PlayerVersus(ctx, "1", 0)
	contractEqual(t, "PlayerVersus", versus, expectedContractVersus(), err)

	incidentRankings := stats.(StatsIncidentRankingStore)
	pveRanking, err := incidentRankings.CarAlarmRanking(ctx, RankingQuery{Mode: "pve"})
	contractEqual(t, "PVE car alarm ranking", pveRanking, []RankingEntry{{Rank: 1, SteamID: "1", PlayerName: "Alice", Value: 3, ActiveSeconds: 90}}, err)
	versusRanking, err := incidentRankings.CarAlarmRanking(ctx, RankingQuery{Mode: "versus_survivor"})
	contractEqual(t, "Versus car alarm ranking", versusRanking, []RankingEntry{{Rank: 1, SteamID: "1", PlayerName: "Alice", Value: 2, ActiveSeconds: 70}}, err)

	ended := contractBaseTime + 300
	sessions, err := stats.PlayerSessions(ctx, "1", 0, "", 20)
	contractEqual(t, "PlayerSessions", sessions, []PlayerSession{{
		SessionID: "session-alice", ServerKey: "one", PlayerName: "Alice", StartedAt: contractBaseTime,
		EndedAt: &ended, ConnectedSeconds: 300, ActiveSeconds: 240, Status: "closed", DisconnectReason: "quit",
	}}, err)

	chapters, err := stats.PlayerChapters(ctx, "1", 0, "", 20)
	chapterEnd1, chapterEnd2, chapterEnd3 := contractBaseTime+210, contractBaseTime+200, contractBaseTime+180
	contractEqual(t, "PlayerChapters", chapters, []PlayerChapter{
		{SegmentID: "segment-vi", ServerKey: "one", ModeFamily: "versus", GameMode: "versus", MapName: "c5m1_waterfront", Side: "infected", StartedAt: contractBaseTime + 50, EndedAt: &chapterEnd1, ActiveSeconds: 60, Status: "closed"},
		{SegmentID: "segment-vs", ServerKey: "one", ModeFamily: "versus", GameMode: "versus", MapName: "c5m1_waterfront", Side: "survivor", StartedAt: contractBaseTime + 40, EndedAt: &chapterEnd2, ActiveSeconds: 70, Status: "closed"},
		{SegmentID: "segment-pve", ServerKey: "one", ModeFamily: "pve", GameMode: "coop", MapName: "c1m1_hotel", Side: "survivor", StartedAt: contractBaseTime + 30, EndedAt: &chapterEnd3, ActiveSeconds: 90, Status: "closed"},
	}, err)

	changes, err := stats.AggregateChanges(ctx, 0)
	if err != nil {
		t.Fatalf("AggregateChanges: %v", err)
	}
	sort.Slice(changes.Rows, func(i, j int) bool {
		return aggregateContractKey(changes.Rows[i]) < aggregateContractKey(changes.Rows[j])
	})
	if changes.SourceWatermark != contractWatermark || changes.SourceRows != 15 || !changes.Full || !reflect.DeepEqual(changes.Days, []int64{contractBaseTime / 86400}) {
		t.Fatalf("AggregateChanges metadata differs: %+v", changes)
	}
	expectedKinds := []string{"activity", "mode_activity", "pve_combat", "pve_detail", "pve_equipment", "run_result", "versus_infected", "versus_infected_class", "versus_result", "versus_survivor", "versus_survivor_class"}
	actualKinds := make([]string, 0, len(expectedKinds))
	for _, row := range changes.Rows {
		if row.Version != AggregateContractVersion {
			t.Fatalf("AggregateChanges row %s has version %d", row.Kind, row.Version)
		}
		if len(actualKinds) == 0 || actualKinds[len(actualKinds)-1] != row.Kind {
			actualKinds = append(actualKinds, row.Kind)
		}
	}
	contractEqual(t, "AggregateChanges kinds", actualKinds, expectedKinds, nil)
	assertContractAggregateMetric(t, changes.Rows, "pve_combat", "", "common_kills", 100)
	assertContractAggregateMetric(t, changes.Rows, "pve_equipment", "7", "actions", 15)
	assertContractAggregateMetric(t, changes.Rows, "versus_infected_class", "3", "human_survivor_controls", 11)
	quality, err := stats.DeepDataQuality(ctx, contractBaseTime-15*60)
	if err != nil {
		t.Fatalf("DeepDataQuality: %v", err)
	}
	if quality.StaleActiveBoots.Count != 0 || quality.UnknownStatsVersion.Count != 0 || quality.LifecycleLinks.Count != 0 || quality.ModeSideMismatch.Count != 0 || quality.PVETotalMismatch.Count != 1 {
		t.Fatalf("DeepDataQuality differs: %+v", quality)
	}

	cutoff := contractBaseTime + 1000
	plan, err := stats.RetentionPlan(ctx, cutoff, cutoff, cutoff)
	if err != nil {
		t.Fatalf("RetentionPlan: %v", err)
	}
	if plan.EquipmentRowsEligible != 1 || plan.VersusClassRowsEligible != 2 || plan.SessionRowsEligible != 2 || plan.VersusRoundResultsEligible != 1 || plan.VersusRunResultsEligible != 1 || plan.SourceWatermark != contractWatermark {
		t.Fatalf("RetentionPlan differs: %+v", plan)
	}
}

func expectedContractPVE() PlayerPVE {
	return PlayerPVE{
		CommonKills: 100, SpecialKills: 12, TankKills: 2, WitchKills: 1,
		DamageToSpecial: 1200, DamageToTank: 400, DamageToWitch: 200, DamageTaken: 300,
		FriendlyFire: 15, FriendlyFireTaken: 7, Incapacitations: 3, Deaths: 1,
		Revives: 9, IncapRevives: 2, LedgeRescues: 3, DefibRevives: 4, RescuesReceived: 1,
		MedkitsUsed: 3, MedkitsUsedSelf: 2, MedkitsUsedOnOthers: 1,
		Healing: 120, MedkitHealingSelf: 80, MedkitHealingOthers: 40,
		PillsUsed: 3, AdrenalineUsed: 2, TemporaryHealth: 50,
		ChapterParticipations: 1, ChapterCompletions: 1, ChapterCompletedAlive: 1, CampaignCompletions: 1,
		TongueSelfCuts: 1, TankRocksDestroyed: 2, WitchOneShots: 1, WitchSoloKills: 1,
		TankEncounters: 2, TankParticipations: 1, WitchEncounters: 1, WitchParticipations: 1,
		IncendiaryPacks: 1, ExplosivePacks: 1, ObjectiveInteractions: 2, AmmoPileUses: 4,
		IncapacitatedSeconds: 20, LedgeHangingSeconds: 10, BlackWhiteRestored: 1, CarAlarmsTriggered: 3,
		Classes: []PVEInfectedClass{
			{ClassID: 1, Kills: 4, Damage: 500, ControlsReceived: 2, ControlledSeconds: 30, Saves: 3},
			{ClassID: 2}, {ClassID: 3}, {ClassID: 4}, {ClassID: 5}, {ClassID: 6},
		},
		Equipment: []PVEEquipment{{EquipmentID: 7, Actions: 15, CommonKills: 25, SpecialKills: 4, TankKills: 1, WitchKills: 1, HeadshotKills: 8, DamageToSpecial: 300, DamageToTank: 100, DamageToWitch: 50}},
	}
}

func expectedContractVersus() PlayerVersus {
	return PlayerVersus{
		SurvivorCommonKills: 40, HumanSpecialKills: 7, BotSpecialKills: 3, HumanTankKills: 2, BotTankKills: 1, SurvivorDamage: 425,
		SurvivorDamageTaken: 80, SurvivorFriendlyFire: 6, SurvivorFriendlyFireTaken: 3, SurvivorIncapacitations: 2, SurvivorDeaths: 1,
		SurvivorRevives: 4, SurvivorIncapRevives: 1, SurvivorLedgeRescues: 2, SurvivorDefibRevives: 1, SurvivorRescuesReceived: 2,
		SurvivorMedkitsSelf: 1, SurvivorMedkitsOthers: 2, SurvivorHealingSelf: 30, SurvivorHealingOthers: 50,
		SurvivorPills: 2, SurvivorAdrenaline: 1, SurvivorTemporaryHealth: 20, SurvivorWitchKills: 1, SurvivorWitchDamage: 90,
		MolotovsThrown: 1, PipeBombsThrown: 2, VomitJarsThrown: 3, SurvivorIncendiaryPacks: 1, SurvivorExplosivePacks: 2,
		SurvivorTongueSelfCuts: 1, SurvivorTankRocksDestroyed: 1, SurvivorWitchOneShots: 1, SurvivorWitchSoloKills: 1, SurvivorCarAlarmsTriggered: 2,
		InfectedSpawns: 6, DamageToHumanSurvivors: 450, DamageToBotSurvivors: 120, HumanSurvivorIncaps: 3, BotSurvivorIncaps: 2,
		HumanSurvivorKills: 1, BotSurvivorKills: 1, HumanSurvivorControls: 11, HumanSurvivorControlSeconds: 75,
		SurvivorClasses: []VersusSurvivorClass{{ClassID: 3, HumanControllerKills: 5, BotControllerKills: 2, DamageToHumanControllers: 120, DamageToBotControllers: 60}},
		InfectedClasses: []VersusInfectedClass{{ClassID: 3, Spawns: 6, DamageToHumanSurvivors: 450, DamageToBotSurvivors: 120, HumanSurvivorIncaps: 3, BotSurvivorIncaps: 2, HumanSurvivorKills: 1, BotSurvivorKills: 1, HumanSurvivorControls: 11, BotSurvivorControls: 4, HumanSurvivorControlSeconds: 75, BotSurvivorControlSeconds: 20, HumanSurvivorAbilityHits: 9, BotSurvivorAbilityHits: 3, HumanSurvivorAbilityDamage: 180, BotSurvivorAbilityDamage: 40}},
	}
}

func aggregateContractKey(row AggregateRow) string {
	return row.Kind + "\x00" + row.ServerKey + "\x00" + row.SteamID + "\x00" + row.Mode + "\x00" + row.Dimension
}

func assertContractAggregateMetric(t *testing.T, rows []AggregateRow, kind, dimension, metric string, expected int64) {
	t.Helper()
	for _, row := range rows {
		if row.Kind == kind && row.Dimension == dimension {
			if row.Metrics[metric] != expected {
				t.Fatalf("%s[%s].%s = %d, want %d", kind, dimension, metric, row.Metrics[metric], expected)
			}
			return
		}
	}
	t.Fatalf("missing aggregate row %s[%s]", kind, dimension)
}

func contractEqual(t *testing.T, name string, actual, expected any, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s differs\nactual:   %#v\nexpected: %#v", name, actual, expected)
	}
}
