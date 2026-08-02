#include <sourcemod>

#pragma semicolon 1
#pragma newdecls required

#include <l4d2_player_stats/definitions>
#include <l4d2_player_stats/config>
#include <l4d2_player_stats/runtime>
#include <l4d2_player_stats/logging>
#include <l4d2_player_stats/modes>
#include <l4d2_player_stats/sessions>
#include <l4d2_player_stats/migrations>
#include <l4d2_player_stats/database>
#include <l4d2_player_stats/commands>

public Plugin myinfo =
{
	name = "L4D2 Player Stats",
	author = "gofurry",
	description = "Persistent L4D2 player identity, session, and gameplay statistics.",
	version = LPS_VERSION,
	url = "https://github.com/gofurry/l4d2-plugin-stats"
};

public APLRes AskPluginLoad2(Handle myself, bool late, char[] error, int errorLength)
{
	if (GetEngineVersion() != Engine_Left4Dead2)
	{
		strcopy(error, errorLength, "L4D2 Player Stats only supports Left 4 Dead 2.");
		return APLRes_SilentFailure;
	}

	return APLRes_Success;
}

public void OnPluginStart()
{
	LPS_CreateConfig();
	LPS_ResetRuntime();
	LPS_InitializeModes();
	LPS_InitializeSessions();
	LPS_RegisterAdminCommands();
	AutoExecConfig(true, "l4d2_player_stats");

	LogMessage("Version %s loaded; waiting for SourceMod configuration.", LPS_VERSION);
}

public void OnConfigsExecuted()
{
	LPS_ApplyConfiguration();
}

public void OnPluginEnd()
{
	LPS_ShutdownSessions();
	LPS_ShutdownRuntime();
}
