package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
		ShowHighlights:   row.ShowHighlights == 1,
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
		ShowHighlights:   boolInt(settings.ShowHighlights),
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

func (s *dashboardStore) IngameServerSettings(ctx context.Context, serverID string) (IngameServerSettings, error) {
	row, err := s.q.GetIngameServerSettings(ctx, serverID)
	if errors.Is(err, sql.ErrNoRows) {
		return IngameServerSettings{
			ServerID: serverID, TitleMode: "inherit", DescriptionMode: "inherit",
			BannerMode: "inherit", BackgroundMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit",
		}, nil
	}
	if err != nil {
		return IngameServerSettings{}, fmt.Errorf("get server in-game settings: %w", err)
	}
	return IngameServerSettings{
		ServerID: row.ServerID, TitleMode: row.TitleMode, Title: row.Title,
		DescriptionMode: row.DescriptionMode, Description: row.Description,
		BannerMode: row.BannerMode, BannerURL: row.BannerUrl,
		BackgroundMode: row.BackgroundMode, BackgroundURL: row.BackgroundUrl,
		WebsiteMode: row.WebsiteMode, WebsiteURL: row.WebsiteUrl,
		HighlightMode:    row.HighlightMode,
		HighlightMetrics: [3]string{row.HighlightMetric1, row.HighlightMetric2, row.HighlightMetric3},
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) UpdateIngameServerSettings(ctx context.Context, settings IngameServerSettings) (IngameServerSettings, error) {
	settings.UpdatedAt = time.Now().Unix()
	err := s.q.UpsertIngameServerSettings(ctx, dashsql.UpsertIngameServerSettingsParams{
		ServerID: settings.ServerID, TitleMode: settings.TitleMode, Title: settings.Title,
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
	return s.IngameServerSettings(ctx, settings.ServerID)
}

func (s *dashboardStore) DeleteIngameServerSettings(ctx context.Context, serverID string) error {
	if err := s.q.DeleteIngameServerSettings(ctx, serverID); err != nil {
		return fmt.Errorf("delete server in-game settings: %w", err)
	}
	return nil
}

func (s *dashboardStore) ListServerDocuments(ctx context.Context, serverID string) ([]ServerDocument, error) {
	rows, err := s.q.ListServerDocuments(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("list server documents: %w", err)
	}
	documents := make([]ServerDocument, 0, len(rows))
	for _, row := range rows {
		documents = append(documents, ServerDocument{
			ServerID: row.ServerID, Key: row.Key, Mode: row.Mode,
			ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt,
		})
	}
	return documents, nil
}

func (s *dashboardStore) GetServerDocument(ctx context.Context, serverID, key string) (ServerDocument, error) {
	row, err := s.q.GetServerDocument(ctx, dashsql.GetServerDocumentParams{ServerID: serverID, Key: key})
	if errors.Is(err, sql.ErrNoRows) {
		return ServerDocument{ServerID: serverID, Key: key, Mode: "inherit"}, nil
	}
	if err != nil {
		return ServerDocument{}, fmt.Errorf("get server document: %w", err)
	}
	return ServerDocument{
		ServerID: row.ServerID, Key: row.Key, Mode: row.Mode,
		ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) UpdateServerDocument(ctx context.Context, document ServerDocument) (ServerDocument, error) {
	document.UpdatedAt = time.Now().Unix()
	err := s.q.UpsertServerDocument(ctx, dashsql.UpsertServerDocumentParams{
		ServerID: document.ServerID, Key: document.Key, Mode: document.Mode,
		ContentMarkdown: document.ContentMarkdown, UpdatedAt: document.UpdatedAt,
	})
	if err != nil {
		return ServerDocument{}, fmt.Errorf("update server document: %w", err)
	}
	return s.GetServerDocument(ctx, document.ServerID, document.Key)
}

func (s *dashboardStore) DeleteServerDocuments(ctx context.Context, serverID string) error {
	if err := s.q.DeleteServerDocuments(ctx, serverID); err != nil {
		return fmt.Errorf("delete server documents: %w", err)
	}
	return nil
}
