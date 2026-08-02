#if defined _l4d2_player_stats_lifecycles_included
	#endinput
#endif
#define _l4d2_player_stats_lifecycles_included

enum struct LPSRunSnapshot
{
	char runId[LPS_MAX_RUN_ID];
	char bootId[LPS_MAX_BOOT_ID];
	char serverKey[LPS_MAX_SERVER_KEY];
	char modeFamily[16];
	char gameMode[LPS_MAX_GAME_MODE];
	char campaignKey[LPS_MAX_CAMPAIGN_KEY];
	char status[LPS_MAX_LIFECYCLE_STATUS];
	int startedAt;
	int endedAt;
	int roundCount;
	int completedRoundCount;
	int failedRoundCount;
	int revision;
}

enum struct LPSRoundSnapshot
{
	char roundId[LPS_MAX_ROUND_ID];
	char runId[LPS_MAX_RUN_ID];
	char serverKey[LPS_MAX_SERVER_KEY];
	char modeFamily[16];
	char mapName[LPS_MAX_MAP_NAME];
	char status[LPS_MAX_LIFECYCLE_STATUS];
	int roundSeq;
	int mapSeq;
	int attemptNo;
	int halfNo;
	int startedAt;
	int endedAt;
	int revision;
}

ArrayList g_LPSClosedRuns;
ArrayList g_LPSClosedRounds;
int g_LPSRunSequence;
int g_LPSRoundSequence;
bool g_LPSLifecycleShuttingDown;
bool g_LPSLifecycleQueueOverflowReported;
LPSContinuationKind g_LPSContinuation;

bool g_LPSRunActive;
bool g_LPSRunDirty;
char g_LPSRunId[LPS_MAX_RUN_ID];
char g_LPSRunModeFamily[16];
char g_LPSRunGameMode[LPS_MAX_GAME_MODE];
char g_LPSRunCampaignKey[LPS_MAX_CAMPAIGN_KEY];
int g_LPSRunStartedAt;
int g_LPSRunRoundCount;
int g_LPSRunCompletedRoundCount;
int g_LPSRunFailedRoundCount;
int g_LPSRunMapSeq;
int g_LPSRunCurrentRevision;

bool g_LPSRoundActive;
bool g_LPSRoundDirty;
char g_LPSRoundId[LPS_MAX_ROUND_ID];
char g_LPSRoundMapName[LPS_MAX_MAP_NAME];
int g_LPSRoundRoundSeq;
int g_LPSRoundMapSeq;
int g_LPSRoundAttemptNo;
int g_LPSRoundHalfNo;
int g_LPSRoundStartedAt;
int g_LPSRoundCurrentRevision;

char g_LPSLastRoundMapName[LPS_MAX_MAP_NAME];
char g_LPSLastRoundStatus[LPS_MAX_LIFECYCLE_STATUS];
int g_LPSLastRoundAttemptNo;
int g_LPSLastRoundHalfNo;
bool g_LPSHasLastClosedRoundSnapshot;
LPSRoundSnapshot g_LPSLastClosedRoundSnapshot;

int g_LPSFlushClosedRunCount;
int g_LPSFlushClosedRoundCount;
bool g_LPSFlushCapturedRun;
bool g_LPSFlushCapturedRound;
char g_LPSFlushRunId[LPS_MAX_RUN_ID];
char g_LPSFlushRoundId[LPS_MAX_ROUND_ID];
int g_LPSFlushRunRevision;
int g_LPSFlushRoundRevision;

void LPS_InitializeLifecycles()
{
	g_LPSClosedRuns = new ArrayList(sizeof(LPSRunSnapshot));
	g_LPSClosedRounds = new ArrayList(sizeof(LPSRoundSnapshot));
	g_LPSRunSequence = 0;
	g_LPSRoundSequence = 0;
	g_LPSLifecycleShuttingDown = false;
	g_LPSLifecycleQueueOverflowReported = false;
	g_LPSContinuation = LPSContinuation_None;
	LPS_ResetRunState();
	LPS_ResetRoundState();
	LPS_ResetLifecycleFlushCapture();

	HookEvent("round_start", LPS_Event_RoundStart, EventHookMode_PostNoCopy);
	HookEvent("round_end", LPS_Event_RoundEnd, EventHookMode_Post);
	HookEvent("map_transition", LPS_Event_MapTransition, EventHookMode_PostNoCopy);
	HookEvent("mission_lost", LPS_Event_MissionLost, EventHookMode_PostNoCopy);
	HookEvent("finale_win", LPS_Event_FinaleWin, EventHookMode_PostNoCopy);
}

bool LPS_CanCollectLifecycles()
{
	return !g_LPSLifecycleShuttingDown
		&& g_LPSEnabled.BoolValue
		&& g_LPSServerKey[0] != '\0'
		&& g_LPSBootId[0] != '\0'
		&& LPS_IsSupportedMode();
}

void LPS_OnCollectorConfigured()
{
	if (LPS_CanCollectLifecycles())
	{
		LPS_EnsureLifecycleStarted(true);
	}
}

void LPS_OnLifecycleModeChanging(
	LPSModeFamily previousFamily,
	const char[] previousMode,
	LPSModeFamily nextFamily,
	const char[] nextMode
)
{
	if (previousFamily == LPSModeFamily_Unsupported)
	{
		return;
	}

	if (previousFamily != nextFamily || !StrEqual(previousMode, nextMode, true))
	{
		LPS_EndAllActiveSegments("abandoned");
		LPS_EndRun("abandoned");
	}
}

void LPS_OnLifecycleModeChanged(
	LPSModeFamily previousFamily,
	const char[] previousMode,
	LPSModeFamily nextFamily,
	const char[] nextMode
)
{
	if (nextFamily == LPSModeFamily_Unsupported)
	{
		return;
	}

	if (previousFamily != nextFamily || !StrEqual(previousMode, nextMode, true))
	{
		LPS_EnsureLifecycleStarted(true);
	}
}

void LPS_OnLifecycleMapStart()
{
	if (!LPS_CanCollectLifecycles())
	{
		return;
	}

	char mapName[LPS_MAX_MAP_NAME];
	GetCurrentMap(mapName, sizeof(mapName));

	if (g_LPSRunActive
		&& (g_LPSContinuation == LPSContinuation_Retry
			|| g_LPSContinuation == LPSContinuation_VersusHalf)
		&& g_LPSLastRoundMapName[0] != '\0'
		&& !StrEqual(mapName, g_LPSLastRoundMapName, true))
	{
		LPS_EndRun("abandoned");
	}

	if (g_LPSRoundActive && !StrEqual(mapName, g_LPSRoundMapName, true))
	{
		LPS_EndRun("abandoned");
	}

	LPS_EnsureLifecycleStarted(false);
}

void LPS_OnLifecycleMapEnd()
{
	if (!g_LPSRunActive)
	{
		return;
	}

	if (!g_LPSRoundActive
		&& (g_LPSContinuation == LPSContinuation_MapTransition
			|| g_LPSContinuation == LPSContinuation_Retry))
	{
		return;
	}

	LPS_EndRun("abandoned");
}

public void LPS_Event_RoundStart(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanCollectLifecycles())
	{
		return;
	}

	LPS_EnsureLifecycleStarted(false);
	if (!g_LPSRoundActive)
	{
		LPS_StartRound();
	}
}

public void LPS_Event_RoundEnd(Event event, const char[] name, bool dontBroadcast)
{
	if (g_LPSModeFamily != LPSModeFamily_Versus || !g_LPSRoundActive)
	{
		return;
	}

	LPS_EndRound("completed");
	g_LPSContinuation = LPSContinuation_VersusHalf;
	LPS_RequestFlush();
}

public void LPS_Event_MapTransition(Event event, const char[] name, bool dontBroadcast)
{
	if (!g_LPSRunActive)
	{
		return;
	}

	if (g_LPSRoundActive)
	{
		LPS_EndRound("completed");
	}

	g_LPSContinuation = LPSContinuation_MapTransition;
	LPS_RequestFlush();
}

public void LPS_Event_MissionLost(Event event, const char[] name, bool dontBroadcast)
{
	if (g_LPSModeFamily != LPSModeFamily_PvE || !g_LPSRunActive)
	{
		return;
	}

	if (g_LPSRoundActive)
	{
		LPS_EndRound("failed");
	}

	g_LPSContinuation = LPSContinuation_Retry;
	LPS_RequestFlush();
}

public void LPS_Event_FinaleWin(Event event, const char[] name, bool dontBroadcast)
{
	if (!g_LPSRunActive)
	{
		return;
	}

	if (g_LPSRoundActive)
	{
		LPS_EndRound("completed");
	}

	LPS_EndRun("completed");
	LPS_RequestFlush();
}

void LPS_EnsureLifecycleStarted(bool startRound)
{
	if (!LPS_CanCollectLifecycles())
	{
		return;
	}

	if (!g_LPSRunActive)
	{
		LPS_StartRun();
	}

	if (startRound && !g_LPSRoundActive)
	{
		LPS_StartRound();
	}
}

void LPS_StartRun()
{
	if (!LPS_CanCollectLifecycles() || g_LPSRunActive)
	{
		return;
	}

	g_LPSRunSequence++;
	Format(g_LPSRunId, sizeof(g_LPSRunId), "%s:run:%d", g_LPSBootId, g_LPSRunSequence);
	LPS_GetModeFamilyName(g_LPSRunModeFamily, sizeof(g_LPSRunModeFamily));
	strcopy(g_LPSRunGameMode, sizeof(g_LPSRunGameMode), g_LPSGameMode);

	char mapName[LPS_MAX_MAP_NAME];
	GetCurrentMap(mapName, sizeof(mapName));
	LPS_BuildCampaignKey(mapName, g_LPSRunCampaignKey, sizeof(g_LPSRunCampaignKey));

	g_LPSRunStartedAt = GetTime();
	g_LPSRunRoundCount = 0;
	g_LPSRunCompletedRoundCount = 0;
	g_LPSRunFailedRoundCount = 0;
	g_LPSRunMapSeq = 0;
	g_LPSRunCurrentRevision = 1;
	g_LPSRunDirty = true;
	g_LPSRunActive = true;
	g_LPSContinuation = LPSContinuation_None;
	g_LPSLastRoundMapName[0] = '\0';
	g_LPSLastRoundStatus[0] = '\0';
	g_LPSLastRoundAttemptNo = 0;
	g_LPSLastRoundHalfNo = 0;

	LPS_RequestFlush();
}

void LPS_StartRound()
{
	if (!g_LPSRunActive || g_LPSRoundActive)
	{
		return;
	}

	char mapName[LPS_MAX_MAP_NAME];
	GetCurrentMap(mapName, sizeof(mapName));
	int halfNo = g_LPSModeFamily == LPSModeFamily_Versus ? LPS_GetVersusHalfNo() : 0;
	if (g_LPSModeFamily == LPSModeFamily_Versus
		&& g_LPSContinuation == LPSContinuation_VersusHalf
		&& g_LPSLastRoundMapName[0] != '\0'
		&& StrEqual(mapName, g_LPSLastRoundMapName, true)
		&& halfNo == g_LPSLastRoundHalfNo)
	{
		LPS_ReclassifyLastVersusRoundAsAbandoned();
	}

	if (g_LPSLastRoundMapName[0] == '\0')
	{
		g_LPSRunMapSeq = 1;
	}
	else if (!StrEqual(mapName, g_LPSLastRoundMapName, true))
	{
		g_LPSRunMapSeq++;
	}

	int attemptNo = 1;
	if (g_LPSLastRoundMapName[0] != '\0'
		&& StrEqual(mapName, g_LPSLastRoundMapName, true)
		&& halfNo == g_LPSLastRoundHalfNo)
	{
		attemptNo = g_LPSLastRoundAttemptNo + 1;
	}

	g_LPSRoundSequence++;
	g_LPSRunRoundCount++;
	Format(g_LPSRoundId, sizeof(g_LPSRoundId), "%s:round:%d", g_LPSBootId, g_LPSRoundSequence);
	strcopy(g_LPSRoundMapName, sizeof(g_LPSRoundMapName), mapName);
	g_LPSRoundRoundSeq = g_LPSRunRoundCount;
	g_LPSRoundMapSeq = g_LPSRunMapSeq;
	g_LPSRoundAttemptNo = attemptNo;
	g_LPSRoundHalfNo = halfNo;
	g_LPSRoundStartedAt = GetTime();
	g_LPSRoundCurrentRevision = 1;
	g_LPSRoundDirty = true;
	g_LPSRoundActive = true;
	g_LPSRunCurrentRevision++;
	g_LPSRunDirty = true;
	g_LPSContinuation = LPSContinuation_None;

	LPS_ReconcileAllClientSegments();
	LPS_RequestFlush();
}

int LPS_GetVersusHalfNo()
{
	return GameRules_GetProp("m_bInSecondHalfOfRound", 1) != 0 ? 2 : 1;
}

void LPS_EndRound(const char[] status)
{
	if (!g_LPSRoundActive)
	{
		return;
	}

	LPS_EndAllActiveSegments(StrEqual(status, "abandoned", true) ? "abandoned" : "closed");

	LPSRoundSnapshot snapshot;
	LPS_FillRoundSnapshot(snapshot, true, status);
	LPS_QueueClosedRound(snapshot);
	g_LPSLastClosedRoundSnapshot = snapshot;
	g_LPSHasLastClosedRoundSnapshot = true;

	strcopy(g_LPSLastRoundMapName, sizeof(g_LPSLastRoundMapName), g_LPSRoundMapName);
	strcopy(g_LPSLastRoundStatus, sizeof(g_LPSLastRoundStatus), status);
	g_LPSLastRoundAttemptNo = g_LPSRoundAttemptNo;
	g_LPSLastRoundHalfNo = g_LPSRoundHalfNo;

	if (StrEqual(status, "completed", true))
	{
		g_LPSRunCompletedRoundCount++;
	}
	else if (StrEqual(status, "failed", true))
	{
		g_LPSRunFailedRoundCount++;
	}

	g_LPSRunCurrentRevision++;
	g_LPSRunDirty = true;
	LPS_ResetRoundState();

	if (!g_LPSLifecycleShuttingDown)
	{
		LPS_RequestFlush();
	}
}

void LPS_ReclassifyLastVersusRoundAsAbandoned()
{
	if (!g_LPSHasLastClosedRoundSnapshot
		|| !StrEqual(g_LPSLastClosedRoundSnapshot.status, "completed", true))
	{
		return;
	}

	strcopy(
		g_LPSLastClosedRoundSnapshot.status,
		sizeof(g_LPSLastClosedRoundSnapshot.status),
		"abandoned"
	);
	g_LPSLastClosedRoundSnapshot.revision++;
	LPS_QueueClosedRound(g_LPSLastClosedRoundSnapshot);
	strcopy(g_LPSLastRoundStatus, sizeof(g_LPSLastRoundStatus), "abandoned");
	if (g_LPSRunCompletedRoundCount > 0)
	{
		g_LPSRunCompletedRoundCount--;
	}
	g_LPSRunCurrentRevision++;
	g_LPSRunDirty = true;
}

void LPS_EndRun(const char[] status)
{
	if (!g_LPSRunActive)
	{
		return;
	}

	if (g_LPSRoundActive)
	{
		LPS_EndRound("abandoned");
	}

	LPSRunSnapshot snapshot;
	LPS_FillRunSnapshot(snapshot, true, status);
	LPS_QueueClosedRun(snapshot);
	LPS_ResetRunState();
	g_LPSContinuation = LPSContinuation_None;

	if (!g_LPSLifecycleShuttingDown)
	{
		LPS_RequestFlush();
	}
}

void LPS_BuildCampaignKey(const char[] mapName, char[] campaignKey, int maxLength)
{
	campaignKey[0] = '\0';
	if (mapName[0] != 'c' && mapName[0] != 'C')
	{
		return;
	}

	int index = 1;
	while (mapName[index] >= '0' && mapName[index] <= '9')
	{
		index++;
	}

	if (index <= 1 || (mapName[index] != 'm' && mapName[index] != 'M') || index >= maxLength)
	{
		return;
	}

	for (int copyIndex = 0; copyIndex < index; copyIndex++)
	{
		campaignKey[copyIndex] = mapName[copyIndex];
	}
	campaignKey[index] = '\0';
}

void LPS_FillRunSnapshot(LPSRunSnapshot snapshot, bool closed, const char[] status)
{
	strcopy(snapshot.runId, sizeof(snapshot.runId), g_LPSRunId);
	strcopy(snapshot.bootId, sizeof(snapshot.bootId), g_LPSBootId);
	strcopy(snapshot.serverKey, sizeof(snapshot.serverKey), g_LPSServerKey);
	strcopy(snapshot.modeFamily, sizeof(snapshot.modeFamily), g_LPSRunModeFamily);
	strcopy(snapshot.gameMode, sizeof(snapshot.gameMode), g_LPSRunGameMode);
	strcopy(snapshot.campaignKey, sizeof(snapshot.campaignKey), g_LPSRunCampaignKey);
	strcopy(snapshot.status, sizeof(snapshot.status), closed ? status : "active");
	snapshot.startedAt = g_LPSRunStartedAt;
	snapshot.endedAt = closed ? GetTime() : 0;
	snapshot.roundCount = g_LPSRunRoundCount;
	snapshot.completedRoundCount = g_LPSRunCompletedRoundCount;
	snapshot.failedRoundCount = g_LPSRunFailedRoundCount;
	snapshot.revision = g_LPSRunCurrentRevision + (closed ? 1 : 0);
}

void LPS_FillRoundSnapshot(LPSRoundSnapshot snapshot, bool closed, const char[] status)
{
	strcopy(snapshot.roundId, sizeof(snapshot.roundId), g_LPSRoundId);
	strcopy(snapshot.runId, sizeof(snapshot.runId), g_LPSRunId);
	strcopy(snapshot.serverKey, sizeof(snapshot.serverKey), g_LPSServerKey);
	strcopy(snapshot.modeFamily, sizeof(snapshot.modeFamily), g_LPSRunModeFamily);
	strcopy(snapshot.mapName, sizeof(snapshot.mapName), g_LPSRoundMapName);
	strcopy(snapshot.status, sizeof(snapshot.status), closed ? status : "active");
	snapshot.roundSeq = g_LPSRoundRoundSeq;
	snapshot.mapSeq = g_LPSRoundMapSeq;
	snapshot.attemptNo = g_LPSRoundAttemptNo;
	snapshot.halfNo = g_LPSRoundHalfNo;
	snapshot.startedAt = g_LPSRoundStartedAt;
	snapshot.endedAt = closed ? GetTime() : 0;
	snapshot.revision = g_LPSRoundCurrentRevision + (closed ? 1 : 0);
}

bool LPS_HasLifecycleQueueCapacity()
{
	int total = g_LPSClosedRuns.Length + g_LPSClosedRounds.Length + LPS_GetPendingClosedSegmentCount();
	if (total < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		return true;
	}

	if (!g_LPSLifecycleQueueOverflowReported)
	{
		LogError(
			"Closed lifecycle memory queue reached its limit (%d); additional snapshots will be dropped until a flush succeeds.",
			g_LPSClosedLifecycleQueueLimit.IntValue
		);
		g_LPSLifecycleQueueOverflowReported = true;
	}
	return false;
}

void LPS_QueueClosedRun(LPSRunSnapshot snapshot)
{
	if (LPS_HasLifecycleQueueCapacity())
	{
		g_LPSClosedRuns.PushArray(snapshot, sizeof(snapshot));
	}
}

void LPS_QueueClosedRound(LPSRoundSnapshot snapshot)
{
	if (LPS_HasLifecycleQueueCapacity())
	{
		g_LPSClosedRounds.PushArray(snapshot, sizeof(snapshot));
	}
}

bool LPS_AddLifecycleFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_ResetLifecycleFlushCapture();
	g_LPSFlushClosedRunCount = g_LPSClosedRuns.Length;
	g_LPSFlushClosedRoundCount = g_LPSClosedRounds.Length;

	LPSRunSnapshot runSnapshot;
	for (int index = 0; index < g_LPSFlushClosedRunCount; index++)
	{
		g_LPSClosedRuns.GetArray(index, runSnapshot, sizeof(runSnapshot));
		if (!LPS_AddRunSnapshotQuery(transaction, runSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	LPSRoundSnapshot roundSnapshot;
	for (int index = 0; index < g_LPSFlushClosedRoundCount; index++)
	{
		g_LPSClosedRounds.GetArray(index, roundSnapshot, sizeof(roundSnapshot));
		if (!LPS_AddRoundSnapshotQuery(transaction, roundSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	if (g_LPSRunActive && g_LPSRunDirty)
	{
		LPS_FillRunSnapshot(runSnapshot, false, "active");
		if (!LPS_AddRunSnapshotQuery(transaction, runSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
		g_LPSFlushCapturedRun = true;
		g_LPSFlushRunRevision = runSnapshot.revision;
		strcopy(g_LPSFlushRunId, sizeof(g_LPSFlushRunId), runSnapshot.runId);
	}

	if (g_LPSRoundActive && g_LPSRoundDirty)
	{
		LPS_FillRoundSnapshot(roundSnapshot, false, "active");
		if (!LPS_AddRoundSnapshotQuery(transaction, roundSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
		g_LPSFlushCapturedRound = true;
		g_LPSFlushRoundRevision = roundSnapshot.revision;
		strcopy(g_LPSFlushRoundId, sizeof(g_LPSFlushRoundId), roundSnapshot.roundId);
	}

	return true;
}

bool LPS_AddRunSnapshotQuery(
	Transaction transaction,
	LPSRunSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[4096];
	if (!LPS_BuildRunUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Run value exceeded its SQL escape buffer");
		return false;
	}
	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_AddRoundSnapshotQuery(
	Transaction transaction,
	LPSRoundSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[4096];
	if (!LPS_BuildRoundUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Round value exceeded its SQL escape buffer");
		return false;
	}
	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_BuildRunUpsertQuery(LPSRunSnapshot snapshot, char[] query, int maxLength)
{
	char runId[(LPS_MAX_RUN_ID * 2) + 1];
	char bootId[(LPS_MAX_BOOT_ID * 2) + 1];
	char serverKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char modeFamily[33];
	char gameMode[(LPS_MAX_GAME_MODE * 2) + 1];
	char campaignKey[(LPS_MAX_CAMPAIGN_KEY * 2) + 1];
	char status[(LPS_MAX_LIFECYCLE_STATUS * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.runId, runId, sizeof(runId))
		|| !g_LPSDatabase.Escape(snapshot.bootId, bootId, sizeof(bootId))
		|| !g_LPSDatabase.Escape(snapshot.serverKey, serverKey, sizeof(serverKey))
		|| !g_LPSDatabase.Escape(snapshot.modeFamily, modeFamily, sizeof(modeFamily))
		|| !g_LPSDatabase.Escape(snapshot.gameMode, gameMode, sizeof(gameMode))
		|| !g_LPSDatabase.Escape(snapshot.campaignKey, campaignKey, sizeof(campaignKey))
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
		Format(query, maxLength, "INSERT INTO lps_runs (run_id, boot_id, server_key, mode_family, game_mode, campaign_key, started_at, ended_at, last_saved_at, status, round_count, completed_round_count, failed_round_count, revision) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', %d, %s, %d, '%s', %d, %d, %d, %d) ON DUPLICATE KEY UPDATE ended_at = VALUES(ended_at), last_saved_at = VALUES(last_saved_at), status = VALUES(status), round_count = VALUES(round_count), completed_round_count = VALUES(completed_round_count), failed_round_count = VALUES(failed_round_count), revision = VALUES(revision)", runId, bootId, serverKey, modeFamily, gameMode, campaignKey, snapshot.startedAt, endedAt, savedAt, status, snapshot.roundCount, snapshot.completedRoundCount, snapshot.failedRoundCount, snapshot.revision);
	}
	else
	{
		Format(query, maxLength, "INSERT INTO lps_runs (run_id, boot_id, server_key, mode_family, game_mode, campaign_key, started_at, ended_at, last_saved_at, status, round_count, completed_round_count, failed_round_count, revision) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', %d, %s, %d, '%s', %d, %d, %d, %d) ON CONFLICT(run_id) DO UPDATE SET ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, status = excluded.status, round_count = excluded.round_count, completed_round_count = excluded.completed_round_count, failed_round_count = excluded.failed_round_count, revision = excluded.revision", runId, bootId, serverKey, modeFamily, gameMode, campaignKey, snapshot.startedAt, endedAt, savedAt, status, snapshot.roundCount, snapshot.completedRoundCount, snapshot.failedRoundCount, snapshot.revision);
	}
	return true;
}

bool LPS_BuildRoundUpsertQuery(LPSRoundSnapshot snapshot, char[] query, int maxLength)
{
	char roundId[(LPS_MAX_ROUND_ID * 2) + 1];
	char runId[(LPS_MAX_RUN_ID * 2) + 1];
	char serverKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char modeFamily[33];
	char mapName[(LPS_MAX_MAP_NAME * 2) + 1];
	char status[(LPS_MAX_LIFECYCLE_STATUS * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.roundId, roundId, sizeof(roundId))
		|| !g_LPSDatabase.Escape(snapshot.runId, runId, sizeof(runId))
		|| !g_LPSDatabase.Escape(snapshot.serverKey, serverKey, sizeof(serverKey))
		|| !g_LPSDatabase.Escape(snapshot.modeFamily, modeFamily, sizeof(modeFamily))
		|| !g_LPSDatabase.Escape(snapshot.mapName, mapName, sizeof(mapName))
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
		Format(query, maxLength, "INSERT INTO lps_rounds (round_id, run_id, server_key, mode_family, map_name, round_seq, map_seq, attempt_no, half_no, started_at, ended_at, last_saved_at, status, revision) VALUES ('%s', '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d, %s, %d, '%s', %d) ON DUPLICATE KEY UPDATE ended_at = VALUES(ended_at), last_saved_at = VALUES(last_saved_at), status = VALUES(status), revision = VALUES(revision)", roundId, runId, serverKey, modeFamily, mapName, snapshot.roundSeq, snapshot.mapSeq, snapshot.attemptNo, snapshot.halfNo, snapshot.startedAt, endedAt, savedAt, status, snapshot.revision);
	}
	else
	{
		Format(query, maxLength, "INSERT INTO lps_rounds (round_id, run_id, server_key, mode_family, map_name, round_seq, map_seq, attempt_no, half_no, started_at, ended_at, last_saved_at, status, revision) VALUES ('%s', '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d, %s, %d, '%s', %d) ON CONFLICT(round_id) DO UPDATE SET ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, status = excluded.status, revision = excluded.revision", roundId, runId, serverKey, modeFamily, mapName, snapshot.roundSeq, snapshot.mapSeq, snapshot.attemptNo, snapshot.halfNo, snapshot.startedAt, endedAt, savedAt, status, snapshot.revision);
	}
	return true;
}

void LPS_OnLifecycleFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedRunCount && g_LPSClosedRuns.Length > 0; index++)
	{
		g_LPSClosedRuns.Erase(0);
	}
	for (int index = 0; index < g_LPSFlushClosedRoundCount && g_LPSClosedRounds.Length > 0; index++)
	{
		g_LPSClosedRounds.Erase(0);
	}

	if (g_LPSFlushCapturedRun && g_LPSRunActive && StrEqual(g_LPSRunId, g_LPSFlushRunId, true))
	{
		if (g_LPSRunCurrentRevision <= g_LPSFlushRunRevision) g_LPSRunDirty = false;
	}
	if (g_LPSFlushCapturedRound && g_LPSRoundActive && StrEqual(g_LPSRoundId, g_LPSFlushRoundId, true))
	{
		if (g_LPSRoundCurrentRevision <= g_LPSFlushRoundRevision) g_LPSRoundDirty = false;
	}

	if (LPS_GetPendingClosedLifecycleCount() < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		g_LPSLifecycleQueueOverflowReported = false;
	}
	LPS_ResetLifecycleFlushCapture();
}

void LPS_ResetLifecycleFlushCapture()
{
	g_LPSFlushClosedRunCount = 0;
	g_LPSFlushClosedRoundCount = 0;
	g_LPSFlushCapturedRun = false;
	g_LPSFlushCapturedRound = false;
	g_LPSFlushRunId[0] = '\0';
	g_LPSFlushRoundId[0] = '\0';
	g_LPSFlushRunRevision = 0;
	g_LPSFlushRoundRevision = 0;
}

void LPS_ResetRunState()
{
	g_LPSRunActive = false;
	g_LPSRunDirty = false;
	g_LPSRunId[0] = '\0';
	g_LPSRunModeFamily[0] = '\0';
	g_LPSRunGameMode[0] = '\0';
	g_LPSRunCampaignKey[0] = '\0';
	g_LPSRunStartedAt = 0;
	g_LPSRunRoundCount = 0;
	g_LPSRunCompletedRoundCount = 0;
	g_LPSRunFailedRoundCount = 0;
	g_LPSRunMapSeq = 0;
	g_LPSRunCurrentRevision = 0;
	g_LPSHasLastClosedRoundSnapshot = false;
}

void LPS_ResetRoundState()
{
	g_LPSRoundActive = false;
	g_LPSRoundDirty = false;
	g_LPSRoundId[0] = '\0';
	g_LPSRoundMapName[0] = '\0';
	g_LPSRoundRoundSeq = 0;
	g_LPSRoundMapSeq = 0;
	g_LPSRoundAttemptNo = 0;
	g_LPSRoundHalfNo = 0;
	g_LPSRoundStartedAt = 0;
	g_LPSRoundCurrentRevision = 0;
}

int LPS_GetPendingClosedLifecycleCount()
{
	int runCount = g_LPSClosedRuns == null ? 0 : g_LPSClosedRuns.Length;
	int roundCount = g_LPSClosedRounds == null ? 0 : g_LPSClosedRounds.Length;
	return runCount + roundCount + LPS_GetPendingClosedSegmentCount();
}

void LPS_OnLifecycleDatabaseReady()
{
	LPS_EnsureLifecycleStarted(true);
	LPS_RequestFlush();
}

void LPS_ShutdownLifecycles()
{
	g_LPSLifecycleShuttingDown = true;
	LPS_EndAllActiveSegments("abandoned");
	LPS_EndRun("abandoned");
}
