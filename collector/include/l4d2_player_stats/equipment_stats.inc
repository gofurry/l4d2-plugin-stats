#if defined _l4d2_player_stats_equipment_stats_included
	#endinput
#endif
#define _l4d2_player_stats_equipment_stats_included

enum struct LPSEquipmentStatsValue
{
	int actions;
	int commonKills;
	int specialKills;
	int tankKills;
	int witchKills;
	int headshotKills;
	int damageToSpecial;
	int damageToTank;
	int damageToWitch;
	int revision;
}

enum struct LPSEquipmentStatsSnapshot
{
	char segmentId[LPS_MAX_SEGMENT_ID];
	int equipmentId;
	int actions;
	int commonKills;
	int specialKills;
	int tankKills;
	int witchKills;
	int headshotKills;
	int damageToSpecial;
	int damageToTank;
	int damageToWitch;
	int revision;
}

StringMap g_LPSEquipmentNames;
StringMap g_LPSMeleeNames;
ArrayList g_LPSClosedEquipmentStats;
bool g_LPSEquipmentQueueOverflowReported;

LPSEquipmentStatsValue g_LPSClientEquipmentStats[MAXPLAYERS + 1][LPSEquipment_Count];
bool g_LPSClientEquipmentDirty[MAXPLAYERS + 1][LPSEquipment_Count];
int g_LPSClientEquipmentPersistedRevision[MAXPLAYERS + 1][LPSEquipment_Count];

bool g_LPSFlushCapturedEquipment[MAXPLAYERS + 1][LPSEquipment_Count];
int g_LPSFlushCapturedEquipmentRevision[MAXPLAYERS + 1][LPSEquipment_Count];
char g_LPSFlushCapturedEquipmentSegmentId[MAXPLAYERS + 1][LPS_MAX_SEGMENT_ID];
int g_LPSFlushClosedEquipmentCount;

LPSEquipment g_LPSEntityEquipment[LPS_MAX_ENTITY_INDEX + 1];
LPSEquipment g_LPSLastDamageEquipment[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSLastDamageAttackerUserId[LPS_MAX_ENTITY_INDEX + 1];

void LPS_InitializeEquipmentStats()
{
	g_LPSEquipmentNames = new StringMap();
	g_LPSMeleeNames = new StringMap();
	g_LPSClosedEquipmentStats = new ArrayList(sizeof(LPSEquipmentStatsSnapshot));
	g_LPSEquipmentQueueOverflowReported = false;
	g_LPSFlushClosedEquipmentCount = 0;

	LPS_RegisterEquipmentName("pistol", LPSEquipment_Pistol);
	LPS_RegisterEquipmentName("pistol_magnum", LPSEquipment_Magnum);
	LPS_RegisterEquipmentName("smg", LPSEquipment_SMG);
	LPS_RegisterEquipmentName("smg_silenced", LPSEquipment_SilencedSMG);
	LPS_RegisterEquipmentName("smg_mp5", LPSEquipment_MP5);
	LPS_RegisterEquipmentName("pumpshotgun", LPSEquipment_PumpShotgun);
	LPS_RegisterEquipmentName("shotgun_chrome", LPSEquipment_ChromeShotgun);
	LPS_RegisterEquipmentName("autoshotgun", LPSEquipment_AutoShotgun);
	LPS_RegisterEquipmentName("shotgun_spas", LPSEquipment_SPAS);
	LPS_RegisterEquipmentName("rifle", LPSEquipment_M16);
	LPS_RegisterEquipmentName("rifle_ak47", LPSEquipment_AK47);
	LPS_RegisterEquipmentName("rifle_desert", LPSEquipment_SCAR);
	LPS_RegisterEquipmentName("rifle_sg552", LPSEquipment_SG552);
	LPS_RegisterEquipmentName("hunting_rifle", LPSEquipment_HuntingRifle);
	LPS_RegisterEquipmentName("sniper_military", LPSEquipment_MilitarySniper);
	LPS_RegisterEquipmentName("sniper_scout", LPSEquipment_Scout);
	LPS_RegisterEquipmentName("sniper_awp", LPSEquipment_AWP);
	LPS_RegisterEquipmentName("grenade_launcher", LPSEquipment_GrenadeLauncher);
	LPS_RegisterEquipmentName("rifle_m60", LPSEquipment_M60);
	LPS_RegisterEquipmentName("chainsaw", LPSEquipment_Chainsaw);
	LPS_RegisterEquipmentName("molotov", LPSEquipment_Molotov);
	LPS_RegisterEquipmentName("pipe_bomb", LPSEquipment_PipeBomb);
	LPS_RegisterEquipmentName("vomitjar", LPSEquipment_VomitJar);
	LPS_RegisterEquipmentName("minigun", LPSEquipment_Minigun);
	LPS_RegisterEquipmentName("mounted_machine_gun", LPSEquipment_MountedGun);

	LPS_RegisterMeleeName("baseball_bat", LPSEquipment_BaseballBat);
	LPS_RegisterMeleeName("cricket_bat", LPSEquipment_CricketBat);
	LPS_RegisterMeleeName("crowbar", LPSEquipment_Crowbar);
	LPS_RegisterMeleeName("electric_guitar", LPSEquipment_ElectricGuitar);
	LPS_RegisterMeleeName("fireaxe", LPSEquipment_FireAxe);
	LPS_RegisterMeleeName("frying_pan", LPSEquipment_FryingPan);
	LPS_RegisterMeleeName("golfclub", LPSEquipment_GolfClub);
	LPS_RegisterMeleeName("katana", LPSEquipment_Katana);
	LPS_RegisterMeleeName("knife", LPSEquipment_Knife);
	LPS_RegisterMeleeName("machete", LPSEquipment_Machete);
	LPS_RegisterMeleeName("pitchfork", LPSEquipment_Pitchfork);
	LPS_RegisterMeleeName("shovel", LPSEquipment_Shovel);
	LPS_RegisterMeleeName("tonfa", LPSEquipment_Tonfa);

	HookEvent("weapon_fire", LPS_Event_WeaponFireStats, EventHookMode_Post);

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientEquipmentStats(client);
	}
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_None;
		g_LPSLastDamageEquipment[entity] = LPSEquipment_None;
		g_LPSLastDamageAttackerUserId[entity] = 0;
	}
	LPS_ResetEquipmentFlushCapture();
}

void LPS_RegisterEquipmentName(const char[] name, LPSEquipment equipment)
{
	g_LPSEquipmentNames.SetValue(name, view_as<int>(equipment));

	char classname[64];
	FormatEx(classname, sizeof(classname), "weapon_%s", name);
	g_LPSEquipmentNames.SetValue(classname, view_as<int>(equipment));
}

void LPS_RegisterMeleeName(const char[] name, LPSEquipment equipment)
{
	g_LPSMeleeNames.SetValue(name, view_as<int>(equipment));
}

public void LPS_Event_WeaponFireStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_IsHumanSurvivorClient(client))
	{
		return;
	}

	char weaponName[64];
	event.GetString("weapon", weaponName, sizeof(weaponName));
	LPSEquipment equipment = LPS_GetEquipmentFromWeaponEvent(
		client,
		event.GetInt("weaponid"),
		weaponName,
		true
	);
	// “actions” is reserved for consumable throws. Gunshots, melee swings,
	// ammunition consumption and reloads are intentionally outside v0.5.1.
	// L4D2 does not reliably expose Vomit Jar throws through weapon_fire.
	// Vomit Jars are counted from their owned projectile instead, so keep the
	// two paths mutually exclusive and avoid duplicate actions.
	if (LPS_IsThrowableEquipment(equipment) && equipment != LPSEquipment_VomitJar)
	{
		LPS_AddEquipmentStat(client, equipment, LPSEquipmentStat_Actions, 1);
	}
}

LPSEquipment LPS_GetEquipmentFromWeaponEvent(
	int client,
	int weaponId,
	const char[] weaponName,
	bool allowOtherFirearm
)
{
	LPSEquipment equipment = LPS_GetEquipmentFromWeaponId(client, weaponId);
	if (equipment != LPSEquipment_None)
	{
		return equipment;
	}

	int value;
	if (weaponName[0] != '\0' && g_LPSEquipmentNames.GetValue(weaponName, value))
	{
		return view_as<LPSEquipment>(value);
	}

	if (StrEqual(weaponName, "melee", false) || StrEqual(weaponName, "weapon_melee", false))
	{
		return LPS_GetActiveEquipment(client, false);
	}

	return allowOtherFirearm && weaponName[0] != '\0'
		? LPSEquipment_OtherFirearm
		: LPSEquipment_None;
}

LPSEquipment LPS_GetEquipmentFromWeaponId(int client, int weaponId)
{
	switch (weaponId)
	{
		case 1: return LPS_IsClientUsingDualPistols(client) ? LPSEquipment_DualPistols : LPSEquipment_Pistol;
		case 2: return LPSEquipment_SMG;
		case 3: return LPSEquipment_PumpShotgun;
		case 4: return LPSEquipment_AutoShotgun;
		case 5: return LPSEquipment_M16;
		case 6: return LPSEquipment_HuntingRifle;
		case 7: return LPSEquipment_SilencedSMG;
		case 8: return LPSEquipment_ChromeShotgun;
		case 9: return LPSEquipment_SCAR;
		case 10: return LPSEquipment_MilitarySniper;
		case 11: return LPSEquipment_SPAS;
		case 13: return LPSEquipment_Molotov;
		case 14: return LPSEquipment_PipeBomb;
		case 19: return LPS_GetActiveEquipment(client, false);
		case 20: return LPSEquipment_Chainsaw;
		case 21: return LPSEquipment_GrenadeLauncher;
		case 24: return LPSEquipment_VomitJar;
		case 25: return LPSEquipment_AK47;
		case 31: return LPSEquipment_Magnum;
		case 32: return LPSEquipment_MP5;
		case 33: return LPSEquipment_SG552;
		case 34: return LPSEquipment_AWP;
		case 35: return LPSEquipment_Scout;
		case 36: return LPSEquipment_M60;
	}
	return LPSEquipment_None;
}

LPSEquipment LPS_GetActiveEquipment(int client, bool allowOtherFirearm)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
	{
		return LPSEquipment_None;
	}

	int weapon = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
	if (weapon <= MaxClients || !IsValidEntity(weapon))
	{
		return LPSEquipment_None;
	}

	char classname[64];
	GetEntityClassname(weapon, classname, sizeof(classname));
	if (StrEqual(classname, "weapon_melee", true))
	{
		char scriptName[64];
		if (HasEntProp(weapon, Prop_Data, "m_strMapSetScriptName"))
		{
			GetEntPropString(weapon, Prop_Data, "m_strMapSetScriptName", scriptName, sizeof(scriptName));
		}

		int meleeValue;
		return scriptName[0] != '\0' && g_LPSMeleeNames.GetValue(scriptName, meleeValue)
			? view_as<LPSEquipment>(meleeValue)
			: LPSEquipment_None;
	}

	if (StrEqual(classname, "weapon_pistol", true) && LPS_IsClientUsingDualPistols(client))
	{
		return LPSEquipment_DualPistols;
	}

	int value;
	if (g_LPSEquipmentNames.GetValue(classname, value))
	{
		return view_as<LPSEquipment>(value);
	}

	return allowOtherFirearm && StrContains(classname, "weapon_", false) == 0
		? LPSEquipment_OtherFirearm
		: LPSEquipment_None;
}

bool LPS_IsClientUsingDualPistols(int client)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
	{
		return false;
	}

	int weapon = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
	return weapon > MaxClients
		&& IsValidEntity(weapon)
		&& HasEntProp(weapon, Prop_Send, "m_hasDualWeapons")
		&& GetEntProp(weapon, Prop_Send, "m_hasDualWeapons") != 0;
}

bool LPS_IsFirearmEquipment(LPSEquipment equipment)
{
	return (equipment >= LPSEquipment_OtherFirearm && equipment <= LPSEquipment_M60)
		|| equipment == LPSEquipment_MountedGun
		|| equipment == LPSEquipment_Minigun;
}

bool LPS_IsMeleeEquipment(LPSEquipment equipment)
{
	return equipment >= LPSEquipment_BaseballBat && equipment <= LPSEquipment_Tonfa;
}

bool LPS_IsThrowableEquipment(LPSEquipment equipment)
{
	return equipment >= LPSEquipment_Molotov && equipment <= LPSEquipment_VomitJar;
}

void LPS_OnEquipmentEntityCreated(int entity, const char[] classname)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	g_LPSEntityEquipment[entity] = LPSEquipment_None;
	g_LPSLastDamageEquipment[entity] = LPSEquipment_None;
	g_LPSLastDamageAttackerUserId[entity] = 0;

	if (StrEqual(classname, "molotov_projectile", true) || StrEqual(classname, "inferno", true))
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_Molotov;
	}
	else if (StrEqual(classname, "pipe_bomb_projectile", true))
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_PipeBomb;
	}
	else if (StrEqual(classname, "vomitjar_projectile", true))
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_VomitJar;
		RequestFrame(LPS_Frame_RecordVomitJarThrow, EntIndexToEntRef(entity));
	}
	else if (StrEqual(classname, "grenade_launcher_projectile", true))
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_GrenadeLauncher;
	}
	else if (StrEqual(classname, "prop_minigun", true))
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_Minigun;
	}
	else if (StrEqual(classname, "prop_minigun_l4d1", true)
		|| StrEqual(classname, "prop_mounted_machine_gun", true))
	{
		g_LPSEntityEquipment[entity] = LPSEquipment_MountedGun;
	}
}

public void LPS_Frame_RecordVomitJarThrow(any entityReference)
{
	int entity = EntRefToEntIndex(entityReference);
	if (!LPS_CanRecordPvEEvents()
		|| entity < 1
		|| entity > LPS_MAX_ENTITY_INDEX
		|| !IsValidEntity(entity)
		|| g_LPSEntityEquipment[entity] != LPSEquipment_VomitJar)
	{
		return;
	}

	// Projectile ownership is populated after entity creation. Waiting one
	// frame avoids reading the entity before m_hThrower/m_hOwnerEntity is ready.
	int thrower = LPS_GetEntityPlayerOwner(entity);
	if (LPS_IsHumanSurvivorClient(thrower))
	{
		LPS_AddEquipmentStat(
			thrower,
			LPSEquipment_VomitJar,
			LPSEquipmentStat_Actions,
			1
		);
	}
}

void LPS_OnEquipmentEntityDestroyed(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	g_LPSEntityEquipment[entity] = LPSEquipment_None;
	g_LPSLastDamageEquipment[entity] = LPSEquipment_None;
	g_LPSLastDamageAttackerUserId[entity] = 0;
}

LPSEquipment LPS_ResolveDamageEquipment(int humanAttacker, int attacker, int inflictor)
{
	if (inflictor >= 1
		&& inflictor <= LPS_MAX_ENTITY_INDEX
		&& g_LPSEntityEquipment[inflictor] != LPSEquipment_None)
	{
		return g_LPSEntityEquipment[inflictor];
	}

	if (attacker >= 1
		&& attacker <= LPS_MAX_ENTITY_INDEX
		&& g_LPSEntityEquipment[attacker] != LPSEquipment_None)
	{
		return g_LPSEntityEquipment[attacker];
	}

	return LPS_GetActiveEquipment(humanAttacker, true);
}

void LPS_RecordEquipmentDamage(
	int victim,
	int humanAttacker,
	int attacker,
	int inflictor,
	LPSEquipmentStat stat,
	int amount
)
{
	LPSEquipment equipment = LPS_ResolveDamageEquipment(humanAttacker, attacker, inflictor);
	LPS_AddEquipmentStat(humanAttacker, equipment, stat, amount);

	if (victim >= 1 && victim <= LPS_MAX_ENTITY_INDEX && equipment != LPSEquipment_None)
	{
		g_LPSLastDamageEquipment[victim] = equipment;
		g_LPSLastDamageAttackerUserId[victim] = GetClientUserId(humanAttacker);
	}
}

LPSEquipment LPS_ResolveKillEquipment(
	int victim,
	int attacker,
	int weaponId,
	const char[] weaponName,
	bool minigun
)
{
	if (minigun)
	{
		return LPSEquipment_Minigun;
	}

	if (victim >= 1
		&& victim <= LPS_MAX_ENTITY_INDEX
		&& g_LPSLastDamageAttackerUserId[victim] == GetClientUserId(attacker)
		&& g_LPSLastDamageEquipment[victim] != LPSEquipment_None)
	{
		return g_LPSLastDamageEquipment[victim];
	}

	return LPS_GetEquipmentFromWeaponEvent(attacker, weaponId, weaponName, true);
}

void LPS_RecordEquipmentKill(
	int victim,
	int attacker,
	int weaponId,
	const char[] weaponName,
	bool minigun,
	LPSEquipmentStat killStat,
	bool headshot
)
{
	if (!LPS_IsHumanSurvivorClient(attacker))
	{
		return;
	}

	LPSEquipment equipment = LPS_ResolveKillEquipment(
		victim,
		attacker,
		weaponId,
		weaponName,
		minigun
	);
	LPS_AddEquipmentStat(attacker, equipment, killStat, 1);
	if (headshot && LPS_IsFirearmEquipment(equipment))
	{
		LPS_AddEquipmentStat(attacker, equipment, LPSEquipmentStat_HeadshotKills, 1);
	}

	if (victim >= 1 && victim <= LPS_MAX_ENTITY_INDEX)
	{
		g_LPSLastDamageEquipment[victim] = LPSEquipment_None;
		g_LPSLastDamageAttackerUserId[victim] = 0;
	}
}

void LPS_AddEquipmentStat(int client, LPSEquipment equipment, LPSEquipmentStat stat, int amount)
{
	if (amount <= 0
		|| client < 1
		|| client > MaxClients
		|| equipment <= LPSEquipment_None
		|| equipment >= LPSEquipment_Count
		|| !g_LPSClientPvEStatsActive[client]
		|| !g_LPSClientSegmentActive[client])
	{
		return;
	}

	switch (stat)
	{
		case LPSEquipmentStat_Actions: g_LPSClientEquipmentStats[client][equipment].actions += amount;
		case LPSEquipmentStat_CommonKills: g_LPSClientEquipmentStats[client][equipment].commonKills += amount;
		case LPSEquipmentStat_SpecialKills: g_LPSClientEquipmentStats[client][equipment].specialKills += amount;
		case LPSEquipmentStat_TankKills: g_LPSClientEquipmentStats[client][equipment].tankKills += amount;
		case LPSEquipmentStat_WitchKills: g_LPSClientEquipmentStats[client][equipment].witchKills += amount;
		case LPSEquipmentStat_HeadshotKills: g_LPSClientEquipmentStats[client][equipment].headshotKills += amount;
		case LPSEquipmentStat_DamageToSpecial: g_LPSClientEquipmentStats[client][equipment].damageToSpecial += amount;
		case LPSEquipmentStat_DamageToTank: g_LPSClientEquipmentStats[client][equipment].damageToTank += amount;
		case LPSEquipmentStat_DamageToWitch: g_LPSClientEquipmentStats[client][equipment].damageToWitch += amount;
	}

	g_LPSClientEquipmentStats[client][equipment].revision++;
	g_LPSClientEquipmentDirty[client][equipment] = true;
}

void LPS_OnClientEquipmentSegmentStarted(int client)
{
	LPS_ResetClientEquipmentStats(client);
}

void LPS_OnClientEquipmentSegmentEnding(int client)
{
	if (client < 1 || client > MaxClients || !g_LPSClientPvEStatsActive[client])
	{
		return;
	}

	LPSEquipmentStatsSnapshot snapshot;
	for (int equipment = 1; equipment < view_as<int>(LPSEquipment_Count); equipment++)
	{
		if (g_LPSClientEquipmentStats[client][equipment].revision <= 0)
		{
			continue;
		}

		LPS_FillEquipmentStatsSnapshot(client, view_as<LPSEquipment>(equipment), snapshot, true);
		LPS_QueueClosedEquipmentStats(snapshot);
	}
	LPS_ResetClientEquipmentStats(client);
}

void LPS_FillEquipmentStatsSnapshot(
	int client,
	LPSEquipment equipment,
	LPSEquipmentStatsSnapshot snapshot,
	bool closed
)
{
	strcopy(snapshot.segmentId, sizeof(snapshot.segmentId), g_LPSClientPvEStats[client].segmentId);
	snapshot.equipmentId = view_as<int>(equipment);
	snapshot.actions = g_LPSClientEquipmentStats[client][equipment].actions;
	snapshot.commonKills = g_LPSClientEquipmentStats[client][equipment].commonKills;
	snapshot.specialKills = g_LPSClientEquipmentStats[client][equipment].specialKills;
	snapshot.tankKills = g_LPSClientEquipmentStats[client][equipment].tankKills;
	snapshot.witchKills = g_LPSClientEquipmentStats[client][equipment].witchKills;
	snapshot.headshotKills = g_LPSClientEquipmentStats[client][equipment].headshotKills;
	snapshot.damageToSpecial = g_LPSClientEquipmentStats[client][equipment].damageToSpecial;
	snapshot.damageToTank = g_LPSClientEquipmentStats[client][equipment].damageToTank;
	snapshot.damageToWitch = g_LPSClientEquipmentStats[client][equipment].damageToWitch;
	snapshot.revision = g_LPSClientEquipmentStats[client][equipment].revision + (closed ? 1 : 0);
}

void LPS_QueueClosedEquipmentStats(LPSEquipmentStatsSnapshot snapshot)
{
	if (g_LPSClosedEquipmentStats.Length >= g_LPSClosedEquipmentQueueLimit.IntValue)
	{
		if (!g_LPSEquipmentQueueOverflowReported)
		{
			LogError(
				"Closed equipment stats queue reached its limit (%d); additional snapshots will be dropped until a flush succeeds.",
				g_LPSClosedEquipmentQueueLimit.IntValue
			);
			g_LPSEquipmentQueueOverflowReported = true;
		}
		return;
	}

	g_LPSClosedEquipmentStats.PushArray(snapshot, sizeof(snapshot));
}

bool LPS_AddEquipmentStatsFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_ResetEquipmentFlushCapture();
	g_LPSFlushClosedEquipmentCount = g_LPSClosedEquipmentStats.Length;

	LPSEquipmentStatsSnapshot snapshot;
	for (int index = 0; index < g_LPSFlushClosedEquipmentCount; index++)
	{
		g_LPSClosedEquipmentStats.GetArray(index, snapshot, sizeof(snapshot));
		if (!LPS_AddEquipmentStatsSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientPvEStatsActive[client])
		{
			continue;
		}

		for (int equipment = 1; equipment < view_as<int>(LPSEquipment_Count); equipment++)
		{
			if (!g_LPSClientEquipmentDirty[client][equipment])
			{
				continue;
			}

			LPS_FillEquipmentStatsSnapshot(client, view_as<LPSEquipment>(equipment), snapshot, false);
			if (!LPS_AddEquipmentStatsSnapshotQuery(transaction, snapshot, queryCount, error, errorLength))
			{
				return false;
			}

			g_LPSFlushCapturedEquipment[client][equipment] = true;
			g_LPSFlushCapturedEquipmentRevision[client][equipment] = snapshot.revision;
			strcopy(
				g_LPSFlushCapturedEquipmentSegmentId[client],
				sizeof(g_LPSFlushCapturedEquipmentSegmentId[]),
				snapshot.segmentId
			);
		}
	}
	return true;
}

bool LPS_AddEquipmentStatsSnapshotQuery(
	Transaction transaction,
	LPSEquipmentStatsSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[2048];
	if (!LPS_BuildEquipmentStatsUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "an equipment stats value exceeded its SQL escape buffer");
		return false;
	}

	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_BuildEquipmentStatsUpsertQuery(LPSEquipmentStatsSnapshot snapshot, char[] query, int maxLength)
{
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_pve_segment_equipment_stats (segment_id, equipment_id, stats_version, last_saved_at, actions, common_kills, special_kills, tank_kills, witch_kills, headshot_kills, damage_to_special, damage_to_tank, damage_to_witch, revision) VALUES ('%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d) ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), actions = VALUES(actions), common_kills = VALUES(common_kills), special_kills = VALUES(special_kills), tank_kills = VALUES(tank_kills), witch_kills = VALUES(witch_kills), headshot_kills = VALUES(headshot_kills), damage_to_special = VALUES(damage_to_special), damage_to_tank = VALUES(damage_to_tank), damage_to_witch = VALUES(damage_to_witch), revision = VALUES(revision)",
			segmentId,
			snapshot.equipmentId,
			LPS_STATS_VERSION,
			GetTime(),
			snapshot.actions,
			snapshot.commonKills,
			snapshot.specialKills,
			snapshot.tankKills,
			snapshot.witchKills,
			snapshot.headshotKills,
			snapshot.damageToSpecial,
			snapshot.damageToTank,
			snapshot.damageToWitch,
			snapshot.revision
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_pve_segment_equipment_stats (segment_id, equipment_id, stats_version, last_saved_at, actions, common_kills, special_kills, tank_kills, witch_kills, headshot_kills, damage_to_special, damage_to_tank, damage_to_witch, revision) VALUES ('%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d) ON CONFLICT(segment_id, equipment_id) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, actions = excluded.actions, common_kills = excluded.common_kills, special_kills = excluded.special_kills, tank_kills = excluded.tank_kills, witch_kills = excluded.witch_kills, headshot_kills = excluded.headshot_kills, damage_to_special = excluded.damage_to_special, damage_to_tank = excluded.damage_to_tank, damage_to_witch = excluded.damage_to_witch, revision = excluded.revision",
			segmentId,
			snapshot.equipmentId,
			LPS_STATS_VERSION,
			GetTime(),
			snapshot.actions,
			snapshot.commonKills,
			snapshot.specialKills,
			snapshot.tankKills,
			snapshot.witchKills,
			snapshot.headshotKills,
			snapshot.damageToSpecial,
			snapshot.damageToTank,
			snapshot.damageToWitch,
			snapshot.revision
		);
	}
	return true;
}

void LPS_OnEquipmentStatsFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedEquipmentCount && g_LPSClosedEquipmentStats.Length > 0; index++)
	{
		g_LPSClosedEquipmentStats.Erase(0);
	}

	if (g_LPSClosedEquipmentStats.Length < g_LPSClosedEquipmentQueueLimit.IntValue)
	{
		g_LPSEquipmentQueueOverflowReported = false;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!g_LPSClientPvEStatsActive[client]
			|| !StrEqual(
				g_LPSClientPvEStats[client].segmentId,
				g_LPSFlushCapturedEquipmentSegmentId[client],
				true
			))
		{
			continue;
		}

		for (int equipment = 1; equipment < view_as<int>(LPSEquipment_Count); equipment++)
		{
			if (!g_LPSFlushCapturedEquipment[client][equipment])
			{
				continue;
			}

			g_LPSClientEquipmentPersistedRevision[client][equipment] =
				g_LPSFlushCapturedEquipmentRevision[client][equipment];
			if (g_LPSClientEquipmentStats[client][equipment].revision
				<= g_LPSFlushCapturedEquipmentRevision[client][equipment])
			{
				g_LPSClientEquipmentDirty[client][equipment] = false;
			}
		}
	}
	LPS_ResetEquipmentFlushCapture();
}

void LPS_ResetEquipmentFlushCapture()
{
	g_LPSFlushClosedEquipmentCount = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSFlushCapturedEquipmentSegmentId[client][0] = '\0';
		for (int equipment = 1; equipment < view_as<int>(LPSEquipment_Count); equipment++)
		{
			g_LPSFlushCapturedEquipment[client][equipment] = false;
			g_LPSFlushCapturedEquipmentRevision[client][equipment] = 0;
		}
	}
}

void LPS_ResetClientEquipmentStats(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	for (int equipment = 1; equipment < view_as<int>(LPSEquipment_Count); equipment++)
	{
		g_LPSClientEquipmentStats[client][equipment].actions = 0;
		g_LPSClientEquipmentStats[client][equipment].commonKills = 0;
		g_LPSClientEquipmentStats[client][equipment].specialKills = 0;
		g_LPSClientEquipmentStats[client][equipment].tankKills = 0;
		g_LPSClientEquipmentStats[client][equipment].witchKills = 0;
		g_LPSClientEquipmentStats[client][equipment].headshotKills = 0;
		g_LPSClientEquipmentStats[client][equipment].damageToSpecial = 0;
		g_LPSClientEquipmentStats[client][equipment].damageToTank = 0;
		g_LPSClientEquipmentStats[client][equipment].damageToWitch = 0;
		g_LPSClientEquipmentStats[client][equipment].revision = 0;
		g_LPSClientEquipmentDirty[client][equipment] = false;
		g_LPSClientEquipmentPersistedRevision[client][equipment] = 0;
	}
}

int LPS_GetPendingClosedEquipmentStatsCount()
{
	return g_LPSClosedEquipmentStats == null ? 0 : g_LPSClosedEquipmentStats.Length;
}

void LPS_ShutdownEquipmentStats()
{
	delete g_LPSEquipmentNames;
	delete g_LPSMeleeNames;
	g_LPSEquipmentNames = null;
	g_LPSMeleeNames = null;
}
