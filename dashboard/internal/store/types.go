package store

import (
	"context"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
)

type FooterLink struct {
	Label      string `json:"label"`
	URL        string `json:"url"`
	OpenNewTab bool   `json:"open_new_tab"`
}

type Site struct {
	Title      string       `json:"title"`
	FooterText string       `json:"footer_text"`
	Links      []FooterLink `json:"footer_links"`
}

type GameServer struct {
	ID             int64  `json:"-"`
	ServerKey      string `json:"server_key"`
	DisplayName    string `json:"display_name"`
	ConnectAddress string `json:"connect_address"`
	QueryAddress   string `json:"-"`
	Primary        bool   `json:"primary"`
	Enabled        bool   `json:"enabled"`
	SortOrder      int64  `json:"sort_order"`
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
	ConfiguredName string    `json:"configured_name"`
	ConnectAddress string    `json:"connect_address"`
	Online         bool      `json:"online"`
	Stale          bool      `json:"stale"`
	Name           string    `json:"name,omitempty"`
	Map            string    `json:"map,omitempty"`
	Players        int       `json:"players"`
	MaxPlayers     int       `json:"max_players"`
	Bots           int       `json:"bots"`
	LatencyMS      int64     `json:"latency_ms,omitempty"`
	LastSuccessAt  time.Time `json:"last_success_at,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

type DashboardStore interface {
	Ping(context.Context) error
	MigrationVersion(context.Context) (int64, error)
	Bootstrap(context.Context, config.BootstrapConfig, bool) (bool, error)
	Site(context.Context) (Site, error)
	PrimaryServer(context.Context) (*GameServer, error)
	Close() error
}

type StatsStore interface {
	Ping(context.Context) error
	SchemaVersion(context.Context) (int64, error)
	Overview(context.Context, time.Time) (Overview, error)
	Close() error
}

type ServerStatusProvider interface {
	PrimaryStatus(context.Context) (*ServerStatus, error)
}
