package store

import "context"

const (
	IngameDocumentIntroduction = "introduction"
	IngameDocumentCommands     = "commands"
	IngameDocumentResources    = "resources"
)

type IngameSettings struct {
	Enabled             bool      `json:"enabled"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	BannerURL           string    `json:"banner_url"`
	BackgroundURL       string    `json:"background_url"`
	WebsiteURL          string    `json:"website_url"`
	ShowAnnouncements   bool      `json:"show_announcements"`
	ShowPlayers         bool      `json:"show_players"`
	ShowHighlights      bool      `json:"show_highlights"`
	ShowServerIntro     bool      `json:"show_server_intro"`
	ShowServerStatus    bool      `json:"show_server_status"`
	HighlightMetrics    [3]string `json:"highlight_metrics"`
	HomeCacheSeconds    int64     `json:"home_cache_seconds"`
	PlayerCacheSeconds  int64     `json:"player_cache_seconds"`
	RankingCacheSeconds int64     `json:"ranking_cache_seconds"`
	ContentCacheSeconds int64     `json:"content_cache_seconds"`
	UpdatedAt           int64     `json:"updated_at"`
}

type IngameServerSettings struct {
	ServerKey        string    `json:"server_key"`
	TitleMode        string    `json:"title_mode"`
	Title            string    `json:"title"`
	DescriptionMode  string    `json:"description_mode"`
	Description      string    `json:"description"`
	BannerMode       string    `json:"banner_mode"`
	BannerURL        string    `json:"banner_url"`
	BackgroundMode   string    `json:"background_mode"`
	BackgroundURL    string    `json:"background_url"`
	WebsiteMode      string    `json:"website_mode"`
	WebsiteURL       string    `json:"website_url"`
	HighlightMode    string    `json:"highlight_mode"`
	HighlightMetrics [3]string `json:"highlight_metrics"`
	UpdatedAt        int64     `json:"updated_at"`
}

type IngameMapName struct {
	MapName     string `json:"map_name"`
	DisplayName string `json:"display_name"`
	UpdatedAt   int64  `json:"updated_at"`
}

type ServerDocument struct {
	ServerKey       string `json:"server_key"`
	Key             string `json:"key"`
	Mode            string `json:"mode"`
	ContentMarkdown string `json:"content_markdown"`
	UpdatedAt       int64  `json:"updated_at"`
}

type IngameQuickLink struct {
	ServerKey string `json:"server_key"`
	Label     string `json:"label"`
	URL       string `json:"url"`
	SortOrder int64  `json:"sort_order"`
	Enabled   bool   `json:"enabled"`
}

type DashboardIngameStore interface {
	IngameSettings(context.Context) (IngameSettings, error)
	UpdateIngameSettings(context.Context, IngameSettings) (IngameSettings, error)
	IngameServerSettings(context.Context, string) (IngameServerSettings, error)
	ListIngameServerSettings(context.Context) ([]IngameServerSettings, error)
	UpdateIngameServerSettings(context.Context, IngameServerSettings) (IngameServerSettings, error)
	DeleteIngameServerSettings(context.Context, string) error
	ListServerDocuments(context.Context, string) ([]ServerDocument, error)
	GetServerDocument(context.Context, string, string) (ServerDocument, error)
	UpdateServerDocument(context.Context, ServerDocument) (ServerDocument, error)
	DeleteServerDocuments(context.Context, string) error
	ListServerQuickLinks(context.Context, string) ([]IngameQuickLink, error)
	ReplaceServerQuickLinks(context.Context, string, []IngameQuickLink) ([]IngameQuickLink, error)
	ListIngameMapNames(context.Context) ([]IngameMapName, error)
	ReplaceIngameMapNames(context.Context, []IngameMapName) ([]IngameMapName, error)
}
