#if defined _l4d2_player_stats_runtime_included
	#endinput
#endif
#define _l4d2_player_stats_runtime_included

Database g_LPSDatabase;
LPSDatabaseState g_LPSDatabaseState;
LPSDatabaseDialect g_LPSDatabaseDialect;

Handle g_LPSRetryTimer;
Handle g_LPSFlushTimer;

int g_LPSConnectionGeneration;
int g_LPSRetryAttempt;
int g_LPSNextRetryDelay;
int g_LPSSchemaVersion;
int g_LPSApplyingMigrationVersion;
int g_LPSBootStartedAt;
int g_LPSLastSuccessfulWriteAt;

bool g_LPSHeartbeatInFlight;
bool g_LPSHeartbeatQueued;
bool g_LPSShuttingDown;

char g_LPSDriverName[LPS_MAX_DRIVER_NAME];
char g_LPSServerKey[LPS_MAX_SERVER_KEY];
char g_LPSBootId[LPS_MAX_BOOT_ID];
char g_LPSLastDatabaseError[LPS_MAX_DATABASE_ERROR];

void LPS_ResetRuntime()
{
	g_LPSDatabase = null;
	g_LPSDatabaseState = LPSDatabaseState_Disconnected;
	g_LPSDatabaseDialect = LPSDatabaseDialect_Unknown;
	g_LPSRetryTimer = null;
	g_LPSFlushTimer = null;
	g_LPSConnectionGeneration = 0;
	g_LPSRetryAttempt = 0;
	g_LPSNextRetryDelay = 0;
	g_LPSSchemaVersion = 0;
	g_LPSApplyingMigrationVersion = 0;
	g_LPSBootStartedAt = 0;
	g_LPSLastSuccessfulWriteAt = 0;
	g_LPSHeartbeatInFlight = false;
	g_LPSHeartbeatQueued = false;
	g_LPSShuttingDown = false;
	g_LPSDriverName[0] = '\0';
	g_LPSServerKey[0] = '\0';
	g_LPSBootId[0] = '\0';
	g_LPSLastDatabaseError[0] = '\0';
}
