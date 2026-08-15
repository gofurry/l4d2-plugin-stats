#if defined _l4d2_player_stats_pve_support_included
	#endinput
#endif
#define _l4d2_player_stats_pve_support_included

public void LPS_Event_ReviveSuccessStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_RecordRelationshipRevive(rescuer, subject, event.GetBool("ledge_hang"));
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
	LPS_RecordRelationshipDefib(rescuer, subject);
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
	LPS_RecordRelationshipMedkit(healer, subject, restored);
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
	LPS_ClearAssistContributor(client);
	LPS_ResetAssistTarget(client);
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
	LPS_ClearAssistContributor(client);
	LPS_ResetAssistTarget(client);
	LPS_ClearBossContributionsForClient(client);
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
		case LPSPvEStat_CarAlarmsTriggered: g_LPSClientPvEStats[client].carAlarmsTriggered += amount;
		case LPSPvEStat_SpecialAssists: g_LPSClientPvEStats[client].specialAssists += amount;
		case LPSPvEStat_SmokerAssists: g_LPSClientPvEStats[client].smokerAssists += amount;
		case LPSPvEStat_BoomerAssists: g_LPSClientPvEStats[client].boomerAssists += amount;
		case LPSPvEStat_HunterAssists: g_LPSClientPvEStats[client].hunterAssists += amount;
		case LPSPvEStat_SpitterAssists: g_LPSClientPvEStats[client].spitterAssists += amount;
		case LPSPvEStat_JockeyAssists: g_LPSClientPvEStats[client].jockeyAssists += amount;
		case LPSPvEStat_ChargerAssists: g_LPSClientPvEStats[client].chargerAssists += amount;
		case LPSPvEStat_FallDeaths: g_LPSClientPvEStats[client].fallDeaths += amount;
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
	snapshot.carAlarmsTriggered = g_LPSClientPvEStats[client].carAlarmsTriggered;
	snapshot.specialAssists = g_LPSClientPvEStats[client].specialAssists;
	snapshot.smokerAssists = g_LPSClientPvEStats[client].smokerAssists;
	snapshot.boomerAssists = g_LPSClientPvEStats[client].boomerAssists;
	snapshot.hunterAssists = g_LPSClientPvEStats[client].hunterAssists;
	snapshot.spitterAssists = g_LPSClientPvEStats[client].spitterAssists;
	snapshot.jockeyAssists = g_LPSClientPvEStats[client].jockeyAssists;
	snapshot.chargerAssists = g_LPSClientPvEStats[client].chargerAssists;
	snapshot.fallDeaths = g_LPSClientPvEStats[client].fallDeaths;
	snapshot.revision = g_LPSClientPvEStats[client].revision + (closed ? 1 : 0);
}
