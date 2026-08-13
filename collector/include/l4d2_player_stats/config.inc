#if defined _l4d2_player_stats_config_included
	#endinput
#endif
#define _l4d2_player_stats_config_included

ConVar g_LPSEnabled;
ConVar g_LPSDatabaseConfig;
ConVar g_LPSServerKeyCvar;
ConVar g_LPSServerName;
ConVar g_LPSFlushInterval;
ConVar g_LPSFlushJitter;
ConVar g_LPSRetryMaximum;
ConVar g_LPSLogSuppressionWindow;
ConVar g_LPSClosedSessionQueueLimit;
ConVar g_LPSClosedLifecycleQueueLimit;
ConVar g_LPSClosedEquipmentQueueLimit;
ConVar g_LPSVersusStatsEnabled;
ConVar g_LPSIncidentsEnabled;
ConVar g_LPSIncidentQueueLimit;
ConVar g_LPSRulesetName;

void LPS_CreateConfig()
{
	g_LPSEnabled = CreateConVar(
		"sm_lps_enabled",
		"1",
		"Enable the L4D2 Player Stats collector.",
		FCVAR_NONE,
		true,
		0.0,
		true,
		1.0
	);

	g_LPSDatabaseConfig = CreateConVar(
		"sm_lps_database_config",
		"l4d2_player_stats",
		"Named database entry in addons/sourcemod/configs/databases.cfg."
	);

	g_LPSServerKeyCvar = CreateConVar(
		"sm_lps_server_key",
		"change-me",
		"Globally unique server key (1-64 characters: A-Z, a-z, 0-9, dot, underscore, hyphen).",
		FCVAR_NOTIFY
	);

	g_LPSServerName = CreateConVar(
		"sm_lps_server_name",
		"",
		"Display name stored for this server. Empty uses hostname."
	);

	g_LPSFlushInterval = CreateConVar(
		"sm_lps_flush_interval",
		"300",
		"Base database flush interval in seconds.",
		FCVAR_NONE,
		true,
		60.0,
		true,
		3600.0
	);

	g_LPSFlushJitter = CreateConVar(
		"sm_lps_flush_jitter",
		"60",
		"Maximum random seconds added to each flush interval.",
		FCVAR_NONE,
		true,
		0.0,
		true,
		600.0
	);

	g_LPSRetryMaximum = CreateConVar(
		"sm_lps_retry_maximum",
		"60",
		"Maximum database reconnect delay in seconds.",
		FCVAR_NONE,
		true,
		15.0,
		true,
		600.0
	);

	g_LPSLogSuppressionWindow = CreateConVar(
		"sm_lps_log_suppression_window",
		"300",
		"Seconds before repeating equivalent database errors in SourceMod logs.",
		FCVAR_NONE,
		true,
		60.0,
		true,
		3600.0
	);

	g_LPSClosedSessionQueueLimit = CreateConVar(
		"sm_lps_closed_session_queue_limit",
		"256",
		"Maximum closed Session snapshots retained in memory while the database is unavailable.",
		FCVAR_NONE,
		true,
		16.0,
		true,
		4096.0
	);

	g_LPSClosedLifecycleQueueLimit = CreateConVar(
		"sm_lps_closed_lifecycle_queue_limit",
		"512",
		"Maximum total closed Run, Round, and Segment snapshots retained while the database is unavailable.",
		FCVAR_NONE,
		true,
		32.0,
		true,
		8192.0
	);

	g_LPSClosedEquipmentQueueLimit = CreateConVar(
		"sm_lps_closed_equipment_queue_limit",
		"4096",
		"Maximum closed per-equipment snapshots retained while the database is unavailable.",
		FCVAR_NONE,
		true,
		128.0,
		true,
		65536.0
	);

	g_LPSVersusStatsEnabled = CreateConVar(
		"sm_lps_versus_stats_enabled",
		"1",
		"Collect Versus survivor and infected gameplay statistics. Takes effect for new Segments.",
		FCVAR_NONE,
		true,
		0.0,
		true,
		1.0
	);

	g_LPSIncidentsEnabled = CreateConVar(
		"sm_lps_incidents_enabled",
		"1",
		"Collect low-frequency Incident Contract v1 analysis facts. Latched at Round start.",
		FCVAR_NONE,
		true,
		0.0,
		true,
		1.0
	);

	g_LPSIncidentQueueLimit = CreateConVar(
		"sm_lps_incident_queue_limit",
		"2048",
		"Maximum pending Incident rows retained in memory; overflow drops analysis only.",
		FCVAR_NONE,
		true,
		256.0,
		true,
		16384.0
	);

	g_LPSRulesetName = CreateConVar(
		"sm_lps_ruleset_name",
		"",
		"Optional operator label stored in Round Context v1 (maximum 64 characters)."
	);
}

bool LPS_GetValidatedServerKey(char[] serverKey, int maxLength)
{
	g_LPSServerKeyCvar.GetString(serverKey, maxLength);
	TrimString(serverKey);

	int length = strlen(serverKey);
	if (length < 1 || length > 64 || StrEqual(serverKey, "change-me", false))
	{
		return false;
	}

	for (int index = 0; index < length; index++)
	{
		int value = serverKey[index];
		bool alphaNumeric = (value >= 'A' && value <= 'Z')
			|| (value >= 'a' && value <= 'z')
			|| (value >= '0' && value <= '9');

		if (!alphaNumeric && value != '.' && value != '_' && value != '-')
		{
			return false;
		}
	}

	return true;
}

void LPS_GetServerDisplayName(char[] displayName, int maxLength)
{
	g_LPSServerName.GetString(displayName, maxLength);
	TrimString(displayName);

	if (displayName[0] != '\0')
	{
		return;
	}

	ConVar hostname = FindConVar("hostname");
	if (hostname != null)
	{
		hostname.GetString(displayName, maxLength);
		TrimString(displayName);
	}

	if (displayName[0] == '\0')
	{
		strcopy(displayName, maxLength, "Left 4 Dead 2 Server");
	}
}
