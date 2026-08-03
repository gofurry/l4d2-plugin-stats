#if defined _l4d2_player_stats_pve_stats_included
	#endinput
#endif
#define _l4d2_player_stats_pve_stats_included

enum struct LPSPvEStatsSnapshot
{
	char segmentId[LPS_MAX_SEGMENT_ID];
	int commonKills;
	int specialKills;
	int tankKills;
	int witchKills;
	int damageToSpecial;
	int damageToTank;
	int damageToWitch;
	int damageTakenInfected;
	int friendlyFireToHumans;
	int friendlyFireToBots;
	int friendlyFireTaken;
	int incapacitations;
	int deaths;
	int incapRevives;
	int ledgeRescues;
	int defibRevives;
	int rescuesReceived;
	int medkitsUsedSelf;
	int medkitsUsedOnOthers;
	int medkitHealingSelf;
	int medkitHealingOthers;
	int pillsUsed;
	int adrenalineUsed;
	int temporaryHealthReceived;
	int chapterParticipations;
	int chapterCompletionsAlive;
	int chapterCompletionsDead;
	int campaignCompletions;
	int smokerKills;
	int boomerKills;
	int hunterKills;
	int spitterKills;
	int jockeyKills;
	int chargerKills;
	int damageToSmoker;
	int damageToBoomer;
	int damageToHunter;
	int damageToSpitter;
	int damageToJockey;
	int damageToCharger;
	int smokerControlsReceived;
	int hunterControlsReceived;
	int jockeyControlsReceived;
	int chargerControlsReceived;
	int smokerControlledSeconds;
	int hunterControlledSeconds;
	int jockeyControlledSeconds;
	int chargerControlledSeconds;
	int smokerSaves;
	int hunterSaves;
	int jockeySaves;
	int chargerSaves;
	int meleeTongueSelfCuts;
	int tankRocksDestroyed;
	int witchOneShots;
	int witchSoloKills;
	int tankEncounters;
	int tankKillParticipations;
	int witchEncounters;
	int witchKillParticipations;
	int incendiaryPacksDeployed;
	int explosivePacksDeployed;
	int objectiveInteractions;
	int ammoPileUses;
	int incapacitatedSeconds;
	int ledgeHangingSeconds;
	int blackWhiteTeammatesRestored;
	int revision;
}

ArrayList g_LPSClosedPvEStats;
ConVar g_LPSPainPillsDecayRate;
bool g_LPSPvEStatsQueueOverflowReported;

bool g_LPSClientPvEStatsActive[MAXPLAYERS + 1];
bool g_LPSClientPvEStatsDirty[MAXPLAYERS + 1];
LPSPvEStatsSnapshot g_LPSClientPvEStats[MAXPLAYERS + 1];
int g_LPSClientPvEStatsPersistedRevision[MAXPLAYERS + 1];

bool g_LPSFlushCapturedPvEStats[MAXPLAYERS + 1];
char g_LPSFlushCapturedPvEStatsSegmentId[MAXPLAYERS + 1][LPS_MAX_SEGMENT_ID];
int g_LPSFlushCapturedPvEStatsRevision[MAXPLAYERS + 1];
int g_LPSFlushClosedPvEStatsCount;

bool g_LPSDamageHooked[LPS_MAX_ENTITY_INDEX + 1];
bool g_LPSDamageHealthCaptured[LPS_MAX_ENTITY_INDEX + 1];
bool g_LPSDamageVictimWasIncapacitated[LPS_MAX_ENTITY_INDEX + 1];
float g_LPSDamageHealthBefore[LPS_MAX_ENTITY_INDEX + 1];
float g_LPSClientEffectiveHealthBeforeCommand[MAXPLAYERS + 1];

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

void LPS_HookStatsClient(int client)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
	{
		return;
	}

	LPS_HookStatsDamageEntity(client);
}

void LPS_HookStatsDamageEntity(int entity)
{
	if (entity < 1
		|| entity > LPS_MAX_ENTITY_INDEX
		|| !IsValidEntity(entity)
		|| g_LPSDamageHooked[entity])
	{
		return;
	}

	SDKHook(entity, SDKHook_OnTakeDamageAlive, LPS_OnTakeDamageAlivePre);
	SDKHook(entity, SDKHook_OnTakeDamageAlivePost, LPS_OnTakeDamageAlivePost);
	g_LPSDamageHooked[entity] = true;
}

void LPS_ResetStatsDamageEntity(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	g_LPSDamageHooked[entity] = false;
	g_LPSDamageHealthCaptured[entity] = false;
	g_LPSDamageVictimWasIncapacitated[entity] = false;
	g_LPSDamageHealthBefore[entity] = 0.0;
	if (entity <= MaxClients)
	{
		g_LPSClientEffectiveHealthBeforeCommand[entity] = 0.0;
	}
}

public Action LPS_OnTakeDamageAlivePre(
	int victim,
	int &attacker,
	int &inflictor,
	float &damage,
	int &damageType
)
{
	if (!LPS_CanCaptureStatsDamage(victim, damage))
	{
		return Plugin_Continue;
	}

	float health = LPS_GetTrackedEffectiveHealth(victim);
	if (health <= 0.0)
	{
		return Plugin_Continue;
	}

	g_LPSDamageHealthBefore[victim] = health;
	g_LPSDamageVictimWasIncapacitated[victim] = LPS_IsSurvivorIncapacitated(victim);
	g_LPSDamageHealthCaptured[victim] = true;
	return Plugin_Continue;
}

public void LPS_OnTakeDamageAlivePost(
	int victim,
	int attacker,
	int inflictor,
	float damage,
	int damageType
)
{
	if (victim < 1
		|| victim > LPS_MAX_ENTITY_INDEX
		|| !g_LPSDamageHealthCaptured[victim])
	{
		return;
	}

	float before = g_LPSDamageHealthBefore[victim];
	bool wasIncapacitated = g_LPSDamageVictimWasIncapacitated[victim];
	g_LPSDamageHealthCaptured[victim] = false;
	g_LPSDamageVictimWasIncapacitated[victim] = false;
	g_LPSDamageHealthBefore[victim] = 0.0;

	if (!LPS_CanCaptureStatsDamage(victim, damage)
		|| LPS_IsExcludedCombatDamage(damageType)
		|| attacker == victim)
	{
		return;
	}

	float after = LPS_GetTrackedEffectiveHealth(victim);
	float lost = before - after;
	int actualDamage = lost > 0.25 ? RoundToNearest(lost) : 0;

	// Entering the incapacitated state replaces the standing health pool with
	// incap health. In that one transition, before-after is not meaningful, so
	// cap the engine's applied damage by the pre-hit effective health instead.
	if (actualDamage <= 0
		&& !wasIncapacitated
		&& LPS_IsSurvivorIncapacitated(victim))
	{
		actualDamage = RoundToCeil(damage < before ? damage : before);
	}

	int maximumLoss = RoundToCeil(before);
	if (actualDamage > maximumLoss)
	{
		actualDamage = maximumLoss;
	}
	if (actualDamage <= 0)
	{
		return;
	}

	if (g_LPSModeFamily == LPSModeFamily_PvE)
	{
		LPS_RecordPvEDamage(victim, attacker, inflictor, actualDamage);
	}
	else if (g_LPSModeFamily == LPSModeFamily_Versus)
	{
		LPS_RecordVersusDamage(victim, attacker, inflictor, actualDamage);
	}
}

bool LPS_IsSurvivorIncapacitated(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& GetClientTeam(client) == LPS_TEAM_SURVIVOR
		&& HasEntProp(client, Prop_Send, "m_isIncapacitated")
		&& GetEntProp(client, Prop_Send, "m_isIncapacitated") != 0;
}

bool LPS_CanCaptureStatsDamage(int victim, float damage)
{
	bool supportedStatsMode = g_LPSModeFamily == LPSModeFamily_PvE
		|| (g_LPSModeFamily == LPSModeFamily_Versus
			&& g_LPSVersusStatsEnabled.BoolValue);
	if (!supportedStatsMode
		|| !g_LPSRunActive
		|| !g_LPSRoundActive
		|| damage <= 0.0
		|| victim < 1
		|| victim > LPS_MAX_ENTITY_INDEX)
	{
		return false;
	}

	if (victim <= MaxClients)
	{
		return IsClientInGame(victim)
			&& (GetClientTeam(victim) == LPS_TEAM_SURVIVOR
				|| GetClientTeam(victim) == LPS_TEAM_INFECTED);
	}

	return (g_LPSModeFamily == LPSModeFamily_PvE
			|| (g_LPSModeFamily == LPSModeFamily_Versus
				&& g_LPSVersusStatsEnabled.BoolValue))
		&& LPS_IsWitchEntity(victim);
}

float LPS_GetTrackedEffectiveHealth(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return 0.0;
	}

	if (entity <= MaxClients)
	{
		if (!IsClientInGame(entity))
		{
			return 0.0;
		}

		float health = float(GetClientHealth(entity));
		if (GetClientTeam(entity) != LPS_TEAM_SURVIVOR
			|| !HasEntProp(entity, Prop_Send, "m_healthBuffer"))
		{
			return health;
		}

		float buffer = GetEntPropFloat(entity, Prop_Send, "m_healthBuffer");
		if (buffer <= 0.0)
		{
			return health;
		}

		if (g_LPSPainPillsDecayRate != null
			&& HasEntProp(entity, Prop_Send, "m_healthBufferTime"))
		{
			float elapsed = GetGameTime() - GetEntPropFloat(entity, Prop_Send, "m_healthBufferTime");
			if (elapsed > 0.0)
			{
				buffer -= elapsed * g_LPSPainPillsDecayRate.FloatValue;
			}
		}

		if (buffer < 0.0)
		{
			buffer = 0.0;
		}
		return health + buffer;
	}

	if (HasEntProp(entity, Prop_Data, "m_iHealth"))
	{
		return float(GetEntProp(entity, Prop_Data, "m_iHealth"));
	}
	return 0.0;
}

bool LPS_IsExcludedCombatDamage(int damageType)
{
	int excluded = DMG_CRUSH
		| DMG_VEHICLE
		| DMG_FALL
		| DMG_DROWN
		| DMG_DROWNRECOVER
		| DMG_DISSOLVE;
	return (damageType & excluded) != 0;
}

void LPS_RecordPvEDamage(int victim, int attacker, int inflictor, int actualDamage)
{
	int humanAttacker = LPS_ResolveHumanSurvivorAttacker(attacker, inflictor);

	if (victim <= MaxClients && IsClientInGame(victim))
	{
		int victimTeam = GetClientTeam(victim);
		if (victimTeam == LPS_TEAM_INFECTED && humanAttacker > 0)
		{
			int zombieClass = LPS_GetZombieClass(victim);
			if (zombieClass == 8)
			{
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_DamageToTank, actualDamage);
				LPS_RecordEquipmentDamage(
					victim,
					humanAttacker,
					attacker,
					inflictor,
					LPSEquipmentStat_DamageToTank,
					actualDamage
				);
				LPS_RecordBossContribution(victim, humanAttacker);
			}
			else if (zombieClass >= 1 && zombieClass <= 6)
			{
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_DamageToSpecial, actualDamage);
				LPS_AddSpecialClassDamage(humanAttacker, zombieClass, actualDamage);
				LPS_RecordEquipmentDamage(
					victim,
					humanAttacker,
					attacker,
					inflictor,
					LPSEquipmentStat_DamageToSpecial,
					actualDamage
				);
			}
			return;
		}

		if (victimTeam != LPS_TEAM_SURVIVOR)
		{
			return;
		}

		if (humanAttacker > 0 && humanAttacker != victim)
		{
			if (IsFakeClient(victim))
			{
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_FriendlyFireToBots, actualDamage);
			}
			else
			{
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_FriendlyFireToHumans, actualDamage);
				LPS_AddPvEStat(victim, LPSPvEStat_FriendlyFireTaken, actualDamage);
			}
			return;
		}

		if (LPS_IsInfectedDamageSource(attacker) || LPS_IsInfectedDamageSource(inflictor))
		{
			LPS_AddPvEStat(victim, LPSPvEStat_DamageTakenInfected, actualDamage);
		}
		return;
	}

	if (humanAttacker > 0 && LPS_IsWitchEntity(victim))
	{
		LPS_AddPvEStat(humanAttacker, LPSPvEStat_DamageToWitch, actualDamage);
		LPS_RecordEquipmentDamage(
			victim,
			humanAttacker,
			attacker,
			inflictor,
			LPSEquipmentStat_DamageToWitch,
			actualDamage
		);
		LPS_RecordBossContribution(victim, humanAttacker);
	}
}

int LPS_ResolveHumanSurvivorAttacker(int attacker, int inflictor)
{
	if (LPS_IsHumanSurvivorClient(attacker))
	{
		return attacker;
	}

	int owner = LPS_GetEntityPlayerOwner(attacker);
	if (LPS_IsHumanSurvivorClient(owner))
	{
		return owner;
	}

	owner = LPS_GetEntityPlayerOwner(inflictor);
	return LPS_IsHumanSurvivorClient(owner) ? owner : 0;
}

int LPS_GetEntityPlayerOwner(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return 0;
	}

	int owner = 0;
	if (HasEntProp(entity, Prop_Send, "m_hOwnerEntity"))
	{
		owner = GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity");
	}
	else if (HasEntProp(entity, Prop_Data, "m_hOwnerEntity"))
	{
		owner = GetEntPropEnt(entity, Prop_Data, "m_hOwnerEntity");
	}

	if (owner >= 1 && owner <= MaxClients)
	{
		return owner;
	}

	if (HasEntProp(entity, Prop_Send, "m_hThrower"))
	{
		owner = GetEntPropEnt(entity, Prop_Send, "m_hThrower");
	}
	else if (HasEntProp(entity, Prop_Data, "m_hThrower"))
	{
		owner = GetEntPropEnt(entity, Prop_Data, "m_hThrower");
	}

	return owner >= 1 && owner <= MaxClients ? owner : 0;
}

bool LPS_IsHumanSurvivorClient(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& !IsFakeClient(client)
		&& GetClientTeam(client) == LPS_TEAM_SURVIVOR;
}

bool LPS_IsInfectedDamageSource(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return false;
	}

	if (entity <= MaxClients)
	{
		return IsClientInGame(entity) && GetClientTeam(entity) == LPS_TEAM_INFECTED;
	}

	char classname[64];
	GetEntityClassname(entity, classname, sizeof(classname));
	return StrEqual(classname, "infected", true)
		|| StrEqual(classname, "witch", true)
		|| StrEqual(classname, "witch_bride", true)
		|| StrEqual(classname, "tank_rock", true)
		|| StrEqual(classname, "insect_swarm", true);
}

bool LPS_IsWitchEntity(int entity)
{
	if (entity <= MaxClients
		|| entity < 1
		|| entity > LPS_MAX_ENTITY_INDEX
		|| !IsValidEntity(entity))
	{
		return false;
	}

	char classname[32];
	GetEntityClassname(entity, classname, sizeof(classname));
	return StrEqual(classname, "witch", true) || StrEqual(classname, "witch_bride", true);
}

int LPS_GetZombieClass(int client)
{
	if (client < 1
		|| client > MaxClients
		|| !IsClientInGame(client)
		|| !HasEntProp(client, Prop_Send, "m_zombieClass"))
	{
		return 0;
	}
	return GetEntProp(client, Prop_Send, "m_zombieClass");
}

void LPS_AddSpecialClassDamage(int client, int zombieClass, int amount)
{
	switch (zombieClass)
	{
		case 1: LPS_AddPvEStat(client, LPSPvEStat_DamageToSmoker, amount);
		case 2: LPS_AddPvEStat(client, LPSPvEStat_DamageToBoomer, amount);
		case 3: LPS_AddPvEStat(client, LPSPvEStat_DamageToHunter, amount);
		case 4: LPS_AddPvEStat(client, LPSPvEStat_DamageToSpitter, amount);
		case 5: LPS_AddPvEStat(client, LPSPvEStat_DamageToJockey, amount);
		case 6: LPS_AddPvEStat(client, LPSPvEStat_DamageToCharger, amount);
	}
}

void LPS_AddSpecialClassKill(int client, int zombieClass)
{
	switch (zombieClass)
	{
		case 1: LPS_AddPvEStat(client, LPSPvEStat_SmokerKills, 1);
		case 2: LPS_AddPvEStat(client, LPSPvEStat_BoomerKills, 1);
		case 3: LPS_AddPvEStat(client, LPSPvEStat_HunterKills, 1);
		case 4: LPS_AddPvEStat(client, LPSPvEStat_SpitterKills, 1);
		case 5: LPS_AddPvEStat(client, LPSPvEStat_JockeyKills, 1);
		case 6: LPS_AddPvEStat(client, LPSPvEStat_ChargerKills, 1);
	}
}

public void LPS_Event_InfectedDeathStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	LPS_AddPvEStat(attacker, LPSPvEStat_CommonKills, 1);
	LPS_RecordEquipmentKill(
		event.GetInt("infected_id"),
		attacker,
		event.GetInt("weapon_id"),
		"",
		event.GetBool("minigun"),
		LPSEquipmentStat_CommonKills,
		event.GetBool("headshot")
	);
}

public void LPS_Event_PlayerDeathStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	if (victim < 1 || victim > MaxClients || !IsClientInGame(victim))
	{
		return;
	}

	int victimTeam = GetClientTeam(victim);
	if (victimTeam == LPS_TEAM_SURVIVOR)
	{
		LPS_EndControl(victim);
		if (!event.GetBool("abort"))
		{
			LPS_AddPvEStat(victim, LPSPvEStat_Deaths, 1);
		}
		return;
	}

	if (victimTeam != LPS_TEAM_INFECTED)
	{
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	int zombieClass = LPS_GetZombieClass(victim);
	LPS_OnSpecialInfectedDeathForExtendedStats(victim, attacker);

	char weaponName[64];
	event.GetString("weapon", weaponName, sizeof(weaponName));
	if (zombieClass == 8)
	{
		LPS_AddPvEStat(attacker, LPSPvEStat_TankKills, 1);
		LPS_RecordEquipmentKill(
			victim,
			attacker,
			event.GetInt("weaponid"),
			weaponName,
			false,
			LPSEquipmentStat_TankKills,
			event.GetBool("headshot")
		);
		LPS_ResolveBossParticipation(victim, true, attacker, false);
	}
	else if (zombieClass >= 1 && zombieClass <= 6)
	{
		LPS_AddPvEStat(attacker, LPSPvEStat_SpecialKills, 1);
		LPS_AddSpecialClassKill(attacker, zombieClass);
		LPS_RecordEquipmentKill(
			victim,
			attacker,
			event.GetInt("weaponid"),
			weaponName,
			false,
			LPSEquipmentStat_SpecialKills,
			event.GetBool("headshot")
		);
	}
}

public void LPS_Event_WitchKilledStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		LPS_RecordVersusWitchKilled(event);
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("userid"));
	LPS_AddPvEStat(attacker, LPSPvEStat_WitchKills, 1);
	int witch = event.GetInt("witchid");
	LPS_RecordEquipmentKill(
		witch,
		attacker,
		0,
		"",
		false,
		LPSEquipmentStat_WitchKills,
		false
	);
	LPS_ResolveBossParticipation(witch, false, attacker, event.GetBool("oneshot"));
}

public void LPS_Event_PlayerIncapacitatedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	LPS_AddPvEStat(victim, LPSPvEStat_Incapacitations, 1);
}

public void LPS_Event_ReviveSuccessStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	if (event.GetBool("ledge_hang"))
	{
		LPS_AddPvEStat(rescuer, LPSPvEStat_LedgeRescues, 1);
	}
	else
	{
		LPS_AddPvEStat(rescuer, LPSPvEStat_IncapRevives, 1);
	}
	LPS_AddPvEStat(subject, LPSPvEStat_RescuesReceived, 1);
}

public void LPS_Event_DefibrillatorUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_AddPvEStat(rescuer, LPSPvEStat_DefibRevives, 1);
	LPS_AddPvEStat(subject, LPSPvEStat_RescuesReceived, 1);
}

public void LPS_Event_HealSuccessStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int healer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	int restored = event.GetInt("health_restored");
	if (healer == subject)
	{
		LPS_AddPvEStat(healer, LPSPvEStat_MedkitsUsedSelf, 1);
		LPS_AddPvEStat(healer, LPSPvEStat_MedkitHealingSelf, restored);
	}
	else
	{
		LPS_AddPvEStat(healer, LPSPvEStat_MedkitsUsedOnOthers, 1);
		LPS_AddPvEStat(healer, LPSPvEStat_MedkitHealingOthers, restored);
	}
}

public void LPS_Event_PillsUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	// The subject is the survivor who actually swallowed and received the pills.
	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_AddPvEStat(subject, LPSPvEStat_PillsUsed, 1);
	LPS_RecordTemporaryHealthAfterUse(subject);
}

public void LPS_Event_AdrenalineUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int client = GetClientOfUserId(event.GetInt("userid"));
	LPS_AddPvEStat(client, LPSPvEStat_AdrenalineUsed, 1);
	LPS_RecordTemporaryHealthAfterUse(client);
}

void LPS_RecordTemporaryHealthAfterUse(int client)
{
	if (client < 1
		|| client > MaxClients
		|| !g_LPSClientPvEStatsActive[client]
		|| !IsClientInGame(client))
	{
		return;
	}

	float before = g_LPSClientEffectiveHealthBeforeCommand[client];
	float current = LPS_GetTrackedEffectiveHealth(client);
	if (before <= 0.0)
	{
		// A synthetic event without a preceding player command cannot provide a
		// reliable baseline, so keep the use count but do not invent health.
		return;
	}

	if (current - before > 0.25)
	{
		LPS_AddPvEStat(
			client,
			LPSPvEStat_TemporaryHealthReceived,
			RoundToNearest(current - before)
		);
		return;
	}

	DataPack pack;
	CreateDataTimer(0.0, LPS_Timer_RecordTemporaryHealth, pack, TIMER_FLAG_NO_MAPCHANGE);
	pack.WriteCell(GetClientUserId(client));
	pack.WriteFloat(before);
	pack.WriteString(g_LPSClientPvEStats[client].segmentId);
}

public Action LPS_Timer_RecordTemporaryHealth(Handle timer, DataPack pack)
{
	pack.Reset();
	int client = GetClientOfUserId(pack.ReadCell());
	float before = pack.ReadFloat();
	char segmentId[LPS_MAX_SEGMENT_ID];
	pack.ReadString(segmentId, sizeof(segmentId));

	if (client < 1
		|| client > MaxClients
		|| !g_LPSClientPvEStatsActive[client]
		|| !StrEqual(g_LPSClientPvEStats[client].segmentId, segmentId, true))
	{
		return Plugin_Stop;
	}

	float gained = LPS_GetTrackedEffectiveHealth(client) - before;
	if (gained > 0.25)
	{
		LPS_AddPvEStat(
			client,
			LPSPvEStat_TemporaryHealthReceived,
			RoundToNearest(gained)
		);
	}
	return Plugin_Stop;
}

void LPS_OnPvERoundCompleting(bool campaignCompleted)
{
	if (g_LPSModeFamily != LPSModeFamily_PvE)
	{
		return;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientPvEStatsActive[client]
			|| !g_LPSClientSegmentActive[client]
			|| !IsClientInGame(client)
			|| IsFakeClient(client)
			|| GetClientTeam(client) != LPS_TEAM_SURVIVOR)
		{
			continue;
		}

		LPS_AddPvEStat(client, LPSPvEStat_ChapterParticipations, 1);
		if (IsPlayerAlive(client))
		{
			LPS_AddPvEStat(client, LPSPvEStat_ChapterCompletionsAlive, 1);
		}
		else
		{
			LPS_AddPvEStat(client, LPSPvEStat_ChapterCompletionsDead, 1);
		}

		if (campaignCompleted)
		{
			LPS_AddPvEStat(client, LPSPvEStat_CampaignCompletions, 1);
		}
	}
}

bool LPS_CanRecordPvEEvents()
{
	return g_LPSModeFamily == LPSModeFamily_PvE && g_LPSRunActive && g_LPSRoundActive;
}

void LPS_OnClientSegmentStartedForStats(int client)
{
	LPS_ResetClientPvEStatsState(client);
	LPS_ResetClientVersusStatsState(client);
	if (g_LPSModeFamily == LPSModeFamily_Versus)
	{
		LPS_OnClientVersusSegmentStarted(client);
		return;
	}

	if (g_LPSModeFamily != LPSModeFamily_PvE
		|| !g_LPSClientSegmentActive[client]
		|| !StrEqual(g_LPSClientSegmentSide[client], "survivor", true))
	{
		return;
	}

	strcopy(
		g_LPSClientPvEStats[client].segmentId,
		LPS_MAX_SEGMENT_ID,
		g_LPSClientSegmentId[client]
	);
	g_LPSClientPvEStats[client].revision = 1;
	g_LPSClientPvEStatsPersistedRevision[client] = 0;
	g_LPSClientPvEStatsDirty[client] = true;
	g_LPSClientPvEStatsActive[client] = true;
	g_LPSClientEffectiveHealthBeforeCommand[client] = LPS_GetTrackedEffectiveHealth(client);
	LPS_OnClientEquipmentSegmentStarted(client);
	LPS_OnClientExtendedSegmentStarted(client);
	LPS_OnClientPvEInteractionSegmentStarted(client);
}

void LPS_OnClientSegmentEndingForStats(int client)
{
	if (g_LPSClientPvEStatsActive[client])
	{
		LPS_OnClientPvEInteractionSegmentEnding(client);
		LPS_OnClientExtendedSegmentEnding(client);
		LPS_OnClientEquipmentSegmentEnding(client);
		LPSPvEStatsSnapshot snapshot;
		LPS_FillClientPvEStatsSnapshot(client, snapshot, true);
		LPS_QueueClosedPvEStats(snapshot);
		LPS_ResetClientPvEStatsState(client);
	}

	LPS_OnClientVersusSegmentEnding(client);
}

void LPS_AddPvEStat(int client, LPSPvEStat stat, int amount)
{
	if (amount <= 0
		|| client < 1
		|| client > MaxClients
		|| !g_LPSClientPvEStatsActive[client]
		|| !g_LPSClientSegmentActive[client]
		|| !StrEqual(g_LPSClientPvEStats[client].segmentId, g_LPSClientSegmentId[client], true))
	{
		return;
	}

	switch (stat)
	{
		case LPSPvEStat_CommonKills: g_LPSClientPvEStats[client].commonKills += amount;
		case LPSPvEStat_SpecialKills: g_LPSClientPvEStats[client].specialKills += amount;
		case LPSPvEStat_TankKills: g_LPSClientPvEStats[client].tankKills += amount;
		case LPSPvEStat_WitchKills: g_LPSClientPvEStats[client].witchKills += amount;
		case LPSPvEStat_DamageToSpecial: g_LPSClientPvEStats[client].damageToSpecial += amount;
		case LPSPvEStat_DamageToTank: g_LPSClientPvEStats[client].damageToTank += amount;
		case LPSPvEStat_DamageToWitch: g_LPSClientPvEStats[client].damageToWitch += amount;
		case LPSPvEStat_DamageTakenInfected: g_LPSClientPvEStats[client].damageTakenInfected += amount;
		case LPSPvEStat_FriendlyFireToHumans: g_LPSClientPvEStats[client].friendlyFireToHumans += amount;
		case LPSPvEStat_FriendlyFireToBots: g_LPSClientPvEStats[client].friendlyFireToBots += amount;
		case LPSPvEStat_FriendlyFireTaken: g_LPSClientPvEStats[client].friendlyFireTaken += amount;
		case LPSPvEStat_Incapacitations: g_LPSClientPvEStats[client].incapacitations += amount;
		case LPSPvEStat_Deaths: g_LPSClientPvEStats[client].deaths += amount;
		case LPSPvEStat_IncapRevives: g_LPSClientPvEStats[client].incapRevives += amount;
		case LPSPvEStat_LedgeRescues: g_LPSClientPvEStats[client].ledgeRescues += amount;
		case LPSPvEStat_DefibRevives: g_LPSClientPvEStats[client].defibRevives += amount;
		case LPSPvEStat_RescuesReceived: g_LPSClientPvEStats[client].rescuesReceived += amount;
		case LPSPvEStat_MedkitsUsedSelf: g_LPSClientPvEStats[client].medkitsUsedSelf += amount;
		case LPSPvEStat_MedkitsUsedOnOthers: g_LPSClientPvEStats[client].medkitsUsedOnOthers += amount;
		case LPSPvEStat_MedkitHealingSelf: g_LPSClientPvEStats[client].medkitHealingSelf += amount;
		case LPSPvEStat_MedkitHealingOthers: g_LPSClientPvEStats[client].medkitHealingOthers += amount;
		case LPSPvEStat_PillsUsed: g_LPSClientPvEStats[client].pillsUsed += amount;
		case LPSPvEStat_AdrenalineUsed: g_LPSClientPvEStats[client].adrenalineUsed += amount;
		case LPSPvEStat_TemporaryHealthReceived: g_LPSClientPvEStats[client].temporaryHealthReceived += amount;
		case LPSPvEStat_ChapterParticipations: g_LPSClientPvEStats[client].chapterParticipations += amount;
		case LPSPvEStat_ChapterCompletionsAlive: g_LPSClientPvEStats[client].chapterCompletionsAlive += amount;
		case LPSPvEStat_ChapterCompletionsDead: g_LPSClientPvEStats[client].chapterCompletionsDead += amount;
		case LPSPvEStat_CampaignCompletions: g_LPSClientPvEStats[client].campaignCompletions += amount;
		case LPSPvEStat_SmokerKills: g_LPSClientPvEStats[client].smokerKills += amount;
		case LPSPvEStat_BoomerKills: g_LPSClientPvEStats[client].boomerKills += amount;
		case LPSPvEStat_HunterKills: g_LPSClientPvEStats[client].hunterKills += amount;
		case LPSPvEStat_SpitterKills: g_LPSClientPvEStats[client].spitterKills += amount;
		case LPSPvEStat_JockeyKills: g_LPSClientPvEStats[client].jockeyKills += amount;
		case LPSPvEStat_ChargerKills: g_LPSClientPvEStats[client].chargerKills += amount;
		case LPSPvEStat_DamageToSmoker: g_LPSClientPvEStats[client].damageToSmoker += amount;
		case LPSPvEStat_DamageToBoomer: g_LPSClientPvEStats[client].damageToBoomer += amount;
		case LPSPvEStat_DamageToHunter: g_LPSClientPvEStats[client].damageToHunter += amount;
		case LPSPvEStat_DamageToSpitter: g_LPSClientPvEStats[client].damageToSpitter += amount;
		case LPSPvEStat_DamageToJockey: g_LPSClientPvEStats[client].damageToJockey += amount;
		case LPSPvEStat_DamageToCharger: g_LPSClientPvEStats[client].damageToCharger += amount;
		case LPSPvEStat_SmokerControlsReceived: g_LPSClientPvEStats[client].smokerControlsReceived += amount;
		case LPSPvEStat_HunterControlsReceived: g_LPSClientPvEStats[client].hunterControlsReceived += amount;
		case LPSPvEStat_JockeyControlsReceived: g_LPSClientPvEStats[client].jockeyControlsReceived += amount;
		case LPSPvEStat_ChargerControlsReceived: g_LPSClientPvEStats[client].chargerControlsReceived += amount;
		case LPSPvEStat_SmokerControlledSeconds: g_LPSClientPvEStats[client].smokerControlledSeconds += amount;
		case LPSPvEStat_HunterControlledSeconds: g_LPSClientPvEStats[client].hunterControlledSeconds += amount;
		case LPSPvEStat_JockeyControlledSeconds: g_LPSClientPvEStats[client].jockeyControlledSeconds += amount;
		case LPSPvEStat_ChargerControlledSeconds: g_LPSClientPvEStats[client].chargerControlledSeconds += amount;
		case LPSPvEStat_SmokerSaves: g_LPSClientPvEStats[client].smokerSaves += amount;
		case LPSPvEStat_HunterSaves: g_LPSClientPvEStats[client].hunterSaves += amount;
		case LPSPvEStat_JockeySaves: g_LPSClientPvEStats[client].jockeySaves += amount;
		case LPSPvEStat_ChargerSaves: g_LPSClientPvEStats[client].chargerSaves += amount;
		case LPSPvEStat_MeleeTongueSelfCuts: g_LPSClientPvEStats[client].meleeTongueSelfCuts += amount;
		case LPSPvEStat_TankRocksDestroyed: g_LPSClientPvEStats[client].tankRocksDestroyed += amount;
		case LPSPvEStat_WitchOneShots: g_LPSClientPvEStats[client].witchOneShots += amount;
		case LPSPvEStat_WitchSoloKills: g_LPSClientPvEStats[client].witchSoloKills += amount;
		case LPSPvEStat_TankEncounters: g_LPSClientPvEStats[client].tankEncounters += amount;
		case LPSPvEStat_TankKillParticipations: g_LPSClientPvEStats[client].tankKillParticipations += amount;
		case LPSPvEStat_WitchEncounters: g_LPSClientPvEStats[client].witchEncounters += amount;
		case LPSPvEStat_WitchKillParticipations: g_LPSClientPvEStats[client].witchKillParticipations += amount;
		case LPSPvEStat_IncendiaryPacksDeployed: g_LPSClientPvEStats[client].incendiaryPacksDeployed += amount;
		case LPSPvEStat_ExplosivePacksDeployed: g_LPSClientPvEStats[client].explosivePacksDeployed += amount;
		case LPSPvEStat_ObjectiveInteractions: g_LPSClientPvEStats[client].objectiveInteractions += amount;
		case LPSPvEStat_AmmoPileUses: g_LPSClientPvEStats[client].ammoPileUses += amount;
		case LPSPvEStat_IncapacitatedSeconds: g_LPSClientPvEStats[client].incapacitatedSeconds += amount;
		case LPSPvEStat_LedgeHangingSeconds: g_LPSClientPvEStats[client].ledgeHangingSeconds += amount;
		case LPSPvEStat_BlackWhiteTeammatesRestored: g_LPSClientPvEStats[client].blackWhiteTeammatesRestored += amount;
	}

	g_LPSClientPvEStats[client].revision++;
	g_LPSClientPvEStatsDirty[client] = true;
}

void LPS_FillClientPvEStatsSnapshot(int client, LPSPvEStatsSnapshot snapshot, bool closed)
{
	strcopy(snapshot.segmentId, sizeof(snapshot.segmentId), g_LPSClientPvEStats[client].segmentId);
	snapshot.commonKills = g_LPSClientPvEStats[client].commonKills;
	snapshot.specialKills = g_LPSClientPvEStats[client].specialKills;
	snapshot.tankKills = g_LPSClientPvEStats[client].tankKills;
	snapshot.witchKills = g_LPSClientPvEStats[client].witchKills;
	snapshot.damageToSpecial = g_LPSClientPvEStats[client].damageToSpecial;
	snapshot.damageToTank = g_LPSClientPvEStats[client].damageToTank;
	snapshot.damageToWitch = g_LPSClientPvEStats[client].damageToWitch;
	snapshot.damageTakenInfected = g_LPSClientPvEStats[client].damageTakenInfected;
	snapshot.friendlyFireToHumans = g_LPSClientPvEStats[client].friendlyFireToHumans;
	snapshot.friendlyFireToBots = g_LPSClientPvEStats[client].friendlyFireToBots;
	snapshot.friendlyFireTaken = g_LPSClientPvEStats[client].friendlyFireTaken;
	snapshot.incapacitations = g_LPSClientPvEStats[client].incapacitations;
	snapshot.deaths = g_LPSClientPvEStats[client].deaths;
	snapshot.incapRevives = g_LPSClientPvEStats[client].incapRevives;
	snapshot.ledgeRescues = g_LPSClientPvEStats[client].ledgeRescues;
	snapshot.defibRevives = g_LPSClientPvEStats[client].defibRevives;
	snapshot.rescuesReceived = g_LPSClientPvEStats[client].rescuesReceived;
	snapshot.medkitsUsedSelf = g_LPSClientPvEStats[client].medkitsUsedSelf;
	snapshot.medkitsUsedOnOthers = g_LPSClientPvEStats[client].medkitsUsedOnOthers;
	snapshot.medkitHealingSelf = g_LPSClientPvEStats[client].medkitHealingSelf;
	snapshot.medkitHealingOthers = g_LPSClientPvEStats[client].medkitHealingOthers;
	snapshot.pillsUsed = g_LPSClientPvEStats[client].pillsUsed;
	snapshot.adrenalineUsed = g_LPSClientPvEStats[client].adrenalineUsed;
	snapshot.temporaryHealthReceived = g_LPSClientPvEStats[client].temporaryHealthReceived;
	snapshot.chapterParticipations = g_LPSClientPvEStats[client].chapterParticipations;
	snapshot.chapterCompletionsAlive = g_LPSClientPvEStats[client].chapterCompletionsAlive;
	snapshot.chapterCompletionsDead = g_LPSClientPvEStats[client].chapterCompletionsDead;
	snapshot.campaignCompletions = g_LPSClientPvEStats[client].campaignCompletions;
	snapshot.smokerKills = g_LPSClientPvEStats[client].smokerKills;
	snapshot.boomerKills = g_LPSClientPvEStats[client].boomerKills;
	snapshot.hunterKills = g_LPSClientPvEStats[client].hunterKills;
	snapshot.spitterKills = g_LPSClientPvEStats[client].spitterKills;
	snapshot.jockeyKills = g_LPSClientPvEStats[client].jockeyKills;
	snapshot.chargerKills = g_LPSClientPvEStats[client].chargerKills;
	snapshot.damageToSmoker = g_LPSClientPvEStats[client].damageToSmoker;
	snapshot.damageToBoomer = g_LPSClientPvEStats[client].damageToBoomer;
	snapshot.damageToHunter = g_LPSClientPvEStats[client].damageToHunter;
	snapshot.damageToSpitter = g_LPSClientPvEStats[client].damageToSpitter;
	snapshot.damageToJockey = g_LPSClientPvEStats[client].damageToJockey;
	snapshot.damageToCharger = g_LPSClientPvEStats[client].damageToCharger;
	snapshot.smokerControlsReceived = g_LPSClientPvEStats[client].smokerControlsReceived;
	snapshot.hunterControlsReceived = g_LPSClientPvEStats[client].hunterControlsReceived;
	snapshot.jockeyControlsReceived = g_LPSClientPvEStats[client].jockeyControlsReceived;
	snapshot.chargerControlsReceived = g_LPSClientPvEStats[client].chargerControlsReceived;
	snapshot.smokerControlledSeconds = g_LPSClientPvEStats[client].smokerControlledSeconds;
	snapshot.hunterControlledSeconds = g_LPSClientPvEStats[client].hunterControlledSeconds;
	snapshot.jockeyControlledSeconds = g_LPSClientPvEStats[client].jockeyControlledSeconds;
	snapshot.chargerControlledSeconds = g_LPSClientPvEStats[client].chargerControlledSeconds;
	snapshot.smokerSaves = g_LPSClientPvEStats[client].smokerSaves;
	snapshot.hunterSaves = g_LPSClientPvEStats[client].hunterSaves;
	snapshot.jockeySaves = g_LPSClientPvEStats[client].jockeySaves;
	snapshot.chargerSaves = g_LPSClientPvEStats[client].chargerSaves;
	snapshot.meleeTongueSelfCuts = g_LPSClientPvEStats[client].meleeTongueSelfCuts;
	snapshot.tankRocksDestroyed = g_LPSClientPvEStats[client].tankRocksDestroyed;
	snapshot.witchOneShots = g_LPSClientPvEStats[client].witchOneShots;
	snapshot.witchSoloKills = g_LPSClientPvEStats[client].witchSoloKills;
	snapshot.tankEncounters = g_LPSClientPvEStats[client].tankEncounters;
	snapshot.tankKillParticipations = g_LPSClientPvEStats[client].tankKillParticipations;
	snapshot.witchEncounters = g_LPSClientPvEStats[client].witchEncounters;
	snapshot.witchKillParticipations = g_LPSClientPvEStats[client].witchKillParticipations;
	snapshot.incendiaryPacksDeployed = g_LPSClientPvEStats[client].incendiaryPacksDeployed;
	snapshot.explosivePacksDeployed = g_LPSClientPvEStats[client].explosivePacksDeployed;
	snapshot.objectiveInteractions = g_LPSClientPvEStats[client].objectiveInteractions;
	snapshot.ammoPileUses = g_LPSClientPvEStats[client].ammoPileUses;
	snapshot.incapacitatedSeconds = g_LPSClientPvEStats[client].incapacitatedSeconds;
	snapshot.ledgeHangingSeconds = g_LPSClientPvEStats[client].ledgeHangingSeconds;
	snapshot.blackWhiteTeammatesRestored = g_LPSClientPvEStats[client].blackWhiteTeammatesRestored;
	snapshot.revision = g_LPSClientPvEStats[client].revision + (closed ? 1 : 0);
}

void LPS_QueueClosedPvEStats(LPSPvEStatsSnapshot snapshot)
{
	if (g_LPSClosedPvEStats.Length >= g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		if (!g_LPSPvEStatsQueueOverflowReported)
		{
			LogError(
				"Closed PvE stats memory queue reached its limit (%d); additional snapshots will be dropped until a flush succeeds.",
				g_LPSClosedLifecycleQueueLimit.IntValue
			);
			g_LPSPvEStatsQueueOverflowReported = true;
		}
		return;
	}

	g_LPSClosedPvEStats.PushArray(snapshot, sizeof(snapshot));
}

bool LPS_AddPvEStatsFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_AccumulateAllPvEInteractionDurations();
	LPS_ResetPvEStatsFlushCapture();
	g_LPSFlushClosedPvEStatsCount = g_LPSClosedPvEStats.Length;

	LPSPvEStatsSnapshot snapshot;
	for (int index = 0; index < g_LPSFlushClosedPvEStatsCount; index++)
	{
		g_LPSClosedPvEStats.GetArray(index, snapshot, sizeof(snapshot));
		if (!LPS_AddPvEStatsSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientPvEStatsActive[client] || !g_LPSClientPvEStatsDirty[client])
		{
			continue;
		}

		LPS_FillClientPvEStatsSnapshot(client, snapshot, false);
		if (!LPS_AddPvEStatsSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}

		g_LPSFlushCapturedPvEStats[client] = true;
		g_LPSFlushCapturedPvEStatsRevision[client] = snapshot.revision;
		strcopy(
			g_LPSFlushCapturedPvEStatsSegmentId[client],
			sizeof(g_LPSFlushCapturedPvEStatsSegmentId[]),
			snapshot.segmentId
		);
	}
	return true;
}

bool LPS_AddPvEStatsSnapshotQuery(
	Transaction transaction,
	LPSPvEStatsSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[LPS_MAX_QUERY_LENGTH];
	if (!LPS_BuildPvEStatsUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a PvE stats value exceeded its SQL escape buffer");
		return false;
	}

	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_BuildPvEStatsUpsertQuery(LPSPvEStatsSnapshot snapshot, char[] query, int maxLength)
{
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	char values[4096];
	int written = FormatEx(
		values,
		sizeof(values),
		"'%s', %d, %d",
		segmentId,
		LPS_STATS_VERSION,
		GetTime()
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.commonKills,
		snapshot.specialKills,
		snapshot.tankKills,
		snapshot.witchKills,
		snapshot.damageToSpecial,
		snapshot.damageToTank,
		snapshot.damageToWitch,
		snapshot.damageTakenInfected,
		snapshot.friendlyFireToHumans,
		snapshot.friendlyFireToBots,
		snapshot.friendlyFireTaken,
		snapshot.incapacitations,
		snapshot.deaths,
		snapshot.incapRevives
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.ledgeRescues,
		snapshot.defibRevives,
		snapshot.rescuesReceived,
		snapshot.medkitsUsedSelf,
		snapshot.medkitsUsedOnOthers,
		snapshot.medkitHealingSelf,
		snapshot.medkitHealingOthers,
		snapshot.pillsUsed,
		snapshot.adrenalineUsed,
		snapshot.temporaryHealthReceived,
		snapshot.chapterParticipations,
		snapshot.chapterCompletionsAlive,
		snapshot.chapterCompletionsDead,
		snapshot.campaignCompletions
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.smokerKills,
		snapshot.boomerKills,
		snapshot.hunterKills,
		snapshot.spitterKills,
		snapshot.jockeyKills,
		snapshot.chargerKills,
		snapshot.damageToSmoker,
		snapshot.damageToBoomer,
		snapshot.damageToHunter,
		snapshot.damageToSpitter,
		snapshot.damageToJockey,
		snapshot.damageToCharger,
		snapshot.smokerControlsReceived,
		snapshot.hunterControlsReceived,
		snapshot.jockeyControlsReceived,
		snapshot.chargerControlsReceived,
		snapshot.smokerControlledSeconds
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.hunterControlledSeconds,
		snapshot.jockeyControlledSeconds,
		snapshot.chargerControlledSeconds,
		snapshot.smokerSaves,
		snapshot.hunterSaves,
		snapshot.jockeySaves,
		snapshot.chargerSaves,
		snapshot.meleeTongueSelfCuts,
		snapshot.tankRocksDestroyed,
		snapshot.witchOneShots,
		snapshot.witchSoloKills,
		snapshot.tankEncounters,
		snapshot.tankKillParticipations,
		snapshot.witchEncounters,
		snapshot.witchKillParticipations,
		snapshot.incendiaryPacksDeployed,
		snapshot.explosivePacksDeployed,
		snapshot.objectiveInteractions,
		snapshot.ammoPileUses,
		snapshot.incapacitatedSeconds,
		snapshot.ledgeHangingSeconds,
		snapshot.blackWhiteTeammatesRestored,
		snapshot.revision
	);
	if (written >= sizeof(values) - 1)
	{
		return false;
	}

	char columns[] = "segment_id, stats_version, last_saved_at, common_kills, special_kills, tank_kills, witch_kills, damage_to_special, damage_to_tank, damage_to_witch, damage_taken_infected, friendly_fire_to_humans, friendly_fire_to_bots, friendly_fire_taken, incapacitations, deaths, incap_revives, ledge_rescues, defib_revives, rescues_received, medkits_used_self, medkits_used_on_others, medkit_healing_self, medkit_healing_others, pills_used, adrenaline_used, temporary_health_received, chapter_participations, chapter_completions_alive, chapter_completions_dead, campaign_completions, smoker_kills, boomer_kills, hunter_kills, spitter_kills, jockey_kills, charger_kills, damage_to_smoker, damage_to_boomer, damage_to_hunter, damage_to_spitter, damage_to_jockey, damage_to_charger, smoker_controls_received, hunter_controls_received, jockey_controls_received, charger_controls_received, smoker_controlled_seconds, hunter_controlled_seconds, jockey_controlled_seconds, charger_controlled_seconds, smoker_saves, hunter_saves, jockey_saves, charger_saves, melee_tongue_self_cuts, tank_rocks_destroyed, witch_oneshots, witch_solo_kills, tank_encounters, tank_kill_participations, witch_encounters, witch_kill_participations, incendiary_packs_deployed, explosive_packs_deployed, objective_interactions, ammo_pile_uses, incapacitated_seconds, ledge_hanging_seconds, black_white_teammates_restored, revision";
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_pve_segment_stats (%s) VALUES (%s) ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), common_kills = VALUES(common_kills), special_kills = VALUES(special_kills), tank_kills = VALUES(tank_kills), witch_kills = VALUES(witch_kills), damage_to_special = VALUES(damage_to_special), damage_to_tank = VALUES(damage_to_tank), damage_to_witch = VALUES(damage_to_witch), damage_taken_infected = VALUES(damage_taken_infected), friendly_fire_to_humans = VALUES(friendly_fire_to_humans), friendly_fire_to_bots = VALUES(friendly_fire_to_bots), friendly_fire_taken = VALUES(friendly_fire_taken), incapacitations = VALUES(incapacitations), deaths = VALUES(deaths), incap_revives = VALUES(incap_revives), ledge_rescues = VALUES(ledge_rescues), defib_revives = VALUES(defib_revives), rescues_received = VALUES(rescues_received), medkits_used_self = VALUES(medkits_used_self), medkits_used_on_others = VALUES(medkits_used_on_others), medkit_healing_self = VALUES(medkit_healing_self), medkit_healing_others = VALUES(medkit_healing_others), pills_used = VALUES(pills_used), adrenaline_used = VALUES(adrenaline_used), temporary_health_received = VALUES(temporary_health_received), chapter_participations = VALUES(chapter_participations), chapter_completions_alive = VALUES(chapter_completions_alive), chapter_completions_dead = VALUES(chapter_completions_dead), campaign_completions = VALUES(campaign_completions), smoker_kills = VALUES(smoker_kills), boomer_kills = VALUES(boomer_kills), hunter_kills = VALUES(hunter_kills), spitter_kills = VALUES(spitter_kills), jockey_kills = VALUES(jockey_kills), charger_kills = VALUES(charger_kills), damage_to_smoker = VALUES(damage_to_smoker), damage_to_boomer = VALUES(damage_to_boomer), damage_to_hunter = VALUES(damage_to_hunter), damage_to_spitter = VALUES(damage_to_spitter), damage_to_jockey = VALUES(damage_to_jockey), damage_to_charger = VALUES(damage_to_charger), smoker_controls_received = VALUES(smoker_controls_received), hunter_controls_received = VALUES(hunter_controls_received), jockey_controls_received = VALUES(jockey_controls_received), charger_controls_received = VALUES(charger_controls_received), smoker_controlled_seconds = VALUES(smoker_controlled_seconds), hunter_controlled_seconds = VALUES(hunter_controlled_seconds), jockey_controlled_seconds = VALUES(jockey_controlled_seconds), charger_controlled_seconds = VALUES(charger_controlled_seconds), smoker_saves = VALUES(smoker_saves), hunter_saves = VALUES(hunter_saves), jockey_saves = VALUES(jockey_saves), charger_saves = VALUES(charger_saves), melee_tongue_self_cuts = VALUES(melee_tongue_self_cuts), tank_rocks_destroyed = VALUES(tank_rocks_destroyed), witch_oneshots = VALUES(witch_oneshots), witch_solo_kills = VALUES(witch_solo_kills), tank_encounters = VALUES(tank_encounters), tank_kill_participations = VALUES(tank_kill_participations), witch_encounters = VALUES(witch_encounters), witch_kill_participations = VALUES(witch_kill_participations), incendiary_packs_deployed = VALUES(incendiary_packs_deployed), explosive_packs_deployed = VALUES(explosive_packs_deployed), objective_interactions = VALUES(objective_interactions), ammo_pile_uses = VALUES(ammo_pile_uses), incapacitated_seconds = VALUES(incapacitated_seconds), ledge_hanging_seconds = VALUES(ledge_hanging_seconds), black_white_teammates_restored = VALUES(black_white_teammates_restored), revision = VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_pve_segment_stats (%s) VALUES (%s) ON CONFLICT(segment_id) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, common_kills = excluded.common_kills, special_kills = excluded.special_kills, tank_kills = excluded.tank_kills, witch_kills = excluded.witch_kills, damage_to_special = excluded.damage_to_special, damage_to_tank = excluded.damage_to_tank, damage_to_witch = excluded.damage_to_witch, damage_taken_infected = excluded.damage_taken_infected, friendly_fire_to_humans = excluded.friendly_fire_to_humans, friendly_fire_to_bots = excluded.friendly_fire_to_bots, friendly_fire_taken = excluded.friendly_fire_taken, incapacitations = excluded.incapacitations, deaths = excluded.deaths, incap_revives = excluded.incap_revives, ledge_rescues = excluded.ledge_rescues, defib_revives = excluded.defib_revives, rescues_received = excluded.rescues_received, medkits_used_self = excluded.medkits_used_self, medkits_used_on_others = excluded.medkits_used_on_others, medkit_healing_self = excluded.medkit_healing_self, medkit_healing_others = excluded.medkit_healing_others, pills_used = excluded.pills_used, adrenaline_used = excluded.adrenaline_used, temporary_health_received = excluded.temporary_health_received, chapter_participations = excluded.chapter_participations, chapter_completions_alive = excluded.chapter_completions_alive, chapter_completions_dead = excluded.chapter_completions_dead, campaign_completions = excluded.campaign_completions, smoker_kills = excluded.smoker_kills, boomer_kills = excluded.boomer_kills, hunter_kills = excluded.hunter_kills, spitter_kills = excluded.spitter_kills, jockey_kills = excluded.jockey_kills, charger_kills = excluded.charger_kills, damage_to_smoker = excluded.damage_to_smoker, damage_to_boomer = excluded.damage_to_boomer, damage_to_hunter = excluded.damage_to_hunter, damage_to_spitter = excluded.damage_to_spitter, damage_to_jockey = excluded.damage_to_jockey, damage_to_charger = excluded.damage_to_charger, smoker_controls_received = excluded.smoker_controls_received, hunter_controls_received = excluded.hunter_controls_received, jockey_controls_received = excluded.jockey_controls_received, charger_controls_received = excluded.charger_controls_received, smoker_controlled_seconds = excluded.smoker_controlled_seconds, hunter_controlled_seconds = excluded.hunter_controlled_seconds, jockey_controlled_seconds = excluded.jockey_controlled_seconds, charger_controlled_seconds = excluded.charger_controlled_seconds, smoker_saves = excluded.smoker_saves, hunter_saves = excluded.hunter_saves, jockey_saves = excluded.jockey_saves, charger_saves = excluded.charger_saves, melee_tongue_self_cuts = excluded.melee_tongue_self_cuts, tank_rocks_destroyed = excluded.tank_rocks_destroyed, witch_oneshots = excluded.witch_oneshots, witch_solo_kills = excluded.witch_solo_kills, tank_encounters = excluded.tank_encounters, tank_kill_participations = excluded.tank_kill_participations, witch_encounters = excluded.witch_encounters, witch_kill_participations = excluded.witch_kill_participations, incendiary_packs_deployed = excluded.incendiary_packs_deployed, explosive_packs_deployed = excluded.explosive_packs_deployed, objective_interactions = excluded.objective_interactions, ammo_pile_uses = excluded.ammo_pile_uses, incapacitated_seconds = excluded.incapacitated_seconds, ledge_hanging_seconds = excluded.ledge_hanging_seconds, black_white_teammates_restored = excluded.black_white_teammates_restored, revision = excluded.revision",
			columns,
			values
		);
	}
	return true;
}

void LPS_OnPvEStatsFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedPvEStatsCount && g_LPSClosedPvEStats.Length > 0; index++)
	{
		g_LPSClosedPvEStats.Erase(0);
	}

	if (g_LPSClosedPvEStats.Length < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		g_LPSPvEStatsQueueOverflowReported = false;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSFlushCapturedPvEStats[client]
			|| !g_LPSClientPvEStatsActive[client]
			|| !StrEqual(
				g_LPSClientPvEStats[client].segmentId,
				g_LPSFlushCapturedPvEStatsSegmentId[client],
				true
			))
		{
			continue;
		}

		g_LPSClientPvEStatsPersistedRevision[client] = g_LPSFlushCapturedPvEStatsRevision[client];
		if (g_LPSClientPvEStats[client].revision <= g_LPSFlushCapturedPvEStatsRevision[client])
		{
			g_LPSClientPvEStatsDirty[client] = false;
		}
	}
	LPS_ResetPvEStatsFlushCapture();
}

void LPS_ResetPvEStatsFlushCapture()
{
	g_LPSFlushClosedPvEStatsCount = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSFlushCapturedPvEStats[client] = false;
		g_LPSFlushCapturedPvEStatsSegmentId[client][0] = '\0';
		g_LPSFlushCapturedPvEStatsRevision[client] = 0;
	}
}

void LPS_ResetClientPvEStatsState(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	g_LPSClientPvEStatsActive[client] = false;
	g_LPSClientPvEStatsDirty[client] = false;
	g_LPSClientPvEStatsPersistedRevision[client] = 0;
	g_LPSClientPvEStats[client].segmentId[0] = '\0';
	g_LPSClientPvEStats[client].commonKills = 0;
	g_LPSClientPvEStats[client].specialKills = 0;
	g_LPSClientPvEStats[client].tankKills = 0;
	g_LPSClientPvEStats[client].witchKills = 0;
	g_LPSClientPvEStats[client].damageToSpecial = 0;
	g_LPSClientPvEStats[client].damageToTank = 0;
	g_LPSClientPvEStats[client].damageToWitch = 0;
	g_LPSClientPvEStats[client].damageTakenInfected = 0;
	g_LPSClientPvEStats[client].friendlyFireToHumans = 0;
	g_LPSClientPvEStats[client].friendlyFireToBots = 0;
	g_LPSClientPvEStats[client].friendlyFireTaken = 0;
	g_LPSClientPvEStats[client].incapacitations = 0;
	g_LPSClientPvEStats[client].deaths = 0;
	g_LPSClientPvEStats[client].incapRevives = 0;
	g_LPSClientPvEStats[client].ledgeRescues = 0;
	g_LPSClientPvEStats[client].defibRevives = 0;
	g_LPSClientPvEStats[client].rescuesReceived = 0;
	g_LPSClientPvEStats[client].medkitsUsedSelf = 0;
	g_LPSClientPvEStats[client].medkitsUsedOnOthers = 0;
	g_LPSClientPvEStats[client].medkitHealingSelf = 0;
	g_LPSClientPvEStats[client].medkitHealingOthers = 0;
	g_LPSClientPvEStats[client].pillsUsed = 0;
	g_LPSClientPvEStats[client].adrenalineUsed = 0;
	g_LPSClientPvEStats[client].temporaryHealthReceived = 0;
	g_LPSClientPvEStats[client].chapterParticipations = 0;
	g_LPSClientPvEStats[client].chapterCompletionsAlive = 0;
	g_LPSClientPvEStats[client].chapterCompletionsDead = 0;
	g_LPSClientPvEStats[client].campaignCompletions = 0;
	g_LPSClientPvEStats[client].smokerKills = 0;
	g_LPSClientPvEStats[client].boomerKills = 0;
	g_LPSClientPvEStats[client].hunterKills = 0;
	g_LPSClientPvEStats[client].spitterKills = 0;
	g_LPSClientPvEStats[client].jockeyKills = 0;
	g_LPSClientPvEStats[client].chargerKills = 0;
	g_LPSClientPvEStats[client].damageToSmoker = 0;
	g_LPSClientPvEStats[client].damageToBoomer = 0;
	g_LPSClientPvEStats[client].damageToHunter = 0;
	g_LPSClientPvEStats[client].damageToSpitter = 0;
	g_LPSClientPvEStats[client].damageToJockey = 0;
	g_LPSClientPvEStats[client].damageToCharger = 0;
	g_LPSClientPvEStats[client].smokerControlsReceived = 0;
	g_LPSClientPvEStats[client].hunterControlsReceived = 0;
	g_LPSClientPvEStats[client].jockeyControlsReceived = 0;
	g_LPSClientPvEStats[client].chargerControlsReceived = 0;
	g_LPSClientPvEStats[client].smokerControlledSeconds = 0;
	g_LPSClientPvEStats[client].hunterControlledSeconds = 0;
	g_LPSClientPvEStats[client].jockeyControlledSeconds = 0;
	g_LPSClientPvEStats[client].chargerControlledSeconds = 0;
	g_LPSClientPvEStats[client].smokerSaves = 0;
	g_LPSClientPvEStats[client].hunterSaves = 0;
	g_LPSClientPvEStats[client].jockeySaves = 0;
	g_LPSClientPvEStats[client].chargerSaves = 0;
	g_LPSClientPvEStats[client].meleeTongueSelfCuts = 0;
	g_LPSClientPvEStats[client].tankRocksDestroyed = 0;
	g_LPSClientPvEStats[client].witchOneShots = 0;
	g_LPSClientPvEStats[client].witchSoloKills = 0;
	g_LPSClientPvEStats[client].tankEncounters = 0;
	g_LPSClientPvEStats[client].tankKillParticipations = 0;
	g_LPSClientPvEStats[client].witchEncounters = 0;
	g_LPSClientPvEStats[client].witchKillParticipations = 0;
	g_LPSClientPvEStats[client].incendiaryPacksDeployed = 0;
	g_LPSClientPvEStats[client].explosivePacksDeployed = 0;
	g_LPSClientPvEStats[client].objectiveInteractions = 0;
	g_LPSClientPvEStats[client].ammoPileUses = 0;
	g_LPSClientPvEStats[client].incapacitatedSeconds = 0;
	g_LPSClientPvEStats[client].ledgeHangingSeconds = 0;
	g_LPSClientPvEStats[client].blackWhiteTeammatesRestored = 0;
	LPS_ResetClientPvEInteractionState(client);
	g_LPSClientPvEStats[client].revision = 0;
}

int LPS_GetActivePvEStatsCount()
{
	int count = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientPvEStatsActive[client])
		{
			count++;
		}
	}
	return count;
}

int LPS_GetPendingClosedPvEStatsCount()
{
	return g_LPSClosedPvEStats == null ? 0 : g_LPSClosedPvEStats.Length;
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
