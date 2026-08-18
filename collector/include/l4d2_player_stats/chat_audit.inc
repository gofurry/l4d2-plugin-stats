#if defined _l4d2_player_stats_chat_audit_included
	#endinput
#endif
#define _l4d2_player_stats_chat_audit_included

enum struct LPSChatMessage
{
	char messageId[LPS_MAX_CHAT_MESSAGE_ID];
	char bootId[LPS_MAX_BOOT_ID];
	char serverKey[LPS_MAX_SERVER_KEY];
	int chatSeq;
	char sessionId[LPS_MAX_SESSION_ID];
	char steamId[LPS_MAX_STEAM_ID];
	int sourceUserId;
	char playerName[LPS_MAX_PLAYER_NAME];
	int occurredAt;
	char mapName[LPS_MAX_MAP_NAME];
	char gameMode[LPS_MAX_GAME_MODE];
	char team[16];
	char channel[16];
	bool alive;
	bool commandLike;
	char content[LPS_MAX_CHAT_CONTENT];
}

ArrayList g_LPSChatQueue;
Handle g_LPSChatFlushTimer;
bool g_LPSChatFlushInFlight;
bool g_LPSChatQueueOverflowReported;
bool g_LPSChatStateDirty;
int g_LPSChatFlushCount;
int g_LPSChatSeq;
int g_LPSChatObservedCount;
int g_LPSChatPersistedCount;
int g_LPSChatDroppedCount;
int g_LPSChatStateRevision;
int g_LPSChatNextPruneAt;

void LPS_InitializeChatAudit()
{
	g_LPSChatQueue = new ArrayList(sizeof(LPSChatMessage));
	g_LPSChatFlushTimer = CreateTimer(2.0, LPS_Timer_ChatFlush, _, TIMER_REPEAT);
	g_LPSChatFlushInFlight = false;
	g_LPSChatQueueOverflowReported = false;
	g_LPSChatStateDirty = true;
	g_LPSChatFlushCount = 0;
	g_LPSChatSeq = 0;
	g_LPSChatObservedCount = 0;
	g_LPSChatPersistedCount = 0;
	g_LPSChatDroppedCount = 0;
	g_LPSChatStateRevision = 1;
	g_LPSChatNextPruneAt = 0;

	AddCommandListener(LPS_Command_ChatAudit, "say");
	AddCommandListener(LPS_Command_ChatAudit, "say_team");
	g_LPSChatAuditEnabled.AddChangeHook(LPS_OnChatAuditEnabledChanged);
}

void LPS_ShutdownChatAudit()
{
	RemoveCommandListener(LPS_Command_ChatAudit, "say");
	RemoveCommandListener(LPS_Command_ChatAudit, "say_team");
	if (g_LPSChatFlushTimer != null)
	{
		delete g_LPSChatFlushTimer;
		g_LPSChatFlushTimer = null;
	}
}

public void LPS_OnChatAuditEnabledChanged(ConVar convar, const char[] oldValue, const char[] newValue)
{
	g_LPSChatStateRevision++;
	g_LPSChatStateDirty = true;
}

public Action LPS_Command_ChatAudit(int client, const char[] command, int argc)
{
	if (!g_LPSChatAuditEnabled.BoolValue
		|| g_LPSBootId[0] == '\0'
		|| g_LPSServerKey[0] == '\0'
		|| client < 1
		|| client > MaxClients
		|| !IsClientInGame(client)
		|| IsFakeClient(client))
	{
		return Plugin_Continue;
	}

	char content[LPS_MAX_CHAT_CONTENT];
	GetCmdArgString(content, sizeof(content));
	StripQuotes(content);
	TrimString(content);
	if (content[0] == '\0')
	{
		return Plugin_Continue;
	}

	// Advance before queue admission. Any bounded-memory drop therefore leaves
	// a durable sequence gap that the Dashboard can report.
	g_LPSChatSeq++;
	g_LPSChatObservedCount++;
	g_LPSChatStateRevision++;
	g_LPSChatStateDirty = true;

	if (g_LPSChatQueue.Length >= LPS_CHAT_QUEUE_LIMIT)
	{
		g_LPSChatDroppedCount++;
		if (!g_LPSChatQueueOverflowReported)
		{
			LogError("Chat audit memory queue reached %d rows; messages will be dropped until persistence recovers.", LPS_CHAT_QUEUE_LIMIT);
			g_LPSChatQueueOverflowReported = true;
		}
		return Plugin_Continue;
	}

	LPSChatMessage message;
	Format(message.messageId, sizeof(message.messageId), "%s:chat:%d", g_LPSBootId, g_LPSChatSeq);
	strcopy(message.bootId, sizeof(message.bootId), g_LPSBootId);
	strcopy(message.serverKey, sizeof(message.serverKey), g_LPSServerKey);
	message.chatSeq = g_LPSChatSeq;
	if (g_LPSClientSessionActive[client])
	{
		strcopy(message.sessionId, sizeof(message.sessionId), g_LPSClientSessionId[client]);
	}
	else
	{
		message.sessionId[0] = '\0';
	}
	strcopy(message.steamId, sizeof(message.steamId), g_LPSClientSteamId[client]);
	message.sourceUserId = GetClientUserId(client);
	GetClientName(client, message.playerName, sizeof(message.playerName));
	message.occurredAt = GetTime();
	GetCurrentMap(message.mapName, sizeof(message.mapName));
	strcopy(message.gameMode, sizeof(message.gameMode), g_LPSGameMode);
	LPS_GetChatTeamName(client, message.team, sizeof(message.team));
	strcopy(message.channel, sizeof(message.channel), StrEqual(command, "say_team", false) ? "team" : "global");
	message.alive = IsPlayerAlive(client);
	message.commandLike = content[0] == '!' || content[0] == '/';
	strcopy(message.content, sizeof(message.content), content);
	g_LPSChatQueue.PushArray(message, sizeof(message));
	return Plugin_Continue;
}

void LPS_GetChatTeamName(int client, char[] team, int maxLength)
{
	switch (GetClientTeam(client))
	{
		case LPS_TEAM_SURVIVOR: strcopy(team, maxLength, "survivor");
		case LPS_TEAM_INFECTED: strcopy(team, maxLength, "infected");
		case LPS_TEAM_SPECTATOR: strcopy(team, maxLength, "spectator");
		default: strcopy(team, maxLength, "other");
	}
}

public Action LPS_Timer_ChatFlush(Handle timer)
{
	LPS_RequestChatFlush();
	return Plugin_Continue;
}

void LPS_OnChatDatabaseReady()
{
	g_LPSChatStateDirty = true;
	LPS_RequestChatFlush();
}

void LPS_OnChatDatabaseDisconnected()
{
	// A stale callback is ignored by connection generation. Keep the immutable
	// queued rows so the next connection can retry them idempotently.
	g_LPSChatFlushInFlight = false;
	g_LPSChatFlushCount = 0;
	g_LPSChatStateDirty = true;
}

void LPS_OnChatBootStarted()
{
	// Sequence identity is boot-scoped. A runtime server_key replacement is a
	// new bounded audit source and must not reuse counters from the old boot.
	if (g_LPSChatQueue != null)
	{
		g_LPSChatQueue.Clear();
	}
	g_LPSChatSeq = 0;
	g_LPSChatObservedCount = 0;
	g_LPSChatPersistedCount = 0;
	g_LPSChatDroppedCount = 0;
	g_LPSChatStateRevision = 1;
	g_LPSChatStateDirty = true;
	g_LPSChatNextPruneAt = 0;
}

void LPS_RequestChatFlush()
{
	if (g_LPSChatFlushInFlight
		|| g_LPSDatabaseState != LPSDatabaseState_Ready
		|| g_LPSDatabase == null
		|| (!g_LPSChatStateDirty && g_LPSChatQueue.Length == 0 && GetTime() < g_LPSChatNextPruneAt))
	{
		return;
	}

	Transaction transaction = new Transaction();
	g_LPSChatFlushCount = g_LPSChatQueue.Length;
	if (g_LPSChatFlushCount > LPS_CHAT_FLUSH_BATCH_LIMIT)
	{
		g_LPSChatFlushCount = LPS_CHAT_FLUSH_BATCH_LIMIT;
	}

	LPSChatMessage message;
	char query[LPS_MAX_QUERY_LENGTH];
	for (int index = 0; index < g_LPSChatFlushCount; index++)
	{
		g_LPSChatQueue.GetArray(index, message, sizeof(message));
		if (!LPS_BuildChatOutboxInsert(message, query, sizeof(query)))
		{
			delete transaction;
			LPS_ReportDatabaseError("chat audit query build failed", "a chat field exceeded its bounded SQL buffer");
			return;
		}
		transaction.AddQuery(query);
	}

	LPS_BuildChatCaptureStateUpsert(query, sizeof(query), g_LPSChatPersistedCount + g_LPSChatFlushCount);
	transaction.AddQuery(query);
	int now = GetTime();
	if (now >= g_LPSChatNextPruneAt)
	{
		LPS_BuildChatPruneQuery(query, sizeof(query), now - LPS_CHAT_TRANSPORT_RETENTION_SECONDS);
		transaction.AddQuery(query);
		LPS_BuildChatOldestSequenceQuery(query, sizeof(query));
		transaction.AddQuery(query);
		g_LPSChatNextPruneAt = now + 300;
	}

	g_LPSChatFlushInFlight = true;
	g_LPSDatabase.Execute(
		transaction,
		LPS_OnChatFlushSucceeded,
		LPS_OnChatFlushFailed,
		g_LPSConnectionGeneration,
		DBPrio_Low
	);
}

bool LPS_BuildChatOutboxInsert(LPSChatMessage message, char[] query, int maxLength)
{
	char messageId[(LPS_MAX_CHAT_MESSAGE_ID * 2) + 1];
	char bootId[(LPS_MAX_BOOT_ID * 2) + 1];
	char serverKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char sessionId[(LPS_MAX_SESSION_ID * 2) + 1];
	char steamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char playerName[(LPS_MAX_PLAYER_NAME * 2) + 1];
	char mapName[(LPS_MAX_MAP_NAME * 2) + 1];
	char gameMode[(LPS_MAX_GAME_MODE * 2) + 1];
	char content[(LPS_MAX_CHAT_CONTENT * 2) + 1];
	if (!g_LPSDatabase.Escape(message.messageId, messageId, sizeof(messageId))
		|| !g_LPSDatabase.Escape(message.bootId, bootId, sizeof(bootId))
		|| !g_LPSDatabase.Escape(message.serverKey, serverKey, sizeof(serverKey))
		|| !g_LPSDatabase.Escape(message.sessionId, sessionId, sizeof(sessionId))
		|| !g_LPSDatabase.Escape(message.steamId, steamId, sizeof(steamId))
		|| !g_LPSDatabase.Escape(message.playerName, playerName, sizeof(playerName))
		|| !g_LPSDatabase.Escape(message.mapName, mapName, sizeof(mapName))
		|| !g_LPSDatabase.Escape(message.gameMode, gameMode, sizeof(gameMode))
		|| !g_LPSDatabase.Escape(message.content, content, sizeof(content)))
	{
		return false;
	}

	char sessionValue[(LPS_MAX_SESSION_ID * 2) + 8];
	char steamValue[(LPS_MAX_STEAM_ID * 2) + 8];
	if (message.sessionId[0] == '\0') strcopy(sessionValue, sizeof(sessionValue), "NULL");
	else Format(sessionValue, sizeof(sessionValue), "'%s'", sessionId);
	if (message.steamId[0] == '\0') strcopy(steamValue, sizeof(steamValue), "NULL");
	else Format(steamValue, sizeof(steamValue), "'%s'", steamId);

	char suffix[128];
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		strcopy(suffix, sizeof(suffix), "ON DUPLICATE KEY UPDATE message_id = VALUES(message_id)");
	}
	else
	{
		strcopy(suffix, sizeof(suffix), "ON CONFLICT(message_id) DO NOTHING");
	}
	FormatEx(
		query,
		maxLength,
		"INSERT INTO lps_chat_outbox (message_id, boot_id, server_key, chat_seq, session_id, steam_id, source_user_id, player_name, occurred_at, map_name, game_mode, team, channel, alive, command_like, content) VALUES ('%s','%s','%s',%d,%s,%s,%d,'%s',%d,'%s','%s','%s','%s',%d,%d,'%s') %s",
		messageId, bootId, serverKey, message.chatSeq, sessionValue, steamValue,
		message.sourceUserId, playerName, message.occurredAt, mapName, gameMode,
		message.team, message.channel, message.alive ? 1 : 0,
		message.commandLike ? 1 : 0, content, suffix
	);
	return true;
}

void LPS_BuildChatCaptureStateUpsert(char[] query, int maxLength, int persistedCount)
{
	char suffix[1024];
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		strcopy(suffix, sizeof(suffix), "ON DUPLICATE KEY UPDATE capture_enabled=VALUES(capture_enabled), ended_at=VALUES(ended_at), last_saved_at=VALUES(last_saved_at), observed_count=VALUES(observed_count), persisted_count=VALUES(persisted_count), dropped_count=VALUES(dropped_count), last_chat_seq=VALUES(last_chat_seq), revision=VALUES(revision)");
	}
	else
	{
		strcopy(suffix, sizeof(suffix), "ON CONFLICT(boot_id) DO UPDATE SET capture_enabled=excluded.capture_enabled, ended_at=excluded.ended_at, last_saved_at=excluded.last_saved_at, observed_count=excluded.observed_count, persisted_count=excluded.persisted_count, dropped_count=excluded.dropped_count, last_chat_seq=excluded.last_chat_seq, revision=excluded.revision");
	}
	FormatEx(
		query,
		maxLength,
		"INSERT INTO lps_chat_capture_state (boot_id, server_key, capture_version, capture_enabled, started_at, ended_at, last_saved_at, observed_count, persisted_count, dropped_count, last_chat_seq, oldest_retained_seq, revision) VALUES ('%s','%s',%d,%d,%d,NULL,%d,%d,%d,%d,%d,0,%d) %s",
		g_LPSBootId, g_LPSServerKey, LPS_CHAT_CAPTURE_VERSION,
		g_LPSChatAuditEnabled.BoolValue ? 1 : 0, g_LPSBootStartedAt, GetTime(),
		g_LPSChatObservedCount, persistedCount, g_LPSChatDroppedCount,
		g_LPSChatSeq, g_LPSChatStateRevision, suffix
	);
}

void LPS_BuildChatPruneQuery(char[] query, int maxLength, int cutoff)
{
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(query, maxLength, "DELETE FROM lps_chat_outbox WHERE occurred_at < %d ORDER BY occurred_at LIMIT 256", cutoff);
	}
	else
	{
		FormatEx(query, maxLength, "DELETE FROM lps_chat_outbox WHERE message_id IN (SELECT message_id FROM lps_chat_outbox WHERE occurred_at < %d ORDER BY occurred_at LIMIT 256)", cutoff);
	}
}

void LPS_BuildChatOldestSequenceQuery(char[] query, int maxLength)
{
	FormatEx(query, maxLength, "UPDATE lps_chat_capture_state SET oldest_retained_seq = COALESCE((SELECT MIN(chat_seq) FROM lps_chat_outbox WHERE boot_id = '%s'), last_chat_seq + 1) WHERE boot_id = '%s'", g_LPSBootId, g_LPSBootId);
}

public void LPS_OnChatFlushSucceeded(Database database, any data, int queryCount, DBResultSet[] results, any[] queryData)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}
	g_LPSChatFlushInFlight = false;
	for (int index = 0; index < g_LPSChatFlushCount && g_LPSChatQueue.Length > 0; index++)
	{
		g_LPSChatQueue.Erase(0);
	}
	g_LPSChatPersistedCount += g_LPSChatFlushCount;
	g_LPSChatFlushCount = 0;
	g_LPSChatStateDirty = false;
	if (g_LPSChatQueue.Length < LPS_CHAT_QUEUE_LIMIT)
	{
		g_LPSChatQueueOverflowReported = false;
	}
	if (g_LPSChatQueue.Length > 0)
	{
		LPS_RequestChatFlush();
	}
}

public void LPS_OnChatFlushFailed(Database database, any data, int queryCount, const char[] error, int failIndex, any[] queryData)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}
	g_LPSChatFlushInFlight = false;
	g_LPSChatFlushCount = 0;
	g_LPSChatStateDirty = true;
	// Never forward the raw driver error: it may contain the rejected INSERT
	// and therefore private chat content.
	LPS_ReportDatabaseError("chat audit persistence failed", "the transient chat transaction was rejected");
}
