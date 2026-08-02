#if defined _l4d2_player_stats_segments_included
	#endinput
#endif
#define _l4d2_player_stats_segments_included

enum struct LPSSegmentSnapshot
{
	char segmentId[LPS_MAX_SEGMENT_ID];
	char sessionId[LPS_MAX_SESSION_ID];
	char runId[LPS_MAX_RUN_ID];
	char roundId[LPS_MAX_ROUND_ID];
	char serverKey[LPS_MAX_SERVER_KEY];
	char steamId[LPS_MAX_STEAM_ID];
	char side[LPS_MAX_SIDE_NAME];
	char status[LPS_MAX_LIFECYCLE_STATUS];
	int startedAt;
	int endedAt;
	int activePlaySeconds;
	int revision;
}

ArrayList g_LPSClosedSegments;
int g_LPSSegmentSequence;
bool g_LPSSegmentBatchEnding;

bool g_LPSClientSegmentActive[MAXPLAYERS + 1];
bool g_LPSClientSegmentDirty[MAXPLAYERS + 1];
char g_LPSClientSegmentId[MAXPLAYERS + 1][LPS_MAX_SEGMENT_ID];
char g_LPSClientSegmentSessionId[MAXPLAYERS + 1][LPS_MAX_SESSION_ID];
char g_LPSClientSegmentRunId[MAXPLAYERS + 1][LPS_MAX_RUN_ID];
char g_LPSClientSegmentRoundId[MAXPLAYERS + 1][LPS_MAX_ROUND_ID];
char g_LPSClientSegmentSide[MAXPLAYERS + 1][LPS_MAX_SIDE_NAME];
int g_LPSClientSegmentStartedAt[MAXPLAYERS + 1];
int g_LPSClientSegmentLastClockAt[MAXPLAYERS + 1];
int g_LPSClientSegmentActiveSeconds[MAXPLAYERS + 1];
int g_LPSClientSegmentCurrentRevision[MAXPLAYERS + 1];
int g_LPSClientSegmentPersistedRevision[MAXPLAYERS + 1];

bool g_LPSFlushCapturedSegment[MAXPLAYERS + 1];
char g_LPSFlushCapturedSegmentId[MAXPLAYERS + 1][LPS_MAX_SEGMENT_ID];
int g_LPSFlushCapturedSegmentRevision[MAXPLAYERS + 1];
int g_LPSFlushClosedSegmentCount;

void LPS_InitializeSegments()
{
	g_LPSClosedSegments = new ArrayList(sizeof(LPSSegmentSnapshot));
	g_LPSSegmentSequence = 0;
	g_LPSSegmentBatchEnding = false;
	g_LPSFlushClosedSegmentCount = 0;

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientSegmentState(client);
	}
	LPS_ResetSegmentFlushCapture();
}

void LPS_ReconcileAllClientSegments()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ReconcileClientSegment(client);
	}
}

void LPS_ReconcileClientSegment(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	if (g_LPSClientSegmentActive[client])
	{
		LPS_UpdateClientSegmentClock(client);
	}

	char side[LPS_MAX_SIDE_NAME];
	bool shouldBeActive = LPS_GetClientSegmentSide(client, side, sizeof(side));
	bool referencesCurrentRound = g_LPSClientSegmentActive[client]
		&& StrEqual(g_LPSClientSegmentSessionId[client], g_LPSClientSessionId[client], true)
		&& StrEqual(g_LPSClientSegmentRunId[client], g_LPSRunId, true)
		&& StrEqual(g_LPSClientSegmentRoundId[client], g_LPSRoundId, true);

	if (g_LPSClientSegmentActive[client]
		&& (!shouldBeActive
			|| !referencesCurrentRound
			|| !StrEqual(g_LPSClientSegmentSide[client], side, true)))
	{
		LPS_EndClientSegment(client, "closed");
	}

	if (!g_LPSClientSegmentActive[client] && shouldBeActive)
	{
		LPS_StartClientSegment(client, side);
	}
}

bool LPS_GetClientSegmentSide(int client, char[] side, int maxLength)
{
	side[0] = '\0';
	if (!g_LPSRunActive
		|| !g_LPSRoundActive
		|| !g_LPSClientSessionActive[client]
		|| !LPS_IsClientActivelyPlaying(client))
	{
		return false;
	}

	int team = GetClientTeam(client);
	if (team == LPS_TEAM_SURVIVOR)
	{
		strcopy(side, maxLength, "survivor");
		return true;
	}
	if (team == LPS_TEAM_INFECTED)
	{
		strcopy(side, maxLength, "infected");
		return true;
	}
	return false;
}

void LPS_StartClientSegment(int client, const char[] side)
{
	if (g_LPSClientSegmentActive[client]
		|| !g_LPSRunActive
		|| !g_LPSRoundActive
		|| !g_LPSClientSessionActive[client])
	{
		return;
	}

	g_LPSSegmentSequence++;
	Format(
		g_LPSClientSegmentId[client],
		sizeof(g_LPSClientSegmentId[]),
		"%s:segment:%d",
		g_LPSBootId,
		g_LPSSegmentSequence
	);
	strcopy(g_LPSClientSegmentSessionId[client], sizeof(g_LPSClientSegmentSessionId[]), g_LPSClientSessionId[client]);
	strcopy(g_LPSClientSegmentRunId[client], sizeof(g_LPSClientSegmentRunId[]), g_LPSRunId);
	strcopy(g_LPSClientSegmentRoundId[client], sizeof(g_LPSClientSegmentRoundId[]), g_LPSRoundId);
	strcopy(g_LPSClientSegmentSide[client], sizeof(g_LPSClientSegmentSide[]), side);

	int now = GetTime();
	g_LPSClientSegmentStartedAt[client] = now;
	g_LPSClientSegmentLastClockAt[client] = now;
	g_LPSClientSegmentActiveSeconds[client] = 0;
	g_LPSClientSegmentCurrentRevision[client] = 1;
	g_LPSClientSegmentPersistedRevision[client] = 0;
	g_LPSClientSegmentDirty[client] = true;
	g_LPSClientSegmentActive[client] = true;

	LPS_RequestFlush();
}

void LPS_UpdateClientSegmentClock(int client)
{
	if (!g_LPSClientSegmentActive[client])
	{
		return;
	}

	int now = GetTime();
	int delta = now - g_LPSClientSegmentLastClockAt[client];
	if (delta <= 0)
	{
		return;
	}

	g_LPSClientSegmentActiveSeconds[client] += delta;
	g_LPSClientSegmentLastClockAt[client] = now;
	g_LPSClientSegmentCurrentRevision[client]++;
	g_LPSClientSegmentDirty[client] = true;
}

void LPS_EndClientSegment(int client, const char[] status)
{
	if (client < 1 || client > MaxClients || !g_LPSClientSegmentActive[client])
	{
		return;
	}

	LPS_UpdateClientSegmentClock(client);
	LPSSegmentSnapshot snapshot;
	LPS_FillClientSegmentSnapshot(client, snapshot, true, status);
	LPS_QueueClosedSegment(snapshot);
	LPS_ResetClientSegmentState(client);

	if (!g_LPSLifecycleShuttingDown && !g_LPSSegmentBatchEnding)
	{
		LPS_RequestFlush();
	}
}

void LPS_EndAllActiveSegments(const char[] status)
{
	bool previousBatchState = g_LPSSegmentBatchEnding;
	g_LPSSegmentBatchEnding = true;
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSegmentActive[client])
		{
			LPS_EndClientSegment(client, status);
		}
	}
	g_LPSSegmentBatchEnding = previousBatchState;
}

void LPS_QueueClosedSegment(LPSSegmentSnapshot snapshot)
{
	if (LPS_HasLifecycleQueueCapacity())
	{
		g_LPSClosedSegments.PushArray(snapshot, sizeof(snapshot));
	}
}

void LPS_FillClientSegmentSnapshot(
	int client,
	LPSSegmentSnapshot snapshot,
	bool closed,
	const char[] status
)
{
	strcopy(snapshot.segmentId, sizeof(snapshot.segmentId), g_LPSClientSegmentId[client]);
	strcopy(snapshot.sessionId, sizeof(snapshot.sessionId), g_LPSClientSegmentSessionId[client]);
	strcopy(snapshot.runId, sizeof(snapshot.runId), g_LPSClientSegmentRunId[client]);
	strcopy(snapshot.roundId, sizeof(snapshot.roundId), g_LPSClientSegmentRoundId[client]);
	strcopy(snapshot.serverKey, sizeof(snapshot.serverKey), g_LPSServerKey);
	strcopy(snapshot.steamId, sizeof(snapshot.steamId), g_LPSClientSteamId[client]);
	strcopy(snapshot.side, sizeof(snapshot.side), g_LPSClientSegmentSide[client]);
	strcopy(snapshot.status, sizeof(snapshot.status), closed ? status : "active");
	snapshot.startedAt = g_LPSClientSegmentStartedAt[client];
	snapshot.endedAt = closed ? GetTime() : 0;
	snapshot.activePlaySeconds = g_LPSClientSegmentActiveSeconds[client];
	snapshot.revision = g_LPSClientSegmentCurrentRevision[client] + (closed ? 1 : 0);
}

bool LPS_AddSegmentFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_ResetSegmentFlushCapture();
	LPS_AccumulateAllSegmentClocks();
	g_LPSFlushClosedSegmentCount = g_LPSClosedSegments.Length;

	LPSSegmentSnapshot snapshot;
	for (int index = 0; index < g_LPSFlushClosedSegmentCount; index++)
	{
		g_LPSClosedSegments.GetArray(index, snapshot, sizeof(snapshot));
		if (!LPS_AddSegmentSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientSegmentActive[client] || !g_LPSClientSegmentDirty[client])
		{
			continue;
		}

		LPS_FillClientSegmentSnapshot(client, snapshot, false, "active");
		if (!LPS_AddSegmentSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}

		g_LPSFlushCapturedSegment[client] = true;
		g_LPSFlushCapturedSegmentRevision[client] = snapshot.revision;
		strcopy(g_LPSFlushCapturedSegmentId[client], sizeof(g_LPSFlushCapturedSegmentId[]), snapshot.segmentId);
	}
	return true;
}

bool LPS_AddSegmentSnapshotQuery(
	Transaction transaction,
	LPSSegmentSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[4096];
	if (!LPS_BuildSegmentUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Segment value exceeded its SQL escape buffer");
		return false;
	}
	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_BuildSegmentUpsertQuery(LPSSegmentSnapshot snapshot, char[] query, int maxLength)
{
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	char sessionId[(LPS_MAX_SESSION_ID * 2) + 1];
	char runId[(LPS_MAX_RUN_ID * 2) + 1];
	char roundId[(LPS_MAX_ROUND_ID * 2) + 1];
	char serverKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char steamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char side[(LPS_MAX_SIDE_NAME * 2) + 1];
	char status[(LPS_MAX_LIFECYCLE_STATUS * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId))
		|| !g_LPSDatabase.Escape(snapshot.sessionId, sessionId, sizeof(sessionId))
		|| !g_LPSDatabase.Escape(snapshot.runId, runId, sizeof(runId))
		|| !g_LPSDatabase.Escape(snapshot.roundId, roundId, sizeof(roundId))
		|| !g_LPSDatabase.Escape(snapshot.serverKey, serverKey, sizeof(serverKey))
		|| !g_LPSDatabase.Escape(snapshot.steamId, steamId, sizeof(steamId))
		|| !g_LPSDatabase.Escape(snapshot.side, side, sizeof(side))
		|| !g_LPSDatabase.Escape(snapshot.status, status, sizeof(status)))
	{
		return false;
	}

	char endedAt[32];
	if (snapshot.endedAt > 0) IntToString(snapshot.endedAt, endedAt, sizeof(endedAt));
	else strcopy(endedAt, sizeof(endedAt), "NULL");
	int savedAt = GetTime();

	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(query, maxLength, "INSERT INTO lps_player_segments (segment_id, session_id, run_id, round_id, server_key, steam_id, side, started_at, ended_at, last_saved_at, active_play_seconds, status, revision) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', %d, %s, %d, %d, '%s', %d) ON DUPLICATE KEY UPDATE ended_at = VALUES(ended_at), last_saved_at = VALUES(last_saved_at), active_play_seconds = VALUES(active_play_seconds), status = VALUES(status), revision = VALUES(revision)", segmentId, sessionId, runId, roundId, serverKey, steamId, side, snapshot.startedAt, endedAt, savedAt, snapshot.activePlaySeconds, status, snapshot.revision);
	}
	else
	{
		Format(query, maxLength, "INSERT INTO lps_player_segments (segment_id, session_id, run_id, round_id, server_key, steam_id, side, started_at, ended_at, last_saved_at, active_play_seconds, status, revision) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', %d, %s, %d, %d, '%s', %d) ON CONFLICT(segment_id) DO UPDATE SET ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, active_play_seconds = excluded.active_play_seconds, status = excluded.status, revision = excluded.revision", segmentId, sessionId, runId, roundId, serverKey, steamId, side, snapshot.startedAt, endedAt, savedAt, snapshot.activePlaySeconds, status, snapshot.revision);
	}
	return true;
}

void LPS_OnSegmentFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedSegmentCount && g_LPSClosedSegments.Length > 0; index++)
	{
		g_LPSClosedSegments.Erase(0);
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSFlushCapturedSegment[client]
			|| !g_LPSClientSegmentActive[client]
			|| !StrEqual(g_LPSClientSegmentId[client], g_LPSFlushCapturedSegmentId[client], true))
		{
			continue;
		}

		g_LPSClientSegmentPersistedRevision[client] = g_LPSFlushCapturedSegmentRevision[client];
		if (g_LPSClientSegmentCurrentRevision[client] <= g_LPSFlushCapturedSegmentRevision[client])
		{
			g_LPSClientSegmentDirty[client] = false;
		}
	}
	LPS_ResetSegmentFlushCapture();
}

void LPS_ResetSegmentFlushCapture()
{
	g_LPSFlushClosedSegmentCount = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSFlushCapturedSegment[client] = false;
		g_LPSFlushCapturedSegmentId[client][0] = '\0';
		g_LPSFlushCapturedSegmentRevision[client] = 0;
	}
}

void LPS_AccumulateAllSegmentClocks()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSegmentActive[client])
		{
			LPS_UpdateClientSegmentClock(client);
		}
	}
}

void LPS_ResetClientSegmentState(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}
	g_LPSClientSegmentActive[client] = false;
	g_LPSClientSegmentDirty[client] = false;
	g_LPSClientSegmentId[client][0] = '\0';
	g_LPSClientSegmentSessionId[client][0] = '\0';
	g_LPSClientSegmentRunId[client][0] = '\0';
	g_LPSClientSegmentRoundId[client][0] = '\0';
	g_LPSClientSegmentSide[client][0] = '\0';
	g_LPSClientSegmentStartedAt[client] = 0;
	g_LPSClientSegmentLastClockAt[client] = 0;
	g_LPSClientSegmentActiveSeconds[client] = 0;
	g_LPSClientSegmentCurrentRevision[client] = 0;
	g_LPSClientSegmentPersistedRevision[client] = 0;
}

int LPS_GetActiveSegmentCount()
{
	int count = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSegmentActive[client]) count++;
	}
	return count;
}

int LPS_GetPendingClosedSegmentCount()
{
	return g_LPSClosedSegments == null ? 0 : g_LPSClosedSegments.Length;
}
