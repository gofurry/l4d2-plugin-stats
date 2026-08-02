#if defined _l4d2_player_stats_database_included
	#endinput
#endif
#define _l4d2_player_stats_database_included

void LPS_ApplyConfiguration()
{
	if (!g_LPSEnabled.BoolValue)
	{
		g_LPSDatabaseState = LPSDatabaseState_Disabled;
		return;
	}

	char configuredServerKey[LPS_MAX_SERVER_KEY];
	if (!LPS_GetValidatedServerKey(configuredServerKey, sizeof(configuredServerKey)))
	{
		g_LPSDatabaseState = LPSDatabaseState_ConfigError;
		LogError(
			"sm_lps_server_key must be changed from 'change-me' to a unique 1-64 character key. Collector is inactive."
		);
		return;
	}

	if (g_LPSBootId[0] != '\0' && StrEqual(g_LPSServerKey, configuredServerKey, true))
	{
		LPS_OnCollectorConfigured();
		LPS_StartEligibleSessions();
		return;
	}

	if (g_LPSBootId[0] != '\0')
	{
		LPS_EndRun("abandoned");
		LPS_EndAllActiveSessions("server_key_changed");
	}

	strcopy(g_LPSServerKey, sizeof(g_LPSServerKey), configuredServerKey);
	LPS_GenerateBootId();
	LPS_OnCollectorConfigured();
	LPS_StartEligibleSessions();
	LPS_StartConnection(true);
}

void LPS_GenerateBootId()
{
	g_LPSBootStartedAt = GetTime();
	int randomValue = GetRandomInt(0, 2147483647);
	Format(
		g_LPSBootId,
		sizeof(g_LPSBootId),
		"%s:%d:%08x",
		g_LPSServerKey,
		g_LPSBootStartedAt,
		randomValue
	);
}

void LPS_StartConnection(bool resetBackoff)
{
	if (g_LPSShuttingDown || !g_LPSEnabled.BoolValue)
	{
		return;
	}

	if (resetBackoff)
	{
		g_LPSRetryAttempt = 0;
		g_LPSNextRetryDelay = 0;
	}

	LPS_CancelRetryTimer();
	LPS_CloseDatabaseHandle();
	g_LPSConnectionGeneration++;
	g_LPSDatabaseState = LPSDatabaseState_Connecting;
	g_LPSDatabaseDialect = LPSDatabaseDialect_Unknown;
	g_LPSDriverName[0] = '\0';
	g_LPSSchemaVersion = 0;

	char databaseConfig[LPS_MAX_DATABASE_CONFIG];
	g_LPSDatabaseConfig.GetString(databaseConfig, sizeof(databaseConfig));
	TrimString(databaseConfig);

	if (databaseConfig[0] == '\0')
	{
		g_LPSDatabaseState = LPSDatabaseState_ConfigError;
		LogError("sm_lps_database_config cannot be empty. Collector is inactive.");
		return;
	}

	LogMessage("Connecting with SourceMod database configuration '%s'.", databaseConfig);
	Database.Connect(LPS_OnDatabaseConnected, databaseConfig, g_LPSConnectionGeneration);
}

public void LPS_OnDatabaseConnected(Database database, const char[] error, any data)
{
	if (data != g_LPSConnectionGeneration || g_LPSShuttingDown)
	{
		if (database != null)
		{
			delete database;
		}
		return;
	}

	if (database == null)
	{
		LPS_HandleDatabaseFailure("connection failed", error);
		return;
	}

	g_LPSDatabase = database;
	g_LPSDatabase.Driver.GetIdentifier(g_LPSDriverName, sizeof(g_LPSDriverName));

	if (StrEqual(g_LPSDriverName, "sqlite", false))
	{
		g_LPSDatabaseDialect = LPSDatabaseDialect_SQLite;
	}
	else if (StrEqual(g_LPSDriverName, "mysql", false))
	{
		g_LPSDatabaseDialect = LPSDatabaseDialect_MySQL;
	}
	else if (StrEqual(g_LPSDriverName, "pgsql", false))
	{
		g_LPSDatabaseDialect = LPSDatabaseDialect_PostgreSQL;
	}
	else
	{
		char unsupportedError[128];
		Format(unsupportedError, sizeof(unsupportedError), "unsupported driver '%s'", g_LPSDriverName);
		g_LPSDatabaseState = LPSDatabaseState_ConfigError;
		LPS_ReportDatabaseError("configuration error", unsupportedError);
		LPS_CloseDatabaseHandle();
		return;
	}

	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		g_LPSDatabase.SetCharset("utf8mb4");
	}
	else if (g_LPSDatabaseDialect == LPSDatabaseDialect_PostgreSQL)
	{
		g_LPSDatabase.SetCharset("utf8");
	}

	LogMessage("Connected using SourceMod database driver '%s'; checking schema.", g_LPSDriverName);
	LPS_StartMigrationCheck(data);
}

void LPS_StartMigrationCheck(int generation)
{
	g_LPSDatabaseState = LPSDatabaseState_Migrating;

	char query[LPS_MAX_QUERY_LENGTH];
	char error[LPS_MAX_DATABASE_ERROR];
	if (!LPS_ReadBootstrapStatement(query, sizeof(query), error, sizeof(error)))
	{
		LPS_HandleDatabaseFailure("migration bootstrap could not be loaded", error);
		return;
	}

	g_LPSDatabase.Query(LPS_OnMigrationTableReady, query, generation, DBPrio_High);
}

public void LPS_OnMigrationTableReady(Database database, DBResultSet results, const char[] error, any data)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	if (results == null)
	{
		LPS_HandleDatabaseFailure("migration table bootstrap failed", error);
		return;
	}

	g_LPSDatabase.Query(
		LPS_OnMigrationVersionLoaded,
		"SELECT version FROM lps_schema_migrations ORDER BY version DESC LIMIT 1",
		data,
		DBPrio_High
	);
}

public void LPS_OnMigrationVersionLoaded(Database database, DBResultSet results, const char[] error, any data)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	if (results == null)
	{
		LPS_HandleDatabaseFailure("schema version query failed", error);
		return;
	}

	g_LPSSchemaVersion = 0;
	if (results.FetchRow())
	{
		g_LPSSchemaVersion = results.FetchInt(0);
	}

	if (g_LPSSchemaVersion >= LPS_SCHEMA_VERSION)
	{
		if (g_LPSSchemaVersion > LPS_SCHEMA_VERSION)
		{
			LogMessage(
				"Database schema version %d is newer than collector schema version %d; continuing without downgrade.",
				g_LPSSchemaVersion,
				LPS_SCHEMA_VERSION
			);
		}

		LPS_RegisterServerBoot(data);
		return;
	}

	char migrationError[LPS_MAX_DATABASE_ERROR];
	int statementCount;
	Transaction migration = LPS_LoadInitialMigration(
		migrationError,
		sizeof(migrationError),
		statementCount
	);

	if (migration == null)
	{
		LPS_HandleDatabaseFailure("initial migration could not be loaded", migrationError);
		return;
	}

	LogMessage("Applying schema migration 0001 (%d SQL statements).", statementCount);
	g_LPSDatabase.Execute(
		migration,
		LPS_OnMigrationSucceeded,
		LPS_OnMigrationFailed,
		data,
		DBPrio_High
	);
}

public void LPS_OnMigrationSucceeded(
	Database database,
	any data,
	int queryCount,
	DBResultSet[] results,
	any[] queryData
)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	g_LPSSchemaVersion = LPS_SCHEMA_VERSION;
	LogMessage("Schema migration 0001 completed successfully.");
	LPS_RegisterServerBoot(data);
}

public void LPS_OnMigrationFailed(
	Database database,
	any data,
	int queryCount,
	const char[] error,
	int failIndex,
	any[] queryData
)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	char context[128];
	Format(context, sizeof(context), "schema migration 0001 failed at statement %d", failIndex + 1);
	LPS_HandleDatabaseFailure(context, error);
}

void LPS_RegisterServerBoot(int generation)
{
	g_LPSDatabaseState = LPSDatabaseState_Registering;

	char escapedServerKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char escapedBootId[(LPS_MAX_BOOT_ID * 2) + 1];
	char displayName[LPS_MAX_SERVER_NAME];
	char escapedDisplayName[(LPS_MAX_SERVER_NAME * 2) + 1];

	LPS_GetServerDisplayName(displayName, sizeof(displayName));
	if (!g_LPSDatabase.Escape(g_LPSServerKey, escapedServerKey, sizeof(escapedServerKey))
		|| !g_LPSDatabase.Escape(g_LPSBootId, escapedBootId, sizeof(escapedBootId))
		|| !g_LPSDatabase.Escape(displayName, escapedDisplayName, sizeof(escapedDisplayName)))
	{
		LPS_HandleDatabaseFailure("server registration failed", "a dynamic value exceeded its SQL escape buffer");
		return;
	}

	Transaction transaction = new Transaction();
	char query[2048];

	Format(
		query,
		sizeof(query),
		"UPDATE lps_player_segments SET status = 'abandoned', ended_at = last_saved_at WHERE status = 'active' AND session_id IN (SELECT session_id FROM lps_sessions WHERE server_key = '%s' AND boot_id <> '%s')",
		escapedServerKey,
		escapedBootId
	);
	transaction.AddQuery(query);

	Format(
		query,
		sizeof(query),
		"UPDATE lps_rounds SET status = 'abandoned', ended_at = last_saved_at WHERE status = 'active' AND run_id IN (SELECT run_id FROM lps_runs WHERE server_key = '%s' AND boot_id <> '%s')",
		escapedServerKey,
		escapedBootId
	);
	transaction.AddQuery(query);

	Format(
		query,
		sizeof(query),
		"UPDATE lps_sessions SET status = 'abandoned', ended_at = last_saved_at WHERE server_key = '%s' AND boot_id <> '%s' AND status = 'active'",
		escapedServerKey,
		escapedBootId
	);
	transaction.AddQuery(query);

	Format(
		query,
		sizeof(query),
		"UPDATE lps_runs SET status = 'abandoned', ended_at = last_saved_at WHERE server_key = '%s' AND boot_id <> '%s' AND status = 'active'",
		escapedServerKey,
		escapedBootId
	);
	transaction.AddQuery(query);

	Format(
		query,
		sizeof(query),
		"UPDATE lps_server_boots SET status = 'abandoned', ended_at = last_heartbeat_at WHERE server_key = '%s' AND boot_id <> '%s' AND status = 'active'",
		escapedServerKey,
		escapedBootId
	);
	transaction.AddQuery(query);

	LPS_BuildServerUpsertQuery(
		query,
		sizeof(query),
		escapedServerKey,
		escapedDisplayName,
		GetTime()
	);
	transaction.AddQuery(query);

	LPS_BuildBootUpsertQuery(
		query,
		sizeof(query),
		escapedBootId,
		escapedServerKey,
		g_LPSBootStartedAt,
		GetTime()
	);
	transaction.AddQuery(query);

	g_LPSDatabase.Execute(
		transaction,
		LPS_OnServerBootRegistered,
		LPS_OnServerBootRegistrationFailed,
		generation,
		DBPrio_High
	);
}

void LPS_BuildServerUpsertQuery(
	char[] query,
	int maxLength,
	const char[] serverKey,
	const char[] displayName,
	int now
)
{
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_servers (server_key, display_name, first_seen_at, last_seen_at) VALUES ('%s', '%s', %d, %d) ON DUPLICATE KEY UPDATE display_name = VALUES(display_name), last_seen_at = VALUES(last_seen_at)",
			serverKey,
			displayName,
			now,
			now
		);
	}
	else
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_servers (server_key, display_name, first_seen_at, last_seen_at) VALUES ('%s', '%s', %d, %d) ON CONFLICT(server_key) DO UPDATE SET display_name = excluded.display_name, last_seen_at = excluded.last_seen_at",
			serverKey,
			displayName,
			now,
			now
		);
	}
}

void LPS_BuildBootUpsertQuery(
	char[] query,
	int maxLength,
	const char[] bootId,
	const char[] serverKey,
	int startedAt,
	int heartbeatAt
)
{
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_server_boots (boot_id, server_key, started_at, ended_at, last_heartbeat_at, status) VALUES ('%s', '%s', %d, NULL, %d, 'active') ON DUPLICATE KEY UPDATE last_heartbeat_at = VALUES(last_heartbeat_at), ended_at = NULL, status = 'active'",
			bootId,
			serverKey,
			startedAt,
			heartbeatAt
		);
	}
	else
	{
		Format(
			query,
			maxLength,
			"INSERT INTO lps_server_boots (boot_id, server_key, started_at, ended_at, last_heartbeat_at, status) VALUES ('%s', '%s', %d, NULL, %d, 'active') ON CONFLICT(boot_id) DO UPDATE SET last_heartbeat_at = excluded.last_heartbeat_at, ended_at = NULL, status = 'active'",
			bootId,
			serverKey,
			startedAt,
			heartbeatAt
		);
	}
}

public void LPS_OnServerBootRegistered(
	Database database,
	any data,
	int queryCount,
	DBResultSet[] results,
	any[] queryData
)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	g_LPSDatabaseState = LPSDatabaseState_Ready;
	g_LPSRetryAttempt = 0;
	g_LPSNextRetryDelay = 0;
	g_LPSLastSuccessfulWriteAt = GetTime();
	LPS_ReportDatabaseRecovery();
	LPS_ScheduleNextFlush();
	LPS_OnLifecycleDatabaseReady();
	LPS_OnSessionDatabaseReady();

	if (g_LPSHeartbeatQueued)
	{
		LPS_RequestFlush();
	}

	LogMessage(
		"Database foundation is ready (driver=%s, schema=%d, server_key=%s).",
		g_LPSDriverName,
		g_LPSSchemaVersion,
		g_LPSServerKey
	);
}

public void LPS_OnServerBootRegistrationFailed(
	Database database,
	any data,
	int queryCount,
	const char[] error,
	int failIndex,
	any[] queryData
)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	LPS_HandleDatabaseFailure("server boot registration failed", error);
}

void LPS_RequestFlush()
{
	if (g_LPSDatabaseState != LPSDatabaseState_Ready || g_LPSDatabase == null)
	{
		g_LPSHeartbeatQueued = true;
		return;
	}

	if (g_LPSHeartbeatInFlight)
	{
		g_LPSHeartbeatQueued = true;
		return;
	}

	g_LPSHeartbeatInFlight = true;
	g_LPSHeartbeatQueued = false;

	char escapedServerKey[(LPS_MAX_SERVER_KEY * 2) + 1];
	char escapedBootId[(LPS_MAX_BOOT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(g_LPSServerKey, escapedServerKey, sizeof(escapedServerKey))
		|| !g_LPSDatabase.Escape(g_LPSBootId, escapedBootId, sizeof(escapedBootId)))
	{
		g_LPSHeartbeatInFlight = false;
		LPS_HandleDatabaseFailure("heartbeat failed", "a dynamic value exceeded its SQL escape buffer");
		return;
	}

	int now = GetTime();
	Transaction transaction = new Transaction();
	char query[512];
	int queryCount = 0;

	Format(
		query,
		sizeof(query),
		"UPDATE lps_servers SET last_seen_at = %d WHERE server_key = '%s'",
		now,
		escapedServerKey
	);
	transaction.AddQuery(query);
	queryCount++;

	Format(
		query,
		sizeof(query),
		"UPDATE lps_server_boots SET last_heartbeat_at = %d WHERE boot_id = '%s' AND status = 'active'",
		now,
		escapedBootId
	);
	transaction.AddQuery(query);
	queryCount++;

	char sessionError[LPS_MAX_DATABASE_ERROR];
	if (!LPS_AddSessionFlushQueries(transaction, queryCount, sessionError, sizeof(sessionError)))
	{
		delete transaction;
		g_LPSHeartbeatInFlight = false;
		g_LPSHeartbeatQueued = true;
		LPS_ReportDatabaseError("flush query build failed", sessionError);
		return;
	}

	char lifecycleError[LPS_MAX_DATABASE_ERROR];
	if (!LPS_AddLifecycleFlushQueries(transaction, queryCount, lifecycleError, sizeof(lifecycleError))
		|| !LPS_AddSegmentFlushQueries(transaction, queryCount, lifecycleError, sizeof(lifecycleError))
		|| !LPS_AddPvEStatsFlushQueries(transaction, queryCount, lifecycleError, sizeof(lifecycleError)))
	{
		delete transaction;
		g_LPSHeartbeatInFlight = false;
		g_LPSHeartbeatQueued = true;
		LPS_ReportDatabaseError("flush query build failed", lifecycleError);
		return;
	}

	g_LPSDatabase.Execute(
		transaction,
		LPS_OnFlushSucceeded,
		LPS_OnFlushFailed,
		g_LPSConnectionGeneration,
		DBPrio_Low
	);
}

public void LPS_OnFlushSucceeded(
	Database database,
	any data,
	int queryCount,
	DBResultSet[] results,
	any[] queryData
)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	g_LPSHeartbeatInFlight = false;
	g_LPSLastSuccessfulWriteAt = GetTime();
	LPS_OnSessionFlushSucceeded();
	LPS_OnLifecycleFlushSucceeded();
	LPS_OnSegmentFlushSucceeded();
	LPS_OnPvEStatsFlushSucceeded();

	if (g_LPSHeartbeatQueued)
	{
		LPS_RequestFlush();
	}
}

public void LPS_OnFlushFailed(
	Database database,
	any data,
	int queryCount,
	const char[] error,
	int failIndex,
	any[] queryData
)
{
	if (!LPS_IsCurrentGeneration(data))
	{
		return;
	}

	g_LPSHeartbeatInFlight = false;
	// A driver may include parts of a rejected query in its error text. Session
	// queries contain private IP data, so never forward this raw error to logs.
	LPS_HandleDatabaseFailure(
		"flush transaction failed",
		"the database rejected the asynchronous collector flush"
	);
}

void LPS_ScheduleNextFlush()
{
	LPS_CancelFlushTimer();

	int baseInterval = g_LPSFlushInterval.IntValue;
	int jitterMaximum = g_LPSFlushJitter.IntValue;
	int jitter = jitterMaximum > 0 ? GetRandomInt(0, jitterMaximum) : 0;
	g_LPSFlushTimer = CreateTimer(float(baseInterval + jitter), LPS_Timer_Flush);
}

public Action LPS_Timer_Flush(Handle timer)
{
	g_LPSFlushTimer = null;
	LPS_RequestFlush();
	LPS_ScheduleNextFlush();
	return Plugin_Stop;
}

void LPS_HandleDatabaseFailure(const char[] context, const char[] error)
{
	if (g_LPSShuttingDown)
	{
		return;
	}

	LPS_ReportDatabaseError(context, error);
	g_LPSDatabaseState = LPSDatabaseState_RetryWait;
	g_LPSHeartbeatInFlight = false;
	g_LPSConnectionGeneration++;
	LPS_CloseDatabaseHandle();
	LPS_CancelFlushTimer();
	LPS_ScheduleReconnect();
}

void LPS_ScheduleReconnect()
{
	if (g_LPSRetryTimer != null || g_LPSShuttingDown)
	{
		return;
	}

	int delays[4] = {5, 15, 30, 60};
	int index = g_LPSRetryAttempt;
	if (index > 3)
	{
		index = 3;
	}

	g_LPSNextRetryDelay = delays[index];
	int maximum = g_LPSRetryMaximum.IntValue;
	if (g_LPSNextRetryDelay > maximum)
	{
		g_LPSNextRetryDelay = maximum;
	}

	g_LPSRetryAttempt++;
	g_LPSRetryTimer = CreateTimer(float(g_LPSNextRetryDelay), LPS_Timer_Reconnect);
}

public Action LPS_Timer_Reconnect(Handle timer)
{
	g_LPSRetryTimer = null;
	g_LPSNextRetryDelay = 0;
	LPS_StartConnection(false);
	return Plugin_Stop;
}

bool LPS_IsCurrentGeneration(any generation)
{
	return !g_LPSShuttingDown
		&& generation == g_LPSConnectionGeneration
		&& g_LPSDatabase != null;
}

void LPS_CloseDatabaseHandle()
{
	if (g_LPSDatabase != null)
	{
		delete g_LPSDatabase;
		g_LPSDatabase = null;
	}
}

void LPS_CancelRetryTimer()
{
	if (g_LPSRetryTimer != null)
	{
		delete g_LPSRetryTimer;
		g_LPSRetryTimer = null;
	}
}

void LPS_CancelFlushTimer()
{
	if (g_LPSFlushTimer != null)
	{
		delete g_LPSFlushTimer;
		g_LPSFlushTimer = null;
	}
}

void LPS_ShutdownRuntime()
{
	g_LPSShuttingDown = true;
	LPS_CancelRetryTimer();
	LPS_CancelFlushTimer();
	LPS_CloseDatabaseHandle();
}

void LPS_GetDatabaseStateName(char[] buffer, int maxLength)
{
	switch (g_LPSDatabaseState)
	{
		case LPSDatabaseState_Disabled: strcopy(buffer, maxLength, "disabled");
		case LPSDatabaseState_ConfigError: strcopy(buffer, maxLength, "configuration-error");
		case LPSDatabaseState_Disconnected: strcopy(buffer, maxLength, "disconnected");
		case LPSDatabaseState_Connecting: strcopy(buffer, maxLength, "connecting");
		case LPSDatabaseState_Migrating: strcopy(buffer, maxLength, "migrating");
		case LPSDatabaseState_Registering: strcopy(buffer, maxLength, "registering-server");
		case LPSDatabaseState_Ready: strcopy(buffer, maxLength, "ready");
		case LPSDatabaseState_RetryWait: strcopy(buffer, maxLength, "waiting-to-retry");
		default: strcopy(buffer, maxLength, "unknown");
	}
}
