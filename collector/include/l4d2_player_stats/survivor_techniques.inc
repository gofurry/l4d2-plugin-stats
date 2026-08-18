#if defined _l4d2_player_stats_survivor_techniques_included
	#endinput
#endif
#define _l4d2_player_stats_survivor_techniques_included

// These detectors intentionally use short, bounded per-client episodes. They
// combine engine events with lifecycle identity instead of trusting a single
// transient send property. Real-game event ordering still requires build
// 10097 validation before the technique semantics are declared verified.
bool g_LPSHunterPounceActive[MAXPLAYERS + 1];
bool g_LPSChargerChargeActive[MAXPLAYERS + 1];
bool g_LPSSurvivorLedgeStateActive[MAXPLAYERS + 1];
int g_LPSHunterPounceUserId[MAXPLAYERS + 1];
int g_LPSChargerChargeUserId[MAXPLAYERS + 1];
float g_LPSHunterPounceExpiresAt[MAXPLAYERS + 1];
float g_LPSChargerChargeExpiresAt[MAXPLAYERS + 1];

void LPS_InitializeSurvivorTechniques()
{
	HookEvent("award_earned", LPS_Event_TeammateProtectionAward, EventHookMode_Post);
	HookEvent("player_ledge_grab", LPS_Event_LedgeGrabTechnique, EventHookMode_Post);
	HookEvent("player_ledge_release", LPS_Event_LedgeReleaseTechnique, EventHookMode_Post);
	HookEvent("revive_success", LPS_Event_LedgeReviveTechnique, EventHookMode_Post);
	HookEvent("ability_use", LPS_Event_TechniqueAbilityUse, EventHookMode_Post);
	HookEvent("lunge_pounce", LPS_Event_HunterPounceSucceeded, EventHookMode_Post);
	HookEvent("pounce_end", LPS_Event_HunterPounceEnded, EventHookMode_Post);
	HookEvent("pounce_stopped", LPS_Event_HunterPounceEnded, EventHookMode_Post);
	HookEvent("charger_charge_start", LPS_Event_ChargerChargeStarted, EventHookMode_Post);
	HookEvent("charger_charge_end", LPS_Event_ChargerChargeEnded, EventHookMode_Post);
	HookEvent("charger_carry_start", LPS_Event_ChargerChargeInvalidated, EventHookMode_Post);
	HookEvent("charger_impact", LPS_Event_ChargerChargeInvalidated, EventHookMode_Post);
	HookEvent("charger_pummel_start", LPS_Event_ChargerChargeInvalidated, EventHookMode_Post);
	HookEvent("player_death", LPS_Event_TechniquePlayerDeath, EventHookMode_Pre);
	HookEvent("player_spawn", LPS_Event_TechniquePlayerReset, EventHookMode_Post);
	HookEvent("player_team", LPS_Event_TechniquePlayerReset, EventHookMode_Post);
	HookEvent("round_start", LPS_Event_TechniqueRoundReset, EventHookMode_PostNoCopy);
	HookEvent("round_end", LPS_Event_TechniqueRoundReset, EventHookMode_PostNoCopy);

	LPS_ResetAllSurvivorTechniqueState();
}

void LPS_ShutdownSurvivorTechniques()
{
	LPS_ResetAllSurvivorTechniqueState();
}

void LPS_ResetAllSurvivorTechniqueState()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetSurvivorTechniqueClient(client);
	}
}

void LPS_ResetSurvivorTechniqueClient(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}
	g_LPSHunterPounceActive[client] = false;
	g_LPSChargerChargeActive[client] = false;
	g_LPSSurvivorLedgeStateActive[client] = false;
	g_LPSHunterPounceUserId[client] = 0;
	g_LPSChargerChargeUserId[client] = 0;
	g_LPSHunterPounceExpiresAt[client] = 0.0;
	g_LPSChargerChargeExpiresAt[client] = 0.0;
}

void LPS_OnClientSurvivorTechniqueSegmentStarted(int client)
{
	// PvE resumes its existing duration state before this hook. Versus only
	// needs the bounded transition bit maintained in this module.
	g_LPSSurvivorLedgeStateActive[client] = LPS_CanRecordVersusEvents()
		&& LPS_IsSurvivorHangingFromLedge(client);
}

void LPS_OnClientSurvivorTechniqueSegmentEnding(int client)
{
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSSurvivorLedgeStateActive[client] = false;
	}
}

void LPS_AddSurvivorTechniqueStat(int client, LPSPvEStat pveStat, LPSVersusSurvivorStat versusStat)
{
	if (LPS_CanRecordPvEEvents())
	{
		LPS_AddPvEStat(client, pveStat, 1);
	}
	else if (LPS_CanRecordVersusEvents())
	{
		LPS_AddVersusSurvivorStat(client, versusStat, 1);
	}
}

public void LPS_Event_TeammateProtectionAward(Event event, const char[] name, bool dontBroadcast)
{
	if (event.GetInt("award") != 67)
	{
		return;
	}
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_IsHumanSurvivorClient(client))
	{
		return;
	}
	LPS_AddSurvivorTechniqueStat(
		client,
		LPSPvEStat_TeammateProtections,
		LPSVersusSurvivorStat_TeammateProtections
	);
}

public void LPS_Event_LedgeGrabTechnique(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_IsHumanSurvivorClient(client))
	{
		return;
	}
	if (LPS_CanRecordPvEEvents())
	{
		// Reuse the existing PvE ledge-duration transition state. A Segment
		// beginning while already hanging has this bit set without counting.
		if (g_LPSLedgeTimerActive[client])
		{
			return;
		}
	}
	else if (g_LPSSurvivorLedgeStateActive[client])
	{
		return;
	}
	g_LPSSurvivorLedgeStateActive[client] = true;
	LPS_AddSurvivorTechniqueStat(client, LPSPvEStat_LedgeGrabs, LPSVersusSurvivorStat_LedgeGrabs);
}

public void LPS_Event_LedgeReleaseTechnique(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSSurvivorLedgeStateActive[client] = false;
	}
}

public void LPS_Event_LedgeReviveTechnique(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("subject"));
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSSurvivorLedgeStateActive[client] = false;
	}
}

public void LPS_Event_TechniqueAbilityUse(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || GetClientTeam(client) != LPS_TEAM_INFECTED)
	{
		return;
	}
	char ability[64];
	event.GetString("ability", ability, sizeof(ability));
	if (StrEqual(ability, "ability_lunge", false) && LPS_GetZombieClass(client) == 3)
	{
		LPS_StartHunterPounceEpisode(client);
	}
	else if (StrEqual(ability, "ability_charge", false) && LPS_GetZombieClass(client) == 6)
	{
		LPS_StartChargerChargeEpisode(client);
	}
}

public void LPS_Event_ChargerChargeStarted(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (client < 1 || client > MaxClients)
	{
		client = GetClientOfUserId(event.GetInt("charger"));
	}
	if (client >= 1 && client <= MaxClients && IsClientInGame(client) && LPS_GetZombieClass(client) == 6)
	{
		LPS_StartChargerChargeEpisode(client);
	}
}

void LPS_StartHunterPounceEpisode(int client)
{
	g_LPSHunterPounceActive[client] = true;
	g_LPSHunterPounceUserId[client] = GetClientUserId(client);
	g_LPSHunterPounceExpiresAt[client] = GetGameTime() + 5.0;
}

void LPS_StartChargerChargeEpisode(int client)
{
	g_LPSChargerChargeActive[client] = true;
	g_LPSChargerChargeUserId[client] = GetClientUserId(client);
	g_LPSChargerChargeExpiresAt[client] = GetGameTime() + 8.0;
}

public void LPS_Event_HunterPounceSucceeded(Event event, const char[] name, bool dontBroadcast)
{
	LPS_InvalidateHunterEpisode(GetClientOfUserId(event.GetInt("userid")));
}

public void LPS_Event_HunterPounceEnded(Event event, const char[] name, bool dontBroadcast)
{
	int hunter = GetClientOfUserId(event.GetInt("userid"));
	if (hunter < 1 || hunter > MaxClients)
	{
		hunter = GetClientOfUserId(event.GetInt("hunter"));
	}
	LPS_InvalidateHunterEpisode(hunter);
}

public void LPS_Event_ChargerChargeEnded(Event event, const char[] name, bool dontBroadcast)
{
	int charger = GetClientOfUserId(event.GetInt("userid"));
	if (charger < 1 || charger > MaxClients)
	{
		charger = GetClientOfUserId(event.GetInt("charger"));
	}
	LPS_InvalidateChargerEpisode(charger);
}

public void LPS_Event_ChargerChargeInvalidated(Event event, const char[] name, bool dontBroadcast)
{
	int charger = GetClientOfUserId(event.GetInt("userid"));
	if (charger < 1 || charger > MaxClients)
	{
		charger = GetClientOfUserId(event.GetInt("charger"));
	}
	LPS_InvalidateChargerEpisode(charger);
}

void LPS_InvalidateHunterEpisode(int client)
{
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSHunterPounceActive[client] = false;
	}
}

void LPS_InvalidateChargerEpisode(int client)
{
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSChargerChargeActive[client] = false;
	}
}

public void LPS_Event_TechniquePlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("userid"));
	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	bool hunterSkeet = g_LPSHunterPounceActive[victim]
		&& g_LPSHunterPounceUserId[victim] == GetClientUserId(victim)
		&& GetGameTime() <= g_LPSHunterPounceExpiresAt[victim]
		&& LPS_GetZombieClass(victim) == 3
		&& LPS_IsHunterInAirbornePounce(victim);
	bool chargerLevel = g_LPSChargerChargeActive[victim]
		&& g_LPSChargerChargeUserId[victim] == GetClientUserId(victim)
		&& GetGameTime() <= g_LPSChargerChargeExpiresAt[victim]
		&& LPS_GetZombieClass(victim) == 6
		&& LPS_IsChargerActivelyCharging(victim);

	LPS_InvalidateHunterEpisode(victim);
	LPS_InvalidateChargerEpisode(victim);
	if (!LPS_IsHumanSurvivorClient(attacker))
	{
		return;
	}

	if (hunterSkeet)
	{
		LPS_AddSurvivorTechniqueStat(attacker, LPSPvEStat_HunterSkeets, LPSVersusSurvivorStat_HunterSkeets);
	}
	if (chargerLevel)
	{
		char weaponName[64];
		event.GetString("weapon", weaponName, sizeof(weaponName));
		LPSEquipment equipment = LPS_GetEquipmentFromWeaponEvent(
			attacker,
			event.GetInt("weaponid"),
			weaponName,
			false
		);
		if (LPS_IsMeleeEquipment(equipment))
		{
			LPS_AddSurvivorTechniqueStat(attacker, LPSPvEStat_ChargerLevels, LPSVersusSurvivorStat_ChargerLevels);
		}
	}
}

bool LPS_IsHunterInAirbornePounce(int client)
{
	// The episode is event-driven, while these properties prove the final
	// blow happened in flight rather than during the wind-up window.
	return HasEntProp(client, Prop_Send, "m_hGroundEntity")
		&& GetEntPropEnt(client, Prop_Send, "m_hGroundEntity") == -1
		&& HasEntProp(client, Prop_Send, "m_isAttemptingToPounce")
		&& GetEntProp(client, Prop_Send, "m_isAttemptingToPounce") != 0;
}

bool LPS_IsChargerActivelyCharging(int client)
{
	// Fail closed if the accepted build no longer exposes the charge property;
	// a missing property must not turn the episode timeout into a heuristic.
	return HasEntProp(client, Prop_Send, "m_isCharging")
		&& GetEntProp(client, Prop_Send, "m_isCharging") != 0;
}

public void LPS_Event_TechniquePlayerReset(Event event, const char[] name, bool dontBroadcast)
{
	LPS_ResetSurvivorTechniqueClient(GetClientOfUserId(event.GetInt("userid")));
}

public void LPS_Event_TechniqueRoundReset(Event event, const char[] name, bool dontBroadcast)
{
	LPS_ResetAllSurvivorTechniqueState();
}
