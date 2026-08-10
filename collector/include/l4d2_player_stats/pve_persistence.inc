#if defined _l4d2_player_stats_pve_persistence_included
	#endinput
#endif
#define _l4d2_player_stats_pve_persistence_included

void LPS_QueueClosedPvEStats(LPSPvEStatsSnapshot snapshot)
{
	if (g_LPSClosedPvEStats.Length >= g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		if (!g_LPSPvEStatsQueueOverflowReported)
		{
			LogError(
				"Closed PvE stats memory queue reached its limit (%d); additional snapshots will be dropped until a flush succeeds.",
				g_LPSClosedLifecycleQueueLimit.IntValue
			);
			g_LPSPvEStatsQueueOverflowReported = true;
		}
		return;
	}

	g_LPSClosedPvEStats.PushArray(snapshot, sizeof(snapshot));
}

bool LPS_AddPvEStatsFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_AccumulateAllPvEInteractionDurations();
	LPS_ResetPvEStatsFlushCapture();
	g_LPSFlushClosedPvEStatsCount = g_LPSClosedPvEStats.Length;

	LPSPvEStatsSnapshot snapshot;
	for (int index = 0; index < g_LPSFlushClosedPvEStatsCount; index++)
	{
		g_LPSClosedPvEStats.GetArray(index, snapshot, sizeof(snapshot));
		if (!LPS_AddPvEStatsSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientPvEStatsActive[client] || !g_LPSClientPvEStatsDirty[client])
		{
			continue;
		}

		LPS_FillClientPvEStatsSnapshot(client, snapshot, false);
		if (!LPS_AddPvEStatsSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}

		g_LPSFlushCapturedPvEStats[client] = true;
		g_LPSFlushCapturedPvEStatsRevision[client] = snapshot.revision;
		strcopy(
			g_LPSFlushCapturedPvEStatsSegmentId[client],
			sizeof(g_LPSFlushCapturedPvEStatsSegmentId[]),
			snapshot.segmentId
		);
	}
	return true;
}

bool LPS_AddPvEStatsSnapshotQuery(
	Transaction transaction,
	LPSPvEStatsSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[LPS_MAX_QUERY_LENGTH];
	if (!LPS_BuildPvEStatsUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a PvE stats value exceeded its SQL escape buffer");
		return false;
	}

	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_BuildPvEStatsUpsertQuery(LPSPvEStatsSnapshot snapshot, char[] query, int maxLength)
{
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	char values[4096];
	int written = FormatEx(
		values,
		sizeof(values),
		"'%s', %d, %d",
		segmentId,
		LPS_STATS_VERSION,
		GetTime()
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.commonKills,
		snapshot.specialKills,
		snapshot.tankKills,
		snapshot.witchKills,
		snapshot.damageToSpecial,
		snapshot.damageToTank,
		snapshot.damageToWitch,
		snapshot.damageTakenInfected,
		snapshot.friendlyFireToHumans,
		snapshot.friendlyFireToBots,
		snapshot.friendlyFireTaken,
		snapshot.incapacitations,
		snapshot.deaths,
		snapshot.incapRevives
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.ledgeRescues,
		snapshot.defibRevives,
		snapshot.rescuesReceived,
		snapshot.medkitsUsedSelf,
		snapshot.medkitsUsedOnOthers,
		snapshot.medkitHealingSelf,
		snapshot.medkitHealingOthers,
		snapshot.pillsUsed,
		snapshot.adrenalineUsed,
		snapshot.temporaryHealthReceived,
		snapshot.chapterParticipations,
		snapshot.chapterCompletionsAlive,
		snapshot.chapterCompletionsDead,
		snapshot.campaignCompletions
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.smokerKills,
		snapshot.boomerKills,
		snapshot.hunterKills,
		snapshot.spitterKills,
		snapshot.jockeyKills,
		snapshot.chargerKills,
		snapshot.damageToSmoker,
		snapshot.damageToBoomer,
		snapshot.damageToHunter,
		snapshot.damageToSpitter,
		snapshot.damageToJockey,
		snapshot.damageToCharger,
		snapshot.smokerControlsReceived,
		snapshot.hunterControlsReceived,
		snapshot.jockeyControlsReceived,
		snapshot.chargerControlsReceived,
		snapshot.smokerControlledSeconds
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.hunterControlledSeconds,
		snapshot.jockeyControlledSeconds,
		snapshot.chargerControlledSeconds,
		snapshot.smokerSaves,
		snapshot.hunterSaves,
		snapshot.jockeySaves,
		snapshot.chargerSaves,
		snapshot.meleeTongueSelfCuts,
		snapshot.tankRocksDestroyed,
		snapshot.witchOneShots,
		snapshot.witchSoloKills,
		snapshot.tankEncounters,
		snapshot.tankKillParticipations,
		snapshot.witchEncounters,
		snapshot.witchKillParticipations,
		snapshot.incendiaryPacksDeployed,
		snapshot.explosivePacksDeployed,
		snapshot.objectiveInteractions,
		snapshot.ammoPileUses,
		snapshot.incapacitatedSeconds,
		snapshot.ledgeHangingSeconds,
		snapshot.blackWhiteTeammatesRestored,
		snapshot.carAlarmsTriggered,
		snapshot.revision
	);
	if (written >= sizeof(values) - 1)
	{
		return false;
	}

	char columns[] = "segment_id, stats_version, last_saved_at, common_kills, special_kills, tank_kills, witch_kills, damage_to_special, damage_to_tank, damage_to_witch, damage_taken_infected, friendly_fire_to_humans, friendly_fire_to_bots, friendly_fire_taken, incapacitations, deaths, incap_revives, ledge_rescues, defib_revives, rescues_received, medkits_used_self, medkits_used_on_others, medkit_healing_self, medkit_healing_others, pills_used, adrenaline_used, temporary_health_received, chapter_participations, chapter_completions_alive, chapter_completions_dead, campaign_completions, smoker_kills, boomer_kills, hunter_kills, spitter_kills, jockey_kills, charger_kills, damage_to_smoker, damage_to_boomer, damage_to_hunter, damage_to_spitter, damage_to_jockey, damage_to_charger, smoker_controls_received, hunter_controls_received, jockey_controls_received, charger_controls_received, smoker_controlled_seconds, hunter_controlled_seconds, jockey_controlled_seconds, charger_controlled_seconds, smoker_saves, hunter_saves, jockey_saves, charger_saves, melee_tongue_self_cuts, tank_rocks_destroyed, witch_oneshots, witch_solo_kills, tank_encounters, tank_kill_participations, witch_encounters, witch_kill_participations, incendiary_packs_deployed, explosive_packs_deployed, objective_interactions, ammo_pile_uses, incapacitated_seconds, ledge_hanging_seconds, black_white_teammates_restored, car_alarms_triggered, revision";
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_pve_segment_stats (%s) VALUES (%s) ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), common_kills = VALUES(common_kills), special_kills = VALUES(special_kills), tank_kills = VALUES(tank_kills), witch_kills = VALUES(witch_kills), damage_to_special = VALUES(damage_to_special), damage_to_tank = VALUES(damage_to_tank), damage_to_witch = VALUES(damage_to_witch), damage_taken_infected = VALUES(damage_taken_infected), friendly_fire_to_humans = VALUES(friendly_fire_to_humans), friendly_fire_to_bots = VALUES(friendly_fire_to_bots), friendly_fire_taken = VALUES(friendly_fire_taken), incapacitations = VALUES(incapacitations), deaths = VALUES(deaths), incap_revives = VALUES(incap_revives), ledge_rescues = VALUES(ledge_rescues), defib_revives = VALUES(defib_revives), rescues_received = VALUES(rescues_received), medkits_used_self = VALUES(medkits_used_self), medkits_used_on_others = VALUES(medkits_used_on_others), medkit_healing_self = VALUES(medkit_healing_self), medkit_healing_others = VALUES(medkit_healing_others), pills_used = VALUES(pills_used), adrenaline_used = VALUES(adrenaline_used), temporary_health_received = VALUES(temporary_health_received), chapter_participations = VALUES(chapter_participations), chapter_completions_alive = VALUES(chapter_completions_alive), chapter_completions_dead = VALUES(chapter_completions_dead), campaign_completions = VALUES(campaign_completions), smoker_kills = VALUES(smoker_kills), boomer_kills = VALUES(boomer_kills), hunter_kills = VALUES(hunter_kills), spitter_kills = VALUES(spitter_kills), jockey_kills = VALUES(jockey_kills), charger_kills = VALUES(charger_kills), damage_to_smoker = VALUES(damage_to_smoker), damage_to_boomer = VALUES(damage_to_boomer), damage_to_hunter = VALUES(damage_to_hunter), damage_to_spitter = VALUES(damage_to_spitter), damage_to_jockey = VALUES(damage_to_jockey), damage_to_charger = VALUES(damage_to_charger), smoker_controls_received = VALUES(smoker_controls_received), hunter_controls_received = VALUES(hunter_controls_received), jockey_controls_received = VALUES(jockey_controls_received), charger_controls_received = VALUES(charger_controls_received), smoker_controlled_seconds = VALUES(smoker_controlled_seconds), hunter_controlled_seconds = VALUES(hunter_controlled_seconds), jockey_controlled_seconds = VALUES(jockey_controlled_seconds), charger_controlled_seconds = VALUES(charger_controlled_seconds), smoker_saves = VALUES(smoker_saves), hunter_saves = VALUES(hunter_saves), jockey_saves = VALUES(jockey_saves), charger_saves = VALUES(charger_saves), melee_tongue_self_cuts = VALUES(melee_tongue_self_cuts), tank_rocks_destroyed = VALUES(tank_rocks_destroyed), witch_oneshots = VALUES(witch_oneshots), witch_solo_kills = VALUES(witch_solo_kills), tank_encounters = VALUES(tank_encounters), tank_kill_participations = VALUES(tank_kill_participations), witch_encounters = VALUES(witch_encounters), witch_kill_participations = VALUES(witch_kill_participations), incendiary_packs_deployed = VALUES(incendiary_packs_deployed), explosive_packs_deployed = VALUES(explosive_packs_deployed), objective_interactions = VALUES(objective_interactions), ammo_pile_uses = VALUES(ammo_pile_uses), incapacitated_seconds = VALUES(incapacitated_seconds), ledge_hanging_seconds = VALUES(ledge_hanging_seconds), black_white_teammates_restored = VALUES(black_white_teammates_restored), car_alarms_triggered = VALUES(car_alarms_triggered), revision = VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_pve_segment_stats (%s) VALUES (%s) ON CONFLICT(segment_id) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, common_kills = excluded.common_kills, special_kills = excluded.special_kills, tank_kills = excluded.tank_kills, witch_kills = excluded.witch_kills, damage_to_special = excluded.damage_to_special, damage_to_tank = excluded.damage_to_tank, damage_to_witch = excluded.damage_to_witch, damage_taken_infected = excluded.damage_taken_infected, friendly_fire_to_humans = excluded.friendly_fire_to_humans, friendly_fire_to_bots = excluded.friendly_fire_to_bots, friendly_fire_taken = excluded.friendly_fire_taken, incapacitations = excluded.incapacitations, deaths = excluded.deaths, incap_revives = excluded.incap_revives, ledge_rescues = excluded.ledge_rescues, defib_revives = excluded.defib_revives, rescues_received = excluded.rescues_received, medkits_used_self = excluded.medkits_used_self, medkits_used_on_others = excluded.medkits_used_on_others, medkit_healing_self = excluded.medkit_healing_self, medkit_healing_others = excluded.medkit_healing_others, pills_used = excluded.pills_used, adrenaline_used = excluded.adrenaline_used, temporary_health_received = excluded.temporary_health_received, chapter_participations = excluded.chapter_participations, chapter_completions_alive = excluded.chapter_completions_alive, chapter_completions_dead = excluded.chapter_completions_dead, campaign_completions = excluded.campaign_completions, smoker_kills = excluded.smoker_kills, boomer_kills = excluded.boomer_kills, hunter_kills = excluded.hunter_kills, spitter_kills = excluded.spitter_kills, jockey_kills = excluded.jockey_kills, charger_kills = excluded.charger_kills, damage_to_smoker = excluded.damage_to_smoker, damage_to_boomer = excluded.damage_to_boomer, damage_to_hunter = excluded.damage_to_hunter, damage_to_spitter = excluded.damage_to_spitter, damage_to_jockey = excluded.damage_to_jockey, damage_to_charger = excluded.damage_to_charger, smoker_controls_received = excluded.smoker_controls_received, hunter_controls_received = excluded.hunter_controls_received, jockey_controls_received = excluded.jockey_controls_received, charger_controls_received = excluded.charger_controls_received, smoker_controlled_seconds = excluded.smoker_controlled_seconds, hunter_controlled_seconds = excluded.hunter_controlled_seconds, jockey_controlled_seconds = excluded.jockey_controlled_seconds, charger_controlled_seconds = excluded.charger_controlled_seconds, smoker_saves = excluded.smoker_saves, hunter_saves = excluded.hunter_saves, jockey_saves = excluded.jockey_saves, charger_saves = excluded.charger_saves, melee_tongue_self_cuts = excluded.melee_tongue_self_cuts, tank_rocks_destroyed = excluded.tank_rocks_destroyed, witch_oneshots = excluded.witch_oneshots, witch_solo_kills = excluded.witch_solo_kills, tank_encounters = excluded.tank_encounters, tank_kill_participations = excluded.tank_kill_participations, witch_encounters = excluded.witch_encounters, witch_kill_participations = excluded.witch_kill_participations, incendiary_packs_deployed = excluded.incendiary_packs_deployed, explosive_packs_deployed = excluded.explosive_packs_deployed, objective_interactions = excluded.objective_interactions, ammo_pile_uses = excluded.ammo_pile_uses, incapacitated_seconds = excluded.incapacitated_seconds, ledge_hanging_seconds = excluded.ledge_hanging_seconds, black_white_teammates_restored = excluded.black_white_teammates_restored, car_alarms_triggered = excluded.car_alarms_triggered, revision = excluded.revision",
			columns,
			values
		);
	}
	return true;
}

void LPS_OnPvEStatsFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedPvEStatsCount && g_LPSClosedPvEStats.Length > 0; index++)
	{
		g_LPSClosedPvEStats.Erase(0);
	}

	if (g_LPSClosedPvEStats.Length < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		g_LPSPvEStatsQueueOverflowReported = false;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSFlushCapturedPvEStats[client]
			|| !g_LPSClientPvEStatsActive[client]
			|| !StrEqual(
				g_LPSClientPvEStats[client].segmentId,
				g_LPSFlushCapturedPvEStatsSegmentId[client],
				true
			))
		{
			continue;
		}

		g_LPSClientPvEStatsPersistedRevision[client] = g_LPSFlushCapturedPvEStatsRevision[client];
		if (g_LPSClientPvEStats[client].revision <= g_LPSFlushCapturedPvEStatsRevision[client])
		{
			g_LPSClientPvEStatsDirty[client] = false;
		}
	}
	LPS_ResetPvEStatsFlushCapture();
}

void LPS_ResetPvEStatsFlushCapture()
{
	g_LPSFlushClosedPvEStatsCount = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSFlushCapturedPvEStats[client] = false;
		g_LPSFlushCapturedPvEStatsSegmentId[client][0] = '\0';
		g_LPSFlushCapturedPvEStatsRevision[client] = 0;
	}
}
