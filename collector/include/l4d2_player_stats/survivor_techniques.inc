#if defined _l4d2_player_stats_survivor_techniques_included
	#endinput
#endif
#define _l4d2_player_stats_survivor_techniques_included

// These detectors intentionally use short, bounded per-client episodes. They
// combine engine events with lifecycle identity instead of trusting a single
// transient send property. Real-game event ordering still requires build
// 10097 validation before the technique semantics are declared verified.
#define LPS_TECHNIQUE_CONFIRM_GRACE 0.25
bool g_LPSHunterPounceActive[MAXPLAYERS + 1];
bool g_LPSChargerChargeActive[MAXPLAYERS + 1];
bool g_LPSHunterAirborneConfirmed[MAXPLAYERS + 1];
bool g_LPSChargerChargingConfirmed[MAXPLAYERS + 1];
bool g_LPSHunterLethalCandidate[MAXPLAYERS + 1];
bool g_LPSChargerLethalCandidate[MAXPLAYERS + 1];
bool g_LPSSurvivorLedgeStateActive[MAXPLAYERS + 1];
int g_LPSHunterPounceUserId[MAXPLAYERS + 1];
int g_LPSChargerChargeUserId[MAXPLAYERS + 1];
float g_LPSHunterPounceExpiresAt[MAXPLAYERS + 1];
float g_LPSChargerChargeExpiresAt[MAXPLAYERS + 1];
float g_LPSHunterLastAirborneAt[MAXPLAYERS + 1];
float g_LPSChargerLastChargingAt[MAXPLAYERS + 1];
float g_LPSTechniqueLethalCandidateAt[MAXPLAYERS + 1];
int g_LPSTechniqueLethalAttackerUserId[MAXPLAYERS + 1];
int g_LPSTechniqueLethalVictimUserId[MAXPLAYERS + 1];

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
	HookEvent("player_hurt", LPS_Event_TechniquePlayerHurt, EventHookMode_Post);
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
	g_LPSHunterAirborneConfirmed[client] = false;
	g_LPSChargerChargingConfirmed[client] = false;
	g_LPSHunterLethalCandidate[client] = false;
	g_LPSChargerLethalCandidate[client] = false;
	g_LPSSurvivorLedgeStateActive[client] = false;
	g_LPSHunterPounceUserId[client] = 0;
	g_LPSChargerChargeUserId[client] = 0;
	g_LPSHunterPounceExpiresAt[client] = 0.0;
	g_LPSChargerChargeExpiresAt[client] = 0.0;
	g_LPSHunterLastAirborneAt[client] = 0.0;
	g_LPSChargerLastChargingAt[client] = 0.0;
	g_LPSTechniqueLethalCandidateAt[client] = 0.0;
	g_LPSTechniqueLethalAttackerUserId[client] = 0;
	g_LPSTechniqueLethalVictimUserId[client] = 0;
}

void LPS_OnSurvivorTechniqueGameFrame()
{
	float now = GetGameTime();
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSTechniqueLethalCandidateAt[client] > 0.0
			&& now - g_LPSTechniqueLethalCandidateAt[client] > 1.0)
		{
			LPS_ClearTechniqueLethalCandidate(client);
		}
		if (g_LPSHunterPounceActive[client])
		{
			if (!LPS_IsTechniqueEpisodeClient(client, 3, g_LPSHunterPounceUserId[client])
				|| now > g_LPSHunterPounceExpiresAt[client])
			{
				LPS_InvalidateHunterEpisode(client);
			}
			else if (LPS_IsHunterAirborne(client))
			{
				if (!g_LPSHunterAirborneConfirmed[client])
				{
					LPS_TechniqueDebug("hunter airborne userid=%d", GetClientUserId(client));
				}
				g_LPSHunterAirborneConfirmed[client] = true;
				g_LPSHunterLastAirborneAt[client] = now;
			}
			else if (g_LPSHunterAirborneConfirmed[client]
				&& now - g_LPSHunterLastAirborneAt[client] > LPS_TECHNIQUE_CONFIRM_GRACE)
			{
				LPS_InvalidateHunterEpisode(client);
			}
		}

		if (g_LPSChargerChargeActive[client])
		{
			if (!LPS_IsTechniqueEpisodeClient(client, 6, g_LPSChargerChargeUserId[client])
				|| now > g_LPSChargerChargeExpiresAt[client])
			{
				LPS_InvalidateChargerEpisode(client);
			}
			else if (LPS_IsChargerActivelyCharging(client))
			{
				if (!g_LPSChargerChargingConfirmed[client])
				{
					LPS_TechniqueDebug("charger charging userid=%d", GetClientUserId(client));
				}
				g_LPSChargerChargingConfirmed[client] = true;
				g_LPSChargerLastChargingAt[client] = now;
			}
			else if (g_LPSChargerChargingConfirmed[client]
				&& now - g_LPSChargerLastChargingAt[client] > LPS_TECHNIQUE_CONFIRM_GRACE)
			{
				LPS_InvalidateChargerEpisode(client);
			}
		}
	}
}

bool LPS_IsTechniqueEpisodeClient(int client, int zombieClass, int userId)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& IsPlayerAlive(client)
		&& GetClientUserId(client) == userId
		&& GetClientTeam(client) == LPS_TEAM_INFECTED
		&& LPS_GetZombieClass(client) == zombieClass;
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
		g_LPSChargerChargingConfirmed[client] = true;
		g_LPSChargerLastChargingAt[client] = GetGameTime();
	}
}

void LPS_StartHunterPounceEpisode(int client)
{
	int userId = GetClientUserId(client);
	bool continuing = g_LPSHunterPounceActive[client]
		&& g_LPSHunterPounceUserId[client] == userId
		&& GetGameTime() <= g_LPSHunterPounceExpiresAt[client];
	g_LPSHunterPounceActive[client] = true;
	g_LPSHunterPounceUserId[client] = userId;
	g_LPSHunterPounceExpiresAt[client] = GetGameTime() + 5.0;
	if (!continuing)
	{
		g_LPSHunterAirborneConfirmed[client] = false;
		g_LPSHunterLastAirborneAt[client] = 0.0;
		LPS_ClearTechniqueLethalCandidate(client);
	}
	LPS_TechniqueDebug("hunter episode userid=%d", userId);
}

void LPS_StartChargerChargeEpisode(int client)
{
	int userId = GetClientUserId(client);
	bool continuing = g_LPSChargerChargeActive[client]
		&& g_LPSChargerChargeUserId[client] == userId
		&& GetGameTime() <= g_LPSChargerChargeExpiresAt[client];
	g_LPSChargerChargeActive[client] = true;
	g_LPSChargerChargeUserId[client] = userId;
	g_LPSChargerChargeExpiresAt[client] = GetGameTime() + 8.0;
	if (!continuing)
	{
		g_LPSChargerChargingConfirmed[client] = false;
		g_LPSChargerLastChargingAt[client] = 0.0;
		LPS_ClearTechniqueLethalCandidate(client);
	}
	LPS_TechniqueDebug("charger episode userid=%d", userId);
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
		g_LPSHunterAirborneConfirmed[client] = false;
		g_LPSHunterPounceUserId[client] = 0;
		g_LPSHunterPounceExpiresAt[client] = 0.0;
		g_LPSHunterLastAirborneAt[client] = 0.0;
	}
}

void LPS_InvalidateChargerEpisode(int client)
{
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSChargerChargeActive[client] = false;
		g_LPSChargerChargingConfirmed[client] = false;
		g_LPSChargerChargeUserId[client] = 0;
		g_LPSChargerChargeExpiresAt[client] = 0.0;
		g_LPSChargerLastChargingAt[client] = 0.0;
	}
}

public void LPS_Event_TechniquePlayerHurt(Event event, const char[] name, bool dontBroadcast)
{
	if (event.GetInt("health") > 0)
	{
		return;
	}
	int victim = GetClientOfUserId(event.GetInt("userid"));
	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	if (victim < 1 || victim > MaxClients || !LPS_IsHumanSurvivorClient(attacker))
	{
		return;
	}
	float now = GetGameTime();
	bool hunter = g_LPSHunterPounceActive[victim]
		&& g_LPSHunterAirborneConfirmed[victim]
		&& g_LPSHunterPounceUserId[victim] == GetClientUserId(victim)
		&& now - g_LPSHunterLastAirborneAt[victim] <= LPS_TECHNIQUE_CONFIRM_GRACE
		&& LPS_GetZombieClass(victim) == 3;
	bool charger = g_LPSChargerChargeActive[victim]
		&& g_LPSChargerChargingConfirmed[victim]
		&& g_LPSChargerChargeUserId[victim] == GetClientUserId(victim)
		&& now - g_LPSChargerLastChargingAt[victim] <= LPS_TECHNIQUE_CONFIRM_GRACE
		&& LPS_GetZombieClass(victim) == 6;
	if (!hunter && !charger)
	{
		return;
	}
	g_LPSHunterLethalCandidate[victim] = hunter;
	g_LPSChargerLethalCandidate[victim] = charger;
	g_LPSTechniqueLethalCandidateAt[victim] = now;
	g_LPSTechniqueLethalAttackerUserId[victim] = GetClientUserId(attacker);
	g_LPSTechniqueLethalVictimUserId[victim] = GetClientUserId(victim);
	LPS_TechniqueDebug(
		"lethal hurt victim=%d attacker=%d hunter=%d charger=%d",
		GetClientUserId(victim),
		GetClientUserId(attacker),
		hunter,
		charger
	);
}

void LPS_ClearTechniqueLethalCandidate(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}
	g_LPSHunterLethalCandidate[client] = false;
	g_LPSChargerLethalCandidate[client] = false;
	g_LPSTechniqueLethalCandidateAt[client] = 0.0;
	g_LPSTechniqueLethalAttackerUserId[client] = 0;
	g_LPSTechniqueLethalVictimUserId[client] = 0;
}

public void LPS_Event_TechniquePlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("userid"));
	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	float now = GetGameTime();
	bool lethalCandidate = g_LPSTechniqueLethalVictimUserId[victim] == GetClientUserId(victim)
		&& g_LPSTechniqueLethalAttackerUserId[victim] == (attacker >= 1 && attacker <= MaxClients ? GetClientUserId(attacker) : 0)
		&& now - g_LPSTechniqueLethalCandidateAt[victim] <= 1.0;
	bool hunterSkeet = (g_LPSHunterPounceActive[victim]
		&& g_LPSHunterAirborneConfirmed[victim]
		&& g_LPSHunterPounceUserId[victim] == GetClientUserId(victim)
		&& now <= g_LPSHunterPounceExpiresAt[victim]
		&& now - g_LPSHunterLastAirborneAt[victim] <= LPS_TECHNIQUE_CONFIRM_GRACE
		&& LPS_GetZombieClass(victim) == 3)
		|| (lethalCandidate && g_LPSHunterLethalCandidate[victim]);
	bool chargerLevel = (g_LPSChargerChargeActive[victim]
		&& g_LPSChargerChargingConfirmed[victim]
		&& g_LPSChargerChargeUserId[victim] == GetClientUserId(victim)
		&& now <= g_LPSChargerChargeExpiresAt[victim]
		&& now - g_LPSChargerLastChargingAt[victim] <= LPS_TECHNIQUE_CONFIRM_GRACE
		&& LPS_GetZombieClass(victim) == 6)
		|| (lethalCandidate && g_LPSChargerLethalCandidate[victim]);
	LPS_TechniqueDebug(
		"death victim=%d attacker=%d class=%d hunter=%d charger=%d",
		GetClientUserId(victim),
		attacker >= 1 && attacker <= MaxClients ? GetClientUserId(attacker) : 0,
		LPS_GetZombieClass(victim),
		hunterSkeet,
		chargerLevel
	);

	LPS_InvalidateHunterEpisode(victim);
	LPS_InvalidateChargerEpisode(victim);
	LPS_ClearTechniqueLethalCandidate(victim);
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
		LPSEquipment equipment = LPS_ResolveKillEquipment(
			victim,
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

bool LPS_IsHunterAirborne(int client)
{
	return HasEntProp(client, Prop_Send, "m_hGroundEntity")
		&& GetEntPropEnt(client, Prop_Send, "m_hGroundEntity") == -1;
}

void LPS_TechniqueDebug(const char[] format, any ...)
{
	if (g_LPSTechniqueDebug == null || !g_LPSTechniqueDebug.BoolValue)
	{
		return;
	}
	char message[256];
	VFormat(message, sizeof(message), format, 2);
	LogMessage("[technique-debug] %s", message);
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
