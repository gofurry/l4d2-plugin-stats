#if defined _l4d2_player_stats_definitions_included
	#endinput
#endif
#define _l4d2_player_stats_definitions_included

#define LPS_VERSION "1.3.4"
#define LPS_SCHEMA_VERSION 6
#define LPS_MIGRATION_NAME "initial_schema"
#define LPS_STATS_VERSION 1

#define LPS_MAX_DATABASE_CONFIG 64
#define LPS_MAX_SERVER_KEY 65
#define LPS_MAX_SERVER_NAME 129
#define LPS_MAX_BOOT_ID 128
#define LPS_MAX_DRIVER_NAME 16
#define LPS_MAX_DATABASE_ERROR 256
#define LPS_MAX_QUERY_LENGTH 8192
#define LPS_MAX_MIGRATION_LINE 1024
#define LPS_MAX_STEAM_ID 32
#define LPS_MAX_PLAYER_NAME 128
#define LPS_MAX_IP_ADDRESS 46
#define LPS_MAX_SESSION_ID 128
#define LPS_MAX_SESSION_STATUS 16
#define LPS_MAX_DISCONNECT_REASON 32
#define LPS_MAX_GAME_MODE 32
#define LPS_MAX_RUN_ID 128
#define LPS_MAX_ROUND_ID 128
#define LPS_MAX_SEGMENT_ID 128
#define LPS_MAX_MAP_NAME 128
#define LPS_MAX_CAMPAIGN_KEY 128
#define LPS_MAX_LIFECYCLE_STATUS 16
#define LPS_MAX_SIDE_NAME 16
#define LPS_SESSION_TRANSFER_TIMEOUT 120
#define LPS_MAX_ENTITY_INDEX 2048
#define LPS_MAX_RULESET_NAME 65
#define LPS_MAX_DIFFICULTY 17
#define LPS_INCIDENT_VERSION 1
#define LPS_CONTEXT_VERSION 1
#define LPS_ANALYSIS_FLUSH_INCIDENT_LIMIT 256

#define LPS_TEAM_SPECTATOR 1
#define LPS_TEAM_SURVIVOR 2
#define LPS_TEAM_INFECTED 3

enum LPSDatabaseState
{
	LPSDatabaseState_Disabled = 0,
	LPSDatabaseState_ConfigError,
	LPSDatabaseState_Disconnected,
	LPSDatabaseState_Connecting,
	LPSDatabaseState_Migrating,
	LPSDatabaseState_Registering,
	LPSDatabaseState_Ready,
	LPSDatabaseState_RetryWait
};

enum LPSDatabaseDialect
{
	LPSDatabaseDialect_Unknown = 0,
	LPSDatabaseDialect_SQLite,
	LPSDatabaseDialect_MySQL,
	LPSDatabaseDialect_PostgreSQL
};

enum LPSModeFamily
{
	LPSModeFamily_Unsupported = 0,
	LPSModeFamily_PvE,
	LPSModeFamily_Versus
};

enum LPSContinuationKind
{
	LPSContinuation_None = 0,
	LPSContinuation_MapTransition,
	LPSContinuation_Retry,
	LPSContinuation_VersusHalf
};

enum LPSPvEStat
{
	LPSPvEStat_CommonKills = 0,
	LPSPvEStat_SpecialKills,
	LPSPvEStat_TankKills,
	LPSPvEStat_WitchKills,
	LPSPvEStat_DamageToSpecial,
	LPSPvEStat_DamageToTank,
	LPSPvEStat_DamageToWitch,
	LPSPvEStat_DamageTakenInfected,
	LPSPvEStat_FriendlyFireToHumans,
	LPSPvEStat_FriendlyFireToBots,
	LPSPvEStat_FriendlyFireTaken,
	LPSPvEStat_Incapacitations,
	LPSPvEStat_Deaths,
	LPSPvEStat_IncapRevives,
	LPSPvEStat_LedgeRescues,
	LPSPvEStat_DefibRevives,
	LPSPvEStat_RescuesReceived,
	LPSPvEStat_MedkitsUsedSelf,
	LPSPvEStat_MedkitsUsedOnOthers,
	LPSPvEStat_MedkitHealingSelf,
	LPSPvEStat_MedkitHealingOthers,
	LPSPvEStat_PillsUsed,
	LPSPvEStat_AdrenalineUsed,
	LPSPvEStat_TemporaryHealthReceived,
	LPSPvEStat_ChapterParticipations,
	LPSPvEStat_ChapterCompletionsAlive,
	LPSPvEStat_ChapterCompletionsDead,
	LPSPvEStat_CampaignCompletions,
	LPSPvEStat_SmokerKills,
	LPSPvEStat_BoomerKills,
	LPSPvEStat_HunterKills,
	LPSPvEStat_SpitterKills,
	LPSPvEStat_JockeyKills,
	LPSPvEStat_ChargerKills,
	LPSPvEStat_DamageToSmoker,
	LPSPvEStat_DamageToBoomer,
	LPSPvEStat_DamageToHunter,
	LPSPvEStat_DamageToSpitter,
	LPSPvEStat_DamageToJockey,
	LPSPvEStat_DamageToCharger,
	LPSPvEStat_SmokerControlsReceived,
	LPSPvEStat_HunterControlsReceived,
	LPSPvEStat_JockeyControlsReceived,
	LPSPvEStat_ChargerControlsReceived,
	LPSPvEStat_SmokerControlledSeconds,
	LPSPvEStat_HunterControlledSeconds,
	LPSPvEStat_JockeyControlledSeconds,
	LPSPvEStat_ChargerControlledSeconds,
	LPSPvEStat_SmokerSaves,
	LPSPvEStat_HunterSaves,
	LPSPvEStat_JockeySaves,
	LPSPvEStat_ChargerSaves,
	LPSPvEStat_MeleeTongueSelfCuts,
	LPSPvEStat_TankRocksDestroyed,
	LPSPvEStat_WitchOneShots,
	LPSPvEStat_WitchSoloKills,
	LPSPvEStat_TankEncounters,
	LPSPvEStat_TankKillParticipations,
	LPSPvEStat_WitchEncounters,
	LPSPvEStat_WitchKillParticipations,
	LPSPvEStat_IncendiaryPacksDeployed,
	LPSPvEStat_ExplosivePacksDeployed,
	LPSPvEStat_ObjectiveInteractions,
	LPSPvEStat_AmmoPileUses,
	LPSPvEStat_IncapacitatedSeconds,
	LPSPvEStat_LedgeHangingSeconds,
	LPSPvEStat_BlackWhiteTeammatesRestored,
	LPSPvEStat_CarAlarmsTriggered,
	LPSPvEStat_SpecialAssists,
	LPSPvEStat_SmokerAssists,
	LPSPvEStat_BoomerAssists,
	LPSPvEStat_HunterAssists,
	LPSPvEStat_SpitterAssists,
	LPSPvEStat_JockeyAssists,
	LPSPvEStat_ChargerAssists,
	LPSPvEStat_FallDeaths
};

// Stable IDs persisted by lps_pve_segment_equipment_stats. Never reuse an ID
// for another item after the public schema is released.
enum LPSEquipment
{
	LPSEquipment_None = 0,
	LPSEquipment_OtherFirearm,
	LPSEquipment_Pistol,
	LPSEquipment_DualPistols,
	LPSEquipment_Magnum,
	LPSEquipment_SMG,
	LPSEquipment_SilencedSMG,
	LPSEquipment_MP5,
	LPSEquipment_PumpShotgun,
	LPSEquipment_ChromeShotgun,
	LPSEquipment_AutoShotgun,
	LPSEquipment_SPAS,
	LPSEquipment_M16,
	LPSEquipment_AK47,
	LPSEquipment_SCAR,
	LPSEquipment_SG552,
	LPSEquipment_HuntingRifle,
	LPSEquipment_MilitarySniper,
	LPSEquipment_Scout,
	LPSEquipment_AWP,
	LPSEquipment_GrenadeLauncher,
	LPSEquipment_M60,
	LPSEquipment_Chainsaw,
	LPSEquipment_MountedGun,
	LPSEquipment_Minigun,
	LPSEquipment_BaseballBat,
	LPSEquipment_CricketBat,
	LPSEquipment_Crowbar,
	LPSEquipment_ElectricGuitar,
	LPSEquipment_FireAxe,
	LPSEquipment_FryingPan,
	LPSEquipment_GolfClub,
	LPSEquipment_Katana,
	LPSEquipment_Knife,
	LPSEquipment_Machete,
	LPSEquipment_Pitchfork,
	LPSEquipment_Shovel,
	LPSEquipment_Tonfa,
	LPSEquipment_Molotov,
	LPSEquipment_PipeBomb,
	LPSEquipment_VomitJar,
	LPSEquipment_Count
};

enum LPSEquipmentStat
{
	LPSEquipmentStat_Actions = 0,
	LPSEquipmentStat_CommonKills,
	LPSEquipmentStat_SpecialKills,
	LPSEquipmentStat_TankKills,
	LPSEquipmentStat_WitchKills,
	LPSEquipmentStat_HeadshotKills,
	LPSEquipmentStat_DamageToSpecial,
	LPSEquipmentStat_DamageToTank,
	LPSEquipmentStat_DamageToWitch
};

enum LPSControlType
{
	LPSControlType_None = 0,
	LPSControlType_Smoker,
	LPSControlType_Hunter,
	LPSControlType_Jockey,
	LPSControlType_Charger
};

// Stable Incident Contract v1 IDs. Never reuse a persisted value.
enum LPSIncidentType
{
	LPSIncident_None = 0,
	LPSIncident_SurvivorControl = 1,
	LPSIncident_SurvivorIncap = 2,
	LPSIncident_SurvivorDeath = 3,
	LPSIncident_SurvivorRevive = 4,
	LPSIncident_LedgeRescue = 5,
	LPSIncident_DefibRevive = 6,
	LPSIncident_CarAlarm = 7,
	LPSIncident_TankSpawn = 8,
	LPSIncident_TankDeath = 9,
	LPSIncident_WitchSpawn = 10,
	LPSIncident_WitchDeath = 11,
	LPSIncident_WitchStartle = 12,
	LPSIncident_MedkitHeal = 13,
	LPSIncident_ObjectiveComplete = 14
};

enum LPSParticipantKind
{
	LPSParticipant_None = 0,
	LPSParticipant_HumanSurvivor = 1,
	LPSParticipant_BotSurvivor = 2,
	LPSParticipant_HumanInfected = 3,
	LPSParticipant_BotInfected = 4,
	LPSParticipant_CommonInfected = 5,
	LPSParticipant_Witch = 6,
	LPSParticipant_World = 7,
	LPSParticipant_Other = 8,
	LPSParticipant_Unknown = 9
};

enum LPSIncidentEndReason
{
	LPSIncidentEnd_Unknown = 0,
	LPSIncidentEnd_Released = 1,
	LPSIncidentEnd_Rescued = 2,
	LPSIncidentEnd_SelfFreed = 3,
	LPSIncidentEnd_ControllerKilled = 4,
	LPSIncidentEnd_TargetDied = 5,
	LPSIncidentEnd_LifecycleEnd = 6
};

enum LPSVersusSurvivorStat
{
	LPSVersusSurvivorStat_CommonKills = 0,
	LPSVersusSurvivorStat_HumanSpecialKills,
	LPSVersusSurvivorStat_BotSpecialKills,
	LPSVersusSurvivorStat_HumanTankKills,
	LPSVersusSurvivorStat_BotTankKills,
	LPSVersusSurvivorStat_DamageToHumanSpecial,
	LPSVersusSurvivorStat_DamageToBotSpecial,
	LPSVersusSurvivorStat_DamageToHumanTank,
	LPSVersusSurvivorStat_DamageToBotTank,
	LPSVersusSurvivorStat_DamageTakenInfected,
	LPSVersusSurvivorStat_FriendlyFireToHumans,
	LPSVersusSurvivorStat_FriendlyFireToBots,
	LPSVersusSurvivorStat_FriendlyFireTaken,
	LPSVersusSurvivorStat_Incapacitations,
	LPSVersusSurvivorStat_Deaths,
	LPSVersusSurvivorStat_IncapRevives,
	LPSVersusSurvivorStat_LedgeRescues,
	LPSVersusSurvivorStat_DefibRevives,
	LPSVersusSurvivorStat_RescuesReceived,
	LPSVersusSurvivorStat_MedkitsUsedSelf,
	LPSVersusSurvivorStat_MedkitsUsedOnOthers,
	LPSVersusSurvivorStat_MedkitHealingSelf,
	LPSVersusSurvivorStat_MedkitHealingOthers,
	LPSVersusSurvivorStat_PillsUsed,
	LPSVersusSurvivorStat_AdrenalineUsed,
	LPSVersusSurvivorStat_TemporaryHealthReceived,
	LPSVersusSurvivorStat_WitchKills,
	LPSVersusSurvivorStat_DamageToWitch,
	LPSVersusSurvivorStat_MolotovsThrown,
	LPSVersusSurvivorStat_PipeBombsThrown,
	LPSVersusSurvivorStat_VomitJarsThrown,
	LPSVersusSurvivorStat_IncendiaryPacksDeployed,
	LPSVersusSurvivorStat_ExplosivePacksDeployed,
	LPSVersusSurvivorStat_MeleeTongueSelfCuts,
	LPSVersusSurvivorStat_TankRocksDestroyed,
	LPSVersusSurvivorStat_WitchOneShots,
	LPSVersusSurvivorStat_WitchSoloKills,
	LPSVersusSurvivorStat_ObjectiveInteractions,
	LPSVersusSurvivorStat_CarAlarmsTriggered,
	LPSVersusSurvivorStat_HumanSpecialAssists,
	LPSVersusSurvivorStat_BotSpecialAssists,
	LPSVersusSurvivorStat_HumanTankAssists,
	LPSVersusSurvivorStat_BotTankAssists,
	LPSVersusSurvivorStat_WitchEncounters,
	LPSVersusSurvivorStat_WitchKillParticipations,
	LPSVersusSurvivorStat_BlackWhiteTeammatesRestored,
	LPSVersusSurvivorStat_FallDeaths
};

// Fixed per-class combat metrics for a human survivor attacking the seven
// playable infected classes. The controller identity is evaluated for each
// damage or death event, so Tank takeovers remain explainable.
enum LPSVersusSurvivorClassStat
{
	LPSVersusSurvivorClassStat_HumanControllerKills = 0,
	LPSVersusSurvivorClassStat_BotControllerKills,
	LPSVersusSurvivorClassStat_DamageToHumanControllers,
	LPSVersusSurvivorClassStat_DamageToBotControllers,
	LPSVersusSurvivorClassStat_HumanControllerAssists,
	LPSVersusSurvivorClassStat_BotControllerAssists,
	LPSVersusSurvivorClassStat_Count
};

enum LPSVersusInfectedStat
{
	LPSVersusInfectedStat_SpawnCount = 0,
	LPSVersusInfectedStat_DamageToHumanSurvivors,
	LPSVersusInfectedStat_DamageToBotSurvivors,
	LPSVersusInfectedStat_HumanSurvivorIncaps,
	LPSVersusInfectedStat_BotSurvivorIncaps,
	LPSVersusInfectedStat_HumanSurvivorKills,
	LPSVersusInfectedStat_BotSurvivorKills,
	LPSVersusInfectedStat_Count
};

// Fixed per-class ability metrics persisted alongside the seven-class
// breakdown. Only the classes whose abilities match a metric populate it:
// Smoker/Hunter/Jockey/Charger use controls, Boomer uses ability hits, and
// Spitter uses ability damage.
enum LPSVersusInfectedAbilityStat
{
	LPSVersusInfectedAbilityStat_HumanSurvivorControls = 0,
	LPSVersusInfectedAbilityStat_BotSurvivorControls,
	LPSVersusInfectedAbilityStat_HumanSurvivorControlSeconds,
	LPSVersusInfectedAbilityStat_BotSurvivorControlSeconds,
	LPSVersusInfectedAbilityStat_HumanSurvivorAbilityHits,
	LPSVersusInfectedAbilityStat_BotSurvivorAbilityHits,
	LPSVersusInfectedAbilityStat_HumanSurvivorAbilityDamage,
	LPSVersusInfectedAbilityStat_BotSurvivorAbilityDamage,
	LPSVersusInfectedAbilityStat_Count
};

// Stable bounded indexes for the seven playable infected classes. Database
// rows persist the engine zombie-class values (1-6 and 8), never these indexes.
enum LPSVersusInfectedClass
{
	LPSVersusInfectedClass_Smoker = 0,
	LPSVersusInfectedClass_Boomer,
	LPSVersusInfectedClass_Hunter,
	LPSVersusInfectedClass_Spitter,
	LPSVersusInfectedClass_Jockey,
	LPSVersusInfectedClass_Charger,
	LPSVersusInfectedClass_Tank,
	LPSVersusInfectedClass_Count
};
