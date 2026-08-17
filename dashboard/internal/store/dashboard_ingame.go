package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	dashsql "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/dashboard"
)

func (s *dashboardStore) IngameSettings(ctx context.Context) (IngameSettings, error) {
	row, err := s.q.GetIngameSettings(ctx)
	if err != nil {
		return IngameSettings{}, fmt.Errorf("get in-game settings: %w", err)
	}
	return IngameSettings{
		Enabled: row.Enabled == 1, Title: row.Title, Description: row.Description,
		BannerURL: row.BannerUrl, BackgroundURL: row.BackgroundUrl, WebsiteURL: row.WebsiteUrl,
		ShowAnnouncements: row.ShowAnnouncements == 1, ShowPlayers: row.ShowPlayers == 1,
		ShowHighlights: row.ShowHighlights == 1, ShowServerIntro: row.ShowServerIntro == 1,
		ShowServerStatus: row.ShowServerStatus == 1,
		HighlightMetrics: [3]string{row.HighlightMetric1, row.HighlightMetric2, row.HighlightMetric3},
		HomeCacheSeconds: row.HomeCacheSeconds, PlayerCacheSeconds: row.PlayerCacheSeconds,
		RankingCacheSeconds: row.RankingCacheSeconds, ContentCacheSeconds: row.ContentCacheSeconds,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) UpdateIngameSettings(ctx context.Context, settings IngameSettings) (IngameSettings, error) {
	settings.UpdatedAt = time.Now().Unix()
	err := s.q.UpsertIngameSettings(ctx, dashsql.UpsertIngameSettingsParams{
		Enabled: boolInt(settings.Enabled), Title: settings.Title, Description: settings.Description,
		BannerUrl: settings.BannerURL, BackgroundUrl: settings.BackgroundURL, WebsiteUrl: settings.WebsiteURL,
		ShowAnnouncements: boolInt(settings.ShowAnnouncements), ShowPlayers: boolInt(settings.ShowPlayers),
		ShowHighlights: boolInt(settings.ShowHighlights), ShowServerIntro: boolInt(settings.ShowServerIntro),
		ShowServerStatus: boolInt(settings.ShowServerStatus),
		HighlightMetric1: settings.HighlightMetrics[0], HighlightMetric2: settings.HighlightMetrics[1],
		HighlightMetric3: settings.HighlightMetrics[2], HomeCacheSeconds: settings.HomeCacheSeconds,
		PlayerCacheSeconds: settings.PlayerCacheSeconds, RankingCacheSeconds: settings.RankingCacheSeconds,
		ContentCacheSeconds: settings.ContentCacheSeconds, UpdatedAt: settings.UpdatedAt,
	})
	if err != nil {
		return IngameSettings{}, fmt.Errorf("update in-game settings: %w", err)
	}
	return s.IngameSettings(ctx)
}

func (s *dashboardStore) IngameServerSettings(ctx context.Context, serverKey string) (IngameServerSettings, error) {
	row, err := s.q.GetIngameServerSettings(ctx, serverKey)
	if errors.Is(err, sql.ErrNoRows) {
		return IngameServerSettings{
			ServerKey: serverKey, TitleMode: "inherit", DescriptionMode: "inherit",
			BannerMode: "inherit", BackgroundMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit",
		}, nil
	}
	if err != nil {
		return IngameServerSettings{}, fmt.Errorf("get server in-game settings: %w", err)
	}
	return IngameServerSettings{
		ServerKey: row.ServerKey, TitleMode: row.TitleMode, Title: row.Title,
		DescriptionMode: row.DescriptionMode, Description: row.Description,
		BannerMode: row.BannerMode, BannerURL: row.BannerUrl,
		BackgroundMode: row.BackgroundMode, BackgroundURL: row.BackgroundUrl,
		WebsiteMode: row.WebsiteMode, WebsiteURL: row.WebsiteUrl,
		HighlightMode:    row.HighlightMode,
		HighlightMetrics: [3]string{row.HighlightMetric1, row.HighlightMetric2, row.HighlightMetric3},
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) ListIngameServerSettings(ctx context.Context) ([]IngameServerSettings, error) {
	rows, err := s.q.ListIngameServerSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list server-group in-game settings: %w", err)
	}
	result := make([]IngameServerSettings, 0, len(rows))
	for _, row := range rows {
		result = append(result, IngameServerSettings{
			ServerKey: row.ServerKey, TitleMode: row.TitleMode, Title: row.Title,
			DescriptionMode: row.DescriptionMode, Description: row.Description,
			BannerMode: row.BannerMode, BannerURL: row.BannerUrl,
			BackgroundMode: row.BackgroundMode, BackgroundURL: row.BackgroundUrl,
			WebsiteMode: row.WebsiteMode, WebsiteURL: row.WebsiteUrl,
			HighlightMode:    row.HighlightMode,
			HighlightMetrics: [3]string{row.HighlightMetric1, row.HighlightMetric2, row.HighlightMetric3},
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return result, nil
}

func (s *dashboardStore) UpdateIngameServerSettings(ctx context.Context, settings IngameServerSettings) (IngameServerSettings, error) {
	settings.UpdatedAt = time.Now().Unix()
	err := s.q.UpsertIngameServerSettings(ctx, dashsql.UpsertIngameServerSettingsParams{
		ServerKey: settings.ServerKey, TitleMode: settings.TitleMode, Title: settings.Title,
		DescriptionMode: settings.DescriptionMode, Description: settings.Description,
		BannerMode: settings.BannerMode, BannerUrl: settings.BannerURL,
		BackgroundMode: settings.BackgroundMode, BackgroundUrl: settings.BackgroundURL,
		WebsiteMode: settings.WebsiteMode, WebsiteUrl: settings.WebsiteURL,
		HighlightMode: settings.HighlightMode, HighlightMetric1: settings.HighlightMetrics[0],
		HighlightMetric2: settings.HighlightMetrics[1], HighlightMetric3: settings.HighlightMetrics[2],
		UpdatedAt: settings.UpdatedAt,
	})
	if err != nil {
		return IngameServerSettings{}, fmt.Errorf("update server in-game settings: %w", err)
	}
	return s.IngameServerSettings(ctx, settings.ServerKey)
}

func (s *dashboardStore) DeleteIngameServerSettings(ctx context.Context, serverKey string) error {
	if err := s.q.DeleteIngameServerSettings(ctx, serverKey); err != nil {
		return fmt.Errorf("delete server in-game settings: %w", err)
	}
	return nil
}

func (s *dashboardStore) ListServerDocuments(ctx context.Context, serverKey string) ([]ServerDocument, error) {
	rows, err := s.q.ListServerDocuments(ctx, serverKey)
	if err != nil {
		return nil, fmt.Errorf("list server documents: %w", err)
	}
	documents := make([]ServerDocument, 0, len(rows))
	for _, row := range rows {
		documents = append(documents, ServerDocument{
			ServerKey: row.ServerKey, Key: row.Key, Mode: row.Mode,
			ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt,
		})
	}
	return documents, nil
}

func (s *dashboardStore) GetServerDocument(ctx context.Context, serverKey, key string) (ServerDocument, error) {
	row, err := s.q.GetServerDocument(ctx, dashsql.GetServerDocumentParams{ServerKey: serverKey, Key: key})
	if errors.Is(err, sql.ErrNoRows) {
		return ServerDocument{ServerKey: serverKey, Key: key, Mode: "inherit"}, nil
	}
	if err != nil {
		return ServerDocument{}, fmt.Errorf("get server document: %w", err)
	}
	return ServerDocument{
		ServerKey: row.ServerKey, Key: row.Key, Mode: row.Mode,
		ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) UpdateServerDocument(ctx context.Context, document ServerDocument) (ServerDocument, error) {
	document.UpdatedAt = time.Now().Unix()
	err := s.q.UpsertServerDocument(ctx, dashsql.UpsertServerDocumentParams{
		ServerKey: document.ServerKey, Key: document.Key, Mode: document.Mode,
		ContentMarkdown: document.ContentMarkdown, UpdatedAt: document.UpdatedAt,
	})
	if err != nil {
		return ServerDocument{}, fmt.Errorf("update server document: %w", err)
	}
	return s.GetServerDocument(ctx, document.ServerKey, document.Key)
}

func (s *dashboardStore) DeleteServerDocuments(ctx context.Context, serverKey string) error {
	if err := s.q.DeleteServerDocuments(ctx, serverKey); err != nil {
		return fmt.Errorf("delete server documents: %w", err)
	}
	return nil
}

func (s *dashboardStore) ListServerQuickLinks(ctx context.Context, serverKey string) ([]IngameQuickLink, error) {
	rows, err := s.q.ListIngameQuickLinks(ctx, serverKey)
	if err != nil {
		return nil, fmt.Errorf("list server-group quick links: %w", err)
	}
	links := make([]IngameQuickLink, 0, len(rows))
	for _, row := range rows {
		links = append(links, IngameQuickLink{
			ServerKey: row.ServerKey, Label: row.Label, URL: row.Url,
			SortOrder: row.SortOrder, Enabled: row.Enabled == 1,
		})
	}
	return links, nil
}

func (s *dashboardStore) ReplaceServerQuickLinks(ctx context.Context, serverKey string, links []IngameQuickLink) ([]IngameQuickLink, error) {
	if err := validateQuickLinkRows(serverKey, links); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin server-group quick-link update: %w", err)
	}
	defer tx.Rollback()
	queries := s.q.WithTx(tx)
	if err := queries.DeleteIngameQuickLinks(ctx, serverKey); err != nil {
		return nil, fmt.Errorf("delete server-group quick links: %w", err)
	}
	for _, link := range links {
		if err := queries.InsertIngameQuickLink(ctx, dashsql.InsertIngameQuickLinkParams{
			ServerKey: serverKey, Label: link.Label, Url: link.URL,
			SortOrder: link.SortOrder, Enabled: boolInt(link.Enabled),
		}); err != nil {
			return nil, fmt.Errorf("insert server-group quick link: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit server-group quick links: %w", err)
	}
	return s.ListServerQuickLinks(ctx, serverKey)
}

func validateQuickLinkRows(serverKey string, links []IngameQuickLink) error {
	if len(links) > 8 {
		return errors.New("at most 8 server-group quick links are allowed")
	}
	orders := make(map[int64]struct{}, len(links))
	for _, link := range links {
		if link.ServerKey != "" && link.ServerKey != serverKey {
			return errors.New("quick-link server key does not match its group")
		}
		label := strings.TrimSpace(link.Label)
		if count := utf8.RuneCountInString(label); count < 1 || count > 32 {
			return errors.New("quick-link label must contain 1 to 32 characters")
		}
		value := strings.TrimSpace(link.URL)
		parsed, err := url.Parse(value)
		if err != nil || len(value) > 2048 || parsed.Host == "" || parsed.User != nil || parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("quick-link URL must be an absolute credential-free HTTP or HTTPS URL up to 2048 bytes")
		}
		if link.SortOrder < 0 || link.SortOrder > 7 {
			return errors.New("quick-link sort order must be between 0 and 7")
		}
		if _, exists := orders[link.SortOrder]; exists {
			return errors.New("quick-link sort orders must be unique")
		}
		orders[link.SortOrder] = struct{}{}
	}
	return nil
}
