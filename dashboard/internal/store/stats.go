package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
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

func OpenStats(ctx context.Context, cfg config.StatsDatabaseConfig) (StatsDatabase, error) {
	store, err := openStats(ctx, cfg, true)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func OpenStatsMaintenance(ctx context.Context, cfg config.StatsDatabaseConfig) (StatsMaintenanceStore, error) {
	return openStats(ctx, cfg, false)
}

func openStats(ctx context.Context, cfg config.StatsDatabaseConfig, readOnly bool) (*statsStore, error) {
	driver := strings.ToLower(cfg.Driver)
	dsn := cfg.DSN
	sqlDriver := driver
	switch driver {
	case "sqlite":
		sqlDriver = "sqlite"
		if !strings.HasPrefix(dsn, "file:") {
			dsn = "file:" + filepath.ToSlash(dsn)
		}
		if readOnly && !strings.Contains(dsn, "mode=") {
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

func (s *statsStore) DatabaseUsage(ctx context.Context) (DatabaseUsage, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	usage := DatabaseUsage{Driver: s.driver}
	switch s.driver {
	case "sqlite":
		var pageCount, pageSize int64
		if err := s.db.QueryRowContext(queryCtx, "PRAGMA page_count").Scan(&pageCount); err != nil {
			return usage, err
		}
		if err := s.db.QueryRowContext(queryCtx, "PRAGMA page_size").Scan(&pageSize); err != nil {
			return usage, err
		}
		usage.Bytes = pageCount * pageSize
		var path string
		if err := s.db.QueryRowContext(queryCtx, "PRAGMA database_list").Scan(new(int64), new(string), &path); err == nil {
			if info, err := os.Stat(path + "-wal"); err == nil {
				usage.WALBytes = info.Size()
			}
		}
	case "mysql":
		if err := s.db.QueryRowContext(queryCtx, `SELECT COALESCE(SUM(data_length + index_length),0) FROM information_schema.tables WHERE table_schema=DATABASE()`).Scan(&usage.Bytes); err != nil {
			return usage, err
		}
	case "postgres":
		if err := s.db.QueryRowContext(queryCtx, `SELECT pg_database_size(current_database())`).Scan(&usage.Bytes); err != nil {
			return usage, err
		}
	}
	return usage, nil
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

func (s *statsStore) PlayerSummary(ctx context.Context, steamID string) (*PlayerSummary, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var result PlayerSummary
	switch s.driver {
	case "sqlite":
		row, err := s.sqlite.GetPlayerSummary(queryCtx, steamID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		result = PlayerSummary{row.SteamID, row.LastName, row.FirstSeenAt, row.LastSeenAt, row.SessionCount, row.ConnectedSeconds, row.ActivePlaySeconds}
	case "mysql":
		row, err := s.mysql.GetPlayerSummary(queryCtx, steamID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		result = PlayerSummary{row.SteamID, row.LastName, row.FirstSeenAt, row.LastSeenAt, row.SessionCount, row.ConnectedSeconds, row.ActivePlaySeconds}
	default:
		row, err := s.pg.GetPlayerSummary(queryCtx, steamID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		result = PlayerSummary{row.SteamID, row.LastName, row.FirstSeenAt, row.LastSeenAt, row.SessionCount, row.ConnectedSeconds, row.ActivePlaySeconds}
	}
	return &result, nil
}

func (s *statsStore) SearchPlayers(ctx context.Context, query string, limit int32) ([]PlayerIdentity, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	query = strings.TrimSpace(query)
	result := make([]PlayerIdentity, 0, limit)
	switch s.driver {
	case "sqlite":
		rows, err := s.sqlite.SearchPlayers(queryCtx, statssqlite.SearchPlayersParams{SearchQuery: query, PageLimit: int64(limit)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerIdentity{SteamID: row.SteamID, Name: row.LastName})
		}
	case "mysql":
		rows, err := s.mysql.SearchPlayers(queryCtx, statsmysql.SearchPlayersParams{SearchQuery: query, Limit: limit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerIdentity{SteamID: row.SteamID, Name: row.LastName})
		}
	default:
		rows, err := s.pg.SearchPlayers(queryCtx, statspg.SearchPlayersParams{SearchQuery: query, PageLimit: limit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerIdentity{SteamID: row.SteamID, Name: row.LastName})
		}
	}
	return result, nil
}

func (s *statsStore) ActivePlayers(ctx context.Context, serverKey string, freshSince int64) ([]ActivePlayer, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result := make([]ActivePlayer, 0)
	switch s.driver {
	case "sqlite":
		rows, err := s.sqlite.ListActivePlayersByServer(queryCtx, statssqlite.ListActivePlayersByServerParams{ServerKey: serverKey, FreshSince: freshSince})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, ActivePlayer{SteamID: row.SteamID, Name: row.PlayerName, StartedAt: row.StartedAt, LastSavedAt: row.LastSavedAt, ConnectedSeconds: row.ConnectedSeconds})
		}
	case "mysql":
		rows, err := s.mysql.ListActivePlayersByServer(queryCtx, statsmysql.ListActivePlayersByServerParams{ServerKey: serverKey, FreshSince: freshSince})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, ActivePlayer{SteamID: row.SteamID, Name: row.PlayerName, StartedAt: row.StartedAt, LastSavedAt: row.LastSavedAt, ConnectedSeconds: row.ConnectedSeconds})
		}
	default:
		rows, err := s.pg.ListActivePlayersByServer(queryCtx, statspg.ListActivePlayersByServerParams{ServerKey: serverKey, FreshSince: freshSince})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, ActivePlayer{SteamID: row.SteamID, Name: row.PlayerName, StartedAt: row.StartedAt, LastSavedAt: row.LastSavedAt, ConnectedSeconds: row.ConnectedSeconds})
		}
	}
	return result, nil
}

func (s *statsStore) PlayerPVE(ctx context.Context, steamID string, cutoff int64) (PlayerPVE, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	switch s.driver {
	case "sqlite":
		row, err := s.sqlite.GetPlayerPVE(queryCtx, statssqlite.GetPlayerPVEParams{SteamID: steamID, Cutoff: cutoff})
		if err != nil {
			return PlayerPVE{}, err
		}
		return s.enrichPlayerPVE(queryCtx, steamID, PlayerFilter{Cutoff: cutoff}, playerPVE(row.CommonKills, row.SpecialKills, row.TankKills, row.WitchKills, row.DamageToSpecial, row.DamageToTank, row.DamageToWitch, row.DamageTaken, row.FriendlyFire, row.Incapacitations, row.Deaths, row.Revives, row.RescuesReceived, row.MedkitsUsed, row.Healing, row.ChapterParticipations, row.ChapterCompletions, row.CampaignCompletions))
	case "mysql":
		row, err := s.mysql.GetPlayerPVE(queryCtx, statsmysql.GetPlayerPVEParams{SteamID: steamID, Cutoff: cutoff})
		if err != nil {
			return PlayerPVE{}, err
		}
		return s.enrichPlayerPVE(queryCtx, steamID, PlayerFilter{Cutoff: cutoff}, playerPVE(row.CommonKills, row.SpecialKills, row.TankKills, row.WitchKills, row.DamageToSpecial, row.DamageToTank, row.DamageToWitch, row.DamageTaken, row.FriendlyFire, row.Incapacitations, row.Deaths, row.Revives, row.RescuesReceived, row.MedkitsUsed, row.Healing, row.ChapterParticipations, row.ChapterCompletions, row.CampaignCompletions))
	default:
		row, err := s.pg.GetPlayerPVE(queryCtx, statspg.GetPlayerPVEParams{SteamID: steamID, Cutoff: cutoff})
		if err != nil {
			return PlayerPVE{}, err
		}
		return s.enrichPlayerPVE(queryCtx, steamID, PlayerFilter{Cutoff: cutoff}, playerPVE(row.CommonKills, row.SpecialKills, row.TankKills, row.WitchKills, row.DamageToSpecial, row.DamageToTank, row.DamageToWitch, row.DamageTaken, row.FriendlyFire, row.Incapacitations, row.Deaths, row.Revives, row.RescuesReceived, row.MedkitsUsed, row.Healing, row.ChapterParticipations, row.ChapterCompletions, row.CampaignCompletions))
	}
}

func (s *statsStore) PlayerVersus(ctx context.Context, steamID string, cutoff int64) (PlayerVersus, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	switch s.driver {
	case "sqlite":
		row, err := s.sqlite.GetPlayerVersus(queryCtx, statssqlite.GetPlayerVersusParams{SteamID: steamID, Cutoff: cutoff})
		if err != nil {
			return PlayerVersus{}, err
		}
		return s.enrichPlayerVersus(queryCtx, steamID, PlayerFilter{Cutoff: cutoff}, playerVersus(row.SurvivorCommonKills, row.HumanSpecialKills, row.BotSpecialKills, row.HumanTankKills, row.BotTankKills, row.SurvivorDamage, row.SurvivorDeaths, row.SurvivorRevives, row.InfectedSpawns, row.DamageToHumanSurvivors, row.HumanSurvivorIncaps, row.HumanSurvivorKills, row.HumanSurvivorControls, row.HumanSurvivorControlSeconds))
	case "mysql":
		row, err := s.mysql.GetPlayerVersus(queryCtx, statsmysql.GetPlayerVersusParams{SteamID: steamID, Cutoff: cutoff})
		if err != nil {
			return PlayerVersus{}, err
		}
		return s.enrichPlayerVersus(queryCtx, steamID, PlayerFilter{Cutoff: cutoff}, playerVersus(row.CommonKills, row.HumanSpecialKills, row.BotSpecialKills, row.HumanTankKills, row.BotTankKills, row.SurvivorDamage, row.SurvivorDeaths, row.SurvivorRevives, row.InfectedSpawns, row.DamageToHumanSurvivors, row.HumanSurvivorIncaps, row.HumanSurvivorKills, row.HumanSurvivorControls, row.HumanSurvivorControlSeconds))
	default:
		row, err := s.pg.GetPlayerVersus(queryCtx, statspg.GetPlayerVersusParams{SteamID: steamID, Cutoff: cutoff})
		if err != nil {
			return PlayerVersus{}, err
		}
		return s.enrichPlayerVersus(queryCtx, steamID, PlayerFilter{Cutoff: cutoff}, playerVersus(row.CommonKills, row.HumanSpecialKills, row.BotSpecialKills, row.HumanTankKills, row.BotTankKills, row.SurvivorDamage, row.SurvivorDeaths, row.SurvivorRevives, row.InfectedSpawns, row.DamageToHumanSurvivors, row.HumanSurvivorIncaps, row.HumanSurvivorKills, row.HumanSurvivorControls, row.HumanSurvivorControlSeconds))
	}
}

func (s *statsStore) PlayerSessions(ctx context.Context, steamID string, cursorAt int64, cursorID string, limit int32) ([]PlayerSession, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result := make([]PlayerSession, 0)
	switch s.driver {
	case "sqlite":
		rows, err := s.sqlite.ListPlayerSessions(queryCtx, statssqlite.ListPlayerSessionsParams{SteamID: steamID, CursorStartedAt: cursorAt, CursorID: cursorID, PageLimit: int64(limit)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerSession{row.SessionID, row.ServerKey, row.PlayerName, row.StartedAt, interfaceInt64(row.EndedAt), row.ConnectedSeconds, row.ActivePlaySeconds, row.Status, row.DisconnectReason})
		}
	case "mysql":
		rows, err := s.mysql.ListPlayerSessions(queryCtx, statsmysql.ListPlayerSessionsParams{SteamID: steamID, CursorStartedAt: cursorAt, CursorID: cursorID, Limit: limit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerSession{row.SessionID, row.ServerKey, row.PlayerName, row.StartedAt, nullInt64(row.EndedAt), row.ConnectedSeconds, row.ActivePlaySeconds, row.Status, row.DisconnectReason})
		}
	default:
		rows, err := s.pg.ListPlayerSessions(queryCtx, statspg.ListPlayerSessionsParams{SteamID: steamID, CursorStartedAt: cursorAt, CursorID: cursorID, PageLimit: limit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerSession{row.SessionID, row.ServerKey, row.PlayerName, row.StartedAt, nullInt64(row.EndedAt), row.ConnectedSeconds, row.ActivePlaySeconds, row.Status, row.DisconnectReason})
		}
	}
	return result, nil
}

func (s *statsStore) PlayerChapters(ctx context.Context, steamID string, cursorAt int64, cursorID string, limit int32) ([]PlayerChapter, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result := make([]PlayerChapter, 0)
	switch s.driver {
	case "sqlite":
		rows, err := s.sqlite.ListPlayerChapters(queryCtx, statssqlite.ListPlayerChaptersParams{SteamID: steamID, CursorStartedAt: cursorAt, CursorID: cursorID, PageLimit: int64(limit)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerChapter{row.SegmentID, row.ServerKey, row.ModeFamily, row.GameMode, row.MapName, row.Side, row.StartedAt, interfaceInt64(row.EndedAt), row.ActivePlaySeconds, row.Status})
		}
	case "mysql":
		rows, err := s.mysql.ListPlayerChapters(queryCtx, statsmysql.ListPlayerChaptersParams{SteamID: steamID, CursorStartedAt: cursorAt, CursorID: cursorID, Limit: limit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerChapter{row.SegmentID, row.ServerKey, row.ModeFamily, row.GameMode, row.MapName, row.Side, row.StartedAt, nullInt64(row.EndedAt), row.ActivePlaySeconds, row.Status})
		}
	default:
		rows, err := s.pg.ListPlayerChapters(queryCtx, statspg.ListPlayerChaptersParams{SteamID: steamID, CursorStartedAt: cursorAt, CursorID: cursorID, PageLimit: limit})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, PlayerChapter{row.SegmentID, row.ServerKey, row.ModeFamily, row.GameMode, row.MapName, row.Side, row.StartedAt, nullInt64(row.EndedAt), row.ActivePlaySeconds, row.Status})
		}
	}
	return result, nil
}

func playerPVE(v ...int64) PlayerPVE {
	return PlayerPVE{
		CommonKills: v[0], SpecialKills: v[1], TankKills: v[2], WitchKills: v[3],
		DamageToSpecial: v[4], DamageToTank: v[5], DamageToWitch: v[6], DamageTaken: v[7],
		FriendlyFire: v[8], Incapacitations: v[9], Deaths: v[10], Revives: v[11],
		RescuesReceived: v[12], MedkitsUsed: v[13], Healing: v[14], ChapterParticipations: v[15],
		ChapterCompletions: v[16], CampaignCompletions: v[17],
	}
}
func playerVersus(v ...int64) PlayerVersus {
	return PlayerVersus{
		SurvivorCommonKills: v[0], HumanSpecialKills: v[1], BotSpecialKills: v[2], HumanTankKills: v[3],
		BotTankKills: v[4], SurvivorDamage: v[5], SurvivorDeaths: v[6], SurvivorRevives: v[7],
		InfectedSpawns: v[8], DamageToHumanSurvivors: v[9], HumanSurvivorIncaps: v[10],
		HumanSurvivorKills: v[11], HumanSurvivorControls: v[12], HumanSurvivorControlSeconds: v[13],
	}
}
func nullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}
func interfaceInt64(v any) *int64 {
	if v == nil {
		return nil
	}
	if n, ok := v.(int64); ok {
		return &n
	}
	return nil
}

func (s *statsStore) Close() error { return s.db.Close() }
