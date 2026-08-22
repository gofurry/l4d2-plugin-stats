#if defined _l4d2_player_stats_pve_stats_state_included
	#endinput
#endif
#define _l4d2_player_stats_pve_stats_state_included

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
	int carAlarmsTriggered;
	int specialAssists;
	int smokerAssists;
	int boomerAssists;
	int hunterAssists;
	int spitterAssists;
	int jockeyAssists;
	int chargerAssists;
	int fallDeaths;
	int teammateProtections;
	int ledgeGrabs;
	int tankRockHitsReceived;
	int hunterSkeets;
	int chargerLevels;
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
	g_LPSClientPvEStats[client].carAlarmsTriggered = 0;
	g_LPSClientPvEStats[client].specialAssists = 0;
	g_LPSClientPvEStats[client].smokerAssists = 0;
	g_LPSClientPvEStats[client].boomerAssists = 0;
	g_LPSClientPvEStats[client].hunterAssists = 0;
	g_LPSClientPvEStats[client].spitterAssists = 0;
	g_LPSClientPvEStats[client].jockeyAssists = 0;
	g_LPSClientPvEStats[client].chargerAssists = 0;
	g_LPSClientPvEStats[client].fallDeaths = 0;
	g_LPSClientPvEStats[client].teammateProtections = 0;
	g_LPSClientPvEStats[client].ledgeGrabs = 0;
	g_LPSClientPvEStats[client].tankRockHitsReceived = 0;
	g_LPSClientPvEStats[client].hunterSkeets = 0;
	g_LPSClientPvEStats[client].chargerLevels = 0;
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
