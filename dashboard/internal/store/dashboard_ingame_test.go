package store

import (
	"context"
	"database/sql"
	"path/filepath"
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
	if err != nil || !settings.Enabled || settings.HomeCacheSeconds != 30 || settings.HighlightMetrics != [3]string{"active_play_seconds", "special_kills", "rescues"} {
		t.Fatalf("default settings=%+v err=%v", settings, err)
	}
	settings.Title = "In-Game"
	settings.BannerURL = "https://example.com/banner.jpg"
	settings.HomeCacheSeconds = 60
	saved, err := ingame.UpdateIngameSettings(ctx, settings)
	if err != nil || saved.Title != settings.Title || saved.HomeCacheSeconds != 60 || saved.UpdatedAt == 0 {
		t.Fatalf("saved settings=%+v err=%v", saved, err)
	}

	server, err := dashboard.CreateServer(ctx, GameServer{DisplayName: "Main", Address: "127.0.0.1:27015"})
	if err != nil {
		t.Fatal(err)
	}
	serverSettings, err := ingame.IngameServerSettings(ctx, server.ID)
	if err != nil || serverSettings.TitleMode != "inherit" || serverSettings.ServerID != server.ID {
		t.Fatalf("default server settings=%+v err=%v", serverSettings, err)
	}
	serverSettings.TitleMode = "override"
	serverSettings.Title = "Main Portal"
	serverSettings.DescriptionMode = "hidden"
	serverSettings.BannerMode = "inherit"
	serverSettings.WebsiteMode = "hidden"
	serverSettings.HighlightMode = "inherit"
	serverSettings, err = ingame.UpdateIngameServerSettings(ctx, serverSettings)
	if err != nil || serverSettings.Title != "Main Portal" || serverSettings.UpdatedAt == 0 {
		t.Fatalf("saved server settings=%+v err=%v", serverSettings, err)
	}

	document, err := ingame.UpdateServerDocument(ctx, ServerDocument{
		ServerID: server.ID, Key: IngameDocumentCommands, Mode: "override", ContentMarkdown: "- !help",
	})
	if err != nil || document.Mode != "override" || document.UpdatedAt == 0 {
		t.Fatalf("saved document=%+v err=%v", document, err)
	}
	documents, err := ingame.ListServerDocuments(ctx, server.ID)
	if err != nil || len(documents) != 1 || documents[0].Key != IngameDocumentCommands {
		t.Fatalf("documents=%+v err=%v", documents, err)
	}

	if err := dashboard.DeleteServer(ctx, server.ID); err != nil {
		t.Fatal(err)
	}
	store := dashboard.(*dashboardStore)
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingame_server_settings`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("server settings after deletion=%d err=%v", count, err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM server_documents`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("server documents after deletion=%d err=%v", count, err)
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
	if err := db.QueryRowContext(ctx, `SELECT enabled,home_cache_seconds,player_cache_seconds,ranking_cache_seconds,content_cache_seconds FROM ingame_settings WHERE id=1`).Scan(&enabled, &home, &player, &ranking, &content); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 || home != 30 || player != 60 || ranking != 120 || content != 300 {
		t.Fatalf("migration defaults=%d/%d/%d/%d/%d", enabled, home, player, ranking, content)
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
