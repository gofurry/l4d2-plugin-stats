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
	HighlightMetrics    [3]string `json:"highlight_metrics"`
	HomeCacheSeconds    int64     `json:"home_cache_seconds"`
	PlayerCacheSeconds  int64     `json:"player_cache_seconds"`
	RankingCacheSeconds int64     `json:"ranking_cache_seconds"`
	ContentCacheSeconds int64     `json:"content_cache_seconds"`
	UpdatedAt           int64     `json:"updated_at"`
}

type IngameServerSettings struct {
	ServerID         string    `json:"server_id"`
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

type ServerDocument struct {
	ServerID        string `json:"server_id"`
	Key             string `json:"key"`
	Mode            string `json:"mode"`
	ContentMarkdown string `json:"content_markdown"`
	UpdatedAt       int64  `json:"updated_at"`
}

type DashboardIngameStore interface {
	IngameSettings(context.Context) (IngameSettings, error)
	UpdateIngameSettings(context.Context, IngameSettings) (IngameSettings, error)
	IngameServerSettings(context.Context, string) (IngameServerSettings, error)
	UpdateIngameServerSettings(context.Context, IngameServerSettings) (IngameServerSettings, error)
	DeleteIngameServerSettings(context.Context, string) error
	ListServerDocuments(context.Context, string) ([]ServerDocument, error)
	GetServerDocument(context.Context, string, string) (ServerDocument, error)
	UpdateServerDocument(context.Context, ServerDocument) (ServerDocument, error)
	DeleteServerDocuments(context.Context, string) error
}
