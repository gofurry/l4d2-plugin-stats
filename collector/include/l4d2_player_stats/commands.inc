#if defined _l4d2_player_stats_commands_included
	#endinput
#endif
#define _l4d2_player_stats_commands_included

void LPS_RegisterAdminCommands()
{
	RegAdminCmd(
		"sm_lps_status",
		LPS_Command_Status,
		ADMFLAG_ROOT,
		"Show collector database and schema status."
	);
	RegAdminCmd(
		"sm_lps_reconnect",
		LPS_Command_Reconnect,
		ADMFLAG_ROOT,
		"Reconnect the collector database immediately."
	);
	RegAdminCmd(
		"sm_lps_flush",
		LPS_Command_Flush,
		ADMFLAG_ROOT,
		"Request an immediate collector flush/heartbeat."
	);
}

public Action LPS_Command_Status(int client, int args)
{
	char state[32];
	char databaseConfig[LPS_MAX_DATABASE_CONFIG];
	LPS_GetDatabaseStateName(state, sizeof(state));
	g_LPSDatabaseConfig.GetString(databaseConfig, sizeof(databaseConfig));

	ReplyToCommand(client, "[LPS] version=%s state=%s enabled=%d", LPS_VERSION, state, g_LPSEnabled.BoolValue);
	ReplyToCommand(
		client,
		"[LPS] database_config=%s driver=%s schema=%d/%d",
		databaseConfig,
		g_LPSDriverName[0] == '\0' ? "unknown" : g_LPSDriverName,
		g_LPSSchemaVersion,
		LPS_SCHEMA_VERSION
	);
	ReplyToCommand(
		client,
		"[LPS] server_key=%s boot_id=%s",
		g_LPSServerKey[0] == '\0' ? "invalid/not-loaded" : g_LPSServerKey,
		g_LPSBootId[0] == '\0' ? "not-created" : g_LPSBootId
	);
	ReplyToCommand(
		client,
		"[LPS] retry_attempt=%d next_retry=%ds last_success=%d",
		g_LPSRetryAttempt,
		g_LPSNextRetryDelay,
		g_LPSLastSuccessfulWriteAt
	);

	if (g_LPSLastDatabaseError[0] != '\0')
	{
		ReplyToCommand(client, "[LPS] last_database_error=%s", g_LPSLastDatabaseError);
	}

	return Plugin_Handled;
}

public Action LPS_Command_Reconnect(int client, int args)
{
	if (!g_LPSEnabled.BoolValue)
	{
		ReplyToCommand(client, "[LPS] Collector is disabled by sm_lps_enabled.");
		return Plugin_Handled;
	}

	if (!LPS_GetValidatedServerKey(g_LPSServerKey, sizeof(g_LPSServerKey)))
	{
		g_LPSDatabaseState = LPSDatabaseState_ConfigError;
		ReplyToCommand(client, "[LPS] Configure a unique sm_lps_server_key before reconnecting.");
		return Plugin_Handled;
	}

	if (g_LPSBootId[0] == '\0')
	{
		LPS_GenerateBootId();
	}

	LogAction(client, -1, "[LPS] Administrator requested a database reconnect.");
	ReplyToCommand(client, "[LPS] Database reconnect requested.");
	LPS_StartConnection(true);
	return Plugin_Handled;
}

public Action LPS_Command_Flush(int client, int args)
{
	if (g_LPSDatabaseState != LPSDatabaseState_Ready)
	{
		ReplyToCommand(client, "[LPS] Database is not ready; flush request has been queued.");
		g_LPSHeartbeatQueued = true;
		return Plugin_Handled;
	}

	LogAction(client, -1, "[LPS] Administrator requested an immediate database flush.");
	ReplyToCommand(client, "[LPS] Immediate flush requested.");
	LPS_RequestFlush();
	return Plugin_Handled;
}
