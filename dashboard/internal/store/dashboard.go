package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dashboarddb "github.com/gofurry/l4d2-plugin-stats/dashboard/database/dashboard"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	dashsql "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/dashboard"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type dashboardStore struct {
	db *sql.DB
	q  *dashsql.Queries
}

func OpenDashboard(ctx context.Context, path string) (DashboardStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create dashboard database directory: %w", err)
	}
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
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set dashboard migration dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate dashboard database: %w", err)
	}
	return &dashboardStore{db: db, q: dashsql.New(db)}, nil
}

func (s *dashboardStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *dashboardStore) MigrationVersion(ctx context.Context) (int64, error) {
	version, err := goose.GetDBVersionContext(ctx, s.db)
	return version, err
}

func (s *dashboardStore) Bootstrap(ctx context.Context, bootstrap config.BootstrapConfig, replace bool) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	if !replace {
		if _, err := q.GetMetadata(ctx, "bootstrap_applied"); err == nil {
			return false, nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("read bootstrap marker: %w", err)
		}
	}
	siteCount, err := q.CountSiteSettings(ctx)
	if err != nil {
		return false, fmt.Errorf("count site settings: %w", err)
	}
	serverCount, err := q.CountGameServers(ctx)
	if err != nil {
		return false, fmt.Errorf("count game servers: %w", err)
	}
	if !replace && (siteCount > 0 || serverCount > 0) {
		if err := q.UpsertMetadata(ctx, dashsql.UpsertMetadataParams{Key: "bootstrap_applied", Value: "existing"}); err != nil {
			return false, fmt.Errorf("write bootstrap marker: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit bootstrap marker: %w", err)
		}
		return false, nil
	}
	now := time.Now().Unix()
	{
		if err := q.UpsertSiteSettings(ctx, dashsql.UpsertSiteSettingsParams{
			Title: bootstrap.Site.Title, FooterText: bootstrap.Site.FooterText, UpdatedAt: now,
		}); err != nil {
			return false, fmt.Errorf("write site settings: %w", err)
		}
		if err := q.DeleteFooterLinks(ctx); err != nil {
			return false, fmt.Errorf("clear footer links: %w", err)
		}
		for i, link := range bootstrap.Site.FooterLinks {
			if err := q.CreateFooterLink(ctx, dashsql.CreateFooterLinkParams{
				Label: link.Label, Url: link.URL, SortOrder: int64(i),
				OpenNewTab: boolInt(link.OpenNewTab), Enabled: boolInt(link.Enabled),
				CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return false, fmt.Errorf("write footer link %d: %w", i, err)
			}
		}
	}
	{
		if err := q.DeleteGameServers(ctx); err != nil {
			return false, fmt.Errorf("clear game servers: %w", err)
		}
		for i, server := range bootstrap.Servers {
			if err := q.CreateGameServer(ctx, dashsql.CreateGameServerParams{
				ServerKey: server.ServerKey, DisplayName: server.DisplayName,
				ConnectAddress: server.ConnectAddress, QueryAddress: server.QueryAddress,
				IsPrimary: boolInt(server.Primary), Enabled: boolInt(server.Enabled),
				SortOrder: int64(server.SortOrder), CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return false, fmt.Errorf("write game server %d: %w", i, err)
			}
		}
	}
	if err := q.UpsertMetadata(ctx, dashsql.UpsertMetadataParams{Key: "bootstrap_applied", Value: fmt.Sprintf("%d", now)}); err != nil {
		return false, fmt.Errorf("write bootstrap marker: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit bootstrap transaction: %w", err)
	}
	return true, nil
}

func (s *dashboardStore) Site(ctx context.Context) (Site, error) {
	row, err := s.q.GetSiteSettings(ctx)
	if err != nil {
		return Site{}, fmt.Errorf("get site settings: %w", err)
	}
	rows, err := s.q.ListEnabledFooterLinks(ctx)
	if err != nil {
		return Site{}, fmt.Errorf("list footer links: %w", err)
	}
	links := make([]FooterLink, 0, len(rows))
	for _, link := range rows {
		links = append(links, FooterLink{Label: link.Label, URL: link.Url, OpenNewTab: link.OpenNewTab == 1})
	}
	return Site{Title: row.Title, FooterText: row.FooterText, Links: links}, nil
}

func (s *dashboardStore) PrimaryServer(ctx context.Context) (*GameServer, error) {
	row, err := s.q.GetPrimaryServer(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get primary server: %w", err)
	}
	return &GameServer{
		ID: row.ID, ServerKey: row.ServerKey, DisplayName: row.DisplayName,
		ConnectAddress: row.ConnectAddress, QueryAddress: row.QueryAddress,
		Primary: row.IsPrimary == 1, Enabled: row.Enabled == 1, SortOrder: row.SortOrder,
	}, nil
}

func (s *dashboardStore) Close() error { return s.db.Close() }

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
