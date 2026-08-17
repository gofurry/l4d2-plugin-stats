package store

import (
	"context"
	"errors"
	"time"
)

var ErrServerNotFound = errors.New("game server not found")

const (
	DashboardSchemaVersion int64 = 17
	StatsSchemaVersion     int64 = 6
)

type PlayerProfileSection string

const (
	PlayerProfileOverview              PlayerProfileSection = "overview"
	PlayerProfileAchievements          PlayerProfileSection = "achievements"
	PlayerProfileAnalysis              PlayerProfileSection = "analysis"
	PlayerProfilePVE                   PlayerProfileSection = "pve"
	PlayerProfilePVEDetails            PlayerProfileSection = "pve-details"
	PlayerProfileVersusSurvivor        PlayerProfileSection = "versus-survivor"
	PlayerProfileVersusSurvivorDetails PlayerProfileSection = "versus-survivor-details"
	PlayerProfileVersusInfected        PlayerProfileSection = "versus-infected"
	PlayerProfileVersusInfectedDetails PlayerProfileSection = "versus-infected-details"
	PlayerProfileRelationships         PlayerProfileSection = "relationships"
	PlayerProfileHistory               PlayerProfileSection = "history"
)

var PlayerProfileSections = []PlayerProfileSection{
	PlayerProfileOverview,
	PlayerProfileAchievements,
	PlayerProfileAnalysis,
	PlayerProfilePVE,
	PlayerProfilePVEDetails,
	PlayerProfileVersusSurvivor,
	PlayerProfileVersusSurvivorDetails,
	PlayerProfileVersusInfected,
	PlayerProfileVersusInfectedDetails,
	PlayerProfileRelationships,
	PlayerProfileHistory,
}

var DefaultPlayerProfileSections = []PlayerProfileSection{
	PlayerProfileOverview,
	PlayerProfileAnalysis,
	PlayerProfileRelationships,
}

type PlayerProfileVisibility struct {
	VisibleSections []PlayerProfileSection `json:"visible_sections"`
	UpdatedAt       int64                  `json:"updated_at,omitempty"`
}

func (v PlayerProfileVisibility) Visible(section PlayerProfileSection) bool {
	for _, candidate := range v.VisibleSections {
		if candidate == section {
			return true
		}
	}
	return false
}

type FooterLink struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Site struct {
	Language           string       `json:"language"`
	BrowserTitle       string       `json:"browser_title"`
	Theme              string       `json:"theme"`
	FooterEnabled      bool         `json:"footer_enabled"`
	BackgroundImageURL string       `json:"background_image_url"`
	Links              []FooterLink `json:"footer_links"`
	SteamOpenIDEnabled bool         `json:"steam_openid_enabled"`
	A2SRefreshSeconds  int64        `json:"a2s_refresh_seconds"`
	Documents          []string     `json:"site_documents"`
	Configured         bool         `json:"configured"`
}

type SiteSettings struct {
	Language            string       `json:"language"`
	BrowserTitle        string       `json:"browser_title"`
	Theme               string       `json:"theme"`
	FooterEnabled       bool         `json:"footer_enabled"`
	BackgroundImageURL  string       `json:"background_image_url"`
	PublicOrigin        string       `json:"public_origin"`
	SteamOpenIDEnabled  bool         `json:"steam_openid_enabled"`
	SteamOpenIDProxyURL string       `json:"steam_openid_proxy_url"`
	A2SRefreshSeconds   int64        `json:"a2s_refresh_seconds"`
	A2SJitterSeconds    int64        `json:"a2s_jitter_seconds"`
	A2SRetryCount       int64        `json:"a2s_retry_count"`
	SEOEnabled          bool         `json:"seo_enabled"`
	SEODescription      string       `json:"seo_description"`
	SEOImageURL         string       `json:"seo_image_url"`
	Links               []FooterLink `json:"footer_links"`
}

type SiteDocument struct {
	Key             string `json:"key"`
	Enabled         bool   `json:"enabled"`
	ContentMarkdown string `json:"content_markdown"`
	UpdatedAt       int64  `json:"updated_at"`
}

type GameServer struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Address     string `json:"address"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int64  `json:"sort_order"`
}

type AdminAccount struct {
	Username          string `json:"username"`
	PasswordHash      string `json:"-"`
	JWTSecret         string `json:"-"`
	TokenVersion      int64  `json:"-"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
	PasswordChangedAt int64  `json:"password_changed_at"`
}

type Announcement struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type AnnouncementPage struct {
	Items []Announcement `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

type AnnouncementFilter struct {
	Title  string
	Year   int
	Limit  int32
	Offset int32
}

type PlayerSummary struct {
	SteamID          string `json:"steam_id"`
	LastName         string `json:"last_name"`
	FirstSeenAt      int64  `json:"first_seen_at"`
	LastSeenAt       int64  `json:"last_seen_at"`
	SessionCount     int64  `json:"session_count"`
	ConnectedSeconds int64  `json:"connected_seconds"`
	ActiveSeconds    int64  `json:"active_play_seconds"`
}

type PlayerActivityPoint struct {
	Day              int64 `json:"day"`
	SessionCount     int64 `json:"session_count"`
	ConnectedSeconds int64 `json:"connected_seconds"`
	ActiveSeconds    int64 `json:"active_play_seconds"`
}

type PlayerServerActivity struct {
	ServerKey     string `json:"server_key"`
	SessionCount  int64  `json:"session_count"`
	ActiveSeconds int64  `json:"active_play_seconds"`
}

type PlayerActivity struct {
	Timeline []PlayerActivityPoint  `json:"timeline"`
	Servers  []PlayerServerActivity `json:"servers"`
}

type PlayerFilter struct {
	Cutoff    int64
	ServerKey string
	GameMode  string
}

type PVEInfectedClass struct {
	ClassID           int    `json:"class_id"`
	Kills             int64  `json:"kills"`
	Assists           *int64 `json:"assists"`
	Damage            int64  `json:"damage"`
	ControlsReceived  int64  `json:"controls_received"`
	ControlledSeconds int64  `json:"controlled_seconds"`
	Saves             int64  `json:"saves"`
}

type PVEEquipment struct {
	EquipmentID     int64 `json:"equipment_id"`
	Actions         int64 `json:"actions"`
	CommonKills     int64 `json:"common_kills"`
	SpecialKills    int64 `json:"special_kills"`
	TankKills       int64 `json:"tank_kills"`
	WitchKills      int64 `json:"witch_kills"`
	HeadshotKills   int64 `json:"headshot_kills"`
	DamageToSpecial int64 `json:"damage_to_special"`
	DamageToTank    int64 `json:"damage_to_tank"`
	DamageToWitch   int64 `json:"damage_to_witch"`
}

type PlayerPVE struct {
	CommonKills           int64              `json:"common_kills"`
	SpecialKills          int64              `json:"special_kills"`
	TankKills             int64              `json:"tank_kills"`
	WitchKills            int64              `json:"witch_kills"`
	DamageToSpecial       int64              `json:"damage_to_special"`
	DamageToTank          int64              `json:"damage_to_tank"`
	DamageToWitch         int64              `json:"damage_to_witch"`
	DamageTaken           int64              `json:"damage_taken_infected"`
	FriendlyFire          int64              `json:"friendly_fire"`
	Incapacitations       int64              `json:"incapacitations"`
	Deaths                int64              `json:"deaths"`
	Revives               int64              `json:"revives"`
	RescuesReceived       int64              `json:"rescues_received"`
	MedkitsUsed           int64              `json:"medkits_used"`
	Healing               int64              `json:"healing"`
	ChapterParticipations int64              `json:"chapter_participations"`
	ChapterCompletions    int64              `json:"chapter_completions"`
	CampaignCompletions   int64              `json:"campaign_completions"`
	FriendlyFireTaken     int64              `json:"friendly_fire_taken"`
	IncapRevives          int64              `json:"incap_revives"`
	LedgeRescues          int64              `json:"ledge_rescues"`
	DefibRevives          int64              `json:"defib_revives"`
	MedkitsUsedSelf       int64              `json:"medkits_used_self"`
	MedkitsUsedOnOthers   int64              `json:"medkits_used_on_others"`
	MedkitHealingSelf     int64              `json:"medkit_healing_self"`
	MedkitHealingOthers   int64              `json:"medkit_healing_others"`
	PillsUsed             int64              `json:"pills_used"`
	AdrenalineUsed        int64              `json:"adrenaline_used"`
	TemporaryHealth       int64              `json:"temporary_health_received"`
	ChapterCompletedAlive int64              `json:"chapter_completions_alive"`
	ChapterCompletedDead  int64              `json:"chapter_completions_dead"`
	TongueSelfCuts        int64              `json:"melee_tongue_self_cuts"`
	TankRocksDestroyed    int64              `json:"tank_rocks_destroyed"`
	WitchOneShots         int64              `json:"witch_oneshots"`
	WitchSoloKills        int64              `json:"witch_solo_kills"`
	TankEncounters        int64              `json:"tank_encounters"`
	TankParticipations    int64              `json:"tank_kill_participations"`
	WitchEncounters       int64              `json:"witch_encounters"`
	WitchParticipations   int64              `json:"witch_kill_participations"`
	IncendiaryPacks       int64              `json:"incendiary_packs_deployed"`
	ExplosivePacks        int64              `json:"explosive_packs_deployed"`
	ObjectiveInteractions int64              `json:"objective_interactions"`
	AmmoPileUses          int64              `json:"ammo_pile_uses"`
	IncapacitatedSeconds  int64              `json:"incapacitated_seconds"`
	LedgeHangingSeconds   int64              `json:"ledge_hanging_seconds"`
	BlackWhiteRestored    int64              `json:"black_white_teammates_restored"`
	CarAlarmsTriggered    int64              `json:"car_alarms_triggered"`
	SpecialAssists        *int64             `json:"special_assists"`
	TankAssists           int64              `json:"tank_assists"`
	WitchAssists          int64              `json:"witch_assists"`
	AssistCoverage        CollectionCoverage `json:"assist_coverage"`
	Classes               []PVEInfectedClass `json:"infected_classes"`
	Equipment             []PVEEquipment     `json:"equipment"`
}

type VersusSurvivorClass struct {
	ClassID                  int64  `json:"class_id"`
	HumanControllerKills     int64  `json:"human_controller_kills"`
	BotControllerKills       int64  `json:"bot_controller_kills"`
	DamageToHumanControllers int64  `json:"damage_to_human_controllers"`
	DamageToBotControllers   int64  `json:"damage_to_bot_controllers"`
	HumanControllerAssists   *int64 `json:"human_controller_assists"`
	BotControllerAssists     *int64 `json:"bot_controller_assists"`
}

type VersusInfectedClass struct {
	ClassID                     int64 `json:"class_id"`
	Spawns                      int64 `json:"spawns"`
	DamageToHumanSurvivors      int64 `json:"damage_to_human_survivors"`
	DamageToBotSurvivors        int64 `json:"damage_to_bot_survivors"`
	HumanSurvivorIncaps         int64 `json:"human_survivor_incaps"`
	BotSurvivorIncaps           int64 `json:"bot_survivor_incaps"`
	HumanSurvivorKills          int64 `json:"human_survivor_kills"`
	BotSurvivorKills            int64 `json:"bot_survivor_kills"`
	HumanSurvivorControls       int64 `json:"human_survivor_controls"`
	BotSurvivorControls         int64 `json:"bot_survivor_controls"`
	HumanSurvivorControlSeconds int64 `json:"human_survivor_control_seconds"`
	BotSurvivorControlSeconds   int64 `json:"bot_survivor_control_seconds"`
	HumanSurvivorAbilityHits    int64 `json:"human_survivor_ability_hits"`
	BotSurvivorAbilityHits      int64 `json:"bot_survivor_ability_hits"`
	HumanSurvivorAbilityDamage  int64 `json:"human_survivor_ability_damage"`
	BotSurvivorAbilityDamage    int64 `json:"bot_survivor_ability_damage"`
}

type PlayerVersus struct {
	SurvivorCommonKills           int64                 `json:"survivor_common_kills"`
	HumanSpecialKills             int64                 `json:"human_special_kills"`
	BotSpecialKills               int64                 `json:"bot_special_kills"`
	HumanTankKills                int64                 `json:"human_tank_kills"`
	BotTankKills                  int64                 `json:"bot_tank_kills"`
	SurvivorDamage                int64                 `json:"survivor_damage"`
	SurvivorDeaths                int64                 `json:"survivor_deaths"`
	SurvivorRevives               int64                 `json:"survivor_revives"`
	InfectedSpawns                int64                 `json:"infected_spawns"`
	DamageToHumanSurvivors        int64                 `json:"damage_to_human_survivors"`
	HumanSurvivorIncaps           int64                 `json:"human_survivor_incaps"`
	HumanSurvivorKills            int64                 `json:"human_survivor_kills"`
	HumanSurvivorControls         int64                 `json:"human_survivor_controls"`
	HumanSurvivorControlSeconds   int64                 `json:"human_survivor_control_seconds"`
	SurvivorDamageTaken           int64                 `json:"survivor_damage_taken"`
	SurvivorFriendlyFire          int64                 `json:"survivor_friendly_fire"`
	SurvivorFriendlyFireTaken     int64                 `json:"survivor_friendly_fire_taken"`
	SurvivorIncapacitations       int64                 `json:"survivor_incapacitations"`
	SurvivorIncapRevives          int64                 `json:"survivor_incap_revives"`
	SurvivorLedgeRescues          int64                 `json:"survivor_ledge_rescues"`
	SurvivorDefibRevives          int64                 `json:"survivor_defib_revives"`
	SurvivorRescuesReceived       int64                 `json:"survivor_rescues_received"`
	SurvivorMedkitsSelf           int64                 `json:"survivor_medkits_self"`
	SurvivorMedkitsOthers         int64                 `json:"survivor_medkits_others"`
	SurvivorHealingSelf           int64                 `json:"survivor_healing_self"`
	SurvivorHealingOthers         int64                 `json:"survivor_healing_others"`
	SurvivorPills                 int64                 `json:"survivor_pills"`
	SurvivorAdrenaline            int64                 `json:"survivor_adrenaline"`
	SurvivorTemporaryHealth       int64                 `json:"survivor_temporary_health"`
	SurvivorWitchKills            int64                 `json:"survivor_witch_kills"`
	SurvivorWitchDamage           int64                 `json:"survivor_witch_damage"`
	MolotovsThrown                int64                 `json:"molotovs_thrown"`
	PipeBombsThrown               int64                 `json:"pipe_bombs_thrown"`
	VomitJarsThrown               int64                 `json:"vomit_jars_thrown"`
	SurvivorIncendiaryPacks       int64                 `json:"survivor_incendiary_packs"`
	SurvivorExplosivePacks        int64                 `json:"survivor_explosive_packs"`
	SurvivorTongueSelfCuts        int64                 `json:"survivor_tongue_self_cuts"`
	SurvivorTankRocksDestroyed    int64                 `json:"survivor_tank_rocks_destroyed"`
	SurvivorWitchOneShots         int64                 `json:"survivor_witch_oneshots"`
	SurvivorWitchSoloKills        int64                 `json:"survivor_witch_solo_kills"`
	SurvivorObjectiveInteractions int64                 `json:"survivor_objective_interactions"`
	SurvivorCarAlarmsTriggered    int64                 `json:"survivor_car_alarms_triggered"`
	HumanSpecialAssists           *int64                `json:"human_special_assists"`
	BotSpecialAssists             *int64                `json:"bot_special_assists"`
	HumanTankAssists              *int64                `json:"human_tank_assists"`
	BotTankAssists                *int64                `json:"bot_tank_assists"`
	SurvivorWitchEncounters       *int64                `json:"survivor_witch_encounters"`
	SurvivorWitchParticipations   *int64                `json:"survivor_witch_kill_participations"`
	SurvivorWitchAssists          *int64                `json:"survivor_witch_assists"`
	SurvivorBlackWhiteRestored    *int64                `json:"survivor_black_white_teammates_restored"`
	AssistCoverage                CollectionCoverage    `json:"assist_coverage"`
	DamageToBotSurvivors          int64                 `json:"damage_to_bot_survivors"`
	BotSurvivorIncaps             int64                 `json:"bot_survivor_incaps"`
	BotSurvivorKills              int64                 `json:"bot_survivor_kills"`
	SurvivorClasses               []VersusSurvivorClass `json:"survivor_classes"`
	InfectedClasses               []VersusInfectedClass `json:"infected_classes"`
}

type RankingEntry struct {
	Rank          int64   `json:"rank"`
	SteamID       string  `json:"steam_id"`
	PlayerName    string  `json:"player_name"`
	Value         float64 `json:"value"`
	ActiveSeconds int64   `json:"active_play_seconds"`
}

type PlayerIdentity struct {
	SteamID string `json:"steam_id"`
	Name    string `json:"name"`
}

type ActivePlayer struct {
	SteamID          string
	Name             string
	StartedAt        int64
	LastSavedAt      int64
	ConnectedSeconds int64
}

type PlayerCompanion struct {
	PlayerName    string `json:"player_name"`
	SharedSeconds int64  `json:"shared_seconds"`
	SharedRounds  int64  `json:"shared_rounds"`
}

type PlayerPreviewPVE struct {
	Available           bool  `json:"available"`
	CommonKills         int64 `json:"common_kills"`
	SpecialKills        int64 `json:"special_kills"`
	BossKills           int64 `json:"boss_kills"`
	HeadshotKills       int64 `json:"headshot_kills"`
	Rescues             int64 `json:"rescues"`
	CampaignCompletions int64 `json:"campaign_completions"`
}

type PlayerPreviewVersus struct {
	Available               bool  `json:"available"`
	HumanSIKills            int64 `json:"human_si_kills"`
	InfectedDamage          int64 `json:"infected_damage"`
	SurvivorControls        int64 `json:"survivor_controls"`
	SurvivorIncapacitations int64 `json:"survivor_incapacitations"`
}

type PlayerPreview struct {
	SteamID           string               `json:"steam_id"`
	PlayerName        string               `json:"player_name"`
	SessionCount      int64                `json:"session_count"`
	ActivePlaySeconds int64                `json:"active_play_seconds"`
	LastSeenAt        int64                `json:"last_seen_at"`
	PVE               PlayerPreviewPVE     `json:"pve"`
	Versus            PlayerPreviewVersus  `json:"versus"`
	Companions        []PlayerCompanion    `json:"companions"`
	Badges            []PlayerPreviewBadge `json:"badges"`
	// MainBadge mirrors the first showcase slot for older API clients.
	MainBadge *PlayerPreviewBadge `json:"main_badge,omitempty"`
}

type PlayerPreviewBadge struct {
	Slot           int64  `json:"slot"`
	AchievementKey string `json:"achievement_key"`
	Title          string `json:"title"`
	ArtworkKey     string `json:"artwork_key"`
	Tier           int64  `json:"tier,omitempty"`
}

type CollectionCoverage struct {
	CollectedSegments int64 `json:"collected_segments"`
	TotalSegments     int64 `json:"total_segments"`
	Complete          bool  `json:"complete"`
}

type PlayerRelationshipDirection struct {
	IncapRevives            int64    `json:"incap_revives"`
	LedgeRescues            int64    `json:"ledge_rescues"`
	DefibRevives            int64    `json:"defib_revives"`
	SmokerRescues           int64    `json:"smoker_rescues"`
	HunterRescues           int64    `json:"hunter_rescues"`
	JockeyRescues           int64    `json:"jockey_rescues"`
	ChargerRescues          int64    `json:"charger_rescues"`
	SpecialRescues          int64    `json:"special_rescues"`
	SupportActions          int64    `json:"support_actions"`
	ControlRescueDurationMS int64    `json:"control_rescue_duration_ms"`
	AverageControlRescueMS  *float64 `json:"average_control_rescue_ms,omitempty"`
	MedkitsUsed             int64    `json:"medkits_used"`
	MedkitHealing           int64    `json:"medkit_healing"`
	BlackWhiteRestores      int64    `json:"black_white_restores"`
	FriendlyFireDamage      int64    `json:"friendly_fire_damage"`
}

type PlayerRelationship struct {
	PeerSteamID   string                      `json:"peer_steam_id"`
	PeerName      string                      `json:"peer_name"`
	SharedRounds  int64                       `json:"shared_rounds"`
	SharedSeconds int64                       `json:"shared_seconds"`
	Outgoing      PlayerRelationshipDirection `json:"outgoing"`
	Incoming      PlayerRelationshipDirection `json:"incoming"`
	MutualSupport int64                       `json:"mutual_support"`
}

type PlayerRelationshipSummary struct {
	PeerSteamID    string `json:"peer_steam_id"`
	PeerName       string `json:"peer_name"`
	SharedRounds   int64  `json:"shared_rounds"`
	SharedSeconds  int64  `json:"shared_seconds"`
	SupportActions int64  `json:"support_actions"`
}

type PlayerRelationshipSummaries struct {
	MostCompanion   *PlayerRelationshipSummary `json:"most_companion,omitempty"`
	MostSupported   *PlayerRelationshipSummary `json:"most_supported,omitempty"`
	MostSupportedBy *PlayerRelationshipSummary `json:"most_supported_by,omitempty"`
	MostMutual      *PlayerRelationshipSummary `json:"most_mutual,omitempty"`
}

type PlayerRelationshipQuery struct {
	PlayerFilter
	Page     int64
	PageSize int64
	Sort     string
	Order    string
}

type PlayerRelationshipPage struct {
	RelationshipVersion int64                       `json:"relationship_version"`
	Page                int64                       `json:"page"`
	PageSize            int64                       `json:"page_size"`
	Total               int64                       `json:"total"`
	Summaries           PlayerRelationshipSummaries `json:"summaries"`
	Items               []PlayerRelationship        `json:"items"`
}

type AnalysisFilter struct {
	Cutoff      int64
	ServerKey   string
	Mode        string
	CampaignKey string
	MapName     string
	Page        int64
	PageSize    int64
	Sort        string
	Order       string
}

type AnalysisOptions struct {
	Servers   []string `json:"servers"`
	Campaigns []string `json:"campaigns"`
}

type AnalysisMapRow struct {
	MapName                 string   `json:"map_name"`
	EligibleRounds          int64    `json:"eligible_rounds"`
	CompletedRounds         int64    `json:"completed_rounds"`
	FailedRounds            int64    `json:"failed_rounds"`
	AverageCompletedAttempt *float64 `json:"average_completed_attempt,omitempty"`
	AverageDurationSeconds  *float64 `json:"average_duration_seconds,omitempty"`
	CompleteIncidentRounds  int64    `json:"complete_incident_rounds"`
	Controls                int64    `json:"controls"`
	Incaps                  int64    `json:"incaps"`
	Deaths                  int64    `json:"deaths"`
}

type AnalysisMaps struct {
	IncidentVersion          int64            `json:"incident_version"`
	EligibleRounds           int64            `json:"eligible_rounds"`
	CompletionRate           *float64         `json:"completion_rate,omitempty"`
	AverageCompletedAttempt  *float64         `json:"average_completed_attempt,omitempty"`
	CompleteIncidentCoverage float64          `json:"complete_incident_coverage"`
	EarliestIncidentAt       int64            `json:"earliest_incident_at"`
	LatestIncidentAt         int64            `json:"latest_incident_at"`
	Page                     int64            `json:"page"`
	PageSize                 int64            `json:"page_size"`
	Total                    int64            `json:"total"`
	Maps                     []AnalysisMapRow `json:"maps"`
}

type IncidentComposition struct {
	Controls           int64 `json:"controls"`
	Incaps             int64 `json:"incaps"`
	Deaths             int64 `json:"deaths"`
	Revives            int64 `json:"revives"`
	LedgeRescues       int64 `json:"ledge_rescues"`
	DefibRevives       int64 `json:"defib_revives"`
	CarAlarms          int64 `json:"car_alarms"`
	WitchStartles      int64 `json:"witch_startles"`
	MedkitHeals        int64 `json:"medkit_heals"`
	ObjectiveCompletes int64 `json:"objective_completes"`
}

type AnalysisTimelinePoint struct {
	BucketSeconds      int64   `json:"bucket_seconds"`
	RoundsReached      int64   `json:"rounds_reached"`
	Controls           float64 `json:"controls_per_100_rounds"`
	Incaps             float64 `json:"incaps_per_100_rounds"`
	Deaths             float64 `json:"deaths_per_100_rounds"`
	WitchStartles      float64 `json:"witch_startles_per_100_rounds"`
	MedkitHeals        float64 `json:"medkit_heals_per_100_rounds"`
	ObjectiveCompletes float64 `json:"objective_completes_per_100_rounds"`
}

type BossAnalysis struct {
	SpawnCount      int64    `json:"spawn_count"`
	DeathCount      int64    `json:"death_count"`
	MatchedPairs    int64    `json:"matched_pairs"`
	AverageLifetime *float64 `json:"average_lifetime_seconds,omitempty"`
	MaximumLifetime *float64 `json:"maximum_lifetime_seconds,omitempty"`
	OneShotDeaths   int64    `json:"one_shot_deaths,omitempty"`
	StartleCount    int64    `json:"startle_count,omitempty"`
}

type AnalysisIncident struct {
	IncidentType  int64  `json:"incident_type"`
	OccurredAt    int64  `json:"occurred_at"`
	RoundOffsetMS int64  `json:"round_offset_ms"`
	ActorSteamID  string `json:"actor_steam_id,omitempty"`
	ActorName     string `json:"actor_name,omitempty"`
	TargetSteamID string `json:"target_steam_id,omitempty"`
	TargetName    string `json:"target_name,omitempty"`
}

type AnalysisMapDetail struct {
	Summary         AnalysisMapRow          `json:"summary"`
	Composition     IncidentComposition     `json:"incident_composition"`
	Timeline        []AnalysisTimelinePoint `json:"timeline"`
	Tank            BossAnalysis            `json:"tank"`
	Witch           BossAnalysis            `json:"witch"`
	RecentIncidents []AnalysisIncident      `json:"recent_incidents"`
}

type AnalysisContextRow struct {
	Fingerprint            string   `json:"fingerprint"`
	RulesetName            string   `json:"ruleset_name"`
	Difficulty             string   `json:"difficulty"`
	SurvivorLimit          int64    `json:"survivor_limit"`
	MaxPlayerZombies       int64    `json:"max_player_zombies"`
	CommonLimit            int64    `json:"common_limit"`
	TankHealth             int64    `json:"tank_health"`
	WitchHealth            int64    `json:"witch_health"`
	RoundCount             int64    `json:"round_count"`
	CompletedRounds        int64    `json:"completed_rounds"`
	FailedRounds           int64    `json:"failed_rounds"`
	AverageDurationSeconds *float64 `json:"average_duration_seconds,omitempty"`
	CompleteIncidentRounds int64    `json:"complete_incident_rounds"`
}

type AnalysisContexts struct {
	EligibleRounds      int64                `json:"eligible_rounds"`
	StableContextRounds int64                `json:"stable_context_rounds"`
	ChangedRuleRounds   int64                `json:"changed_rule_rounds"`
	NoContextRounds     int64                `json:"no_context_rounds"`
	Page                int64                `json:"page"`
	PageSize            int64                `json:"page_size"`
	Total               int64                `json:"total"`
	Contexts            []AnalysisContextRow `json:"contexts"`
}

type PlayerAnalysisTotals struct {
	ActiveSeconds  int64
	SpecialKills   int64
	Rescues        int64
	Incaps         int64
	Deaths         int64
	FriendlyFire   int64
	TankKills      int64
	WitchKills     int64
	Damage         int64
	Spawns         int64
	Controls       int64
	Kills          int64
	ControlSeconds int64
}

type PlayerIncidentClass struct {
	InfectedClass          int64   `json:"infected_class"`
	Controls               int64   `json:"controls"`
	AverageDurationSeconds float64 `json:"average_duration_seconds"`
}

type PlayerRescuer struct {
	PlayerName string `json:"player_name"`
	Rescues    int64  `json:"rescues"`
}

type PlayerIncidentAnalysis struct {
	EarliestIncidentAt    int64                 `json:"earliest_incident_at"`
	LatestIncidentAt      int64                 `json:"latest_incident_at"`
	ControlsReceived      int64                 `json:"controls_received"`
	AverageControlSeconds *float64              `json:"average_control_seconds,omitempty"`
	Incaps                int64                 `json:"incaps"`
	Deaths                int64                 `json:"deaths"`
	TeammatesRescued      int64                 `json:"teammates_rescued"`
	RescuedByTeammates    int64                 `json:"rescued_by_teammates"`
	ControlClasses        []PlayerIncidentClass `json:"control_classes"`
	TopRescuers           []PlayerRescuer       `json:"top_rescuers"`
	TwoCapEpisodes        int64                 `json:"two_cap_episodes"`
	ThreeCapEpisodes      int64                 `json:"three_cap_episodes"`
	FourCapEpisodes       int64                 `json:"four_cap_episodes"`
}

type PlayerAnalysis struct {
	View          string                 `json:"view"`
	ActiveSeconds int64                  `json:"active_play_seconds"`
	Metrics       map[string]*float64    `json:"metrics"`
	Samples       map[string]int64       `json:"samples"`
	Incidents     PlayerIncidentAnalysis `json:"recent_incidents"`
}

type RankingPage struct {
	Metric         string         `json:"metric"`
	Mode           string         `json:"mode"`
	HigherIsBetter bool           `json:"higher_is_better"`
	LowerIsBetter  bool           `json:"lower_is_better"`
	Items          []RankingEntry `json:"items"`
	Total          int64          `json:"total"`
	Self           *RankingEntry  `json:"self,omitempty"`
	GeneratedAt    time.Time      `json:"generated_at"`
}

type RankingQuery struct {
	Mode             string
	Metric           string
	ServerKey        string
	Cutoff           int64
	MinimumActiveSec int64
	SteamIDs         []string
	SubjectSteamID   string
	Limit            int
	Offset           int
}

type AggregateStatus struct {
	AggregateVersion int64  `json:"aggregate_version"`
	State            string `json:"state"`
	LastStartedAt    int64  `json:"last_started_at"`
	LastFinishedAt   int64  `json:"last_finished_at"`
	SourceRows       int64  `json:"source_rows"`
	AggregateRows    int64  `json:"aggregate_rows"`
	SourceWatermark  int64  `json:"source_watermark"`
	LastDurationMS   int64  `json:"last_duration_ms"`
	LastChangedDays  int64  `json:"last_changed_days"`
	LastBuildMode    string `json:"last_build_mode"`
	LastError        string `json:"last_error,omitempty"`
}

type AggregateGrain string

const (
	AggregateContractVersion int64          = 1
	AggregateGrainDaily      AggregateGrain = "daily"
	AggregateGrainMonthly    AggregateGrain = "monthly"
	AggregateGrainLifetime   AggregateGrain = "lifetime"
)

type RetentionPlan struct {
	AggregateVersion           int64  `json:"aggregate_version"`
	GeneratedAt                int64  `json:"generated_at"`
	DetailCutoff               int64  `json:"detail_cutoff"`
	SessionCutoff              int64  `json:"session_cutoff"`
	ResultCutoff               int64  `json:"result_cutoff"`
	EquipmentRowsEligible      int64  `json:"equipment_rows_eligible"`
	VersusClassRowsEligible    int64  `json:"versus_class_rows_eligible"`
	SessionRowsEligible        int64  `json:"session_rows_eligible"`
	VersusRoundResultsEligible int64  `json:"versus_round_results_eligible"`
	VersusRunResultsEligible   int64  `json:"versus_run_results_eligible"`
	SourceWatermark            int64  `json:"source_watermark"`
	PlanID                     string `json:"plan_id"`
	DeletionEnabled            bool   `json:"deletion_enabled"`
	AggregateCoverageReady     bool   `json:"aggregate_coverage_ready"`
}

type RetentionResult struct {
	RunID                 string `json:"run_id"`
	ExecutedAt            int64  `json:"executed_at"`
	EquipmentRows         int64  `json:"equipment_rows"`
	VersusClassRows       int64  `json:"versus_class_rows"`
	SessionRows           int64  `json:"session_rows"`
	VersusRoundResultRows int64  `json:"versus_round_result_rows"`
	VersusRunResultRows   int64  `json:"versus_run_result_rows"`
}

type DataMaintenanceSettings struct {
	AggregateIntervalMinutes int64 `json:"aggregate_interval_minutes"`
	DetailRetentionDays      int64 `json:"detail_retention_days"`
	SessionRetentionDays     int64 `json:"session_retention_days"`
	ResultRetentionDays      int64 `json:"result_retention_days"`
	IncidentRetentionDays    int64 `json:"incident_retention_days"`
	UpdatedAt                int64 `json:"updated_at"`
}

type AnalysisStatus struct {
	IncidentVersion           int64   `json:"incident_version"`
	IncidentRows              int64   `json:"incident_rows"`
	CaptureEnabledRounds      int64   `json:"capture_enabled_rounds"`
	CompleteRounds            int64   `json:"complete_rounds"`
	CompleteRatio             float64 `json:"complete_ratio"`
	RowsLast30Days            int64   `json:"rows_last_30d"`
	EarliestIncidentAt        int64   `json:"earliest_incident_at"`
	LatestIncidentAt          int64   `json:"latest_incident_at"`
	ProjectedRowsForRetention int64   `json:"projected_rows_for_retention"`
	RetentionDays             int64   `json:"retention_days"`
	CleanupRuns               int64   `json:"cleanup_runs"`
}

type IncidentRetentionPlan struct {
	IncidentVersion      int64  `json:"incident_version"`
	GeneratedAt          int64  `json:"generated_at"`
	Cutoff               int64  `json:"cutoff"`
	IncidentRowsEligible int64  `json:"incident_rows_eligible"`
	UnknownVersionRows   int64  `json:"unknown_version_rows"`
	CandidateWatermark   int64  `json:"candidate_watermark"`
	PlanID               string `json:"plan_id"`
	DeletionEnabled      bool   `json:"deletion_enabled"`
}

type IncidentRetentionResult struct {
	RunID        string `json:"run_id"`
	ExecutedAt   int64  `json:"executed_at"`
	IncidentRows int64  `json:"incident_rows"`
}

type DatabaseUsage struct {
	Driver   string `json:"driver"`
	Bytes    int64  `json:"bytes"`
	WALBytes int64  `json:"wal_bytes,omitempty"`
}

type DataQualityFinding struct {
	Count int64
	IDs   []string
}

type StatsDataQuality struct {
	SourceWatermark      int64
	StaleActiveBoots     DataQualityFinding
	UnknownStatsVersion  DataQualityFinding
	LifecycleLinks       DataQualityFinding
	ModeSideMismatch     DataQualityFinding
	PVETotalMismatch     DataQualityFinding
	ContextContract      DataQualityFinding
	IncidentContract     DataQualityFinding
	IncidentCompleteness DataQualityFinding
	RelationshipContract DataQualityFinding
	PVEAssistContract    DataQualityFinding
	VersusAssistContract DataQualityFinding
	FallDeathContract    DataQualityFinding
}

type PlayerSession struct {
	SessionID        string `json:"-"`
	ServerKey        string `json:"server_key"`
	PlayerName       string `json:"player_name"`
	StartedAt        int64  `json:"started_at"`
	EndedAt          *int64 `json:"ended_at"`
	ConnectedSeconds int64  `json:"connected_seconds"`
	ActiveSeconds    int64  `json:"active_play_seconds"`
	Status           string `json:"status"`
	DisconnectReason string `json:"disconnect_reason"`
}

type PlayerChapter struct {
	SegmentID     string `json:"-"`
	ServerKey     string `json:"server_key"`
	ModeFamily    string `json:"mode_family"`
	GameMode      string `json:"game_mode"`
	MapName       string `json:"map_name"`
	Side          string `json:"side"`
	StartedAt     int64  `json:"started_at"`
	EndedAt       *int64 `json:"ended_at"`
	ActiveSeconds int64  `json:"active_play_seconds"`
	Status        string `json:"status"`
}

type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type CoreOverview struct {
	TotalPlayers           int64 `json:"total_players"`
	ActivePlayers7Days     int64 `json:"active_players_7d"`
	TotalActivePlaySeconds int64 `json:"total_active_play_seconds"`
	CompletedPVERuns       int64 `json:"completed_pve_runs"`
	CompletedVersusRuns    int64 `json:"completed_versus_runs"`
}

type PVEOverview struct {
	CommonKills  int64 `json:"common_kills"`
	SpecialKills int64 `json:"special_kills"`
	TankKills    int64 `json:"tank_kills"`
	WitchKills   int64 `json:"witch_kills"`
	Rescues      int64 `json:"rescues"`
}

type VersusOverview struct {
	CompletedMatches      int64 `json:"completed_matches"`
	CompletedHalves       int64 `json:"completed_halves"`
	HumanControlledKills  int64 `json:"human_controlled_infected_kills"`
	HumanSurvivorControls int64 `json:"human_survivor_controls"`
}

type Overview struct {
	Core      CoreOverview   `json:"core"`
	PVE       PVEOverview    `json:"pve"`
	Versus    VersusOverview `json:"versus"`
	Generated time.Time      `json:"generated_at"`
}

type ServerStatus struct {
	ServerID      string         `json:"server_id"`
	ServerKey     string         `json:"server_key,omitempty"`
	DisplayName   string         `json:"display_name"`
	Address       string         `json:"address"`
	Online        bool           `json:"online"`
	Stale         bool           `json:"stale"`
	Checking      bool           `json:"checking"`
	Name          string         `json:"name,omitempty"`
	Map           string         `json:"map,omitempty"`
	Players       int            `json:"players"`
	MaxPlayers    int            `json:"max_players"`
	Bots          int            `json:"bots"`
	LatencyMS     int64          `json:"latency_ms,omitempty"`
	LastSuccessAt time.Time      `json:"last_success_at,omitempty"`
	CheckedAt     time.Time      `json:"checked_at"`
	PlayerList    []ServerPlayer `json:"player_list,omitempty"`
	Rules         []ServerRule   `json:"rules,omitempty"`
}

type ServerPlayer struct {
	Name            string `json:"name"`
	SteamID         string `json:"steam_id,omitempty"`
	Score           int32  `json:"score"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type ServerRule struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DashboardStore interface {
	Ping(context.Context) error
	MigrationVersion(context.Context) (int64, error)
	Site(context.Context) (Site, error)
	SiteSettings(context.Context) (SiteSettings, error)
	UpdateSite(context.Context, SiteSettings) error
	ListSiteDocuments(context.Context, bool) ([]SiteDocument, error)
	GetSiteDocument(context.Context, string, bool) (SiteDocument, error)
	UpdateSiteDocument(context.Context, SiteDocument) (SiteDocument, error)
	ListServers(context.Context) ([]GameServer, error)
	CreateServer(context.Context, GameServer) (GameServer, error)
	UpdateServer(context.Context, GameServer) error
	SetServerEnabled(context.Context, string, bool) error
	MoveServer(context.Context, string, string) error
	DeleteServer(context.Context, string) error
	AdminConfigured(context.Context) (bool, error)
	CreateAdmin(context.Context, string, string, string) error
	Admin(context.Context) (*AdminAccount, error)
	UpdateAdminUsername(context.Context, string) error
	UpdateAdminPassword(context.Context, string) error
	ListAnnouncements(context.Context, AnnouncementFilter) (AnnouncementPage, error)
	ListAnnouncementYears(context.Context) ([]int, error)
	GetAnnouncement(context.Context, string) (Announcement, error)
	CreateAnnouncement(context.Context, Announcement) (Announcement, error)
	UpdateAnnouncement(context.Context, Announcement) (Announcement, error)
	DeleteAnnouncement(context.Context, string) error
	Close() error
}

type DashboardProfileStore interface {
	PlayerProfileVisibility(context.Context, string) (PlayerProfileVisibility, error)
	ReplacePlayerProfileVisibility(context.Context, string, []PlayerProfileSection, int64) (PlayerProfileVisibility, error)
}

type DashboardAggregateStore interface {
	AggregateStatus(context.Context) (AggregateStatus, error)
	ReplaceAggregateRows(context.Context, []AggregateRow, int64) error
	ApplyAggregateChanges(context.Context, AggregateChangeSet) error
	ListAggregateRows(context.Context, AggregateFilter) ([]AggregateRow, error)
	DataMaintenanceSettings(context.Context) (DataMaintenanceSettings, error)
	UpdateDataMaintenanceSettings(context.Context, DataMaintenanceSettings) error
	DatabaseUsage(context.Context) (DatabaseUsage, error)
	RecordRetentionRun(context.Context, RetentionPlan, RetentionResult) error
	RetentionRunCount(context.Context) (int64, error)
	RecordIncidentRetentionRun(context.Context, IncidentRetentionPlan, IncidentRetentionResult) error
	IncidentRetentionRunCount(context.Context) (int64, error)
}

type DashboardDatabase interface {
	DashboardStore
	DashboardProfileStore
	DashboardAggregateStore
	DashboardAchievementStore
	ServerStatusSnapshotStore
}

type AggregateRow struct {
	Version   int64
	Kind      string
	Day       int64
	ServerKey string
	SteamID   string
	Mode      string
	Dimension string
	Metrics   map[string]int64
}

type AggregateFilter struct {
	Grain     AggregateGrain
	Kinds     []string
	SteamID   string
	ServerKey string
	Mode      string
	CutoffDay int64
}

type StatsStore interface {
	Ping(context.Context) error
	SchemaVersion(context.Context) (int64, error)
	Overview(context.Context, time.Time) (Overview, error)
	PlayerSummary(context.Context, string) (*PlayerSummary, error)
	SearchPlayers(context.Context, string, int32) ([]PlayerIdentity, error)
	PlayerPVE(context.Context, string, int64) (PlayerPVE, error)
	PlayerVersus(context.Context, string, int64) (PlayerVersus, error)
	PlayerActivity(context.Context, string, int64) (PlayerActivity, error)
	PlayerSessions(context.Context, string, int64, string, int32) ([]PlayerSession, error)
	PlayerChapters(context.Context, string, int64, string, int32) ([]PlayerChapter, error)
	Close() error
}

type StatsDoctorStore interface {
	DeepDataQuality(context.Context, int64) (StatsDataQuality, error)
}

type StatsAggregateStore interface {
	AggregateRows(context.Context) ([]AggregateRow, error)
	AggregateChanges(context.Context, int64) (AggregateChangeSet, error)
	RetentionPlan(context.Context, int64, int64, int64) (RetentionPlan, error)
	DatabaseUsage(context.Context) (DatabaseUsage, error)
}

type StatsAnalysisMaintenanceStore interface {
	AnalysisStatus(context.Context, int64) (AnalysisStatus, error)
	IncidentRetentionPlan(context.Context, int64) (IncidentRetentionPlan, error)
}

type AggregateChangeSet struct {
	Rows            []AggregateRow
	Days            []int64
	SourceWatermark int64
	SourceRows      int64
	Full            bool
}

type StatsMaintenanceStore interface {
	StatsAggregateStore
	StatsAnalysisMaintenanceStore
	ApplyRetention(context.Context, RetentionPlan) (RetentionResult, error)
	ApplyIncidentRetention(context.Context, IncidentRetentionPlan) (IncidentRetentionResult, error)
	Close() error
}

type StatsFilteredStore interface {
	PlayerPVEFiltered(context.Context, string, PlayerFilter) (PlayerPVE, error)
	PlayerVersusFiltered(context.Context, string, PlayerFilter) (PlayerVersus, error)
	PlayerActivityFiltered(context.Context, string, PlayerFilter) (PlayerActivity, error)
}

type StatsPresenceStore interface {
	ActivePlayers(context.Context, string, int64) ([]ActivePlayer, error)
}

type StatsIncidentRankingStore interface {
	CarAlarmRanking(context.Context, RankingQuery) ([]RankingEntry, error)
}

type StatsAnalysisStore interface {
	PlayerCompanions(context.Context, string) ([]PlayerCompanion, error)
	AnalysisOptions(context.Context, AnalysisFilter) (AnalysisOptions, error)
	AnalysisMaps(context.Context, AnalysisFilter) (AnalysisMaps, error)
	AnalysisMapDetail(context.Context, AnalysisFilter, string) (AnalysisMapDetail, error)
	AnalysisContexts(context.Context, AnalysisFilter) (AnalysisContexts, error)
	PlayerAnalysisTotals(context.Context, string, PlayerFilter, string) (PlayerAnalysisTotals, error)
	PlayerIncidentAnalysis(context.Context, string, PlayerFilter) (PlayerIncidentAnalysis, error)
}

type StatsRelationshipStore interface {
	PlayerRelationships(context.Context, string, PlayerRelationshipQuery) (PlayerRelationshipPage, error)
}

type StatsDatabase interface {
	StatsStore
	StatsAggregateStore
	StatsFilteredStore
	StatsPresenceStore
	StatsDoctorStore
	StatsAchievementStore
}

type ServerStatusProvider interface {
	Statuses(context.Context) ([]ServerStatus, error)
	LastStatus(context.Context, string) (ServerStatus, bool, error)
	RefreshStatus(context.Context, string) (ServerStatus, error)
	InvalidateServer(string)
}

type ServerStatusSnapshotStore interface {
	ListServerStatusSnapshots(context.Context) ([]ServerStatus, error)
	UpsertServerStatusSnapshot(context.Context, ServerStatus) error
}
