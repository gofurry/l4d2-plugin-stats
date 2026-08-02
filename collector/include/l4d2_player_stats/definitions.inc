#if defined _l4d2_player_stats_definitions_included
	#endinput
#endif
#define _l4d2_player_stats_definitions_included

#define LPS_VERSION "0.1.0"
#define LPS_SCHEMA_VERSION 1
#define LPS_MIGRATION_NAME "initial_schema"

#define LPS_MAX_DATABASE_CONFIG 64
#define LPS_MAX_SERVER_KEY 65
#define LPS_MAX_SERVER_NAME 129
#define LPS_MAX_BOOT_ID 128
#define LPS_MAX_DRIVER_NAME 16
#define LPS_MAX_DATABASE_ERROR 256
#define LPS_MAX_QUERY_LENGTH 8192
#define LPS_MAX_MIGRATION_LINE 1024

enum LPSDatabaseState
{
	LPSDatabaseState_Disabled = 0,
	LPSDatabaseState_ConfigError,
	LPSDatabaseState_Disconnected,
	LPSDatabaseState_Connecting,
	LPSDatabaseState_Migrating,
	LPSDatabaseState_Registering,
	LPSDatabaseState_Ready,
	LPSDatabaseState_RetryWait
};

enum LPSDatabaseDialect
{
	LPSDatabaseDialect_Unknown = 0,
	LPSDatabaseDialect_SQLite,
	LPSDatabaseDialect_MySQL,
	LPSDatabaseDialect_PostgreSQL
};
