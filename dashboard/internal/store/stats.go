package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	statsmysql "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/stats_mysql"
	statspg "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/stats_postgres"
	statssqlite "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/stats_sqlite"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type statsStore struct {
	db      *sql.DB
	driver  string
	timeout time.Duration
	sqlite  *statssqlite.Queries
	mysql   *statsmysql.Queries
	pg      *statspg.Queries
}

func OpenStats(ctx context.Context, cfg config.StatsDatabaseConfig) (StatsStore, error) {
	driver := strings.ToLower(cfg.Driver)
	dsn := cfg.DSN
	sqlDriver := driver
	switch driver {
	case "sqlite":
		sqlDriver = "sqlite"
		if !strings.HasPrefix(dsn, "file:") {
			dsn = "file:" + filepath.ToSlash(dsn) + "?mode=ro"
		} else if !strings.Contains(dsn, "mode=") {
			separator := "?"
			if strings.Contains(dsn, "?") {
				separator = "&"
			}
			dsn += separator + "mode=ro"
		}
	case "mysql":
		sqlDriver = "mysql"
	case "pgsql", "postgres", "postgresql":
		driver = "postgres"
		sqlDriver = "pgx"
	default:
		return nil, fmt.Errorf("unsupported stats database driver %q", cfg.Driver)
	}
	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open stats database: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime.Value())
	pingCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout.Value())
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping stats database: %w", err)
	}
	s := &statsStore{db: db, driver: driver, timeout: cfg.QueryTimeout.Value()}
	switch driver {
	case "sqlite":
		s.sqlite = statssqlite.New(db)
	case "mysql":
		s.mysql = statsmysql.New(db)
	case "postgres":
		s.pg = statspg.New(db)
	}
	return s, nil
}

func (s *statsStore) Ping(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.db.PingContext(queryCtx)
}

func (s *statsStore) SchemaVersion(ctx context.Context) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	switch s.driver {
	case "sqlite":
		return s.sqlite.GetSchemaVersion(queryCtx)
	case "mysql":
		return s.mysql.GetSchemaVersion(queryCtx)
	default:
		return s.pg.GetSchemaVersion(queryCtx)
	}
}

func (s *statsStore) Overview(ctx context.Context, activeSince time.Time) (Overview, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result := Overview{Generated: time.Now().UTC()}
	cutoff := activeSince.Unix()
	switch s.driver {
	case "sqlite":
		core, err := s.sqlite.GetCoreOverview(queryCtx, cutoff)
		if err != nil {
			return Overview{}, fmt.Errorf("query core overview: %w", err)
		}
		pve, err := s.sqlite.GetPVEOverview(queryCtx)
		if err != nil {
			return Overview{}, fmt.Errorf("query pve overview: %w", err)
		}
		versus, err := s.sqlite.GetVersusOverview(queryCtx)
		if err != nil {
			return Overview{}, fmt.Errorf("query versus overview: %w", err)
		}
		result.Core = CoreOverview{core.TotalPlayers, core.ActivePlayers7d, core.TotalActivePlaySeconds, core.CompletedPveRuns, core.CompletedVersusRuns}
		result.PVE = PVEOverview{pve.CommonKills, pve.SpecialKills, pve.TankKills, pve.WitchKills, pve.Rescues}
		result.Versus = VersusOverview{versus.CompletedMatches, versus.CompletedHalves, versus.HumanControlledInfectedKills, versus.HumanSurvivorControls}
	case "mysql":
		core, err := s.mysql.GetCoreOverview(queryCtx, cutoff)
		if err != nil {
			return Overview{}, fmt.Errorf("query core overview: %w", err)
		}
		pve, err := s.mysql.GetPVEOverview(queryCtx)
		if err != nil {
			return Overview{}, fmt.Errorf("query pve overview: %w", err)
		}
		versus, err := s.mysql.GetVersusOverview(queryCtx)
		if err != nil {
			return Overview{}, fmt.Errorf("query versus overview: %w", err)
		}
		result.Core = CoreOverview{core.TotalPlayers, core.ActivePlayers7d, core.TotalActivePlaySeconds, core.CompletedPveRuns, core.CompletedVersusRuns}
		result.PVE = PVEOverview{pve.CommonKills, pve.SpecialKills, pve.TankKills, pve.WitchKills, pve.Rescues}
		result.Versus = VersusOverview{versus.CompletedMatches, versus.CompletedHalves, versus.HumanControlledInfectedKills, versus.HumanSurvivorControls}
	default:
		core, err := s.pg.GetCoreOverview(queryCtx, cutoff)
		if err != nil {
			return Overview{}, fmt.Errorf("query core overview: %w", err)
		}
		pve, err := s.pg.GetPVEOverview(queryCtx)
		if err != nil {
			return Overview{}, fmt.Errorf("query pve overview: %w", err)
		}
		versus, err := s.pg.GetVersusOverview(queryCtx)
		if err != nil {
			return Overview{}, fmt.Errorf("query versus overview: %w", err)
		}
		result.Core = CoreOverview{core.TotalPlayers, core.ActivePlayers7d, core.TotalActivePlaySeconds, core.CompletedPveRuns, core.CompletedVersusRuns}
		result.PVE = PVEOverview{pve.CommonKills, pve.SpecialKills, pve.TankKills, pve.WitchKills, pve.Rescues}
		result.Versus = VersusOverview{versus.CompletedMatches, versus.CompletedHalves, versus.HumanControlledInfectedKills, versus.HumanSurvivorControls}
	}
	return result, nil
}

func (s *statsStore) Close() error { return s.db.Close() }
