#if defined _l4d2_player_stats_sessions_included
	#endinput
#endif
#define _l4d2_player_stats_sessions_included

enum struct LPSSessionSnapshot
{
	char sessionId[LPS_MAX_SESSION_ID];
	char bootId[LPS_MAX_BOOT_ID];
	char serverKey[LPS_MAX_SERVER_KEY];
	char steamId[LPS_MAX_STEAM_ID];
	char playerName[LPS_MAX_PLAYER_NAME];
	char latestName[LPS_MAX_PLAYER_NAME];
	char ipAddress[LPS_MAX_IP_ADDRESS];
	char status[LPS_MAX_SESSION_STATUS];
	char disconnectReason[LPS_MAX_DISCONNECT_REASON];
	int startedAt;
	int endedAt;
	int connectedSeconds;
	int activePlaySeconds;
	int revision;
}

enum struct LPSSuspendedSession
{
	char sessionId[LPS_MAX_SESSION_ID];
	char bootId[LPS_MAX_BOOT_ID];
	char serverKey[LPS_MAX_SERVER_KEY];
	char steamId[LPS_MAX_STEAM_ID];
	char playerName[LPS_MAX_PLAYER_NAME];
	char ipAddress[LPS_MAX_IP_ADDRESS];
	int startedAt;
	int connectedSeconds;
	int activePlaySeconds;
	int currentRevision;
	int persistedRevision;
	int suspendedAt;
}

ArrayList g_LPSClosedSessions;
ArrayList g_LPSSuspendedSessions;
Handle g_LPSSessionClockTimer;
int g_LPSSessionSequence;
bool g_LPSSessionQueueOverflowReported;
bool g_LPSSessionShuttingDown;
bool g_LPSSessionMapEnding;

bool g_LPSClientAuthorized[MAXPLAYERS + 1];
bool g_LPSClientSessionActive[MAXPLAYERS + 1];
bool g_LPSClientSessionDirty[MAXPLAYERS + 1];
bool g_LPSClientWasActive[MAXPLAYERS + 1];
char g_LPSClientSteamId[MAXPLAYERS + 1][LPS_MAX_STEAM_ID];
char g_LPSClientSessionId[MAXPLAYERS + 1][LPS_MAX_SESSION_ID];
char g_LPSClientSessionName[MAXPLAYERS + 1][LPS_MAX_PLAYER_NAME];
char g_LPSClientIpAddress[MAXPLAYERS + 1][LPS_MAX_IP_ADDRESS];
int g_LPSClientSessionStartedAt[MAXPLAYERS + 1];
int g_LPSClientLastClockAt[MAXPLAYERS + 1];
int g_LPSClientConnectedSeconds[MAXPLAYERS + 1];
int g_LPSClientActivePlaySeconds[MAXPLAYERS + 1];
int g_LPSClientCurrentRevision[MAXPLAYERS + 1];
int g_LPSClientPersistedRevision[MAXPLAYERS + 1];

bool g_LPSFlushCapturedClient[MAXPLAYERS + 1];
char g_LPSFlushCapturedSessionId[MAXPLAYERS + 1][LPS_MAX_SESSION_ID];
int g_LPSFlushCapturedRevision[MAXPLAYERS + 1];
int g_LPSFlushClosedSessionCount;

void LPS_InitializeSessions()
{
	g_LPSClosedSessions = new ArrayList(sizeof(LPSSessionSnapshot));
	g_LPSSuspendedSessions = new ArrayList(sizeof(LPSSuspendedSession));
	g_LPSSessionSequence = 0;
	g_LPSSessionQueueOverflowReported = false;
	g_LPSSessionShuttingDown = false;
	g_LPSSessionMapEnding = false;
	g_LPSFlushClosedSessionCount = 0;

	HookEvent("player_team", LPS_Event_PlayerTeam, EventHookMode_Post);
	HookEvent("player_bot_replace", LPS_Event_PlayerBotReplace, EventHookMode_Post);
	HookEvent("bot_player_replace", LPS_Event_BotPlayerReplace, EventHookMode_Post);

	g_LPSSessionClockTimer = CreateTimer(1.0, LPS_Timer_SessionClock, _, TIMER_REPEAT);

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientSessionState(client, true);

		if (IsClientConnected(client) && !IsFakeClient(client))
		{
			LPS_TryCacheClientSteamId(client);
		}
	}
}

public void OnClientConnected(int client)
{
	LPS_OnRelationshipClientSlotReset(client);
	LPS_ResetSurvivorTechniqueClient(client);
	LPS_ClearAssistContributor(client);
	LPS_ResetAssistTarget(client);
	LPS_ClearBossContributionsForClient(client);
	LPS_ResetStatsDamageEntity(client);
	LPS_ResetClientSegmentState(client);
	LPS_ResetClientSessionState(client, true);
}

public void OnClientAuthorized(int client, const char[] auth)
{
	if (client < 1 || client > MaxClients || StrEqual(auth, "BOT", false))
	{
		return;
	}

	LPS_TryCacheClientSteamId(client);
	if (!LPS_TryResumeClientSession(client))
	{
		LPS_TryStartClientSession(client);
	}
}

public void OnClientPutInServer(int client)
{
	LPS_HookStatsClient(client);

	if (IsFakeClient(client))
	{
		return;
	}

	if (!g_LPSClientAuthorized[client])
	{
		LPS_TryCacheClientSteamId(client);
	}

	if (!LPS_TryResumeClientSession(client))
	{
		LPS_TryStartClientSession(client);
	}

	if (g_LPSClientSessionActive[client])
	{
		g_LPSClientWasActive[client] = LPS_IsClientActivelyPlaying(client);
		LPS_ReconcileClientSegment(client);
	}
}

public void OnClientDisconnect(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	LPS_OnRelationshipClientSlotReset(client);
	if (g_LPSClientSessionActive[client])
	{
		if (g_LPSSessionMapEnding)
		{
			LPS_SuspendClientSession(client);
		}
		else
		{
			LPS_EndClientSession(client, "client_disconnect");
		}
	}

	LPS_ResetClientSegmentState(client);
	LPS_ResetClientSessionState(client, true);
	LPS_ResetStatsDamageEntity(client);
}

public Action LPS_Timer_SessionClock(Handle timer)
{
	int now = GetTime();
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSessionActive[client])
		{
			LPS_UpdateClientSessionClock(client, true);
			LPS_ReconcileClientSegment(client);
		}
		else if (g_LPSClientAuthorized[client])
		{
			if (!LPS_TryResumeClientSession(client))
			{
				LPS_TryStartClientSession(client);
			}
		}
	}

	LPS_ExpireSuspendedSessions(now);

	return Plugin_Continue;
}

void LPS_OnSessionsMapStart()
{
	g_LPSSessionMapEnding = false;
}

void LPS_OnSessionsMapEnd()
{
	g_LPSSessionMapEnding = true;
}

void LPS_OnSessionsMapTransition()
{
	// listen servers can disconnect the host before SourceMod emits OnMapEnd.
	// The gameplay event is therefore the earliest reliable transfer boundary.
	g_LPSSessionMapEnding = true;
}

public void LPS_Event_PlayerTeam(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (client >= 1 && client <= MaxClients && g_LPSClientSessionActive[client])
	{
		LPS_UpdateClientSessionClock(client, true);
		LPS_ReconcileClientSegment(client);
	}
}

public void LPS_Event_PlayerBotReplace(Event event, const char[] name, bool dontBroadcast)
{
	LPS_UpdateSessionClockFromUserId(event.GetInt("player"));
}

public void LPS_Event_BotPlayerReplace(Event event, const char[] name, bool dontBroadcast)
{
	LPS_UpdateSessionClockFromUserId(event.GetInt("player"));
}

void LPS_UpdateSessionClockFromUserId(int userId)
{
	int client = GetClientOfUserId(userId);
	if (client >= 1 && client <= MaxClients && g_LPSClientSessionActive[client])
	{
		LPS_UpdateClientSessionClock(client, true);
		LPS_ReconcileClientSegment(client);
	}
}

bool LPS_TryCacheClientSteamId(int client)
{
	if (client < 1 || client > MaxClients || !IsClientConnected(client) || IsFakeClient(client))
	{
		return false;
	}

	char steamId[LPS_MAX_STEAM_ID];
	if (!GetClientAuthId(client, AuthId_SteamID64, steamId, sizeof(steamId), true)
		|| !LPS_IsValidSteamId64(steamId))
	{
		return false;
	}

	strcopy(g_LPSClientSteamId[client], sizeof(g_LPSClientSteamId[]), steamId);
	g_LPSClientAuthorized[client] = true;
	return true;
}

bool LPS_IsValidSteamId64(const char[] steamId)
{
	int length = strlen(steamId);
	if (length < 16 || length >= LPS_MAX_STEAM_ID)
	{
		return false;
	}

	for (int index = 0; index < length; index++)
	{
		if (steamId[index] < '0' || steamId[index] > '9')
		{
			return false;
		}
	}

	return true;
}

bool LPS_CanCollectSessions()
{
	return !g_LPSSessionShuttingDown
		&& g_LPSEnabled.BoolValue
		&& g_LPSServerKey[0] != '\0'
		&& g_LPSBootId[0] != '\0'
		&& LPS_IsSupportedMode();
}

void LPS_StartEligibleSessions()
{
	if (!LPS_CanCollectSessions())
	{
		return;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (IsClientConnected(client) && !IsFakeClient(client))
		{
			if (!g_LPSClientAuthorized[client])
			{
				LPS_TryCacheClientSteamId(client);
			}

			if (!LPS_TryResumeClientSession(client))
			{
				LPS_TryStartClientSession(client);
			}
		}
	}
}

bool LPS_TryResumeClientSession(int client)
{
	if (!LPS_CanCollectSessions()
		|| client < 1
		|| client > MaxClients
		|| g_LPSClientSessionActive[client]
		|| !g_LPSClientAuthorized[client]
		|| !IsClientConnected(client)
		|| IsFakeClient(client)
		|| g_LPSSuspendedSessions == null)
	{
		return false;
	}

	int index = LPS_FindSuspendedSessionBySteamId(g_LPSClientSteamId[client]);
	if (index < 0)
	{
		return false;
	}

	LPSSuspendedSession suspended;
	g_LPSSuspendedSessions.GetArray(index, suspended, sizeof(suspended));
	int now = GetTime();
	if (now - suspended.suspendedAt >= LPS_SESSION_TRANSFER_TIMEOUT)
	{
		LPS_CloseAndRemoveSuspendedSession(index, "map_reconnect_timeout");
		LPS_RequestFlush();
		return false;
	}

	g_LPSSuspendedSessions.Erase(index);

	int transferSeconds = now - suspended.suspendedAt;
	if (transferSeconds < 0)
	{
		transferSeconds = 0;
	}

	strcopy(g_LPSClientSessionId[client], sizeof(g_LPSClientSessionId[]), suspended.sessionId);
	strcopy(g_LPSClientSessionName[client], sizeof(g_LPSClientSessionName[]), suspended.playerName);
	strcopy(g_LPSClientIpAddress[client], sizeof(g_LPSClientIpAddress[]), suspended.ipAddress);
	g_LPSClientSessionStartedAt[client] = suspended.startedAt;
	g_LPSClientLastClockAt[client] = now;
	g_LPSClientConnectedSeconds[client] = suspended.connectedSeconds + transferSeconds;
	g_LPSClientActivePlaySeconds[client] = suspended.activePlaySeconds;
	g_LPSClientCurrentRevision[client] = suspended.currentRevision + 1;
	g_LPSClientPersistedRevision[client] = suspended.persistedRevision;
	g_LPSClientWasActive[client] = LPS_IsClientActivelyPlaying(client);
	g_LPSClientSessionDirty[client] = true;
	g_LPSClientSessionActive[client] = true;

	if (IsClientInGame(client))
	{
		LPS_ReconcileClientSegment(client);
	}

	LPS_RequestFlush();
	return true;
}

bool LPS_TryStartClientSession(int client)
{
	if (!LPS_CanCollectSessions()
		|| client < 1
		|| client > MaxClients
		|| g_LPSClientSessionActive[client]
		|| !g_LPSClientAuthorized[client]
		|| !IsClientInGame(client)
		|| IsFakeClient(client))
	{
		return false;
	}

	char ipAddress[LPS_MAX_IP_ADDRESS];
	if (!GetClientIP(client, ipAddress, sizeof(ipAddress), true))
	{
		return false;
	}

	TrimString(ipAddress);
	if (StrEqual(ipAddress, "loopback", false) || StrEqual(ipAddress, "localhost", false))
	{
		strcopy(ipAddress, sizeof(ipAddress), "127.0.0.1");
	}

	if (ipAddress[0] == '\0')
	{
		return false;
	}

	g_LPSSessionSequence++;
	Format(
		g_LPSClientSessionId[client],
		sizeof(g_LPSClientSessionId[]),
		"%s:session:%d",
		g_LPSBootId,
		g_LPSSessionSequence
	);
	GetClientName(client, g_LPSClientSessionName[client], sizeof(g_LPSClientSessionName[]));
	strcopy(g_LPSClientIpAddress[client], sizeof(g_LPSClientIpAddress[]), ipAddress);

	int now = GetTime();
	g_LPSClientSessionStartedAt[client] = now;
	g_LPSClientLastClockAt[client] = now;
	g_LPSClientConnectedSeconds[client] = 0;
	g_LPSClientActivePlaySeconds[client] = 0;
	g_LPSClientCurrentRevision[client] = 1;
	g_LPSClientPersistedRevision[client] = 0;
	g_LPSClientWasActive[client] = LPS_IsClientActivelyPlaying(client);
	g_LPSClientSessionDirty[client] = true;
	g_LPSClientSessionActive[client] = true;

	LPS_ReconcileClientSegment(client);
	LPS_RequestFlush();
	return true;
}

void LPS_UpdateClientSessionClock(int client, bool refreshActivity)
{
	if (!g_LPSClientSessionActive[client])
	{
		return;
	}

	int now = GetTime();
	int delta = now - g_LPSClientLastClockAt[client];
	if (delta > 0)
	{
		g_LPSClientConnectedSeconds[client] += delta;
		if (g_LPSClientWasActive[client])
		{
			g_LPSClientActivePlaySeconds[client] += delta;
		}

		g_LPSClientLastClockAt[client] = now;
		g_LPSClientCurrentRevision[client]++;
		g_LPSClientSessionDirty[client] = true;
	}

	if (refreshActivity)
	{
		g_LPSClientWasActive[client] = LPS_IsClientActivelyPlaying(client);
	}
}

bool LPS_IsClientActivelyPlaying(int client)
{
	if (!IsClientInGame(client))
	{
		return false;
	}

	int team = GetClientTeam(client);
	if (team != LPS_TEAM_SURVIVOR && team != LPS_TEAM_INFECTED)
	{
		return false;
	}

	int userId = GetClientUserId(client);
	for (int bot = 1; bot <= MaxClients; bot++)
	{
		if (!IsClientInGame(bot) || !IsFakeClient(bot) || GetClientTeam(bot) != LPS_TEAM_SURVIVOR)
		{
			continue;
		}

		if (HasEntProp(bot, Prop_Send, "m_humanSpectatorUserID")
			&& GetEntProp(bot, Prop_Send, "m_humanSpectatorUserID") == userId)
		{
			return false;
		}
	}

	return true;
}

void LPS_EndClientSession(int client, const char[] reason)
{
	if (!g_LPSClientSessionActive[client])
	{
		return;
	}

	LPS_UpdateClientSessionClock(client, false);
	LPS_EndClientSegment(client, "closed");

	LPSSessionSnapshot snapshot;
	LPS_FillClientSessionSnapshot(client, snapshot, true, reason);
	LPS_QueueClosedSession(snapshot);
	LPS_ResetClientSessionState(client, false);

	if (!g_LPSSessionShuttingDown)
	{
		LPS_RequestFlush();
	}
}

void LPS_SuspendClientSession(int client)
{
	if (!g_LPSClientSessionActive[client])
	{
		return;
	}

	LPS_UpdateClientSessionClock(client, false);
	LPS_EndClientSegment(client, "closed");

	int existingIndex = LPS_FindSuspendedSessionBySteamId(g_LPSClientSteamId[client]);
	if (existingIndex >= 0)
	{
		LPS_CloseAndRemoveSuspendedSession(existingIndex, "map_transfer_replaced");
	}

	LPSSuspendedSession suspended;
	strcopy(suspended.sessionId, sizeof(suspended.sessionId), g_LPSClientSessionId[client]);
	strcopy(suspended.bootId, sizeof(suspended.bootId), g_LPSBootId);
	strcopy(suspended.serverKey, sizeof(suspended.serverKey), g_LPSServerKey);
	strcopy(suspended.steamId, sizeof(suspended.steamId), g_LPSClientSteamId[client]);
	strcopy(suspended.playerName, sizeof(suspended.playerName), g_LPSClientSessionName[client]);
	strcopy(suspended.ipAddress, sizeof(suspended.ipAddress), g_LPSClientIpAddress[client]);
	suspended.startedAt = g_LPSClientSessionStartedAt[client];
	suspended.connectedSeconds = g_LPSClientConnectedSeconds[client];
	suspended.activePlaySeconds = g_LPSClientActivePlaySeconds[client];
	suspended.currentRevision = g_LPSClientCurrentRevision[client] + 1;
	suspended.persistedRevision = g_LPSClientPersistedRevision[client];
	suspended.suspendedAt = GetTime();
	g_LPSSuspendedSessions.PushArray(suspended, sizeof(suspended));

	LPS_ResetClientSessionState(client, false);
	if (!g_LPSSessionShuttingDown)
	{
		LPS_RequestFlush();
	}
}

int LPS_FindSuspendedSessionBySteamId(const char[] steamId)
{
	if (g_LPSSuspendedSessions == null || steamId[0] == '\0')
	{
		return -1;
	}

	LPSSuspendedSession suspended;
	for (int index = 0; index < g_LPSSuspendedSessions.Length; index++)
	{
		g_LPSSuspendedSessions.GetArray(index, suspended, sizeof(suspended));
		if (StrEqual(suspended.steamId, steamId, true))
		{
			return index;
		}
	}

	return -1;
}

void LPS_ExpireSuspendedSessions(int now)
{
	if (g_LPSSuspendedSessions == null)
	{
		return;
	}

	bool changed = false;
	LPSSuspendedSession suspended;
	for (int index = g_LPSSuspendedSessions.Length - 1; index >= 0; index--)
	{
		g_LPSSuspendedSessions.GetArray(index, suspended, sizeof(suspended));
		if (now - suspended.suspendedAt < LPS_SESSION_TRANSFER_TIMEOUT)
		{
			continue;
		}

		LPS_CloseAndRemoveSuspendedSession(index, "map_reconnect_timeout");
		changed = true;
	}

	if (changed && !g_LPSSessionShuttingDown)
	{
		LPS_RequestFlush();
	}
}

void LPS_CloseAndRemoveSuspendedSession(int index, const char[] reason)
{
	if (g_LPSSuspendedSessions == null || index < 0 || index >= g_LPSSuspendedSessions.Length)
	{
		return;
	}

	LPSSuspendedSession suspended;
	LPSSessionSnapshot snapshot;
	g_LPSSuspendedSessions.GetArray(index, suspended, sizeof(suspended));
	LPS_FillSuspendedSessionSnapshot(suspended, snapshot, true, reason);
	LPS_QueueClosedSession(snapshot);
	g_LPSSuspendedSessions.Erase(index);
}

void LPS_EndAllActiveSessions(const char[] reason)
{
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSessionActive[client])
		{
			LPS_EndClientSession(client, reason);
		}
	}

	if (g_LPSSuspendedSessions != null)
	{
		for (int index = g_LPSSuspendedSessions.Length - 1; index >= 0; index--)
		{
			LPS_CloseAndRemoveSuspendedSession(index, reason);
		}
	}

	if (!g_LPSSessionShuttingDown)
	{
		LPS_RequestFlush();
	}
}

void LPS_QueueClosedSession(LPSSessionSnapshot snapshot)
{
	int queueLimit = g_LPSClosedSessionQueueLimit.IntValue;
	if (g_LPSClosedSessions.Length >= queueLimit)
	{
		if (!g_LPSSessionQueueOverflowReported)
		{
			LogError(
				"Closed Session memory queue reached its limit (%d); additional closed Sessions will be dropped until a flush succeeds.",
				queueLimit
			);
			g_LPSSessionQueueOverflowReported = true;
		}
		return;
	}

	g_LPSClosedSessions.PushArray(snapshot, sizeof(snapshot));
}

void LPS_FillClientSessionSnapshot(
	int client,
	LPSSessionSnapshot snapshot,
	bool closed,
	const char[] reason
)
{
	strcopy(snapshot.sessionId, sizeof(snapshot.sessionId), g_LPSClientSessionId[client]);
	strcopy(snapshot.bootId, sizeof(snapshot.bootId), g_LPSBootId);
	strcopy(snapshot.serverKey, sizeof(snapshot.serverKey), g_LPSServerKey);
	strcopy(snapshot.steamId, sizeof(snapshot.steamId), g_LPSClientSteamId[client]);
	strcopy(snapshot.playerName, sizeof(snapshot.playerName), g_LPSClientSessionName[client]);
	strcopy(snapshot.latestName, sizeof(snapshot.latestName), g_LPSClientSessionName[client]);
	if (IsClientInGame(client))
	{
		GetClientName(client, snapshot.latestName, sizeof(snapshot.latestName));
	}
	strcopy(snapshot.ipAddress, sizeof(snapshot.ipAddress), g_LPSClientIpAddress[client]);
	strcopy(snapshot.status, sizeof(snapshot.status), closed ? "closed" : "active");
	strcopy(snapshot.disconnectReason, sizeof(snapshot.disconnectReason), closed ? reason : "");
	snapshot.startedAt = g_LPSClientSessionStartedAt[client];
	snapshot.endedAt = closed ? GetTime() : 0;
	snapshot.connectedSeconds = g_LPSClientConnectedSeconds[client];
	snapshot.activePlaySeconds = g_LPSClientActivePlaySeconds[client];
	snapshot.revision = g_LPSClientCurrentRevision[client] + (closed ? 1 : 0);
}

void LPS_FillSuspendedSessionSnapshot(
	LPSSuspendedSession suspended,
	LPSSessionSnapshot snapshot,
	bool closed,
	const char[] reason
)
{
	strcopy(snapshot.sessionId, sizeof(snapshot.sessionId), suspended.sessionId);
	strcopy(snapshot.bootId, sizeof(snapshot.bootId), suspended.bootId);
	strcopy(snapshot.serverKey, sizeof(snapshot.serverKey), suspended.serverKey);
	strcopy(snapshot.steamId, sizeof(snapshot.steamId), suspended.steamId);
	strcopy(snapshot.playerName, sizeof(snapshot.playerName), suspended.playerName);
	strcopy(snapshot.latestName, sizeof(snapshot.latestName), suspended.playerName);
	strcopy(snapshot.ipAddress, sizeof(snapshot.ipAddress), suspended.ipAddress);
	strcopy(snapshot.status, sizeof(snapshot.status), closed ? "closed" : "active");
	strcopy(snapshot.disconnectReason, sizeof(snapshot.disconnectReason), closed ? reason : "");
	snapshot.startedAt = suspended.startedAt;
	snapshot.endedAt = closed ? suspended.suspendedAt : 0;
	snapshot.connectedSeconds = suspended.connectedSeconds;
	snapshot.activePlaySeconds = suspended.activePlaySeconds;
	snapshot.revision = suspended.currentRevision + (closed ? 1 : 0);
}

bool LPS_AddSessionFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_ResetSessionFlushCapture();
	LPS_AccumulateAllSessionClocks();

	g_LPSFlushClosedSessionCount = g_LPSClosedSessions.Length;
	LPSSessionSnapshot snapshot;

	for (int index = 0; index < g_LPSFlushClosedSessionCount; index++)
	{
		g_LPSClosedSessions.GetArray(index, snapshot, sizeof(snapshot));
		if (!LPS_AddSessionSnapshotQueries(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientSessionActive[client] || !g_LPSClientSessionDirty[client])
		{
			continue;
		}

		LPS_FillClientSessionSnapshot(client, snapshot, false, "");
		if (!LPS_AddSessionSnapshotQueries(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}

		g_LPSFlushCapturedClient[client] = true;
		g_LPSFlushCapturedRevision[client] = snapshot.revision;
		strcopy(
			g_LPSFlushCapturedSessionId[client],
			sizeof(g_LPSFlushCapturedSessionId[]),
			snapshot.sessionId
		);
	}

	LPSSuspendedSession suspended;
	for (int index = 0; index < g_LPSSuspendedSessions.Length; index++)
	{
		g_LPSSuspendedSessions.GetArray(index, suspended, sizeof(suspended));
		LPS_FillSuspendedSessionSnapshot(suspended, snapshot, false, "");
		if (!LPS_AddSessionSnapshotQueries(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	return true;
}

bool LPS_AddSessionSnapshotQueries(
	Transaction transaction,
	LPSSessionSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char playerQuery[2048];
	char sessionQuery[4096];
	if (!LPS_BuildPlayerUpsertQuery(snapshot, playerQuery, sizeof(playerQuery))
		|| !LPS_BuildSessionUpsertQuery(snapshot, sessionQuery, sizeof(sessionQuery)))
	{
		strcopy(error, errorLength, "a Session value exceeded its SQL escape buffer");
		return false;
	}

	transaction.AddQuery(playerQuery);
	transaction.AddQuery(sessionQuery);
	queryCount += 2;
	return true;
}

bool LPS_BuildPlayerUpsertQuery(LPSSessionSnapshot snapshot, char[] query, int maxLength)
{
	char steamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char latestName[(LPS_MAX_PLAYER_NAME * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.steamId, steamId, sizeof(steamId))
		|| !g_LPSDatabase.Escape(snapshot.latestName, latestName, sizeof(latestName)))
	{
		return false;
	}

	int lastSeenAt = GetTime();
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_players (steam_id, last_name, first_seen_at, last_seen_at) VALUES ('%s', '%s', %d, %d) ON DUPLICATE KEY UPDATE last_name = IF(VALUES(last_seen_at) >= last_seen_at, VALUES(last_name), last_name), last_seen_at = GREATEST(last_seen_at, VALUES(last_seen_at))",
			steamId,
			latestName,
			snapshot.startedAt,
			lastSeenAt
		);
	}
	else if (g_LPSDatabaseDialect == LPSDatabaseDialect_PostgreSQL)
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_players (steam_id, last_name, first_seen_at, last_seen_at) VALUES ('%s', '%s', %d, %d) ON CONFLICT(steam_id) DO UPDATE SET last_name = CASE WHEN excluded.last_seen_at >= lps_players.last_seen_at THEN excluded.last_name ELSE lps_players.last_name END, last_seen_at = GREATEST(lps_players.last_seen_at, excluded.last_seen_at)",
			steamId,
			latestName,
			snapshot.startedAt,
			lastSeenAt
		);
	}
	else
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_players (steam_id, last_name, first_seen_at, last_seen_at) VALUES ('%s', '%s', %d, %d) ON CONFLICT(steam_id) DO UPDATE SET last_name = CASE WHEN excluded.last_seen_at >= lps_players.last_seen_at THEN excluded.last_name ELSE lps_players.last_name END, last_seen_at = MAX(lps_players.last_seen_at, excluded.last_seen_at)",
			steamId,
			latestName,
			snapshot.startedAt,
			lastSeenAt
		);
	}

	return true;
}

bool LPS_BuildSessionUpsertQuery(LPSSessionSnapshot snapshot, char[] query, int maxLength)
{
	char sessionId[(LPS_MAX_SESSION_ID * 2) + 1];
	char bootId[(LPS_MAX_BOOT_ID * 2) + 1];
	char serverKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char steamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char playerName[(LPS_MAX_PLAYER_NAME * 2) + 1];
	char ipAddress[(LPS_MAX_IP_ADDRESS * 2) + 1];
	char status[(LPS_MAX_SESSION_STATUS * 2) + 1];
	char reason[(LPS_MAX_DISCONNECT_REASON * 2) + 1];

	if (!g_LPSDatabase.Escape(snapshot.sessionId, sessionId, sizeof(sessionId))
		|| !g_LPSDatabase.Escape(snapshot.bootId, bootId, sizeof(bootId))
		|| !g_LPSDatabase.Escape(snapshot.serverKey, serverKey, sizeof(serverKey))
		|| !g_LPSDatabase.Escape(snapshot.steamId, steamId, sizeof(steamId))
		|| !g_LPSDatabase.Escape(snapshot.playerName, playerName, sizeof(playerName))
		|| !g_LPSDatabase.Escape(snapshot.ipAddress, ipAddress, sizeof(ipAddress))
		|| !g_LPSDatabase.Escape(snapshot.status, status, sizeof(status))
		|| !g_LPSDatabase.Escape(snapshot.disconnectReason, reason, sizeof(reason)))
	{
		return false;
	}

	char endedAt[32];
	if (snapshot.endedAt > 0)
	{
		IntToString(snapshot.endedAt, endedAt, sizeof(endedAt));
	}
	else
	{
		strcopy(endedAt, sizeof(endedAt), "NULL");
	}

	int savedAt = GetTime();
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_sessions (session_id, boot_id, server_key, steam_id, player_name, ip_address, started_at, ended_at, last_saved_at, connected_seconds, active_play_seconds, status, disconnect_reason, revision) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', %d, %s, %d, %d, %d, '%s', '%s', %d) ON DUPLICATE KEY UPDATE ended_at = VALUES(ended_at), last_saved_at = VALUES(last_saved_at), connected_seconds = VALUES(connected_seconds), active_play_seconds = VALUES(active_play_seconds), status = VALUES(status), disconnect_reason = VALUES(disconnect_reason), revision = VALUES(revision)",
			sessionId,
			bootId,
			serverKey,
			steamId,
			playerName,
			ipAddress,
			snapshot.startedAt,
			endedAt,
			savedAt,
			snapshot.connectedSeconds,
			snapshot.activePlaySeconds,
			status,
			reason,
			snapshot.revision
		);
	}
	else
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_sessions (session_id, boot_id, server_key, steam_id, player_name, ip_address, started_at, ended_at, last_saved_at, connected_seconds, active_play_seconds, status, disconnect_reason, revision) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', %d, %s, %d, %d, %d, '%s', '%s', %d) ON CONFLICT(session_id) DO UPDATE SET ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, connected_seconds = excluded.connected_seconds, active_play_seconds = excluded.active_play_seconds, status = excluded.status, disconnect_reason = excluded.disconnect_reason, revision = excluded.revision",
			sessionId,
			bootId,
			serverKey,
			steamId,
			playerName,
			ipAddress,
			snapshot.startedAt,
			endedAt,
			savedAt,
			snapshot.connectedSeconds,
			snapshot.activePlaySeconds,
			status,
			reason,
			snapshot.revision
		);
	}

	return true;
}

void LPS_OnSessionFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedSessionCount && g_LPSClosedSessions.Length > 0; index++)
	{
		g_LPSClosedSessions.Erase(0);
	}

	if (g_LPSClosedSessions.Length < g_LPSClosedSessionQueueLimit.IntValue)
	{
		g_LPSSessionQueueOverflowReported = false;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSFlushCapturedClient[client]
			|| !g_LPSClientSessionActive[client]
			|| !StrEqual(g_LPSClientSessionId[client], g_LPSFlushCapturedSessionId[client], true))
		{
			continue;
		}

		if (g_LPSFlushCapturedRevision[client] > g_LPSClientPersistedRevision[client])
		{
			g_LPSClientPersistedRevision[client] = g_LPSFlushCapturedRevision[client];
		}

		if (g_LPSClientCurrentRevision[client] <= g_LPSFlushCapturedRevision[client])
		{
			g_LPSClientSessionDirty[client] = false;
		}
	}

	LPS_ResetSessionFlushCapture();
}

void LPS_ResetSessionFlushCapture()
{
	g_LPSFlushClosedSessionCount = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSFlushCapturedClient[client] = false;
		g_LPSFlushCapturedRevision[client] = 0;
		g_LPSFlushCapturedSessionId[client][0] = '\0';
	}
}

void LPS_AccumulateAllSessionClocks()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSessionActive[client])
		{
			LPS_UpdateClientSessionClock(client, true);
		}
	}
}

void LPS_ResetClientSessionState(int client, bool clearAuthorization)
{
	g_LPSClientSessionActive[client] = false;
	g_LPSClientSessionDirty[client] = false;
	g_LPSClientWasActive[client] = false;
	g_LPSClientSessionId[client][0] = '\0';
	g_LPSClientSessionName[client][0] = '\0';
	g_LPSClientIpAddress[client][0] = '\0';
	g_LPSClientSessionStartedAt[client] = 0;
	g_LPSClientLastClockAt[client] = 0;
	g_LPSClientConnectedSeconds[client] = 0;
	g_LPSClientActivePlaySeconds[client] = 0;
	g_LPSClientCurrentRevision[client] = 0;
	g_LPSClientPersistedRevision[client] = 0;

	if (clearAuthorization)
	{
		g_LPSClientAuthorized[client] = false;
		g_LPSClientSteamId[client][0] = '\0';
	}
}

int LPS_GetActiveSessionCount()
{
	int count = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientSessionActive[client])
		{
			count++;
		}
	}
	return count;
}

int LPS_GetSuspendedSessionCount()
{
	return g_LPSSuspendedSessions == null ? 0 : g_LPSSuspendedSessions.Length;
}

int LPS_GetPendingClosedSessionCount()
{
	return g_LPSClosedSessions == null ? 0 : g_LPSClosedSessions.Length;
}

void LPS_OnSessionDatabaseReady()
{
	LPS_StartEligibleSessions();
	LPS_RequestFlush();
}

void LPS_ShutdownSessions()
{
	g_LPSSessionShuttingDown = true;
	LPS_EndAllActiveSessions("plugin_unload");

	if (g_LPSSessionClockTimer != null)
	{
		delete g_LPSSessionClockTimer;
		g_LPSSessionClockTimer = null;
	}
}
