package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dashboarddb "github.com/gofurry/l4d2-plugin-stats/dashboard/database/dashboard"
	dashsql "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/dashboard"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

const defaultSiteLanguage = "zh-CN"
const defaultBrowserTitle = "L4D2 Stats"

type dashboardStore struct {
	db   *sql.DB
	q    *dashsql.Queries
	path string
}

func OpenDashboard(ctx context.Context, path string) (DashboardDatabase, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create dashboard database directory: %w", err)
	}
	_, statErr := os.Stat(path)
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open dashboard database: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping dashboard database: %w", err)
	}
	if errors.Is(statErr, os.ErrNotExist) {
		_ = os.Chmod(path, 0o600)
	}
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set dashboard migration dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate dashboard database: %w", err)
	}
	return &dashboardStore{db: db, q: dashsql.New(db), path: path}, nil
}

func (s *dashboardStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *dashboardStore) MigrationVersion(ctx context.Context) (int64, error) {
	return goose.GetDBVersionContext(ctx, s.db)
}

func (s *dashboardStore) AggregateStatus(ctx context.Context) (AggregateStatus, error) {
	row, err := s.q.GetAggregateStatus(ctx)
	if err != nil {
		return AggregateStatus{}, fmt.Errorf("get aggregate status: %w", err)
	}
	return AggregateStatus{
		State: row.State, LastStartedAt: row.LastStartedAt, LastFinishedAt: row.LastFinishedAt,
		SourceRows: row.SourceRows, AggregateRows: row.AggregateRows, LastError: row.LastError,
		SourceWatermark: row.SourceWatermark, LastDurationMS: row.LastDurationMs,
		LastChangedDays: row.LastChangedDays, LastBuildMode: row.LastBuildMode,
	}, nil
}

func (s *dashboardStore) ReplaceAggregateRows(ctx context.Context, rows []AggregateRow, sourceRows int64) error {
	return s.ApplyAggregateChanges(ctx, AggregateChangeSet{
		Rows: rows, SourceRows: sourceRows, SourceWatermark: time.Now().Unix(), Full: true,
	})
}

func (s *dashboardStore) ApplyAggregateChanges(ctx context.Context, change AggregateChangeSet) error {
	startedAt := time.Now()
	started := startedAt.Unix()
	if err := s.q.MarkAggregateStarted(ctx, started); err != nil {
		return fmt.Errorf("mark aggregate build started: %w", err)
	}
	fail := func(err error) error {
		_ = s.q.MarkAggregateFailed(context.Background(), err.Error())
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fail(fmt.Errorf("begin aggregate transaction: %w", err))
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	if change.Full {
		if err := q.DeleteAggregateRows(ctx); err != nil {
			return fail(fmt.Errorf("clear aggregate rows: %w", err))
		}
	} else {
		for _, day := range change.Days {
			if err := q.DeleteAggregateRowsForDay(ctx, day); err != nil {
				return fail(fmt.Errorf("clear aggregate day %d: %w", day, err))
			}
		}
	}
	for _, row := range change.Rows {
		metrics, err := json.Marshal(row.Metrics)
		if err != nil {
			return fail(fmt.Errorf("encode aggregate metrics: %w", err))
		}
		if err := q.InsertAggregateRow(ctx, dashsql.InsertAggregateRowParams{
			Kind: row.Kind, Day: row.Day, ServerKey: row.ServerKey, SteamID: row.SteamID,
			Mode: row.Mode, Dimension: row.Dimension, MetricsJson: string(metrics),
		}); err != nil {
			return fail(fmt.Errorf("insert aggregate row: %w", err))
		}
	}
	aggregateRows, err := q.CountAggregateRows(ctx)
	if err != nil {
		return fail(fmt.Errorf("count aggregate rows: %w", err))
	}
	mode := "incremental"
	if change.Full {
		mode = "full"
	}
	if err := q.CompleteAggregateBuild(ctx, dashsql.CompleteAggregateBuildParams{
		LastFinishedAt: time.Now().Unix(), SourceRows: change.SourceRows, AggregateRows: aggregateRows,
		SourceWatermark: change.SourceWatermark, LastDurationMs: time.Since(startedAt).Milliseconds(),
		LastChangedDays: int64(len(change.Days)), LastBuildMode: mode,
	}); err != nil {
		return fail(fmt.Errorf("complete aggregate build: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return fail(fmt.Errorf("commit aggregate build: %w", err))
	}
	return nil
}

func (s *dashboardStore) DataMaintenanceSettings(ctx context.Context) (DataMaintenanceSettings, error) {
	row, err := s.q.GetDataMaintenanceSettings(ctx)
	if err != nil {
		return DataMaintenanceSettings{}, fmt.Errorf("get data maintenance settings: %w", err)
	}
	return DataMaintenanceSettings{
		AggregateIntervalMinutes: row.AggregateIntervalMinutes,
		DetailRetentionDays:      row.DetailRetentionDays,
		SessionRetentionDays:     row.SessionRetentionDays,
		ResultRetentionDays:      row.ResultRetentionDays,
		UpdatedAt:                row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) UpdateDataMaintenanceSettings(ctx context.Context, value DataMaintenanceSettings) error {
	value.UpdatedAt = time.Now().Unix()
	return s.q.UpdateDataMaintenanceSettings(ctx, dashsql.UpdateDataMaintenanceSettingsParams{
		AggregateIntervalMinutes: value.AggregateIntervalMinutes,
		DetailRetentionDays:      value.DetailRetentionDays,
		SessionRetentionDays:     value.SessionRetentionDays,
		ResultRetentionDays:      value.ResultRetentionDays,
		UpdatedAt:                value.UpdatedAt,
	})
}

func (s *dashboardStore) DatabaseUsage(ctx context.Context) (DatabaseUsage, error) {
	var pageCount, pageSize int64
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return DatabaseUsage{}, err
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return DatabaseUsage{}, err
	}
	var walBytes int64
	if info, err := os.Stat(s.path + "-wal"); err == nil {
		walBytes = info.Size()
	}
	return DatabaseUsage{Driver: "sqlite", Bytes: pageCount * pageSize, WALBytes: walBytes}, nil
}

func (s *dashboardStore) RecordRetentionRun(ctx context.Context, plan RetentionPlan, result RetentionResult) error {
	return s.q.CreateRetentionRun(ctx, dashsql.CreateRetentionRunParams{
		ID: result.RunID, ExecutedAt: result.ExecutedAt, SourceWatermark: plan.SourceWatermark,
		DetailCutoff: plan.DetailCutoff, SessionCutoff: plan.SessionCutoff, ResultCutoff: plan.ResultCutoff,
		EquipmentRows: result.EquipmentRows, VersusClassRows: result.VersusClassRows,
		SessionRows: result.SessionRows, VersusRoundResultRows: result.VersusRoundResultRows,
		VersusRunResultRows: result.VersusRunResultRows,
	})
}

func (s *dashboardStore) RetentionRunCount(ctx context.Context) (int64, error) {
	return s.q.CountRetentionRuns(ctx)
}

func (s *dashboardStore) ListAggregateRows(ctx context.Context, filter AggregateFilter) ([]AggregateRow, error) {
	rows, err := s.q.ListAggregateRows(ctx, dashsql.ListAggregateRowsParams{
		Column1: filter.SteamID, Column2: filter.ServerKey, Column3: filter.Mode, Column4: filter.CutoffDay,
	})
	if err != nil {
		return nil, fmt.Errorf("list aggregate rows: %w", err)
	}
	kinds := make(map[string]struct{}, len(filter.Kinds))
	for _, kind := range filter.Kinds {
		kinds[kind] = struct{}{}
	}
	result := make([]AggregateRow, 0, len(rows))
	for _, row := range rows {
		if len(kinds) > 0 {
			if _, ok := kinds[row.Kind]; !ok {
				continue
			}
		}
		metrics := make(map[string]int64)
		if err := json.Unmarshal([]byte(row.MetricsJson), &metrics); err != nil {
			return nil, fmt.Errorf("decode aggregate row %s/%d/%s: %w", row.Kind, row.Day, row.SteamID, err)
		}
		result = append(result, AggregateRow{
			Kind: row.Kind, Day: row.Day, ServerKey: row.ServerKey, SteamID: row.SteamID,
			Mode: row.Mode, Dimension: row.Dimension, Metrics: metrics,
		})
	}
	return result, nil
}

func (s *dashboardStore) Site(ctx context.Context) (Site, error) {
	row, err := s.q.GetSiteSettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return Site{Language: defaultSiteLanguage, BrowserTitle: defaultBrowserTitle, Theme: "light", A2SRefreshSeconds: 30, Links: []FooterLink{}, Documents: []string{}, Configured: false}, nil
	}
	if err != nil {
		return Site{}, fmt.Errorf("get site settings: %w", err)
	}
	rows, err := s.q.ListPublicFooterLinks(ctx)
	if err != nil {
		return Site{}, fmt.Errorf("list footer links: %w", err)
	}
	links := make([]FooterLink, 0, len(rows))
	for _, link := range rows {
		links = append(links, FooterLink{Label: link.Label, URL: link.Url})
	}
	documentRows, err := s.q.ListPublicSiteDocuments(ctx)
	if err != nil {
		return Site{}, fmt.Errorf("list public site documents: %w", err)
	}
	documents := make([]string, 0, len(documentRows))
	for _, document := range documentRows {
		documents = append(documents, document)
	}
	return Site{
		Language: row.Language, BrowserTitle: row.BrowserTitle, Theme: row.Theme, FooterEnabled: row.FooterEnabled == 1,
		BackgroundImageURL: row.BackgroundImageUrl, Links: links,
		SteamOpenIDEnabled: row.SteamOpenidEnabled == 1 && row.PublicOrigin != "",
		A2SRefreshSeconds:  row.A2sRefreshSeconds, Documents: documents, Configured: true,
	}, nil
}

func (s *dashboardStore) SiteSettings(ctx context.Context) (SiteSettings, error) {
	settings := SiteSettings{Language: defaultSiteLanguage, BrowserTitle: defaultBrowserTitle, Theme: "light", A2SRefreshSeconds: 30, A2SJitterSeconds: 2, A2SRetryCount: 1, Links: []FooterLink{}}
	row, err := s.q.GetSiteSettings(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return SiteSettings{}, fmt.Errorf("get site settings: %w", err)
	}
	if err == nil {
		settings.Language = row.Language
		settings.BrowserTitle = row.BrowserTitle
		settings.Theme = row.Theme
		settings.FooterEnabled = row.FooterEnabled == 1
		settings.BackgroundImageURL = row.BackgroundImageUrl
		settings.PublicOrigin = row.PublicOrigin
		settings.SteamOpenIDEnabled = row.SteamOpenidEnabled == 1
		settings.A2SRefreshSeconds = row.A2sRefreshSeconds
		settings.A2SJitterSeconds = row.A2sJitterSeconds
		settings.A2SRetryCount = row.A2sRetryCount
	}
	seo, err := s.q.GetSEOSettings(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return SiteSettings{}, fmt.Errorf("get SEO settings: %w", err)
	}
	if err == nil {
		settings.SEOEnabled = seo.Enabled == 1
		settings.SEODescription = seo.Description
		settings.SEOImageURL = seo.ImageUrl
	}
	links, err := s.q.ListFooterLinks(ctx)
	if err != nil {
		return SiteSettings{}, fmt.Errorf("list footer links: %w", err)
	}
	for _, link := range links {
		settings.Links = append(settings.Links, FooterLink{ID: link.ID, Label: link.Label, URL: link.Url})
	}
	return settings, nil
}

func (s *dashboardStore) UpdateSite(ctx context.Context, settings SiteSettings) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin site transaction: %w", err)
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	now := time.Now().Unix()
	if err := q.UpsertSiteSettings(ctx, dashsql.UpsertSiteSettingsParams{
		Language: settings.Language, FooterEnabled: boolInt(settings.FooterEnabled), BackgroundImageUrl: settings.BackgroundImageURL, PublicOrigin: settings.PublicOrigin,
		SteamOpenidEnabled: boolInt(settings.SteamOpenIDEnabled), BrowserTitle: settings.BrowserTitle, Theme: settings.Theme,
		A2sRefreshSeconds: settings.A2SRefreshSeconds, A2sJitterSeconds: settings.A2SJitterSeconds,
		A2sRetryCount: settings.A2SRetryCount, UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("update site settings: %w", err)
	}
	if err := q.UpsertSEOSettings(ctx, dashsql.UpsertSEOSettingsParams{
		Enabled: boolInt(settings.SEOEnabled), Description: settings.SEODescription,
		ImageUrl: settings.SEOImageURL, UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("update SEO settings: %w", err)
	}
	if err := q.DeleteFooterLinks(ctx); err != nil {
		return fmt.Errorf("clear footer links: %w", err)
	}
	for i, link := range settings.Links {
		if err := q.CreateFooterLink(ctx, dashsql.CreateFooterLinkParams{
			ID: uuid.NewString(), Label: link.Label, Url: link.URL, SortOrder: int64(i), CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return fmt.Errorf("create footer link %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit site transaction: %w", err)
	}
	return nil
}

func (s *dashboardStore) ListSiteDocuments(ctx context.Context, publicOnly bool) ([]SiteDocument, error) {
	if publicOnly {
		rows, err := s.q.ListPublicSiteDocuments(ctx)
		if err != nil {
			return nil, fmt.Errorf("list public site documents: %w", err)
		}
		result := make([]SiteDocument, 0, len(rows))
		for _, key := range rows {
			document, err := s.GetSiteDocument(ctx, key, true)
			if err != nil {
				return nil, err
			}
			result = append(result, document)
		}
		return result, nil
	}
	rows, err := s.q.ListSiteDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list site documents: %w", err)
	}
	result := make([]SiteDocument, 0, len(rows))
	for _, row := range rows {
		result = append(result, SiteDocument{Key: row.Key, Enabled: row.Enabled == 1, ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt})
	}
	return result, nil
}

func (s *dashboardStore) GetSiteDocument(ctx context.Context, key string, publicOnly bool) (SiteDocument, error) {
	if publicOnly {
		row, err := s.q.GetPublicSiteDocument(ctx, key)
		if err != nil {
			return SiteDocument{}, err
		}
		return SiteDocument{Key: row.Key, Enabled: row.Enabled == 1, ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt}, nil
	}
	row, err := s.q.GetSiteDocument(ctx, key)
	if err != nil {
		return SiteDocument{}, err
	}
	return SiteDocument{Key: row.Key, Enabled: row.Enabled == 1, ContentMarkdown: row.ContentMarkdown, UpdatedAt: row.UpdatedAt}, nil
}

func (s *dashboardStore) UpdateSiteDocument(ctx context.Context, document SiteDocument) (SiteDocument, error) {
	rows, err := s.q.UpdateSiteDocument(ctx, dashsql.UpdateSiteDocumentParams{
		Key: document.Key, Enabled: boolInt(document.Enabled), ContentMarkdown: document.ContentMarkdown, UpdatedAt: time.Now().Unix(),
	})
	if err != nil {
		return SiteDocument{}, fmt.Errorf("update site document: %w", err)
	}
	if rows == 0 {
		return SiteDocument{}, sql.ErrNoRows
	}
	return s.GetSiteDocument(ctx, document.Key, false)
}

func (s *dashboardStore) ListServers(ctx context.Context) ([]GameServer, error) {
	rows, err := s.q.ListGameServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list game servers: %w", err)
	}
	servers := make([]GameServer, 0, len(rows))
	for _, row := range rows {
		servers = append(servers, gameServer(row.ID, row.DisplayName, row.Address, row.Enabled, row.SortOrder))
	}
	return servers, nil
}

func (s *dashboardStore) CreateServer(ctx context.Context, server GameServer) (GameServer, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return GameServer{}, fmt.Errorf("begin game server transaction: %w", err)
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	sortOrder, err := q.NextGameServerSortOrder(ctx)
	if err != nil {
		return GameServer{}, fmt.Errorf("get next game server order: %w", err)
	}
	now := time.Now().Unix()
	server.ID = uuid.NewString()
	server.Enabled = true
	server.SortOrder = sortOrder
	if err := q.CreateGameServer(ctx, dashsql.CreateGameServerParams{
		ID: server.ID, DisplayName: server.DisplayName, Address: server.Address,
		SortOrder: sortOrder, CreatedAt: now,
	}); err != nil {
		return GameServer{}, fmt.Errorf("create game server: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return GameServer{}, fmt.Errorf("commit game server transaction: %w", err)
	}
	return server, nil
}

func (s *dashboardStore) UpdateServer(ctx context.Context, server GameServer) error {
	now := time.Now().Unix()
	rows, err := s.q.UpdateGameServer(ctx, dashsql.UpdateGameServerParams{
		ID: server.ID, DisplayName: server.DisplayName, Address: server.Address, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("update game server: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *dashboardStore) SetServerEnabled(ctx context.Context, id string, enabled bool) error {
	rows, err := s.q.SetGameServerEnabled(ctx, dashsql.SetGameServerEnabledParams{ID: id, Enabled: boolInt(enabled), UpdatedAt: time.Now().Unix()})
	if err != nil {
		return fmt.Errorf("set game server enabled: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *dashboardStore) MoveServer(ctx context.Context, id, direction string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin game server move: %w", err)
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	rows, err := q.ListGameServers(ctx)
	if err != nil {
		return fmt.Errorf("list game servers for move: %w", err)
	}
	index := -1
	for i := range rows {
		if rows[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return sql.ErrNoRows
	}
	target := index - 1
	if direction == "down" {
		target = index + 1
	}
	if target < 0 || target >= len(rows) {
		return nil
	}
	now := time.Now().Unix()
	if _, err := q.SetGameServerSortOrder(ctx, dashsql.SetGameServerSortOrderParams{ID: rows[index].ID, SortOrder: rows[target].SortOrder, UpdatedAt: now}); err != nil {
		return fmt.Errorf("move game server: %w", err)
	}
	if _, err := q.SetGameServerSortOrder(ctx, dashsql.SetGameServerSortOrderParams{ID: rows[target].ID, SortOrder: rows[index].SortOrder, UpdatedAt: now}); err != nil {
		return fmt.Errorf("move adjacent game server: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit game server move: %w", err)
	}
	return nil
}

func (s *dashboardStore) DeleteServer(ctx context.Context, id string) error {
	rows, err := s.q.DeleteGameServer(ctx, id)
	if err != nil {
		return fmt.Errorf("delete game server: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *dashboardStore) AdminConfigured(ctx context.Context) (bool, error) {
	count, err := s.q.CountAdminAccounts(ctx)
	return count > 0, err
}

func (s *dashboardStore) CreateAdmin(ctx context.Context, username, passwordHash, jwtSecret string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	count, err := q.CountAdminAccounts(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("administrator is already configured")
	}
	now := time.Now().Unix()
	if err := q.CreateAdminAccount(ctx, dashsql.CreateAdminAccountParams{
		Username: username, PasswordHash: passwordHash, JwtSecret: jwtSecret, CreatedAt: now,
	}); err != nil {
		return fmt.Errorf("create administrator: %w", err)
	}
	return tx.Commit()
}

func (s *dashboardStore) Admin(ctx context.Context) (*AdminAccount, error) {
	row, err := s.q.GetAdminAccount(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &AdminAccount{
		Username: row.Username, PasswordHash: row.PasswordHash, JWTSecret: row.JwtSecret,
		TokenVersion: row.TokenVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		PasswordChangedAt: row.PasswordChangedAt,
	}, nil
}

func (s *dashboardStore) UpdateAdminUsername(ctx context.Context, username string) error {
	return s.q.UpdateAdminUsername(ctx, dashsql.UpdateAdminUsernameParams{Username: username, UpdatedAt: time.Now().Unix()})
}

func (s *dashboardStore) UpdateAdminPassword(ctx context.Context, passwordHash string) error {
	now := time.Now().Unix()
	return s.q.UpdateAdminPassword(ctx, dashsql.UpdateAdminPasswordParams{PasswordHash: passwordHash, UpdatedAt: now})
}

func (s *dashboardStore) ListAnnouncements(ctx context.Context, filter AnnouncementFilter) (AnnouncementPage, error) {
	rows, err := s.q.ListAnnouncements(ctx, dashsql.ListAnnouncementsParams{
		TitleFilter: filter.Title, YearFilter: filter.Year,
		RowLimit: int64(filter.Limit), RowOffset: int64(filter.Offset),
	})
	if err != nil {
		return AnnouncementPage{}, fmt.Errorf("list announcements: %w", err)
	}
	total, err := s.q.CountAnnouncements(ctx, dashsql.CountAnnouncementsParams{TitleFilter: filter.Title, YearFilter: filter.Year})
	if err != nil {
		return AnnouncementPage{}, fmt.Errorf("count announcements: %w", err)
	}
	items := make([]Announcement, 0, len(rows))
	for _, row := range rows {
		items = append(items, announcement(row))
	}
	page := 1
	if filter.Limit > 0 {
		page = int(filter.Offset/filter.Limit) + 1
	}
	return AnnouncementPage{Items: items, Total: total, Page: page, Limit: int(filter.Limit)}, nil
}

func (s *dashboardStore) ListAnnouncementYears(ctx context.Context) ([]int, error) {
	rows, err := s.q.ListAnnouncementYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("list announcement years: %w", err)
	}
	years := make([]int, 0, len(rows))
	for _, year := range rows {
		years = append(years, int(year))
	}
	return years, nil
}

func (s *dashboardStore) GetAnnouncement(ctx context.Context, id string) (Announcement, error) {
	row, err := s.q.GetAnnouncement(ctx, id)
	if err != nil {
		return Announcement{}, err
	}
	return announcement(row), nil
}

func (s *dashboardStore) CreateAnnouncement(ctx context.Context, value Announcement) (Announcement, error) {
	now := time.Now().Unix()
	value.ID = uuid.NewString()
	value.CreatedAt = now
	value.UpdatedAt = now
	if err := s.q.CreateAnnouncement(ctx, dashsql.CreateAnnouncementParams{
		ID: value.ID, Title: value.Title, ContentMarkdown: value.ContentMarkdown, CreatedAt: now,
	}); err != nil {
		return Announcement{}, fmt.Errorf("create announcement: %w", err)
	}
	return value, nil
}

func (s *dashboardStore) UpdateAnnouncement(ctx context.Context, value Announcement) (Announcement, error) {
	now := time.Now().Unix()
	rows, err := s.q.UpdateAnnouncement(ctx, dashsql.UpdateAnnouncementParams{
		ID: value.ID, Title: value.Title, ContentMarkdown: value.ContentMarkdown, UpdatedAt: now,
	})
	if err != nil {
		return Announcement{}, fmt.Errorf("update announcement: %w", err)
	}
	if rows == 0 {
		return Announcement{}, sql.ErrNoRows
	}
	return s.GetAnnouncement(ctx, value.ID)
}

func (s *dashboardStore) DeleteAnnouncement(ctx context.Context, id string) error {
	rows, err := s.q.DeleteAnnouncement(ctx, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *dashboardStore) Close() error { return s.db.Close() }

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func gameServer(id, name, address string, enabled, sort int64) GameServer {
	return GameServer{ID: id, DisplayName: name, Address: address, Enabled: enabled == 1, SortOrder: sort}
}

func announcement(row dashsql.Announcement) Announcement {
	return Announcement{
		ID: row.ID, Title: row.Title, ContentMarkdown: row.ContentMarkdown,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}
