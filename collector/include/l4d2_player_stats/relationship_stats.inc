#if defined _l4d2_player_stats_relationship_stats_included
	#endinput
#endif
#define _l4d2_player_stats_relationship_stats_included

#define LPS_RELATIONSHIP_VERSION 1

enum LPSRelationshipStat
{
	LPSRelationshipStat_IncapRevives = 0,
	LPSRelationshipStat_LedgeRescues,
	LPSRelationshipStat_DefibRevives,
	LPSRelationshipStat_SmokerRescues,
	LPSRelationshipStat_HunterRescues,
	LPSRelationshipStat_JockeyRescues,
	LPSRelationshipStat_ChargerRescues,
	LPSRelationshipStat_ControlRescueDurationMs,
	LPSRelationshipStat_MedkitsUsed,
	LPSRelationshipStat_MedkitHealing,
	LPSRelationshipStat_BlackWhiteRestores,
	LPSRelationshipStat_FriendlyFireDamage
};

enum struct LPSRelationshipSnapshot
{
	char roundId[LPS_MAX_ROUND_ID];
	char actorSteamId[LPS_MAX_STEAM_ID];
	char targetSteamId[LPS_MAX_STEAM_ID];
	int incapRevives;
	int ledgeRescues;
	int defibRevives;
	int smokerRescues;
	int hunterRescues;
	int jockeyRescues;
	int chargerRescues;
	int controlRescueDurationMs;
	int medkitsUsed;
	int medkitHealing;
	int blackWhiteRestores;
	int friendlyFireDamage;
	int revision;
	bool dirty;
}

ArrayList g_LPSActiveRelationshipPairs;
ArrayList g_LPSClosedRelationshipPairs;
ArrayList g_LPSFlushRelationshipIndexes;
ArrayList g_LPSFlushRelationshipRevisions;
int g_LPSRelationshipPairIndex[MAXPLAYERS + 1][MAXPLAYERS + 1];
int g_LPSFlushClosedRelationshipCount;
bool g_LPSRelationshipQueueOverflowReported;

void LPS_InitializeRelationships()
{
	g_LPSActiveRelationshipPairs = new ArrayList(sizeof(LPSRelationshipSnapshot));
	g_LPSClosedRelationshipPairs = new ArrayList(sizeof(LPSRelationshipSnapshot));
	g_LPSFlushRelationshipIndexes = new ArrayList();
	g_LPSFlushRelationshipRevisions = new ArrayList();
	g_LPSFlushClosedRelationshipCount = 0;
	g_LPSRelationshipQueueOverflowReported = false;
	LPS_ResetRelationshipPairCache();
}

void LPS_ShutdownRelationships()
{
	// The lifecycle shutdown queues the active Round before this callback.
	LPS_ResetRelationshipFlushCapture();
}

void LPS_OnRelationshipRoundStarted()
{
	if (g_LPSActiveRelationshipPairs.Length > 0)
	{
		LPS_QueueAllActiveRelationshipPairs();
	}
	g_LPSActiveRelationshipPairs.Clear();
	LPS_ResetRelationshipPairCache();
}

void LPS_OnRelationshipRoundEnding()
{
	LPS_QueueAllActiveRelationshipPairs();
	g_LPSActiveRelationshipPairs.Clear();
	LPS_ResetRelationshipPairCache();
}

void LPS_QueueAllActiveRelationshipPairs()
{
	LPSRelationshipSnapshot snapshot;
	for (int index = 0; index < g_LPSActiveRelationshipPairs.Length; index++)
	{
		g_LPSActiveRelationshipPairs.GetArray(index, snapshot, sizeof(snapshot));
		LPS_QueueClosedRelationshipPair(snapshot);
	}
}

void LPS_OnRelationshipClientSlotReset(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}
	for (int peer = 1; peer <= MaxClients; peer++)
	{
		g_LPSRelationshipPairIndex[client][peer] = -1;
		g_LPSRelationshipPairIndex[peer][client] = -1;
	}
}

void LPS_ResetRelationshipPairCache()
{
	for (int actor = 1; actor <= MaxClients; actor++)
	{
		for (int target = 1; target <= MaxClients; target++)
		{
			g_LPSRelationshipPairIndex[actor][target] = -1;
		}
	}
}

bool LPS_IsRelationshipHumanSurvivor(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& !IsFakeClient(client)
		&& GetClientTeam(client) == LPS_TEAM_SURVIVOR
		&& g_LPSClientAuthorized[client]
		&& g_LPSClientSessionActive[client]
		&& g_LPSClientSteamId[client][0] != '\0'
		&& g_LPSClientSegmentActive[client]
		&& StrEqual(g_LPSClientSegmentSide[client], "survivor", true)
		&& StrEqual(g_LPSClientSegmentRoundId[client], g_LPSRoundId, true);
}

int LPS_GetOrCreateRelationshipPair(int actor, int target)
{
	if (!g_LPSRoundActive
		|| !LPS_IsRelationshipHumanSurvivor(actor)
		|| !LPS_IsRelationshipHumanSurvivor(target)
		|| actor == target)
	{
		return -1;
	}

	int cached = g_LPSRelationshipPairIndex[actor][target];
	LPSRelationshipSnapshot snapshot;
	if (cached >= 0 && cached < g_LPSActiveRelationshipPairs.Length)
	{
		g_LPSActiveRelationshipPairs.GetArray(cached, snapshot, sizeof(snapshot));
		if (StrEqual(snapshot.roundId, g_LPSRoundId, true)
			&& StrEqual(snapshot.actorSteamId, g_LPSClientSteamId[actor], true)
			&& StrEqual(snapshot.targetSteamId, g_LPSClientSteamId[target], true))
		{
			return cached;
		}
	}

	// A reconnect clears the slot cache but must resume an already-created
	// SteamID pair instead of creating a second partial absolute snapshot.
	for (int index = 0; index < g_LPSActiveRelationshipPairs.Length; index++)
	{
		g_LPSActiveRelationshipPairs.GetArray(index, snapshot, sizeof(snapshot));
		if (StrEqual(snapshot.roundId, g_LPSRoundId, true)
			&& StrEqual(snapshot.actorSteamId, g_LPSClientSteamId[actor], true)
			&& StrEqual(snapshot.targetSteamId, g_LPSClientSteamId[target], true))
		{
			g_LPSRelationshipPairIndex[actor][target] = index;
			return index;
		}
	}

	strcopy(snapshot.roundId, sizeof(snapshot.roundId), g_LPSRoundId);
	strcopy(snapshot.actorSteamId, sizeof(snapshot.actorSteamId), g_LPSClientSteamId[actor]);
	strcopy(snapshot.targetSteamId, sizeof(snapshot.targetSteamId), g_LPSClientSteamId[target]);
	snapshot.incapRevives = 0;
	snapshot.ledgeRescues = 0;
	snapshot.defibRevives = 0;
	snapshot.smokerRescues = 0;
	snapshot.hunterRescues = 0;
	snapshot.jockeyRescues = 0;
	snapshot.chargerRescues = 0;
	snapshot.controlRescueDurationMs = 0;
	snapshot.medkitsUsed = 0;
	snapshot.medkitHealing = 0;
	snapshot.blackWhiteRestores = 0;
	snapshot.friendlyFireDamage = 0;
	snapshot.revision = 0;
	snapshot.dirty = false;
	int index = g_LPSActiveRelationshipPairs.PushArray(snapshot, sizeof(snapshot));
	g_LPSRelationshipPairIndex[actor][target] = index;
	return index;
}

void LPS_AddRelationshipStat(int actor, int target, LPSRelationshipStat stat, int amount)
{
	if (amount <= 0)
	{
		return;
	}
	int index = LPS_GetOrCreateRelationshipPair(actor, target);
	if (index < 0)
	{
		return;
	}

	LPSRelationshipSnapshot snapshot;
	g_LPSActiveRelationshipPairs.GetArray(index, snapshot, sizeof(snapshot));
	switch (stat)
	{
		case LPSRelationshipStat_IncapRevives: snapshot.incapRevives += amount;
		case LPSRelationshipStat_LedgeRescues: snapshot.ledgeRescues += amount;
		case LPSRelationshipStat_DefibRevives: snapshot.defibRevives += amount;
		case LPSRelationshipStat_SmokerRescues: snapshot.smokerRescues += amount;
		case LPSRelationshipStat_HunterRescues: snapshot.hunterRescues += amount;
		case LPSRelationshipStat_JockeyRescues: snapshot.jockeyRescues += amount;
		case LPSRelationshipStat_ChargerRescues: snapshot.chargerRescues += amount;
		case LPSRelationshipStat_ControlRescueDurationMs: snapshot.controlRescueDurationMs += amount;
		case LPSRelationshipStat_MedkitsUsed: snapshot.medkitsUsed += amount;
		case LPSRelationshipStat_MedkitHealing: snapshot.medkitHealing += amount;
		case LPSRelationshipStat_BlackWhiteRestores: snapshot.blackWhiteRestores += amount;
		case LPSRelationshipStat_FriendlyFireDamage: snapshot.friendlyFireDamage += amount;
	}
	snapshot.revision++;
	snapshot.dirty = true;
	g_LPSActiveRelationshipPairs.SetArray(index, snapshot, sizeof(snapshot));
}

void LPS_RecordRelationshipRevive(int rescuer, int subject, bool ledge)
{
	LPS_AddRelationshipStat(
		rescuer,
		subject,
		ledge ? LPSRelationshipStat_LedgeRescues : LPSRelationshipStat_IncapRevives,
		1
	);
}

void LPS_RecordRelationshipDefib(int rescuer, int subject)
{
	LPS_AddRelationshipStat(rescuer, subject, LPSRelationshipStat_DefibRevives, 1);
}

void LPS_RecordRelationshipMedkit(int healer, int subject, int restored)
{
	LPS_AddRelationshipStat(healer, subject, LPSRelationshipStat_MedkitsUsed, 1);
	LPS_AddRelationshipStat(healer, subject, LPSRelationshipStat_MedkitHealing, restored);
}

void LPS_RecordRelationshipBlackWhiteRestore(int healer, int subject)
{
	LPS_AddRelationshipStat(healer, subject, LPSRelationshipStat_BlackWhiteRestores, 1);
}

void LPS_RecordRelationshipFriendlyFire(int attacker, int victim, int actualDamage)
{
	LPS_AddRelationshipStat(attacker, victim, LPSRelationshipStat_FriendlyFireDamage, actualDamage);
}

void LPS_RecordRelationshipControlRescue(
	int rescuer,
	int victim,
	LPSControlType type,
	int durationMs
)
{
	LPSRelationshipStat stat;
	switch (type)
	{
		case LPSControlType_Smoker: stat = LPSRelationshipStat_SmokerRescues;
		case LPSControlType_Hunter: stat = LPSRelationshipStat_HunterRescues;
		case LPSControlType_Jockey: stat = LPSRelationshipStat_JockeyRescues;
		case LPSControlType_Charger: stat = LPSRelationshipStat_ChargerRescues;
		default: return;
	}
	LPS_AddRelationshipStat(rescuer, victim, stat, 1);
	LPS_AddRelationshipStat(
		rescuer,
		victim,
		LPSRelationshipStat_ControlRescueDurationMs,
		durationMs
	);
}

void LPS_QueueClosedRelationshipPair(LPSRelationshipSnapshot snapshot)
{
	if (g_LPSClosedRelationshipPairs.Length >= g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		if (!g_LPSRelationshipQueueOverflowReported)
		{
			LogError(
				"Closed relationship memory queue reached its limit (%d); additional snapshots will be dropped until a flush succeeds.",
				g_LPSClosedLifecycleQueueLimit.IntValue
			);
			g_LPSRelationshipQueueOverflowReported = true;
		}
		return;
	}
	snapshot.dirty = true;
	g_LPSClosedRelationshipPairs.PushArray(snapshot, sizeof(snapshot));
}

bool LPS_AddRelationshipFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_ResetRelationshipFlushCapture();
	g_LPSFlushClosedRelationshipCount = g_LPSClosedRelationshipPairs.Length;
	LPSRelationshipSnapshot snapshot;
	for (int index = 0; index < g_LPSFlushClosedRelationshipCount; index++)
	{
		g_LPSClosedRelationshipPairs.GetArray(index, snapshot, sizeof(snapshot));
		if (!LPS_AddRelationshipSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int index = 0; index < g_LPSActiveRelationshipPairs.Length; index++)
	{
		g_LPSActiveRelationshipPairs.GetArray(index, snapshot, sizeof(snapshot));
		if (!snapshot.dirty)
		{
			continue;
		}
		if (!LPS_AddRelationshipSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
		g_LPSFlushRelationshipIndexes.Push(index);
		g_LPSFlushRelationshipRevisions.Push(snapshot.revision);
	}
	return true;
}

bool LPS_AddRelationshipSnapshotQuery(
	Transaction transaction,
	LPSRelationshipSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[LPS_MAX_QUERY_LENGTH];
	if (!LPS_BuildRelationshipUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a relationship value exceeded its SQL escape buffer");
		return false;
	}
	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_BuildRelationshipUpsertQuery(LPSRelationshipSnapshot snapshot, char[] query, int maxLength)
{
	char roundId[(LPS_MAX_ROUND_ID * 2) + 1];
	char actorSteamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char targetSteamId[(LPS_MAX_STEAM_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.roundId, roundId, sizeof(roundId))
		|| !g_LPSDatabase.Escape(snapshot.actorSteamId, actorSteamId, sizeof(actorSteamId))
		|| !g_LPSDatabase.Escape(snapshot.targetSteamId, targetSteamId, sizeof(targetSteamId)))
	{
		return false;
	}

	char columns[] = "round_id, actor_steam_id, target_steam_id, relationship_version, incap_revives, ledge_rescues, defib_revives, smoker_rescues, hunter_rescues, jockey_rescues, charger_rescues, control_rescue_duration_ms, medkits_used, medkit_healing, black_white_restores, friendly_fire_damage, last_saved_at, revision";
	char values[1024];
	FormatEx(
		values,
		sizeof(values),
		"'%s', '%s', '%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		roundId,
		actorSteamId,
		targetSteamId,
		LPS_RELATIONSHIP_VERSION,
		snapshot.incapRevives,
		snapshot.ledgeRescues,
		snapshot.defibRevives,
		snapshot.smokerRescues,
		snapshot.hunterRescues,
		snapshot.jockeyRescues,
		snapshot.chargerRescues,
		snapshot.controlRescueDurationMs,
		snapshot.medkitsUsed,
		snapshot.medkitHealing,
		snapshot.blackWhiteRestores,
		snapshot.friendlyFireDamage,
		GetTime(),
		snapshot.revision
	);

	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_player_round_relationship_stats (%s) VALUES (%s) ON DUPLICATE KEY UPDATE relationship_version=VALUES(relationship_version), incap_revives=VALUES(incap_revives), ledge_rescues=VALUES(ledge_rescues), defib_revives=VALUES(defib_revives), smoker_rescues=VALUES(smoker_rescues), hunter_rescues=VALUES(hunter_rescues), jockey_rescues=VALUES(jockey_rescues), charger_rescues=VALUES(charger_rescues), control_rescue_duration_ms=VALUES(control_rescue_duration_ms), medkits_used=VALUES(medkits_used), medkit_healing=VALUES(medkit_healing), black_white_restores=VALUES(black_white_restores), friendly_fire_damage=VALUES(friendly_fire_damage), last_saved_at=VALUES(last_saved_at), revision=VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_player_round_relationship_stats (%s) VALUES (%s) ON CONFLICT(round_id, actor_steam_id, target_steam_id) DO UPDATE SET relationship_version=excluded.relationship_version, incap_revives=excluded.incap_revives, ledge_rescues=excluded.ledge_rescues, defib_revives=excluded.defib_revives, smoker_rescues=excluded.smoker_rescues, hunter_rescues=excluded.hunter_rescues, jockey_rescues=excluded.jockey_rescues, charger_rescues=excluded.charger_rescues, control_rescue_duration_ms=excluded.control_rescue_duration_ms, medkits_used=excluded.medkits_used, medkit_healing=excluded.medkit_healing, black_white_restores=excluded.black_white_restores, friendly_fire_damage=excluded.friendly_fire_damage, last_saved_at=excluded.last_saved_at, revision=excluded.revision",
			columns,
			values
		);
	}
	return true;
}

void LPS_OnRelationshipFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedRelationshipCount
		&& g_LPSClosedRelationshipPairs.Length > 0; index++)
	{
		g_LPSClosedRelationshipPairs.Erase(0);
	}
	if (g_LPSClosedRelationshipPairs.Length < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		g_LPSRelationshipQueueOverflowReported = false;
	}

	LPSRelationshipSnapshot snapshot;
	for (int capture = 0; capture < g_LPSFlushRelationshipIndexes.Length; capture++)
	{
		int index = g_LPSFlushRelationshipIndexes.Get(capture);
		int revision = g_LPSFlushRelationshipRevisions.Get(capture);
		if (index < 0 || index >= g_LPSActiveRelationshipPairs.Length)
		{
			continue;
		}
		g_LPSActiveRelationshipPairs.GetArray(index, snapshot, sizeof(snapshot));
		if (snapshot.revision <= revision)
		{
			snapshot.dirty = false;
			g_LPSActiveRelationshipPairs.SetArray(index, snapshot, sizeof(snapshot));
		}
	}
	LPS_ResetRelationshipFlushCapture();
}

void LPS_ResetRelationshipFlushCapture()
{
	g_LPSFlushClosedRelationshipCount = 0;
	if (g_LPSFlushRelationshipIndexes != null)
	{
		g_LPSFlushRelationshipIndexes.Clear();
	}
	if (g_LPSFlushRelationshipRevisions != null)
	{
		g_LPSFlushRelationshipRevisions.Clear();
	}
}

int LPS_GetPendingClosedRelationshipCount()
{
	return g_LPSClosedRelationshipPairs == null ? 0 : g_LPSClosedRelationshipPairs.Length;
}
