#include <sourcemod>
#include <sdkhooks>
#include <sdktools>

#pragma semicolon 1
#pragma newdecls required

#include <l4d2_player_stats/definitions>
#include <l4d2_player_stats/config>
#include <l4d2_player_stats/runtime>
#include <l4d2_player_stats/logging>
#include <l4d2_player_stats/modes>
#include <l4d2_player_stats/lifecycles>
#include <l4d2_player_stats/versus_results>
#include <l4d2_player_stats/sessions>
#include <l4d2_player_stats/segments>
#include <l4d2_player_stats/relationship_stats>
#include <l4d2_player_stats/assist_stats>
#include <l4d2_player_stats/round_context>
#include <l4d2_player_stats/incidents>
#include <l4d2_player_stats/pve_stats>
#include <l4d2_player_stats/versus_stats>
#include <l4d2_player_stats/versus_abilities>
#include <l4d2_player_stats/equipment_stats>
#include <l4d2_player_stats/pve_extended>
#include <l4d2_player_stats/versus_survivor_detail>
#include <l4d2_player_stats/pve_interactions>
#include <l4d2_player_stats/survivor_techniques>
#include <l4d2_player_stats/survivor_incidents>
#include <l4d2_player_stats/analysis_persistence>
#include <l4d2_player_stats/chat_audit>
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
	LPS_InitializeVersusResults();
	LPS_InitializeLifecycles();
	LPS_InitializeSegments();
	LPS_InitializeRelationships();
	LPS_InitializeAssistStats();
	LPS_InitializeRoundContext();
	LPS_InitializeIncidents();
	LPS_InitializeAnalysisPersistence();
	LPS_InitializeSessions();
	LPS_InitializePvEStats();
	LPS_InitializeVersusStats();
	LPS_InitializeVersusAbilities();
	LPS_InitializeEquipmentStats();
	LPS_InitializeSurvivorTechniques();
	LPS_InitializeExtendedPvEStats();
	LPS_InitializePvEInteractions();
	LPS_InitializeSurvivorIncidents();
	LPS_InitializeChatAudit();
	LPS_RegisterAdminCommands();
	AutoExecConfig(true, "l4d2_player_stats");

	LogMessage("Version %s loaded; waiting for SourceMod configuration.", LPS_VERSION);
}

public void OnConfigsExecuted()
{
	LPS_ApplyConfiguration();
}

public void OnGameFrame()
{
	LPS_OnSurvivorTechniqueGameFrame();
}

public void OnPluginEnd()
{
	LPS_ShutdownLifecycles();
	LPS_ShutdownVersusResults();
	LPS_ShutdownSurvivorIncidents();
	LPS_ShutdownChatAudit();
	LPS_ShutdownPvEInteractions();
	LPS_ShutdownExtendedPvEStats();
	LPS_ShutdownEquipmentStats();
	LPS_ShutdownSurvivorTechniques();
	LPS_ShutdownVersusAbilities();
	LPS_ShutdownVersusStats();
	LPS_ShutdownPvEStats();
	LPS_ShutdownRelationships();
	LPS_ShutdownSessions();
	LPS_ShutdownRuntime();
}
