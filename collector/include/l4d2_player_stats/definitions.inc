#if defined _l4d2_player_stats_definitions_included
	#endinput
#endif
#define _l4d2_player_stats_definitions_included

#define LPS_VERSION "0.3.0"
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
#define LPS_MAX_STEAM_ID 32
#define LPS_MAX_PLAYER_NAME 128
#define LPS_MAX_IP_ADDRESS 46
#define LPS_MAX_SESSION_ID 128
#define LPS_MAX_SESSION_STATUS 16
#define LPS_MAX_DISCONNECT_REASON 32
#define LPS_MAX_GAME_MODE 32
#define LPS_MAX_RUN_ID 128
#define LPS_MAX_ROUND_ID 128
#define LPS_MAX_SEGMENT_ID 128
#define LPS_MAX_MAP_NAME 128
#define LPS_MAX_CAMPAIGN_KEY 128
#define LPS_MAX_LIFECYCLE_STATUS 16
#define LPS_MAX_SIDE_NAME 16
#define LPS_SESSION_TRANSFER_TIMEOUT 120

#define LPS_TEAM_SPECTATOR 1
#define LPS_TEAM_SURVIVOR 2
#define LPS_TEAM_INFECTED 3

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

enum LPSModeFamily
{
	LPSModeFamily_Unsupported = 0,
	LPSModeFamily_PvE,
	LPSModeFamily_Versus
};

enum LPSContinuationKind
{
	LPSContinuation_None = 0,
	LPSContinuation_MapTransition,
	LPSContinuation_Retry,
	LPSContinuation_VersusHalf
};
