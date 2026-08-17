package service

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type IngameMetricDefinition struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Mode          string `json:"mode"`
	RankingMetric string `json:"ranking_metric"`
	Format        string `json:"format"`
}

var ingameMetricCatalog = []IngameMetricDefinition{
	{Key: "active_play_seconds", Label: "实际游戏时间", Mode: "activity", RankingMetric: "active_time", Format: "duration"},
	{Key: "sessions", Label: "会话次数", Mode: "activity", RankingMetric: "sessions", Format: "integer"},
	{Key: "common_kills", Label: "普通感染者击杀", Mode: "pve", RankingMetric: "common_kills", Format: "integer"},
	{Key: "special_kills", Label: "特殊感染者击杀", Mode: "pve", RankingMetric: "special_kills", Format: "integer"},
	{Key: "boss_kills", Label: "Boss 击杀", Mode: "pve", RankingMetric: "boss_kills", Format: "integer"},
	{Key: "campaign_completions", Label: "战役完成", Mode: "pve", RankingMetric: "campaign_completions", Format: "integer"},
	{Key: "rescues", Label: "队友救援", Mode: "pve", RankingMetric: "rescues", Format: "integer"},
	{Key: "human_si_kills", Label: "真人特感击杀", Mode: "versus_survivor", RankingMetric: "human_si_kills", Format: "integer"},
	{Key: "infected_damage", Label: "感染者伤害", Mode: "versus_infected", RankingMetric: "damage", Format: "integer"},
	{Key: "survivor_controls", Label: "控制幸存者", Mode: "versus_infected", RankingMetric: "controls", Format: "integer"},
	{Key: "survivor_incaps", Label: "击倒幸存者", Mode: "versus_infected", RankingMetric: "incaps", Format: "integer"},
}

func IngameMetricCatalog() []IngameMetricDefinition {
	return append([]IngameMetricDefinition(nil), ingameMetricCatalog...)
}

func IngameMetric(key string) (IngameMetricDefinition, bool) {
	for _, metric := range ingameMetricCatalog {
		if metric.Key == key {
			return metric, true
		}
	}
	return IngameMetricDefinition{}, false
}

func ValidateIngameURL(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > 2048 {
		return errors.New("URL must not exceed 2048 bytes")
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return errors.New("URL must be an absolute credential-free HTTP or HTTPS URL")
	}
	return nil
}

func ValidateIngameServerKey(value string) error {
	value = strings.TrimSpace(value)
	if len(value) < 1 || len(value) > 64 || strings.EqualFold(value, "change-me") {
		return errors.New("server_key must contain 1 to 64 characters and must not be change-me")
	}
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') || (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-' {
			continue
		}
		return errors.New("server_key may only contain letters, numbers, dot, underscore, and hyphen")
	}
	return nil
}

func ValidateIngameSettings(settings store.IngameSettings) error {
	settings.Title = strings.TrimSpace(settings.Title)
	if len(settings.Title) > 128 {
		return errors.New("title must not exceed 128 characters")
	}
	if len(settings.Description) > 1000 {
		return errors.New("description must not exceed 1000 characters")
	}
	if err := ValidateIngameURL(strings.TrimSpace(settings.BannerURL)); err != nil {
		return fmt.Errorf("banner_url: %w", err)
	}
	if err := ValidateIngameURL(strings.TrimSpace(settings.BackgroundURL)); err != nil {
		return fmt.Errorf("background_url: %w", err)
	}
	if err := ValidateIngameURL(strings.TrimSpace(settings.WebsiteURL)); err != nil {
		return fmt.Errorf("website_url: %w", err)
	}
	if err := validateIngameMetrics(settings.HighlightMetrics); err != nil {
		return err
	}
	if !slices.Contains([]int64{10, 30, 60, 120}, settings.HomeCacheSeconds) {
		return errors.New("home_cache_seconds must use an approved preset")
	}
	if !slices.Contains([]int64{30, 60, 120, 300}, settings.PlayerCacheSeconds) {
		return errors.New("player_cache_seconds must use an approved preset")
	}
	if !slices.Contains([]int64{60, 120, 300, 600}, settings.RankingCacheSeconds) {
		return errors.New("ranking_cache_seconds must use an approved preset")
	}
	if !slices.Contains([]int64{60, 300, 600, 1800}, settings.ContentCacheSeconds) {
		return errors.New("content_cache_seconds must use an approved preset")
	}
	return nil
}

func ValidateIngameServerSettings(settings store.IngameServerSettings) error {
	if err := ValidateIngameServerKey(settings.ServerKey); err != nil {
		return err
	}
	if !slices.Contains([]string{"inherit", "override"}, settings.TitleMode) {
		return errors.New("title_mode is invalid")
	}
	if !slices.Contains([]string{"inherit", "override", "hidden"}, settings.DescriptionMode) {
		return errors.New("description_mode is invalid")
	}
	if !slices.Contains([]string{"inherit", "override", "hidden"}, settings.BannerMode) {
		return errors.New("banner_mode is invalid")
	}
	if !slices.Contains([]string{"inherit", "override", "hidden"}, settings.BackgroundMode) {
		return errors.New("background_mode is invalid")
	}
	if !slices.Contains([]string{"inherit", "override", "hidden"}, settings.WebsiteMode) {
		return errors.New("website_mode is invalid")
	}
	if !slices.Contains([]string{"inherit", "override"}, settings.HighlightMode) {
		return errors.New("highlight_mode is invalid")
	}
	if settings.TitleMode == "override" && strings.TrimSpace(settings.Title) == "" {
		return errors.New("title is required in override mode")
	}
	if len(settings.Title) > 128 || len(settings.Description) > 1000 {
		return errors.New("server title or description is too long")
	}
	if settings.BannerMode == "override" {
		if strings.TrimSpace(settings.BannerURL) == "" {
			return errors.New("banner_url is required in override mode")
		}
		if err := ValidateIngameURL(strings.TrimSpace(settings.BannerURL)); err != nil {
			return fmt.Errorf("banner_url: %w", err)
		}
	}
	if settings.BackgroundMode == "override" {
		if strings.TrimSpace(settings.BackgroundURL) == "" {
			return errors.New("background_url is required in override mode")
		}
	}
	if err := ValidateIngameURL(strings.TrimSpace(settings.BackgroundURL)); err != nil {
		return fmt.Errorf("background_url: %w", err)
	}
	if settings.WebsiteMode == "override" {
		if strings.TrimSpace(settings.WebsiteURL) == "" {
			return errors.New("website_url is required in override mode")
		}
		if err := ValidateIngameURL(strings.TrimSpace(settings.WebsiteURL)); err != nil {
			return fmt.Errorf("website_url: %w", err)
		}
	}
	if settings.HighlightMode == "override" {
		return validateIngameMetrics(settings.HighlightMetrics)
	}
	return nil
}

func ValidateServerDocument(document store.ServerDocument) error {
	if err := ValidateIngameServerKey(document.ServerKey); err != nil {
		return err
	}
	if !slices.Contains([]string{store.IngameDocumentIntroduction, store.IngameDocumentCommands, store.IngameDocumentResources}, document.Key) {
		return errors.New("document key is invalid")
	}
	if !slices.Contains([]string{"inherit", "override", "hidden"}, document.Mode) {
		return errors.New("document mode is invalid")
	}
	if len(document.ContentMarkdown) > 102400 {
		return errors.New("content_markdown must not exceed 102400 bytes")
	}
	if document.Mode == "override" && strings.TrimSpace(document.ContentMarkdown) == "" {
		return errors.New("content_markdown is required in override mode")
	}
	return nil
}

func validateIngameMetrics(metrics [3]string) error {
	seen := make(map[string]struct{}, len(metrics))
	for _, key := range metrics {
		if _, ok := IngameMetric(key); !ok {
			return fmt.Errorf("unsupported in-game highlight metric %q", key)
		}
		if _, duplicate := seen[key]; duplicate {
			return errors.New("highlight metrics must be distinct")
		}
		seen[key] = struct{}{}
	}
	return nil
}

type ResolvedIngameAppearance struct {
	Title         string
	Description   string
	BannerURL     string
	BackgroundURL string
	WebsiteURL    string
}

type ResolvedIngameModules struct {
	ShowAnnouncements bool
	ShowPlayers       bool
	ShowHighlights    bool
	ShowServerIntro   bool
	ShowServerStatus  bool
}

type ResolvedIngameConfig struct {
	Appearance ResolvedIngameAppearance
	Modules    ResolvedIngameModules
	Metrics    [3]string
}

func ResolveIngameConfig(global store.IngameSettings, server store.IngameServerSettings, fallbackTitle string) ResolvedIngameConfig {
	resolved := ResolvedIngameConfig{
		Appearance: ResolvedIngameAppearance{
			Title: strings.TrimSpace(global.Title), Description: global.Description,
			BannerURL: strings.TrimSpace(global.BannerURL), BackgroundURL: strings.TrimSpace(global.BackgroundURL), WebsiteURL: strings.TrimSpace(global.WebsiteURL),
		},
		Modules: ResolvedIngameModules{
			ShowAnnouncements: global.ShowAnnouncements,
			ShowPlayers:       global.ShowPlayers,
			ShowHighlights:    global.ShowHighlights,
			ShowServerIntro:   global.ShowServerIntro,
			ShowServerStatus:  global.ShowServerStatus,
		},
		Metrics: global.HighlightMetrics,
	}
	if server.TitleMode == "override" {
		resolved.Appearance.Title = strings.TrimSpace(server.Title)
	}
	if resolved.Appearance.Title == "" {
		resolved.Appearance.Title = fallbackTitle
	}
	resolved.Appearance.Description = resolveMode(server.DescriptionMode, global.Description, server.Description)
	resolved.Appearance.BannerURL = resolveMode(server.BannerMode, global.BannerURL, server.BannerURL)
	resolved.Appearance.BackgroundURL = resolveMode(server.BackgroundMode, global.BackgroundURL, server.BackgroundURL)
	resolved.Appearance.WebsiteURL = resolveMode(server.WebsiteMode, global.WebsiteURL, server.WebsiteURL)
	if server.HighlightMode == "override" {
		resolved.Metrics = server.HighlightMetrics
	}
	return resolved
}

func resolveMode(mode, inherited, overridden string) string {
	switch mode {
	case "override":
		return strings.TrimSpace(overridden)
	case "hidden":
		return ""
	default:
		return strings.TrimSpace(inherited)
	}
}

func ResolveIngameDocument(server store.ServerDocument, site store.SiteDocument) (string, bool) {
	switch server.Mode {
	case "hidden":
		return "", false
	case "override":
		content := strings.TrimSpace(server.ContentMarkdown)
		return content, content != ""
	default:
		content := strings.TrimSpace(site.ContentMarkdown)
		return content, site.Enabled && content != ""
	}
}

func BuildExternalBrowserHref(value string) string {
	if ValidateIngameURL(value) != nil {
		return ""
	}
	return "steam://openurl_external/" + value
}
