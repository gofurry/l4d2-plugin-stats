#if defined _l4d2_player_stats_pve_extended_included
	#endinput
#endif
#define _l4d2_player_stats_pve_extended_included

LPSControlType g_LPSControlType[MAXPLAYERS + 1];
float g_LPSControlStartedAt[MAXPLAYERS + 1];
int g_LPSControlControllerUserId[MAXPLAYERS + 1];
bool g_LPSControlSaveAwarded[MAXPLAYERS + 1];
bool g_LPSTongueStopHandled[MAXPLAYERS + 1];
bool g_LPSChargerPummeling[MAXPLAYERS + 1];

int g_LPSBossContributorUserId[LPS_MAX_ENTITY_INDEX + 1][MAXPLAYERS + 1];
bool g_LPSTankRockDestroyed[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSTankRockVictimMaskLow[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSTankRockVictimMaskHigh[LPS_MAX_ENTITY_INDEX + 1];

// 0 = none, 1 = incendiary, 2 = explosive.
int g_LPSPendingUpgradePackType[MAXPLAYERS + 1];

void LPS_InitializeExtendedPvEStats()
{
	HookEvent("tongue_grab", LPS_Event_TongueGrabStats, EventHookMode_Post);
	HookEvent("tongue_release", LPS_Event_TongueReleaseStats, EventHookMode_Post);
	HookEvent("tongue_pull_stopped", LPS_Event_TongueStoppedStats, EventHookMode_Post);
	HookEvent("choke_stopped", LPS_Event_TongueStoppedStats, EventHookMode_Post);
	HookEvent("lunge_pounce", LPS_Event_LungePounceStats, EventHookMode_Post);
	HookEvent("pounce_end", LPS_Event_PounceEndStats, EventHookMode_Post);
	HookEvent("pounce_stopped", LPS_Event_PounceStoppedStats, EventHookMode_Post);
	HookEvent("jockey_ride", LPS_Event_JockeyRideStats, EventHookMode_Post);
	HookEvent("jockey_ride_end", LPS_Event_JockeyRideEndStats, EventHookMode_Post);
	HookEvent("charger_carry_start", LPS_Event_ChargerCarryStartStats, EventHookMode_Post);
	HookEvent("charger_carry_end", LPS_Event_ChargerCarryEndStats, EventHookMode_Post);
	HookEvent("charger_pummel_start", LPS_Event_ChargerPummelStartStats, EventHookMode_Post);
	HookEvent("charger_pummel_end", LPS_Event_ChargerPummelEndStats, EventHookMode_Post);
	HookEvent("tank_spawn", LPS_Event_TankSpawnStats, EventHookMode_Post);
	HookEvent("witch_spawn", LPS_Event_WitchSpawnStats, EventHookMode_Post);
	HookEvent("upgrade_pack_begin", LPS_Event_UpgradePackBeginStats, EventHookMode_Post);
	HookEvent("upgrade_pack_used", LPS_Event_UpgradePackUsedStats, EventHookMode_Post);

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientExtendedPvEState(client);
	}
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		LPS_ResetBossTracking(entity);
		g_LPSTankRockDestroyed[entity] = false;
		g_LPSTankRockVictimMaskLow[entity] = 0;
		g_LPSTankRockVictimMaskHigh[entity] = 0;
	}

	int entity = -1;
	while ((entity = FindEntityByClassname(entity, "tank_rock")) != -1)
	{
		LPS_HookTankRock(entity);
	}
}

void LPS_OnExtendedEntityCreated(int entity, const char[] classname)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	if (StrEqual(classname, "tank_rock", true))
	{
		g_LPSTankRockDestroyed[entity] = false;
		g_LPSTankRockVictimMaskLow[entity] = 0;
		g_LPSTankRockVictimMaskHigh[entity] = 0;
		LPS_HookTankRock(entity);
	}
	else if (StrEqual(classname, "witch", true) || StrEqual(classname, "witch_bride", true))
	{
		LPS_ResetBossTracking(entity);
	}
}

void LPS_OnExtendedEntityDestroyed(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	LPS_ResetBossTracking(entity);
	g_LPSTankRockDestroyed[entity] = false;
	g_LPSTankRockVictimMaskLow[entity] = 0;
	g_LPSTankRockVictimMaskHigh[entity] = 0;
}

void LPS_RecordTankRockHitReceived(int victim, int attacker, int inflictor, int actualDamage)
{
	if (actualDamage <= 0 || !LPS_IsHumanSurvivorClient(victim))
	{
		return;
	}

	int rock = LPS_IsTankRockEntity(inflictor) ? inflictor : (LPS_IsTankRockEntity(attacker) ? attacker : 0);
	if (rock <= MaxClients || rock > LPS_MAX_ENTITY_INDEX || !LPS_MarkTankRockVictim(rock, victim))
	{
		return;
	}

	if (LPS_CanRecordPvEEvents())
	{
		LPS_AddPvEStat(victim, LPSPvEStat_TankRockHitsReceived, 1);
	}
	else if (LPS_CanRecordVersusEvents())
	{
		LPS_AddVersusSurvivorStat(victim, LPSVersusSurvivorStat_TankRockHitsReceived, 1);
	}
}

bool LPS_IsTankRockEntity(int entity)
{
	if (entity <= MaxClients || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return false;
	}
	char classname[32];
	GetEntityClassname(entity, classname, sizeof(classname));
	return StrEqual(classname, "tank_rock", true);
}

bool LPS_MarkTankRockVictim(int rock, int victim)
{
	int index = victim - 1;
	int bit = 1 << (index % 32);
	if (index < 32)
	{
		if ((g_LPSTankRockVictimMaskLow[rock] & bit) != 0)
		{
			return false;
		}
		g_LPSTankRockVictimMaskLow[rock] |= bit;
		return true;
	}
	if ((g_LPSTankRockVictimMaskHigh[rock] & bit) != 0)
	{
		return false;
	}
	g_LPSTankRockVictimMaskHigh[rock] |= bit;
	return true;
}

void LPS_HookTankRock(int entity)
{
	if (entity <= MaxClients || !IsValidEntity(entity))
	{
		return;
	}
	SDKHook(entity, SDKHook_OnTakeDamage, LPS_OnTankRockTakeDamage);
}

public Action LPS_OnTankRockTakeDamage(
	int victim,
	int &attacker,
	int &inflictor,
	float &damage,
	int &damageType
)
{
	bool recordPvE = LPS_CanRecordPvEEvents();
	bool recordVersus = LPS_CanRecordVersusEvents();
	if ((!recordPvE && !recordVersus)
		|| victim < 1
		|| victim > LPS_MAX_ENTITY_INDEX
		|| g_LPSTankRockDestroyed[victim]
		|| damage <= 0.0)
	{
		return Plugin_Continue;
	}

	int humanAttacker = LPS_ResolveHumanSurvivorAttacker(attacker, inflictor);
	LPSEquipment equipment = LPS_GetActiveEquipment(humanAttacker, true);
	if (!LPS_IsHumanSurvivorClient(humanAttacker)
		|| !LPS_IsFirearmEquipment(equipment)
		|| !HasEntProp(victim, Prop_Data, "m_iHealth"))
	{
		return Plugin_Continue;
	}

	int health = GetEntProp(victim, Prop_Data, "m_iHealth");
	if (health > 0 && damage + 0.01 >= float(health))
	{
		g_LPSTankRockDestroyed[victim] = true;
		if (recordPvE)
		{
			LPS_AddPvEStat(humanAttacker, LPSPvEStat_TankRocksDestroyed, 1);
		}
		else
		{
			LPS_AddVersusSurvivorStat(
				humanAttacker,
				LPSVersusSurvivorStat_TankRocksDestroyed,
				1
			);
		}
	}
	return Plugin_Continue;
}

public void LPS_Event_TongueGrabStats(Event event, const char[] name, bool dontBroadcast)
{
	int controller = GetClientOfUserId(event.GetInt("userid"));
	int victim = GetClientOfUserId(event.GetInt("victim"));
	g_LPSTongueStopHandled[victim] = false;
	LPS_StartControl(victim, controller, LPSControlType_Smoker);
}

public void LPS_Event_TongueReleaseStats(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	LPS_DeferControlEnd(victim, LPSControlType_Smoker);
}

public void LPS_Event_TongueStoppedStats(Event event, const char[] name, bool dontBroadcast)
{
	bool recordPvE = LPS_CanRecordPvEEvents();
	bool recordVersus = LPS_CanRecordVersusEvents();
	if (!recordPvE && !recordVersus)
	{
		return;
	}

	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int victim = GetClientOfUserId(event.GetInt("victim"));
	if (victim < 1 || victim > MaxClients || g_LPSTongueStopHandled[victim])
	{
		return;
	}

	g_LPSTongueStopHandled[victim] = true;
	if (rescuer == victim && LPS_IsHumanSurvivorClient(victim))
	{
		LPSEquipment equipment = LPS_GetActiveEquipment(victim, false);
		if (LPS_IsMeleeEquipment(equipment))
		{
			if (recordPvE)
			{
				LPS_AddPvEStat(victim, LPSPvEStat_MeleeTongueSelfCuts, 1);
			}
			else
			{
				LPS_AddVersusSurvivorStat(
					victim,
					LPSVersusSurvivorStat_MeleeTongueSelfCuts,
					1
				);
			}
		}
	}
	else if (recordPvE)
	{
		LPS_AwardControlSave(rescuer, victim, LPSControlType_Smoker);
	}
	if (recordPvE)
	{
		LPSIncidentEndReason reason = rescuer == victim
			? LPSIncidentEnd_SelfFreed
			: (rescuer > 0 ? LPSIncidentEnd_Rescued : LPSIncidentEnd_Released);
		LPS_EndControl(victim, reason, rescuer);
	}
}

public void LPS_Event_LungePounceStats(Event event, const char[] name, bool dontBroadcast)
{
	LPS_StartControl(
		GetClientOfUserId(event.GetInt("victim")),
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Hunter
	);
}

public void LPS_Event_PounceEndStats(Event event, const char[] name, bool dontBroadcast)
{
	LPS_DeferControlEnd(
		GetClientOfUserId(event.GetInt("victim")),
		LPSControlType_Hunter
	);
}

public void LPS_Event_PounceStoppedStats(Event event, const char[] name, bool dontBroadcast)
{
	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int victim = GetClientOfUserId(event.GetInt("victim"));
	LPS_AwardControlSave(rescuer, victim, LPSControlType_Hunter);
	LPS_EndControl(victim, rescuer > 0 ? LPSIncidentEnd_Rescued : LPSIncidentEnd_Released, rescuer);
}

public void LPS_Event_JockeyRideStats(Event event, const char[] name, bool dontBroadcast)
{
	LPS_StartControl(
		GetClientOfUserId(event.GetInt("victim")),
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Jockey
	);
}

public void LPS_Event_JockeyRideEndStats(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	int rescuer = GetClientOfUserId(event.GetInt("rescuer"));
	LPS_AwardControlSave(rescuer, victim, LPSControlType_Jockey);
	LPS_DeferControlEnd(victim, LPSControlType_Jockey);
}

public void LPS_Event_ChargerCarryStartStats(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	g_LPSChargerPummeling[victim] = false;
	LPS_StartControl(
		victim,
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Charger
	);
}

public void LPS_Event_ChargerCarryEndStats(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	DataPack pack;
	CreateDataTimer(0.1, LPS_Timer_EndChargerCarry, pack, TIMER_FLAG_NO_MAPCHANGE);
	pack.WriteCell(GetClientUserId(victim));
}

public Action LPS_Timer_EndChargerCarry(Handle timer, DataPack pack)
{
	pack.Reset();
	int victim = GetClientOfUserId(pack.ReadCell());
	if (victim >= 1 && victim <= MaxClients && !g_LPSChargerPummeling[victim])
	{
		LPS_EndControl(victim);
	}
	return Plugin_Stop;
}

public void LPS_Event_ChargerPummelStartStats(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	g_LPSChargerPummeling[victim] = true;
	LPS_StartControl(
		victim,
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Charger
	);
}

public void LPS_Event_ChargerPummelEndStats(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	g_LPSChargerPummeling[victim] = false;
	LPS_DeferControlEnd(victim, LPSControlType_Charger);
}

void LPS_DeferControlEnd(int victim, LPSControlType expectedType)
{
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	DataPack pack;
	CreateDataTimer(0.1, LPS_Timer_EndControl, pack, TIMER_FLAG_NO_MAPCHANGE);
	pack.WriteCell(GetClientUserId(victim));
	pack.WriteCell(view_as<int>(expectedType));
}

public Action LPS_Timer_EndControl(Handle timer, DataPack pack)
{
	pack.Reset();
	int victim = GetClientOfUserId(pack.ReadCell());
	LPSControlType expectedType = view_as<LPSControlType>(pack.ReadCell());
	if (victim >= 1
		&& victim <= MaxClients
		&& g_LPSControlType[victim] == expectedType)
	{
		LPS_EndControl(victim);
	}
	return Plugin_Stop;
}

void LPS_StartControl(int victim, int controller, LPSControlType type)
{
	if (!LPS_CanRecordPvEEvents()
		|| victim < 1
		|| victim > MaxClients
		|| !IsClientInGame(victim)
		|| GetClientTeam(victim) != LPS_TEAM_SURVIVOR
		|| controller < 1
		|| controller > MaxClients
		|| !IsClientInGame(controller)
		|| GetClientTeam(controller) != LPS_TEAM_INFECTED
		|| (IsFakeClient(victim) && IsFakeClient(controller)))
	{
		return;
	}

	if (g_LPSControlType[victim] == type
		&& g_LPSControlControllerUserId[victim] == GetClientUserId(controller))
	{
		return;
	}
	LPS_EndControl(victim);

	g_LPSControlType[victim] = type;
	g_LPSControlStartedAt[victim] = GetGameTime();
	g_LPSControlControllerUserId[victim] = GetClientUserId(controller);
	g_LPSControlSaveAwarded[victim] = false;
	LPS_StartControlIncident(victim, controller, type, LPS_ControlTypeToZombieClass(type));

	switch (type)
	{
		case LPSControlType_Smoker: LPS_AddPvEStat(victim, LPSPvEStat_SmokerControlsReceived, 1);
		case LPSControlType_Hunter: LPS_AddPvEStat(victim, LPSPvEStat_HunterControlsReceived, 1);
		case LPSControlType_Jockey: LPS_AddPvEStat(victim, LPSPvEStat_JockeyControlsReceived, 1);
		case LPSControlType_Charger: LPS_AddPvEStat(victim, LPSPvEStat_ChargerControlsReceived, 1);
	}
}

void LPS_EndControl(int victim, LPSIncidentEndReason reason = LPSIncidentEnd_Released, int helper = 0)
{
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}
	if (g_LPSControlType[victim] == LPSControlType_None)
	{
		LPS_EndControlIncident(victim, LPSControlType_None, reason, helper);
		return;
	}

	float elapsed = GetGameTime() - g_LPSControlStartedAt[victim];
	int durationMs = elapsed > 0.0 ? RoundToNearest(elapsed * 1000.0) : 0;
	if ((reason == LPSIncidentEnd_Rescued || reason == LPSIncidentEnd_ControllerKilled)
		&& helper > 0)
	{
		LPS_RecordRelationshipControlRescue(
			helper,
			victim,
			g_LPSControlType[victim],
			durationMs
		);
	}
	int seconds = elapsed > 0.0 ? RoundToNearest(elapsed) : 0;
	if (seconds > 0)
	{
		switch (g_LPSControlType[victim])
		{
			case LPSControlType_Smoker: LPS_AddPvEStat(victim, LPSPvEStat_SmokerControlledSeconds, seconds);
			case LPSControlType_Hunter: LPS_AddPvEStat(victim, LPSPvEStat_HunterControlledSeconds, seconds);
			case LPSControlType_Jockey: LPS_AddPvEStat(victim, LPSPvEStat_JockeyControlledSeconds, seconds);
			case LPSControlType_Charger: LPS_AddPvEStat(victim, LPSPvEStat_ChargerControlledSeconds, seconds);
		}
	}
	LPS_EndControlIncident(victim, g_LPSControlType[victim], reason, helper);

	g_LPSControlType[victim] = LPSControlType_None;
	g_LPSControlStartedAt[victim] = 0.0;
	g_LPSControlControllerUserId[victim] = 0;
	g_LPSControlSaveAwarded[victim] = false;
}

void LPS_AwardControlSave(int rescuer, int victim, LPSControlType expectedType)
{
	if (!LPS_CanRecordPvEEvents()
		|| !LPS_IsHumanSurvivorClient(rescuer)
		|| !LPS_IsHumanSurvivorClient(victim)
		|| rescuer == victim
		|| g_LPSControlType[victim] != expectedType
		|| g_LPSControlSaveAwarded[victim])
	{
		return;
	}

	switch (expectedType)
	{
		case LPSControlType_Smoker: LPS_AddPvEStat(rescuer, LPSPvEStat_SmokerSaves, 1);
		case LPSControlType_Hunter: LPS_AddPvEStat(rescuer, LPSPvEStat_HunterSaves, 1);
		case LPSControlType_Jockey: LPS_AddPvEStat(rescuer, LPSPvEStat_JockeySaves, 1);
		case LPSControlType_Charger: LPS_AddPvEStat(rescuer, LPSPvEStat_ChargerSaves, 1);
	}
	g_LPSControlSaveAwarded[victim] = true;
}

int LPS_ControlTypeToZombieClass(LPSControlType type)
{
	switch (type)
	{
		case LPSControlType_Smoker: return 1;
		case LPSControlType_Hunter: return 3;
		case LPSControlType_Jockey: return 5;
		case LPSControlType_Charger: return 6;
	}
	return 0;
}

void LPS_OnSpecialInfectedDeathForExtendedStats(int infected, int attacker, bool aborted = false)
{
	if (infected < 1 || infected > MaxClients)
	{
		return;
	}

	int controllerUserId = GetClientUserId(infected);
	for (int victim = 1; victim <= MaxClients; victim++)
	{
		if (g_LPSControlControllerUserId[victim] != controllerUserId)
		{
			continue;
		}
		LPS_AwardControlSave(attacker, victim, g_LPSControlType[victim]);
		LPS_EndControl(victim, aborted ? LPSIncidentEnd_Released : LPSIncidentEnd_ControllerKilled, aborted ? 0 : attacker);
	}
}

bool LPS_RecordBossContribution(int boss, int client)
{
	bool recordPvE = LPS_CanRecordPvEEvents();
	bool recordVersusWitch = LPS_CanRecordVersusEvents() && LPS_IsWitchEntity(boss);
	if (boss < 1
		|| boss > LPS_MAX_ENTITY_INDEX
		|| !LPS_IsHumanSurvivorClient(client)
		|| (!recordPvE && !recordVersusWitch))
	{
		return false;
	}

	int userId = GetClientUserId(client);
	for (int slot = 1; slot <= MaxClients; slot++)
	{
		if (g_LPSBossContributorUserId[boss][slot] == userId)
		{
			return false;
		}
	}

	for (int slot = 1; slot <= MaxClients; slot++)
	{
		if (g_LPSBossContributorUserId[boss][slot] != 0)
		{
			continue;
		}
		g_LPSBossContributorUserId[boss][slot] = userId;
		if (!recordPvE)
		{
			return true;
		}
		if (boss <= MaxClients && LPS_GetZombieClass(boss) == 8)
		{
			LPS_AddPvEStat(client, LPSPvEStat_TankEncounters, 1);
		}
		else if (LPS_IsWitchEntity(boss))
		{
			LPS_AddPvEStat(client, LPSPvEStat_WitchEncounters, 1);
		}
		return true;
	}
	return false;
}

void LPS_ClearBossContributionsForClient(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	int userId = GetClientUserId(client);
	if (userId == 0)
	{
		return;
	}

	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		for (int slot = 1; slot <= MaxClients; slot++)
		{
			if (g_LPSBossContributorUserId[entity][slot] == userId)
			{
				g_LPSBossContributorUserId[entity][slot] = 0;
			}
		}
	}
}

void LPS_ResolveBossParticipation(int boss, bool tank, int killer, bool witchOneShot)
{
	if (boss < 1 || boss > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	int contributorCount = 0;
	int soleContributor = 0;
	for (int slot = 1; slot <= MaxClients; slot++)
	{
		int client = GetClientOfUserId(g_LPSBossContributorUserId[boss][slot]);
		if (!LPS_IsHumanSurvivorClient(client))
		{
			continue;
		}
		contributorCount++;
		soleContributor = client;
		LPS_AddPvEStat(
			client,
			tank ? LPSPvEStat_TankKillParticipations : LPSPvEStat_WitchKillParticipations,
			1
		);
	}

	if (!tank && LPS_IsHumanSurvivorClient(killer))
	{
		if (witchOneShot)
		{
			LPS_AddPvEStat(killer, LPSPvEStat_WitchOneShots, 1);
		}
		if (contributorCount == 1 && soleContributor == killer)
		{
			LPS_AddPvEStat(killer, LPSPvEStat_WitchSoloKills, 1);
		}
	}
	LPS_ResetBossTracking(boss);
}

void LPS_ResetBossTracking(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}
	for (int slot = 1; slot <= MaxClients; slot++)
	{
		g_LPSBossContributorUserId[entity][slot] = 0;
	}
}

public void LPS_Event_TankSpawnStats(Event event, const char[] name, bool dontBroadcast)
{
	int tank = event.GetInt("tankid");
	if (tank <= 0)
	{
		tank = GetClientOfUserId(event.GetInt("userid"));
	}
	LPS_ResetBossTracking(tank);
	LPS_RecordBossSpawnIncident(tank, true);
}

public void LPS_Event_WitchSpawnStats(Event event, const char[] name, bool dontBroadcast)
{
	LPS_ResetBossTracking(event.GetInt("witchid"));
	LPS_RecordBossSpawnIncident(event.GetInt("witchid"), false);
}

public void LPS_Event_UpgradePackBeginStats(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if ((!LPS_CanRecordPvEEvents() && !LPS_CanRecordVersusEvents())
		|| !LPS_IsHumanSurvivorClient(client))
	{
		return;
	}

	g_LPSPendingUpgradePackType[client] = 0;
	int weapon = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
	if (weapon <= MaxClients || !IsValidEntity(weapon))
	{
		return;
	}

	char classname[64];
	GetEntityClassname(weapon, classname, sizeof(classname));
	if (StrEqual(classname, "weapon_upgradepack_incendiary", true))
	{
		g_LPSPendingUpgradePackType[client] = 1;
	}
	else if (StrEqual(classname, "weapon_upgradepack_explosive", true))
	{
		g_LPSPendingUpgradePackType[client] = 2;
	}
}

public void LPS_Event_UpgradePackUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	bool recordPvE = LPS_CanRecordPvEEvents();
	bool recordVersus = LPS_CanRecordVersusEvents();
	if ((!recordPvE && !recordVersus) || !LPS_IsHumanSurvivorClient(client))
	{
		return;
	}

	if (g_LPSPendingUpgradePackType[client] == 1)
	{
		if (recordPvE)
		{
			LPS_AddPvEStat(client, LPSPvEStat_IncendiaryPacksDeployed, 1);
		}
		else
		{
			LPS_AddVersusSurvivorStat(
				client,
				LPSVersusSurvivorStat_IncendiaryPacksDeployed,
				1
			);
		}
	}
	else if (g_LPSPendingUpgradePackType[client] == 2)
	{
		if (recordPvE)
		{
			LPS_AddPvEStat(client, LPSPvEStat_ExplosivePacksDeployed, 1);
		}
		else
		{
			LPS_AddVersusSurvivorStat(
				client,
				LPSVersusSurvivorStat_ExplosivePacksDeployed,
				1
			);
		}
	}
	g_LPSPendingUpgradePackType[client] = 0;
}

void LPS_OnClientExtendedSegmentStarted(int client)
{
	LPS_ResetClientExtendedPvEState(client);
}

void LPS_OnClientExtendedSegmentEnding(int client)
{
	LPS_EndControl(client, LPSIncidentEnd_LifecycleEnd);
	LPS_ResetClientExtendedPvEState(client);
}

void LPS_ResetClientExtendedPvEState(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}
	g_LPSControlType[client] = LPSControlType_None;
	g_LPSControlStartedAt[client] = 0.0;
	g_LPSControlControllerUserId[client] = 0;
	g_LPSControlSaveAwarded[client] = false;
	g_LPSTongueStopHandled[client] = false;
	g_LPSChargerPummeling[client] = false;
	g_LPSPendingUpgradePackType[client] = 0;
}

void LPS_ShutdownExtendedPvEStats()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientExtendedPvEState(client);
	}
}
