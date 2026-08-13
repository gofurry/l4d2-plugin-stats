#if defined _l4d2_player_stats_pve_stats_included
	#endinput
#endif
#define _l4d2_player_stats_pve_stats_included

#include <l4d2_player_stats/pve_stats_state>
#include <l4d2_player_stats/pve_damage>
#include <l4d2_player_stats/pve_combat>
#include <l4d2_player_stats/pve_support>
#include <l4d2_player_stats/pve_persistence>

void LPS_InitializePvEStats()
{
	g_LPSClosedPvEStats = new ArrayList(sizeof(LPSPvEStatsSnapshot));
	g_LPSPainPillsDecayRate = FindConVar("pain_pills_decay_rate");
	g_LPSPvEStatsQueueOverflowReported = false;
	g_LPSFlushClosedPvEStatsCount = 0;

	HookEvent("infected_death", LPS_Event_InfectedDeathStats, EventHookMode_Post);
	HookEvent("player_death", LPS_Event_PlayerDeathStats, EventHookMode_Post);
	HookEvent("witch_killed", LPS_Event_WitchKilledStats, EventHookMode_Post);
	HookEvent("player_incapacitated", LPS_Event_PlayerIncapacitatedStats, EventHookMode_Post);
	HookEvent("revive_success", LPS_Event_ReviveSuccessStats, EventHookMode_Post);
	HookEvent("defibrillator_used", LPS_Event_DefibrillatorUsedStats, EventHookMode_Post);
	HookEvent("heal_success", LPS_Event_HealSuccessStats, EventHookMode_Post);
	HookEvent("pills_used", LPS_Event_PillsUsedStats, EventHookMode_Post);
	HookEvent("adrenaline_used", LPS_Event_AdrenalineUsedStats, EventHookMode_Post);

	for (int entity = 0; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSDamageHooked[entity] = false;
		g_LPSDamageHealthCaptured[entity] = false;
		g_LPSDamageVictimWasIncapacitated[entity] = false;
		g_LPSDamageHealthBefore[entity] = 0.0;
		if (entity <= MaxClients)
		{
			g_LPSClientEffectiveHealthBeforeCommand[entity] = 0.0;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientPvEStatsState(client);
		if (IsClientInGame(client))
		{
			LPS_HookStatsClient(client);
		}
	}

	int entity = -1;
	while ((entity = FindEntityByClassname(entity, "witch")) != -1)
	{
		LPS_HookStatsDamageEntity(entity);
	}
	entity = -1;
	while ((entity = FindEntityByClassname(entity, "witch_bride")) != -1)
	{
		LPS_HookStatsDamageEntity(entity);
	}

	LPS_ResetPvEStatsFlushCapture();
}

public void OnEntityCreated(int entity, const char[] classname)
{
	LPS_OnEquipmentEntityCreated(entity, classname);
	LPS_OnExtendedEntityCreated(entity, classname);
	LPS_OnInteractionEntityCreated(entity);
	LPS_OnVersusAbilityEntityCreated(entity, classname);

	if (StrEqual(classname, "witch", true) || StrEqual(classname, "witch_bride", true))
	{
		LPS_HookStatsDamageEntity(entity);
	}
}

public void OnEntityDestroyed(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	LPS_OnEquipmentEntityDestroyed(entity);
	LPS_OnExtendedEntityDestroyed(entity);
	LPS_OnInteractionEntityDestroyed(entity);
	LPS_OnVersusAbilityEntityDestroyed(entity);
	LPS_OnIncidentEntityDestroyed(entity);
	g_LPSDamageHooked[entity] = false;
	g_LPSDamageHealthCaptured[entity] = false;
	g_LPSDamageVictimWasIncapacitated[entity] = false;
	g_LPSDamageHealthBefore[entity] = 0.0;
	if (entity <= MaxClients)
	{
		g_LPSClientEffectiveHealthBeforeCommand[entity] = 0.0;
	}
}

public Action OnPlayerRunCmd(
	int client,
	int &buttons,
	int &impulse,
	float velocity[3],
	float angles[3],
	int &weapon,
	int &subtype,
	int &commandNumber,
	int &tickCount,
	int &randomSeed,
	int mouse[2]
)
{
	if (client >= 1 && client <= MaxClients && IsClientInGame(client))
	{
		int team = GetClientTeam(client);
		if (team == LPS_TEAM_SURVIVOR)
		{
			g_LPSClientEffectiveHealthBeforeCommand[client] = LPS_GetTrackedEffectiveHealth(client);
		}
		else if (team == LPS_TEAM_INFECTED)
		{
			LPS_RecordVersusSpawnIfReady(client);
		}
	}
	return Plugin_Continue;
}


void LPS_ShutdownPvEStats()
{
	// Active stats are closed by LPS_ShutdownLifecycles before this callback.
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSDamageHealthCaptured[entity] = false;
		g_LPSDamageVictimWasIncapacitated[entity] = false;
		g_LPSDamageHealthBefore[entity] = 0.0;
		if (entity <= MaxClients)
		{
			g_LPSClientEffectiveHealthBeforeCommand[entity] = 0.0;
		}
	}
}
