package store

import (
	"context"
	"errors"
	"time"
)

var ErrServerNotFound = errors.New("game server not found")

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
	Language           string       `json:"language"`
	BrowserTitle       string       `json:"browser_title"`
	Theme              string       `json:"theme"`
	FooterEnabled      bool         `json:"footer_enabled"`
	BackgroundImageURL string       `json:"background_image_url"`
	PublicOrigin       string       `json:"public_origin"`
	SteamOpenIDEnabled bool         `json:"steam_openid_enabled"`
	A2SRefreshSeconds  int64        `json:"a2s_refresh_seconds"`
	A2SJitterSeconds   int64        `json:"a2s_jitter_seconds"`
	A2SRetryCount      int64        `json:"a2s_retry_count"`
	SEOEnabled         bool         `json:"seo_enabled"`
	SEODescription     string       `json:"seo_description"`
	SEOImageURL        string       `json:"seo_image_url"`
	Links              []FooterLink `json:"footer_links"`
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
	ClassID           int   `json:"class_id"`
	Kills             int64 `json:"kills"`
	Damage            int64 `json:"damage"`
	ControlsReceived  int64 `json:"controls_received"`
	ControlledSeconds int64 `json:"controlled_seconds"`
	Saves             int64 `json:"saves"`
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
	Classes               []PVEInfectedClass `json:"infected_classes"`
	Equipment             []PVEEquipment     `json:"equipment"`
}

type VersusSurvivorClass struct {
	ClassID                  int64 `json:"class_id"`
	HumanControllerKills     int64 `json:"human_controller_kills"`
	BotControllerKills       int64 `json:"bot_controller_kills"`
	DamageToHumanControllers int64 `json:"damage_to_human_controllers"`
	DamageToBotControllers   int64 `json:"damage_to_bot_controllers"`
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
	SurvivorCommonKills         int64                 `json:"survivor_common_kills"`
	HumanSpecialKills           int64                 `json:"human_special_kills"`
	BotSpecialKills             int64                 `json:"bot_special_kills"`
	HumanTankKills              int64                 `json:"human_tank_kills"`
	BotTankKills                int64                 `json:"bot_tank_kills"`
	SurvivorDamage              int64                 `json:"survivor_damage"`
	SurvivorDeaths              int64                 `json:"survivor_deaths"`
	SurvivorRevives             int64                 `json:"survivor_revives"`
	InfectedSpawns              int64                 `json:"infected_spawns"`
	DamageToHumanSurvivors      int64                 `json:"damage_to_human_survivors"`
	HumanSurvivorIncaps         int64                 `json:"human_survivor_incaps"`
	HumanSurvivorKills          int64                 `json:"human_survivor_kills"`
	HumanSurvivorControls       int64                 `json:"human_survivor_controls"`
	HumanSurvivorControlSeconds int64                 `json:"human_survivor_control_seconds"`
	SurvivorDamageTaken         int64                 `json:"survivor_damage_taken"`
	SurvivorFriendlyFire        int64                 `json:"survivor_friendly_fire"`
	SurvivorFriendlyFireTaken   int64                 `json:"survivor_friendly_fire_taken"`
	SurvivorIncapacitations     int64                 `json:"survivor_incapacitations"`
	SurvivorIncapRevives        int64                 `json:"survivor_incap_revives"`
	SurvivorLedgeRescues        int64                 `json:"survivor_ledge_rescues"`
	SurvivorDefibRevives        int64                 `json:"survivor_defib_revives"`
	SurvivorRescuesReceived     int64                 `json:"survivor_rescues_received"`
	SurvivorMedkitsSelf         int64                 `json:"survivor_medkits_self"`
	SurvivorMedkitsOthers       int64                 `json:"survivor_medkits_others"`
	SurvivorHealingSelf         int64                 `json:"survivor_healing_self"`
	SurvivorHealingOthers       int64                 `json:"survivor_healing_others"`
	SurvivorPills               int64                 `json:"survivor_pills"`
	SurvivorAdrenaline          int64                 `json:"survivor_adrenaline"`
	SurvivorTemporaryHealth     int64                 `json:"survivor_temporary_health"`
	SurvivorWitchKills          int64                 `json:"survivor_witch_kills"`
	SurvivorWitchDamage         int64                 `json:"survivor_witch_damage"`
	MolotovsThrown              int64                 `json:"molotovs_thrown"`
	PipeBombsThrown             int64                 `json:"pipe_bombs_thrown"`
	VomitJarsThrown             int64                 `json:"vomit_jars_thrown"`
	SurvivorIncendiaryPacks     int64                 `json:"survivor_incendiary_packs"`
	SurvivorExplosivePacks      int64                 `json:"survivor_explosive_packs"`
	SurvivorTongueSelfCuts      int64                 `json:"survivor_tongue_self_cuts"`
	SurvivorTankRocksDestroyed  int64                 `json:"survivor_tank_rocks_destroyed"`
	SurvivorWitchOneShots       int64                 `json:"survivor_witch_oneshots"`
	SurvivorWitchSoloKills      int64                 `json:"survivor_witch_solo_kills"`
	DamageToBotSurvivors        int64                 `json:"damage_to_bot_survivors"`
	BotSurvivorIncaps           int64                 `json:"bot_survivor_incaps"`
	BotSurvivorKills            int64                 `json:"bot_survivor_kills"`
	SurvivorClasses             []VersusSurvivorClass `json:"survivor_classes"`
	InfectedClasses             []VersusInfectedClass `json:"infected_classes"`
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

type RankingPage struct {
	Metric      string         `json:"metric"`
	Mode        string         `json:"mode"`
	Items       []RankingEntry `json:"items"`
	Total       int64          `json:"total"`
	Self        *RankingEntry  `json:"self,omitempty"`
	GeneratedAt time.Time      `json:"generated_at"`
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
	State          string `json:"state"`
	LastStartedAt  int64  `json:"last_started_at"`
	LastFinishedAt int64  `json:"last_finished_at"`
	SourceRows     int64  `json:"source_rows"`
	AggregateRows  int64  `json:"aggregate_rows"`
	LastError      string `json:"last_error,omitempty"`
}

type RetentionPlan struct {
	GeneratedAt             int64 `json:"generated_at"`
	DetailCutoff            int64 `json:"detail_cutoff"`
	SegmentCutoff           int64 `json:"segment_cutoff"`
	EquipmentRowsEligible   int64 `json:"equipment_rows_eligible"`
	VersusClassRowsEligible int64 `json:"versus_class_rows_eligible"`
	SegmentRowsEligible     int64 `json:"segment_rows_eligible"`
	DeletionEnabled         bool  `json:"deletion_enabled"`
	AggregateCoverageReady  bool  `json:"aggregate_coverage_ready"`
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
	DisplayName   string         `json:"display_name"`
	Address       string         `json:"address"`
	Online        bool           `json:"online"`
	Stale         bool           `json:"stale"`
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

type DashboardAggregateStore interface {
	AggregateStatus(context.Context) (AggregateStatus, error)
	ReplaceAggregateRows(context.Context, []AggregateRow, int64) error
	ListAggregateRows(context.Context, AggregateFilter) ([]AggregateRow, error)
}

type DashboardDatabase interface {
	DashboardStore
	DashboardAggregateStore
}

type AggregateRow struct {
	Kind      string
	Day       int64
	ServerKey string
	SteamID   string
	Mode      string
	Dimension string
	Metrics   map[string]int64
}

type AggregateFilter struct {
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

type StatsAggregateStore interface {
	AggregateRows(context.Context) ([]AggregateRow, error)
	RetentionPlan(context.Context, int64, int64) (RetentionPlan, error)
}

type StatsFilteredStore interface {
	PlayerPVEFiltered(context.Context, string, PlayerFilter) (PlayerPVE, error)
	PlayerVersusFiltered(context.Context, string, PlayerFilter) (PlayerVersus, error)
	PlayerActivityFiltered(context.Context, string, PlayerFilter) (PlayerActivity, error)
}

type StatsDatabase interface {
	StatsStore
	StatsAggregateStore
	StatsFilteredStore
}

type ServerStatusProvider interface {
	Statuses(context.Context) ([]ServerStatus, error)
	LastStatus(context.Context, string) (ServerStatus, bool, error)
	RefreshStatus(context.Context, string) (ServerStatus, error)
}
