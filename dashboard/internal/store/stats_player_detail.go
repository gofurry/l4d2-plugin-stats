package store

import (
	"context"
	"fmt"
	"strings"
)

const playerActivityTimelineLimit = 30

func (s *statsStore) PlayerActivity(ctx context.Context, steamID string, cutoff int64) (PlayerActivity, error) {
	return s.PlayerActivityFiltered(ctx, steamID, PlayerFilter{Cutoff: cutoff})
}

func (s *statsStore) PlayerActivityFiltered(ctx context.Context, steamID string, filter PlayerFilter) (PlayerActivity, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	p1, p2 := s.bind(1), s.bind(2)
	day := s.dayExpression("started_at")
	serverCondition := ""
	args := []any{steamID, filter.Cutoff}
	if s.driver != "postgres" {
		args = []any{steamID, filter.Cutoff, filter.Cutoff}
	}
	if filter.ServerKey != "" {
		serverPosition := 3
		if s.driver != "postgres" {
			serverPosition = 4
		}
		serverCondition = " AND server_key=" + s.bind(serverPosition)
		args = append(args, filter.ServerKey)
	}
	timelineQuery := fmt.Sprintf(`SELECT %s AS day, COUNT(*), COALESCE(SUM(connected_seconds),0), COALESCE(SUM(active_play_seconds),0)
FROM lps_sessions WHERE steam_id=%s AND (%s=0 OR started_at >= %s)%s
GROUP BY %s ORDER BY %s`, day, p1, p2, p2, serverCondition, day, day)
	rows, err := s.db.QueryContext(queryCtx, timelineQuery, args...)
	if err != nil {
		return PlayerActivity{}, err
	}
	result := PlayerActivity{Timeline: []PlayerActivityPoint{}, Servers: []PlayerServerActivity{}}
	for rows.Next() {
		var dayValue, sessions, connected, active any
		if err := rows.Scan(&dayValue, &sessions, &connected, &active); err != nil {
			rows.Close()
			return PlayerActivity{}, err
		}
		result.Timeline = append(result.Timeline, PlayerActivityPoint{Day: integerValue(dayValue), SessionCount: integerValue(sessions), ConnectedSeconds: integerValue(connected), ActiveSeconds: integerValue(active)})
	}
	if err := rows.Close(); err != nil {
		return PlayerActivity{}, err
	}
	// The summary remains all-time, but charts never need an unbounded number of
	// daily points. Keep at most the newest 30 points for legibility.
	if len(result.Timeline) > playerActivityTimelineLimit {
		result.Timeline = result.Timeline[len(result.Timeline)-playerActivityTimelineLimit:]
	}
	serverQuery := fmt.Sprintf(`SELECT server_key, COUNT(*), COALESCE(SUM(active_play_seconds),0)
FROM lps_sessions WHERE steam_id=%s AND (%s=0 OR started_at >= %s)
GROUP BY server_key ORDER BY COALESCE(SUM(active_play_seconds),0) DESC, server_key`, p1, p2, p2)
	serverArgs := []any{steamID, filter.Cutoff}
	if s.driver != "postgres" {
		serverArgs = []any{steamID, filter.Cutoff, filter.Cutoff}
	}
	rows, err = s.db.QueryContext(queryCtx, serverQuery, serverArgs...)
	if err != nil {
		return PlayerActivity{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var server string
		var sessions, active any
		if err := rows.Scan(&server, &sessions, &active); err != nil {
			return PlayerActivity{}, err
		}
		result.Servers = append(result.Servers, PlayerServerActivity{ServerKey: server, SessionCount: integerValue(sessions), ActiveSeconds: integerValue(active)})
	}
	return result, rows.Err()
}

func (s *statsStore) PlayerPVEFiltered(ctx context.Context, steamID string, filter PlayerFilter) (PlayerPVE, error) {
	return s.enrichPlayerPVE(ctx, steamID, filter, PlayerPVE{})
}

func (s *statsStore) enrichPlayerPVE(ctx context.Context, steamID string, filter PlayerFilter, result PlayerPVE) (PlayerPVE, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	where, args := s.playerSegmentWhere("p", "pve", "survivor", steamID, filter)
	core, err := s.queryMetricTotals(queryCtx, "lps_pve_segment_stats p "+where+" AND p.stats_version=1", "p", pveCombatMetrics, args...)
	if err != nil {
		return PlayerPVE{}, err
	}
	detail, err := s.queryMetricTotals(queryCtx, "lps_pve_segment_stats p "+where+" AND p.stats_version=1", "p", pveDetailMetrics, args...)
	if err != nil {
		return PlayerPVE{}, err
	}
	incidents, err := s.queryMetricTotals(queryCtx, "lps_pve_segment_stats p "+where+" AND p.stats_version=1", "p", []string{"car_alarms_triggered"}, args...)
	if err != nil {
		return PlayerPVE{}, err
	}
	result.CommonKills = core["common_kills"]
	result.SpecialKills = core["special_kills"]
	result.TankKills = core["tank_kills"]
	result.WitchKills = core["witch_kills"]
	result.DamageToSpecial = core["damage_to_special"]
	result.DamageToTank = core["damage_to_tank"]
	result.DamageToWitch = core["damage_to_witch"]
	result.DamageTaken = core["damage_taken_infected"]
	result.FriendlyFire = core["friendly_fire_to_humans"] + core["friendly_fire_to_bots"]
	result.FriendlyFireTaken = core["friendly_fire_taken"]
	result.Incapacitations = core["incapacitations"]
	result.Deaths = core["deaths"]
	result.IncapRevives = core["incap_revives"]
	result.LedgeRescues = core["ledge_rescues"]
	result.DefibRevives = core["defib_revives"]
	result.Revives = result.IncapRevives + result.LedgeRescues + result.DefibRevives
	result.RescuesReceived = core["rescues_received"]
	result.MedkitsUsedSelf = core["medkits_used_self"]
	result.MedkitsUsedOnOthers = core["medkits_used_on_others"]
	result.MedkitsUsed = result.MedkitsUsedSelf + result.MedkitsUsedOnOthers
	result.MedkitHealingSelf = core["medkit_healing_self"]
	result.MedkitHealingOthers = core["medkit_healing_others"]
	result.Healing = result.MedkitHealingSelf + result.MedkitHealingOthers
	result.PillsUsed = core["pills_used"]
	result.AdrenalineUsed = core["adrenaline_used"]
	result.TemporaryHealth = core["temporary_health_received"]
	result.ChapterParticipations = core["chapter_participations"]
	result.ChapterCompletedAlive = core["chapter_completions_alive"]
	result.ChapterCompletedDead = core["chapter_completions_dead"]
	result.ChapterCompletions = result.ChapterCompletedAlive + result.ChapterCompletedDead
	result.CampaignCompletions = core["campaign_completions"]
	result.TongueSelfCuts = detail["melee_tongue_self_cuts"]
	result.TankRocksDestroyed = detail["tank_rocks_destroyed"]
	result.WitchOneShots = detail["witch_oneshots"]
	result.WitchSoloKills = detail["witch_solo_kills"]
	result.TankEncounters = detail["tank_encounters"]
	result.TankParticipations = detail["tank_kill_participations"]
	result.WitchEncounters = detail["witch_encounters"]
	result.WitchParticipations = detail["witch_kill_participations"]
	result.IncendiaryPacks = detail["incendiary_packs_deployed"]
	result.ExplosivePacks = detail["explosive_packs_deployed"]
	result.ObjectiveInteractions = detail["objective_interactions"]
	result.AmmoPileUses = detail["ammo_pile_uses"]
	result.IncapacitatedSeconds = detail["incapacitated_seconds"]
	result.LedgeHangingSeconds = detail["ledge_hanging_seconds"]
	result.BlackWhiteRestored = detail["black_white_teammates_restored"]
	result.CarAlarmsTriggered = incidents["car_alarms_triggered"]
	classNames := []string{"smoker", "boomer", "hunter", "spitter", "jockey", "charger"}
	result.Classes = make([]PVEInfectedClass, 0, len(classNames))
	for index, name := range classNames {
		result.Classes = append(result.Classes, PVEInfectedClass{
			ClassID: index + 1, Kills: detail[name+"_kills"], Damage: detail["damage_to_"+name],
			ControlsReceived: detail[name+"_controls_received"], ControlledSeconds: detail[name+"_controlled_seconds"], Saves: detail[name+"_saves"],
		})
	}
	equipmentRows, err := s.queryGroupedMetrics(queryCtx, "lps_pve_segment_equipment_stats p "+where+" AND p.stats_version=1", "p", "p.equipment_id", equipmentMetrics, args...)
	if err != nil {
		return PlayerPVE{}, err
	}
	result.Equipment = make([]PVEEquipment, 0, len(equipmentRows))
	for _, row := range equipmentRows {
		m := row.metrics
		result.Equipment = append(result.Equipment, PVEEquipment{EquipmentID: row.dimension, Actions: m["actions"], CommonKills: m["common_kills"], SpecialKills: m["special_kills"], TankKills: m["tank_kills"], WitchKills: m["witch_kills"], HeadshotKills: m["headshot_kills"], DamageToSpecial: m["damage_to_special"], DamageToTank: m["damage_to_tank"], DamageToWitch: m["damage_to_witch"]})
	}
	return result, nil
}

func (s *statsStore) PlayerVersusFiltered(ctx context.Context, steamID string, filter PlayerFilter) (PlayerVersus, error) {
	return s.enrichPlayerVersus(ctx, steamID, filter, PlayerVersus{})
}

func (s *statsStore) enrichPlayerVersus(ctx context.Context, steamID string, filter PlayerFilter, result PlayerVersus) (PlayerVersus, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	survivorWhere, args := s.playerSegmentWhere("p", "versus", "survivor", steamID, filter)
	survivor, err := s.queryMetricTotals(queryCtx, "lps_versus_survivor_stats p "+survivorWhere+" AND p.stats_version=1", "p", versusSurvivorMetrics, args...)
	if err != nil {
		return PlayerVersus{}, err
	}
	survivorIncidents, err := s.queryMetricTotals(queryCtx, "lps_versus_survivor_stats p "+survivorWhere+" AND p.stats_version=1", "p", []string{"objective_interactions", "car_alarms_triggered"}, args...)
	if err != nil {
		return PlayerVersus{}, err
	}
	result.SurvivorCommonKills = survivor["common_kills"]
	result.HumanSpecialKills = survivor["human_special_kills"]
	result.BotSpecialKills = survivor["bot_special_kills"]
	result.HumanTankKills = survivor["human_tank_kills"]
	result.BotTankKills = survivor["bot_tank_kills"]
	result.SurvivorDamage = survivor["damage_to_human_special"] + survivor["damage_to_bot_special"] + survivor["damage_to_human_tank"] + survivor["damage_to_bot_tank"]
	result.SurvivorDamageTaken = survivor["damage_taken_infected"]
	result.SurvivorFriendlyFire = survivor["friendly_fire_to_humans"] + survivor["friendly_fire_to_bots"]
	result.SurvivorFriendlyFireTaken = survivor["friendly_fire_taken"]
	result.SurvivorIncapacitations = survivor["incapacitations"]
	result.SurvivorDeaths = survivor["deaths"]
	result.SurvivorIncapRevives = survivor["incap_revives"]
	result.SurvivorLedgeRescues = survivor["ledge_rescues"]
	result.SurvivorDefibRevives = survivor["defib_revives"]
	result.SurvivorRevives = result.SurvivorIncapRevives + result.SurvivorLedgeRescues + result.SurvivorDefibRevives
	result.SurvivorRescuesReceived = survivor["rescues_received"]
	result.SurvivorMedkitsSelf = survivor["medkits_used_self"]
	result.SurvivorMedkitsOthers = survivor["medkits_used_on_others"]
	result.SurvivorHealingSelf = survivor["medkit_healing_self"]
	result.SurvivorHealingOthers = survivor["medkit_healing_others"]
	result.SurvivorPills = survivor["pills_used"]
	result.SurvivorAdrenaline = survivor["adrenaline_used"]
	result.SurvivorTemporaryHealth = survivor["temporary_health_received"]
	result.SurvivorWitchKills = survivor["witch_kills"]
	result.SurvivorWitchDamage = survivor["damage_to_witch"]
	result.MolotovsThrown = survivor["molotovs_thrown"]
	result.PipeBombsThrown = survivor["pipe_bombs_thrown"]
	result.VomitJarsThrown = survivor["vomit_jars_thrown"]
	result.SurvivorIncendiaryPacks = survivor["incendiary_packs_deployed"]
	result.SurvivorExplosivePacks = survivor["explosive_packs_deployed"]
	result.SurvivorTongueSelfCuts = survivor["melee_tongue_self_cuts"]
	result.SurvivorTankRocksDestroyed = survivor["tank_rocks_destroyed"]
	result.SurvivorWitchOneShots = survivor["witch_oneshots"]
	result.SurvivorWitchSoloKills = survivor["witch_solo_kills"]
	result.SurvivorObjectiveInteractions = survivorIncidents["objective_interactions"]
	result.SurvivorCarAlarmsTriggered = survivorIncidents["car_alarms_triggered"]
	survivorClasses, err := s.queryGroupedMetrics(queryCtx, "lps_versus_survivor_infected_class_stats p "+survivorWhere+" AND p.stats_version=1", "p", "p.infected_class", versusSurvivorClassMetrics, args...)
	if err != nil {
		return PlayerVersus{}, err
	}
	result.SurvivorClasses = make([]VersusSurvivorClass, 0, len(survivorClasses))
	for _, row := range survivorClasses {
		m := row.metrics
		result.SurvivorClasses = append(result.SurvivorClasses, VersusSurvivorClass{ClassID: row.dimension, HumanControllerKills: m["human_controller_kills"], BotControllerKills: m["bot_controller_kills"], DamageToHumanControllers: m["damage_to_human_controllers"], DamageToBotControllers: m["damage_to_bot_controllers"]})
	}
	infectedWhere, infectedArgs := s.playerSegmentWhere("p", "versus", "infected", steamID, filter)
	infected, err := s.queryMetricTotals(queryCtx, "lps_versus_infected_stats p "+infectedWhere+" AND p.stats_version=1", "p", versusInfectedMetrics, infectedArgs...)
	if err != nil {
		return PlayerVersus{}, err
	}
	result.InfectedSpawns = infected["spawn_count"]
	result.DamageToHumanSurvivors = infected["damage_to_human_survivors"]
	result.DamageToBotSurvivors = infected["damage_to_bot_survivors"]
	result.HumanSurvivorIncaps = infected["human_survivor_incaps"]
	result.BotSurvivorIncaps = infected["bot_survivor_incaps"]
	result.HumanSurvivorKills = infected["human_survivor_kills"]
	result.BotSurvivorKills = infected["bot_survivor_kills"]
	infectedClasses, err := s.queryGroupedMetrics(queryCtx, "lps_versus_infected_class_stats p "+infectedWhere+" AND p.stats_version=1", "p", "p.infected_class", versusInfectedClassMetrics, infectedArgs...)
	if err != nil {
		return PlayerVersus{}, err
	}
	result.InfectedClasses = make([]VersusInfectedClass, 0, len(infectedClasses))
	result.HumanSurvivorControls = 0
	result.HumanSurvivorControlSeconds = 0
	for _, row := range infectedClasses {
		m := row.metrics
		result.InfectedClasses = append(result.InfectedClasses, VersusInfectedClass{ClassID: row.dimension, Spawns: m["spawn_count"], DamageToHumanSurvivors: m["damage_to_human_survivors"], DamageToBotSurvivors: m["damage_to_bot_survivors"], HumanSurvivorIncaps: m["human_survivor_incaps"], BotSurvivorIncaps: m["bot_survivor_incaps"], HumanSurvivorKills: m["human_survivor_kills"], BotSurvivorKills: m["bot_survivor_kills"], HumanSurvivorControls: m["human_survivor_controls"], BotSurvivorControls: m["bot_survivor_controls"], HumanSurvivorControlSeconds: m["human_survivor_control_seconds"], BotSurvivorControlSeconds: m["bot_survivor_control_seconds"], HumanSurvivorAbilityHits: m["human_survivor_ability_hits"], BotSurvivorAbilityHits: m["bot_survivor_ability_hits"], HumanSurvivorAbilityDamage: m["human_survivor_ability_damage"], BotSurvivorAbilityDamage: m["bot_survivor_ability_damage"]})
		result.HumanSurvivorControls += m["human_survivor_controls"]
		result.HumanSurvivorControlSeconds += m["human_survivor_control_seconds"]
	}
	return result, nil
}

func (s *statsStore) playerSegmentWhere(alias, mode, side, steamID string, filter PlayerFilter) (string, []any) {
	p1, p2 := s.bind(1), s.bind(2)
	modeCondition := "r.mode_family='pve' AND r.game_mode IN ('coop','realism')"
	if mode == "versus" {
		modeCondition = "r.mode_family='versus' AND r.game_mode='versus'"
	}
	args := []any{steamID, filter.Cutoff}
	if s.driver != "postgres" {
		args = []any{steamID, filter.Cutoff, filter.Cutoff}
	}
	extra := ""
	nextPosition := 3
	if s.driver != "postgres" {
		nextPosition = 4
	}
	if filter.ServerKey != "" {
		extra += " AND s.server_key=" + s.bind(nextPosition)
		args = append(args, filter.ServerKey)
		nextPosition++
	}
	if mode == "pve" && (filter.GameMode == "coop" || filter.GameMode == "realism") {
		extra += " AND r.game_mode=" + s.bind(nextPosition)
		args = append(args, filter.GameMode)
	}
	return fmt.Sprintf(`JOIN lps_player_segments s ON s.segment_id=%s.segment_id
JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.steam_id=%s AND s.side='%s' AND %s AND (%s=0 OR s.started_at >= %s)%s`, alias, p1, side, modeCondition, p2, p2, extra), args
}

func (s *statsStore) queryMetricTotals(ctx context.Context, source, alias string, fields []string, args ...any) (map[string]int64, error) {
	selects := make([]string, 0, len(fields))
	for _, field := range fields {
		selects = append(selects, fmt.Sprintf("COALESCE(SUM(%s.%s),0)", alias, field))
	}
	row := s.db.QueryRowContext(ctx, "SELECT "+strings.Join(selects, ",")+" FROM "+source, args...)
	values := make([]any, len(fields))
	pointers := make([]any, len(fields))
	for i := range values {
		pointers[i] = &values[i]
	}
	if err := row.Scan(pointers...); err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(fields))
	for i, field := range fields {
		result[field] = integerValue(values[i])
	}
	return result, nil
}

type groupedMetrics struct {
	dimension int64
	metrics   map[string]int64
}

func (s *statsStore) queryGroupedMetrics(ctx context.Context, source, alias, dimension string, fields []string, args ...any) ([]groupedMetrics, error) {
	selects := make([]string, 0, len(fields))
	for _, field := range fields {
		selects = append(selects, fmt.Sprintf("COALESCE(SUM(%s.%s),0)", alias, field))
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+dimension+","+strings.Join(selects, ",")+" FROM "+source+" GROUP BY "+dimension+" ORDER BY "+dimension, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]groupedMetrics, 0)
	for rows.Next() {
		values := make([]any, len(fields)+1)
		pointers := make([]any, len(values))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		metrics := make(map[string]int64, len(fields))
		for i, field := range fields {
			metrics[field] = integerValue(values[i+1])
		}
		result = append(result, groupedMetrics{dimension: integerValue(values[0]), metrics: metrics})
	}
	return result, rows.Err()
}

func (s *statsStore) bind(position int) string {
	if s.driver == "postgres" {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}
