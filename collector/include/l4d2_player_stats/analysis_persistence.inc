#if defined _l4d2_player_stats_analysis_persistence_included
	#endinput
#endif
#define _l4d2_player_stats_analysis_persistence_included

bool g_LPSAnalysisFlushInFlight;
bool g_LPSAnalysisFlushScheduled;
int g_LPSAnalysisCapturedIncidentCount;
int g_LPSAnalysisCapturedContextCount;
bool g_LPSAnalysisCapturedActiveContext;
int g_LPSAnalysisCapturedActiveRevision;
int g_LPSLastAnalysisFlushRows;
int g_LPSLastAnalysisFlushMs;
int g_LPSLastAnalysisSuccess;
char g_LPSLastAnalysisError[LPS_MAX_DATABASE_ERROR];
float g_LPSAnalysisFlushStartedGameTime;
int g_LPSLastAnalysisErrorLogAt;
int g_LPSSuppressedAnalysisErrors;

void LPS_InitializeAnalysisPersistence()
{
	g_LPSAnalysisFlushInFlight = false;
	g_LPSAnalysisFlushScheduled = false;
	g_LPSAnalysisCapturedIncidentCount = 0;
	g_LPSAnalysisCapturedContextCount = 0;
	g_LPSAnalysisCapturedActiveContext = false;
	g_LPSAnalysisCapturedActiveRevision = 0;
	g_LPSLastAnalysisFlushRows = 0;
	g_LPSLastAnalysisFlushMs = 0;
	g_LPSLastAnalysisSuccess = 0;
	g_LPSLastAnalysisError[0] = '\0';
	g_LPSAnalysisFlushStartedGameTime = 0.0;
	g_LPSLastAnalysisErrorLogAt = 0;
	g_LPSSuppressedAnalysisErrors = 0;
}

void LPS_RequestAnalysisFlush()
{
	if (g_LPSAnalysisFlushInFlight
		|| g_LPSDatabaseState != LPSDatabaseState_Ready
		|| g_LPSDatabase == null)
	{
		return;
	}

	int pendingIncidents = LPS_GetPendingIncidentCount();
	g_LPSAnalysisCapturedIncidentCount = pendingIncidents;
	if (g_LPSAnalysisCapturedIncidentCount > LPS_ANALYSIS_FLUSH_INCIDENT_LIMIT)
	{
		g_LPSAnalysisCapturedIncidentCount = LPS_ANALYSIS_FLUSH_INCIDENT_LIMIT;
	}

	Transaction transaction = new Transaction();
	int queryCount = 0;
	char error[LPS_MAX_DATABASE_ERROR];
	if (!LPS_AddAnalysisContextQueries(transaction, queryCount, error, sizeof(error))
		|| !LPS_AddIncidentBatchQueries(transaction, queryCount, error, sizeof(error)))
	{
		delete transaction;
		LPS_RecordAnalysisError("analysis query build failed");
		LPS_ResetAnalysisFlushCapture();
		return;
	}

	if (queryCount == 0)
	{
		delete transaction;
		LPS_ResetAnalysisFlushCapture();
		return;
	}

	g_LPSAnalysisFlushInFlight = true;
	g_LPSAnalysisFlushStartedGameTime = GetGameTime();
	g_LPSDatabase.Execute(
		transaction,
		LPS_OnAnalysisFlushSucceeded,
		LPS_OnAnalysisFlushFailed,
		g_LPSConnectionGeneration,
		DBPrio_Low
	);
}

bool LPS_AddAnalysisContextQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	g_LPSAnalysisCapturedContextCount = 0;
	g_LPSAnalysisCapturedActiveContext = false;
	g_LPSAnalysisCapturedActiveRevision = 0;

	LPSRoundContextSnapshot snapshot;
	for (int index = 0; index < g_LPSPendingRoundContexts.Length; index++)
	{
		g_LPSPendingRoundContexts.GetArray(index, snapshot, sizeof(snapshot));
		if (LPS_HasIncidentForRoundAfterCapture(snapshot.roundId, g_LPSAnalysisCapturedIncidentCount))
		{
			break;
		}
		snapshot.incidentCaptureComplete = snapshot.incidentCaptureEnabled
			&& snapshot.incidentDroppedCount == 0;
		if (!LPS_AddRoundContextQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
		g_LPSAnalysisCapturedContextCount++;
	}

	if (g_LPSRoundContextActive && g_LPSRoundContextDirty)
	{
		snapshot = g_LPSActiveRoundContext;
		snapshot.incidentCaptureComplete = false;
		if (!LPS_AddRoundContextQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
		g_LPSAnalysisCapturedActiveContext = true;
		g_LPSAnalysisCapturedActiveRevision = snapshot.revision;
	}
	return true;
}

bool LPS_HasIncidentForRoundAfterCapture(const char[] roundId, int captureCount)
{
	LPSIncidentSnapshot incident;
	for (int index = captureCount; index < g_LPSPendingIncidents.Length; index++)
	{
		g_LPSPendingIncidents.GetArray(index, incident, sizeof(incident));
		if (StrEqual(incident.roundId, roundId, true))
		{
			return true;
		}
	}
	return false;
}

bool LPS_AddRoundContextQuery(Transaction transaction, LPSRoundContextSnapshot snapshot, int &queryCount, char[] error, int errorLength)
{
	char roundId[(LPS_MAX_ROUND_ID * 2) + 1];
	char collectorVersion[65];
	char rulesetName[(LPS_MAX_RULESET_NAME * 2) + 1];
	char difficulty[(LPS_MAX_DIFFICULTY * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.roundId, roundId, sizeof(roundId))
		|| !g_LPSDatabase.Escape(snapshot.collectorVersion, collectorVersion, sizeof(collectorVersion))
		|| !g_LPSDatabase.Escape(snapshot.rulesetName, rulesetName, sizeof(rulesetName))
		|| !g_LPSDatabase.Escape(snapshot.difficulty, difficulty, sizeof(difficulty)))
	{
		strcopy(error, errorLength, "a Round Context value exceeded its SQL escape buffer");
		return false;
	}

	char query[4096];
	char columns[] = "round_id, context_version, captured_at, last_saved_at, collector_version, ruleset_name, difficulty, survivor_limit, max_player_zombies, common_limit, tank_health, witch_health, change_mask, incident_capture_enabled, incident_capture_complete, incident_expected_count, incident_dropped_count, revision";
	char values[2048];
	Format(values, sizeof(values), "'%s', %d, %d, %d, '%s', '%s', '%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d", roundId, LPS_CONTEXT_VERSION, snapshot.capturedAt, GetTime(), collectorVersion, rulesetName, difficulty, snapshot.survivorLimit, snapshot.maxPlayerZombies, snapshot.commonLimit, snapshot.tankHealth, snapshot.witchHealth, snapshot.changeMask, snapshot.incidentCaptureEnabled ? 1 : 0, snapshot.incidentCaptureComplete ? 1 : 0, snapshot.incidentExpectedCount, snapshot.incidentDroppedCount, snapshot.revision);
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(query, sizeof(query), "INSERT INTO lps_round_contexts (%s) VALUES (%s) ON DUPLICATE KEY UPDATE last_saved_at=VALUES(last_saved_at), change_mask=VALUES(change_mask), incident_capture_enabled=VALUES(incident_capture_enabled), incident_capture_complete=VALUES(incident_capture_complete), incident_expected_count=VALUES(incident_expected_count), incident_dropped_count=VALUES(incident_dropped_count), revision=VALUES(revision)", columns, values);
	}
	else
	{
		Format(query, sizeof(query), "INSERT INTO lps_round_contexts (%s) VALUES (%s) ON CONFLICT(round_id) DO UPDATE SET last_saved_at=excluded.last_saved_at, change_mask=excluded.change_mask, incident_capture_enabled=excluded.incident_capture_enabled, incident_capture_complete=excluded.incident_capture_complete, incident_expected_count=excluded.incident_expected_count, incident_dropped_count=excluded.incident_dropped_count, revision=excluded.revision", columns, values);
	}
	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_AddIncidentBatchQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	if (g_LPSAnalysisCapturedIncidentCount <= 0)
	{
		return true;
	}
	char prefix[] = "INSERT INTO lps_incidents (round_id, incident_seq, incident_version, incident_type, occurred_at, round_offset_ms, duration_ms, actor_kind, actor_steam_id, target_kind, target_steam_id, helper_kind, helper_steam_id, infected_class, end_reason, detail_flags, related_incident_seq, pos_x, pos_y, pos_z, end_pos_x, end_pos_y, end_pos_z) VALUES ";
	char suffix[160];
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		strcopy(suffix, sizeof(suffix), " ON DUPLICATE KEY UPDATE incident_seq=VALUES(incident_seq)");
	}
	else
	{
		strcopy(suffix, sizeof(suffix), " ON CONFLICT(round_id, incident_seq) DO NOTHING");
	}

	char query[LPS_MAX_QUERY_LENGTH];
	strcopy(query, sizeof(query), prefix);
	int rowsInQuery = 0;
	LPSIncidentSnapshot incident;
	char row[2048];
	for (int index = 0; index < g_LPSAnalysisCapturedIncidentCount; index++)
	{
		g_LPSPendingIncidents.GetArray(index, incident, sizeof(incident));
		if (!LPS_BuildIncidentValues(incident, row, sizeof(row)))
		{
			strcopy(error, errorLength, "an Incident value exceeded its SQL escape buffer");
			return false;
		}
		int required = strlen(query) + strlen(row) + strlen(suffix) + 2;
		if (rowsInQuery > 0 && required >= sizeof(query))
		{
			StrCat(query, sizeof(query), suffix);
			transaction.AddQuery(query);
			queryCount++;
			strcopy(query, sizeof(query), prefix);
			rowsInQuery = 0;
		}
		if (strlen(query) + strlen(row) + strlen(suffix) + 2 >= sizeof(query))
		{
			strcopy(error, errorLength, "a single encoded Incident exceeds the query buffer");
			return false;
		}
		if (rowsInQuery > 0) StrCat(query, sizeof(query), ",");
		StrCat(query, sizeof(query), row);
		rowsInQuery++;
	}
	if (rowsInQuery > 0)
	{
		StrCat(query, sizeof(query), suffix);
		transaction.AddQuery(query);
		queryCount++;
	}
	return true;
}

bool LPS_BuildIncidentValues(LPSIncidentSnapshot incident, char[] row, int maxLength)
{
	char roundId[(LPS_MAX_ROUND_ID * 2) + 1];
	char actorSteamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char targetSteamId[(LPS_MAX_STEAM_ID * 2) + 1];
	char helperSteamId[(LPS_MAX_STEAM_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(incident.roundId, roundId, sizeof(roundId))
		|| !g_LPSDatabase.Escape(incident.actorSteamId, actorSteamId, sizeof(actorSteamId))
		|| !g_LPSDatabase.Escape(incident.targetSteamId, targetSteamId, sizeof(targetSteamId))
		|| !g_LPSDatabase.Escape(incident.helperSteamId, helperSteamId, sizeof(helperSteamId)))
	{
		return false;
	}
	char posX[24], posY[24], posZ[24], endPosX[24], endPosY[24], endPosZ[24];
	LPS_FormatNullableInt(incident.hasPosition, incident.posX, posX, sizeof(posX));
	LPS_FormatNullableInt(incident.hasPosition, incident.posY, posY, sizeof(posY));
	LPS_FormatNullableInt(incident.hasPosition, incident.posZ, posZ, sizeof(posZ));
	LPS_FormatNullableInt(incident.hasEndPosition, incident.endPosX, endPosX, sizeof(endPosX));
	LPS_FormatNullableInt(incident.hasEndPosition, incident.endPosY, endPosY, sizeof(endPosY));
	LPS_FormatNullableInt(incident.hasEndPosition, incident.endPosZ, endPosZ, sizeof(endPosZ));
	Format(row, maxLength, "('%s',%d,%d,%d,%d,%d,%d,%d,'%s',%d,'%s',%d,'%s',%d,%d,%d,%d,%s,%s,%s,%s,%s,%s)", roundId, incident.sequence, LPS_INCIDENT_VERSION, incident.type, incident.occurredAt, incident.roundOffsetMs, incident.durationMs, incident.actorKind, actorSteamId, incident.targetKind, targetSteamId, incident.helperKind, helperSteamId, incident.infectedClass, incident.endReason, incident.detailFlags, incident.relatedSequence, posX, posY, posZ, endPosX, endPosY, endPosZ);
	return strlen(row) < maxLength - 1;
}

void LPS_FormatNullableInt(bool available, int value, char[] output, int maxLength)
{
	if (available) IntToString(value, output, maxLength);
	else strcopy(output, maxLength, "NULL");
}

public void LPS_OnAnalysisFlushSucceeded(Database database, any data, int queryCount, DBResultSet[] results, any[] queryData)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}
	g_LPSAnalysisFlushInFlight = false;
	for (int index = 0; index < g_LPSAnalysisCapturedIncidentCount && g_LPSPendingIncidents.Length > 0; index++)
	{
		g_LPSPendingIncidents.Erase(0);
	}
	for (int index = 0; index < g_LPSAnalysisCapturedContextCount && g_LPSPendingRoundContexts.Length > 0; index++)
	{
		g_LPSPendingRoundContexts.Erase(0);
	}
	if (g_LPSAnalysisCapturedActiveContext && g_LPSRoundContextActive
		&& g_LPSActiveRoundContext.revision <= g_LPSAnalysisCapturedActiveRevision)
	{
		g_LPSRoundContextDirty = false;
	}
	g_LPSLastAnalysisFlushRows = g_LPSAnalysisCapturedIncidentCount;
	g_LPSLastAnalysisFlushMs = RoundToNearest((GetGameTime() - g_LPSAnalysisFlushStartedGameTime) * 1000.0);
	g_LPSLastAnalysisSuccess = GetTime();
	g_LPSLastAnalysisError[0] = '\0';
	LPS_ResetAnalysisFlushCapture();
	if (LPS_GetPendingIncidentCount() > 0 || LPS_GetPendingRoundContextCount() > 0 || g_LPSRoundContextDirty)
	{
		LPS_ScheduleAnalysisFlush();
	}
}

public void LPS_OnAnalysisFlushFailed(Database database, any data, int queryCount, const char[] error, int failIndex, any[] queryData)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}
	g_LPSAnalysisFlushInFlight = false;
	LPS_RecordAnalysisError("analysis transaction failed; queued facts retained");
	LPS_ResetAnalysisFlushCapture();
}

void LPS_ScheduleAnalysisFlush()
{
	if (!g_LPSAnalysisFlushScheduled)
	{
		g_LPSAnalysisFlushScheduled = true;
		RequestFrame(LPS_Frame_ContinueAnalysisFlush);
	}
}

public void LPS_Frame_ContinueAnalysisFlush(any data)
{
	g_LPSAnalysisFlushScheduled = false;
	LPS_RequestAnalysisFlush();
}

void LPS_RecordAnalysisError(const char[] message)
{
	strcopy(g_LPSLastAnalysisError, sizeof(g_LPSLastAnalysisError), message);
	int now = GetTime();
	if (g_LPSLastAnalysisErrorLogAt > 0 && now - g_LPSLastAnalysisErrorLogAt < g_LPSLogSuppressionWindow.IntValue)
	{
		g_LPSSuppressedAnalysisErrors++;
		return;
	}
	if (g_LPSSuppressedAnalysisErrors > 0)
	{
		LogError("Suppressed %d repeated analysis persistence error(s).", g_LPSSuppressedAnalysisErrors);
	}
	LogError("Analysis persistence failed; core statistics remain committed and queued analysis will retry after a later core success.");
	g_LPSLastAnalysisErrorLogAt = now;
	g_LPSSuppressedAnalysisErrors = 0;
}

void LPS_ResetAnalysisFlushCapture()
{
	g_LPSAnalysisCapturedIncidentCount = 0;
	g_LPSAnalysisCapturedContextCount = 0;
	g_LPSAnalysisCapturedActiveContext = false;
	g_LPSAnalysisCapturedActiveRevision = 0;
	g_LPSAnalysisFlushStartedGameTime = 0.0;
}

void LPS_OnAnalysisConnectionLost()
{
	g_LPSAnalysisFlushInFlight = false;
	g_LPSAnalysisFlushScheduled = false;
	LPS_ResetAnalysisFlushCapture();
}
