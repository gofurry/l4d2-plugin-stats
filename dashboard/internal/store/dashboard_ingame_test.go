package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	dashboarddb "github.com/gofurry/l4d2-plugin-stats/dashboard/database/dashboard"
	"github.com/pressly/goose/v3"
)

func TestDashboardIngameSettingsAndServerDocuments(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	ingame := dashboard.(DashboardIngameStore)
	settings, err := ingame.IngameSettings(ctx)
	if err != nil || !settings.Enabled || !settings.ShowServerIntro || !settings.ShowServerStatus || settings.HomeCacheSeconds != 30 || settings.HighlightMetrics != [3]string{"active_play_seconds", "special_kills", "rescues"} {
		t.Fatalf("default settings=%+v err=%v", settings, err)
	}
	settings.Title = "In-Game"
	settings.BannerURL = "https://example.com/banner.jpg"
	settings.BackgroundURL = "https://example.com/background.jpg?ver=2"
	settings.ShowServerIntro = false
	settings.ShowServerStatus = false
	settings.HomeCacheSeconds = 60
	saved, err := ingame.UpdateIngameSettings(ctx, settings)
	if err != nil || saved.Title != settings.Title || saved.BackgroundURL != settings.BackgroundURL || saved.ShowServerIntro || saved.ShowServerStatus || saved.HomeCacheSeconds != 60 || saved.UpdatedAt == 0 {
		t.Fatalf("saved settings=%+v err=%v", saved, err)
	}

	server, err := dashboard.CreateServer(ctx, GameServer{DisplayName: "Main", Address: "127.0.0.1:27015"})
	if err != nil {
		t.Fatal(err)
	}
	serverKey := "community.one"
	serverSettings, err := ingame.IngameServerSettings(ctx, serverKey)
	if err != nil || serverSettings.TitleMode != "inherit" || serverSettings.ServerKey != serverKey {
		t.Fatalf("default server settings=%+v err=%v", serverSettings, err)
	}
	serverSettings.TitleMode = "override"
	serverSettings.Title = "Main Portal"
	serverSettings.DescriptionMode = "hidden"
	serverSettings.BannerMode = "inherit"
	serverSettings.BackgroundMode = "override"
	serverSettings.BackgroundURL = "https://example.com/server-background.jpg"
	serverSettings.WebsiteMode = "hidden"
	serverSettings.HighlightMode = "inherit"
	serverSettings, err = ingame.UpdateIngameServerSettings(ctx, serverSettings)
	if err != nil || serverSettings.Title != "Main Portal" || serverSettings.BackgroundMode != "override" || serverSettings.BackgroundURL == "" || serverSettings.UpdatedAt == 0 {
		t.Fatalf("saved server settings=%+v err=%v", serverSettings, err)
	}
	for _, mode := range []string{"hidden", "inherit", "override"} {
		serverSettings.BackgroundMode = mode
		savedMode, saveErr := ingame.UpdateIngameServerSettings(ctx, serverSettings)
		if saveErr != nil || savedMode.BackgroundMode != mode || savedMode.BackgroundURL != serverSettings.BackgroundURL {
			t.Fatalf("persist background mode %q: settings=%+v err=%v", mode, savedMode, saveErr)
		}
		serverSettings = savedMode
	}

	document, err := ingame.UpdateServerDocument(ctx, ServerDocument{
		ServerKey: serverKey, Key: IngameDocumentCommands, Mode: "override", ContentMarkdown: "- !help",
	})
	if err != nil || document.Mode != "override" || document.UpdatedAt == 0 {
		t.Fatalf("saved document=%+v err=%v", document, err)
	}
	documents, err := ingame.ListServerDocuments(ctx, serverKey)
	if err != nil || len(documents) != 1 || documents[0].Key != IngameDocumentCommands {
		t.Fatalf("documents=%+v err=%v", documents, err)
	}
	links, err := ingame.ReplaceServerQuickLinks(ctx, serverKey, []IngameQuickLink{
		{Label: "问题反馈", URL: "https://example.com/issues", SortOrder: 1, Enabled: false},
		{Label: "地图合集", URL: "https://example.com/maps", SortOrder: 0, Enabled: true},
	})
	if err != nil || len(links) != 2 || links[0].Label != "地图合集" || links[1].Enabled || links[0].ServerKey != serverKey {
		t.Fatalf("quick links=%+v err=%v", links, err)
	}
	if _, err := ingame.ReplaceServerQuickLinks(ctx, serverKey, []IngameQuickLink{{Label: "Steam", URL: "steam://connect/127.0.0.1", SortOrder: 0, Enabled: true}}); err == nil {
		t.Fatal("store accepted unsafe quick-link URL")
	}
	mapNames, err := ingame.ReplaceIngameMapNames(ctx, []IngameMapName{{MapName: " C1M1_HOTEL ", DisplayName: " 自定义大厅 "}, {MapName: "custom_map", DisplayName: "三方图第一章"}})
	if err != nil || len(mapNames) != 2 || mapNames[0].MapName != "c1m1_hotel" || mapNames[0].DisplayName != "自定义大厅" || mapNames[0].UpdatedAt == 0 {
		t.Fatalf("map names=%+v err=%v", mapNames, err)
	}
	if _, err := ingame.ReplaceIngameMapNames(ctx, []IngameMapName{{MapName: "Map", DisplayName: "One"}, {MapName: "map", DisplayName: "Two"}}); err == nil {
		t.Fatal("store accepted duplicate case-insensitive map_name")
	}

	if err := dashboard.DeleteServer(ctx, server.ID); err != nil {
		t.Fatal(err)
	}
	store := dashboard.(*dashboardStore)
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingame_server_settings`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("server settings after deletion=%d err=%v", count, err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM server_documents`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("server documents after deletion=%d err=%v", count, err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingame_quick_links`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("server quick links after instance deletion=%d err=%v", count, err)
	}
}

func TestIngameMigrationNineteenToTwentyOne(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "dashboard-19.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 19); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_server_settings(server_key,updated_at) VALUES('shared.key',1)`); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 20); err != nil {
		t.Fatal(err)
	}
	var shortDescription string
	if err := db.QueryRowContext(ctx, `SELECT short_description FROM ingame_server_settings WHERE server_key='shared.key'`).Scan(&shortDescription); err != nil || shortDescription != "" {
		t.Fatalf("schema 20 short_description=%q err=%v", shortDescription, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE ingame_server_settings SET short_description=? WHERE server_key='shared.key'`, strings.Repeat("字", 81)); err == nil {
		t.Fatal("schema 20 accepted an overlong short description")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_map_names(map_name,display_name,updated_at) VALUES('c1m1_hotel','自定义大厅',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_map_names(map_name,display_name,updated_at) VALUES('','invalid',1)`); err == nil {
		t.Fatal("schema accepted an empty map_name")
	}
	if err := goose.UpToContext(ctx, db, "migrations", 21); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT short_description FROM ingame_server_settings`); err == nil {
		t.Fatal("short_description survived schema 21 migration")
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM ingame_map_names`); err != nil {
		t.Fatalf("schema 21 removed map names: %v", err)
	}
	if err := goose.DownToContext(ctx, db, "migrations", 20); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT short_description FROM ingame_server_settings WHERE server_key='shared.key'`).Scan(&shortDescription); err != nil || shortDescription != "" {
		t.Fatalf("schema 21 Down short_description=%q err=%v", shortDescription, err)
	}
	if err := goose.DownToContext(ctx, db, "migrations", 19); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM ingame_map_names`); err == nil {
		t.Fatal("map-name table survived Down migration")
	}
	if _, err := db.ExecContext(ctx, `SELECT short_description FROM ingame_server_settings`); err == nil {
		t.Fatal("short_description survived schema 20 Down migration")
	}
}

func TestIngameMigrationSixteenToSeventeen(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "dashboard-16.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 16); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM ingame_settings`); err == nil {
		t.Fatal("schema 16 unexpectedly contains in-game settings")
	}
	if err := goose.UpToContext(ctx, db, "migrations", 17); err != nil {
		t.Fatal(err)
	}
	var enabled, home, player, ranking, content int64
	var background string
	if err := db.QueryRowContext(ctx, `SELECT enabled,background_url,home_cache_seconds,player_cache_seconds,ranking_cache_seconds,content_cache_seconds FROM ingame_settings WHERE id=1`).Scan(&enabled, &background, &home, &player, &ranking, &content); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 || background != "" || home != 30 || player != 60 || ranking != 120 || content != 300 {
		t.Fatalf("migration defaults=%d/%q/%d/%d/%d/%d", enabled, background, home, player, ranking, content)
	}
	serverID := "00000000-0000-0000-0000-000000000001"
	if _, err := db.ExecContext(ctx, `INSERT INTO game_servers(id,display_name,address,enabled,sort_order,created_at,updated_at) VALUES(?, 'Main', '127.0.0.1:27015', 1, 0, unixepoch(), unixepoch())`, serverID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_server_settings(server_id,updated_at) VALUES(?,unixepoch())`, serverID); err != nil {
		t.Fatal(err)
	}
	var backgroundMode, serverBackground string
	if err := db.QueryRowContext(ctx, `SELECT background_mode,background_url FROM ingame_server_settings WHERE server_id=?`, serverID).Scan(&backgroundMode, &serverBackground); err != nil {
		t.Fatal(err)
	}
	if backgroundMode != "inherit" || serverBackground != "" {
		t.Fatalf("server background defaults=%q/%q", backgroundMode, serverBackground)
	}
	if _, err := db.ExecContext(ctx, `UPDATE ingame_server_settings SET background_mode='invalid' WHERE server_id=?`, serverID); err == nil {
		t.Fatal("background_mode CHECK accepted invalid value")
	}
	for _, statement := range []string{
		`UPDATE ingame_settings SET home_cache_seconds=5 WHERE id=1`,
		`UPDATE ingame_settings SET player_cache_seconds=10 WHERE id=1`,
		`UPDATE ingame_settings SET ranking_cache_seconds=10 WHERE id=1`,
		`UPDATE ingame_settings SET content_cache_seconds=10 WHERE id=1`,
	} {
		if _, err := db.ExecContext(ctx, statement); err == nil {
			t.Fatalf("preset CHECK accepted %q", statement)
		}
	}
	if err := goose.DownToContext(ctx, db, "migrations", 16); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM ingame_settings`); err == nil {
		t.Fatal("schema 17 in-game tables survived Down migration")
	}
}

func TestOpenDashboardCompletesPrereleaseSchemaSeventeen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "dashboard-prerelease-17.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 17); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`ALTER TABLE ingame_settings DROP COLUMN background_url`,
		`ALTER TABLE ingame_server_settings DROP COLUMN background_url`,
		`ALTER TABLE ingame_server_settings DROP COLUMN background_mode`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("prepare pre-release schema 17 with %q: %v", statement, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	dashboard, err := OpenDashboard(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	if version, versionErr := dashboard.MigrationVersion(ctx); versionErr != nil || version != DashboardSchemaVersion {
		t.Fatalf("migration version=%d err=%v", version, versionErr)
	}
	ingame := dashboard.(DashboardIngameStore)
	settings, err := ingame.IngameSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings.BackgroundURL != "" || settings.HighlightMetrics != [3]string{"active_play_seconds", "special_kills", "rescues"} || settings.HomeCacheSeconds != 30 || settings.PlayerCacheSeconds != 60 || settings.RankingCacheSeconds != 120 || settings.ContentCacheSeconds != 300 {
		t.Fatalf("completed global settings=%+v", settings)
	}
	serverSettings, err := ingame.IngameServerSettings(ctx, "community.one")
	if err != nil || serverSettings.BackgroundMode != "inherit" || serverSettings.BackgroundURL != "" {
		t.Fatalf("completed server settings=%+v err=%v", serverSettings, err)
	}
}

func TestIngameMigrationEighteenToNineteenAddsQuickLinks(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "dashboard-18.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 18); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM ingame_quick_links`); err == nil {
		t.Fatal("schema 18 unexpectedly contains quick links")
	}
	if err := goose.UpToContext(ctx, db, "migrations", 19); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_quick_links(server_key,label,url,sort_order,enabled) VALUES
		('shared.key','地图合集','https://example.com/maps',1,1),
		('shared.key','问题反馈','https://example.com/issues',0,0)`); err != nil {
		t.Fatal(err)
	}
	var first string
	if err := db.QueryRowContext(ctx, `SELECT label FROM ingame_quick_links WHERE server_key='shared.key' ORDER BY sort_order,id LIMIT 1`).Scan(&first); err != nil || first != "问题反馈" {
		t.Fatalf("stable quick-link order=%q err=%v", first, err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_quick_links(server_key,label,url,sort_order,enabled) VALUES('shared.key','overflow','https://example.com',8,1)`); err == nil {
		t.Fatal("schema accepted a ninth sort position")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_quick_links(server_key,label,url,sort_order,enabled) VALUES('shared.key','', 'https://example.com',2,1)`); err == nil {
		t.Fatal("schema accepted an empty label")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_quick_links(server_key,label,url,sort_order,enabled) VALUES('shared.key','Steam','steam://connect/127.0.0.1',2,1)`); err == nil {
		t.Fatal("schema accepted a non-HTTP quick link")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_quick_links(server_key,label,url,sort_order,enabled) VALUES('shared.key','Credential','https://user:pass@example.com',2,1)`); err == nil {
		t.Fatal("schema accepted a credential-bearing quick link")
	}
	if err := goose.DownToContext(ctx, db, "migrations", 18); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SELECT 1 FROM ingame_quick_links`); err == nil {
		t.Fatal("quick-link table survived Down migration")
	}
}

func TestIngameMigrationSeventeenToEighteenCollapsesByServerKey(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "dashboard-17.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 17); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ id, name, address string }{
		{"00000000-0000-0000-0000-000000000001", "One", "127.0.0.1:27015"},
		{"00000000-0000-0000-0000-000000000002", "Two", "127.0.0.1:27016"},
		{"00000000-0000-0000-0000-000000000003", "Unknown", "127.0.0.1:27017"},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO game_servers(id,display_name,address,enabled,sort_order,created_at,updated_at) VALUES(?,?,?,1,0,1,1)`, row.id, row.name, row.address); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO a2s_status_snapshots(server_id,status_json,checked_at,updated_at) VALUES
		(?, '{"server_id":"00000000-0000-0000-0000-000000000001","server_key":"shared.key"}', 1, 1),
		(?, '{"server_id":"00000000-0000-0000-0000-000000000002","server_key":"shared.key"}', 1, 1),
		(?, '{"server_id":"00000000-0000-0000-0000-000000000003"}', 1, 1)`,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ingame_server_settings(server_id,title_mode,title,updated_at) VALUES
		(?, 'override', 'Older', 10), (?, 'override', 'Newer', 20), (?, 'override', 'Discard me', 30)`,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO server_documents(server_id,key,mode,content_markdown,updated_at) VALUES
		(?, 'commands', 'override', 'low id', 40), (?, 'commands', 'override', 'high id', 40)`,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 18); err != nil {
		t.Fatal(err)
	}
	var intro, status int
	if err := db.QueryRowContext(ctx, `SELECT show_server_intro,show_server_status FROM ingame_settings WHERE id=1`).Scan(&intro, &status); err != nil || intro != 1 || status != 1 {
		t.Fatalf("module defaults=%d/%d err=%v", intro, status, err)
	}
	var key, title string
	if err := db.QueryRowContext(ctx, `SELECT server_key,title FROM ingame_server_settings`).Scan(&key, &title); err != nil || key != "shared.key" || title != "Newer" {
		t.Fatalf("collapsed settings=%q/%q err=%v", key, title, err)
	}
	var content string
	if err := db.QueryRowContext(ctx, `SELECT content_markdown FROM server_documents WHERE server_key='shared.key' AND key='commands'`).Scan(&content); err != nil || content != "high id" {
		t.Fatalf("deterministic document winner=%q err=%v", content, err)
	}
	if err := goose.DownToContext(ctx, db, "migrations", 17); err != nil {
		t.Fatal(err)
	}
	var restored int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingame_server_settings WHERE title='Newer'`).Scan(&restored); err != nil || restored != 2 {
		t.Fatalf("restored legacy rows=%d err=%v", restored, err)
	}
	if _, err := db.ExecContext(ctx, `SELECT show_server_intro FROM ingame_settings`); err == nil {
		t.Fatal("schema 18 module columns survived Down migration")
	}
}
