package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	dashboarddb "github.com/gofurry/l4d2-plugin-stats/dashboard/database/dashboard"
	"github.com/pressly/goose/v3"
)

func TestDashboardInitialSchemaAndManagement(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "dashboard.db")
	dashboard, err := OpenDashboard(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	version, err := dashboard.MigrationVersion(ctx)
	if err != nil || version != 9 {
		t.Fatalf("migration version=%d err=%v", version, err)
	}
	site, err := dashboard.Site(ctx)
	if err != nil || site.Language != "zh-CN" || site.Theme != "light" || site.Configured || site.FooterEnabled {
		t.Fatalf("default Site()=%#v err=%v", site, err)
	}
	settings := SiteSettings{Language: "en", BrowserTitle: "My Stats", Theme: "dark", FooterEnabled: true, BackgroundImageURL: "https://example.com/background.jpg", PublicOrigin: "https://example.com", SteamOpenIDEnabled: true, A2SRefreshSeconds: 10, A2SJitterSeconds: 5, A2SRetryCount: 2, SEOEnabled: true, SEODescription: "Server statistics", SEOImageURL: "https://example.com/share.jpg", Links: []FooterLink{{Label: "ICP", URL: "https://example.com/icp"}}}
	if err := dashboard.UpdateSite(ctx, settings); err != nil {
		t.Fatal(err)
	}
	site, err = dashboard.Site(ctx)
	if err != nil || site.Language != "en" || site.BrowserTitle != "My Stats" || site.Theme != "dark" || site.A2SRefreshSeconds != 10 || !site.Configured || !site.FooterEnabled || !site.SteamOpenIDEnabled || site.BackgroundImageURL == "" || len(site.Links) != 1 {
		t.Fatalf("updated Site()=%#v err=%v", site, err)
	}
	savedSettings, err := dashboard.SiteSettings(ctx)
	if err != nil || savedSettings.Theme != "dark" || savedSettings.A2SJitterSeconds != 5 || savedSettings.A2SRetryCount != 2 || !savedSettings.SEOEnabled || savedSettings.SEODescription == "" {
		t.Fatalf("updated SiteSettings()=%#v err=%v", savedSettings, err)
	}
	document, err := dashboard.UpdateSiteDocument(ctx, SiteDocument{Key: "introduction", Enabled: true, ContentMarkdown: "# Welcome"})
	if err != nil || !document.Enabled {
		t.Fatalf("UpdateSiteDocument()=%#v err=%v", document, err)
	}
	publicSite, err := dashboard.Site(ctx)
	if err != nil || len(publicSite.Documents) != 1 || publicSite.Documents[0] != "introduction" {
		t.Fatalf("Site().Documents=%#v err=%v", publicSite.Documents, err)
	}
	created, err := dashboard.CreateServer(ctx, GameServer{DisplayName: "Main", Address: "127.0.0.1:27015"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || !created.Enabled || created.SortOrder != 0 {
		t.Fatalf("created server=%#v", created)
	}
	second, err := dashboard.CreateServer(ctx, GameServer{DisplayName: "Second", Address: "127.0.0.1:27016"})
	if err != nil || second.SortOrder != 1 {
		t.Fatalf("second server=%#v err=%v", second, err)
	}
	servers, err := dashboard.ListServers(ctx)
	if err != nil || len(servers) != 2 || servers[0].ID != created.ID || servers[1].ID != second.ID {
		t.Fatalf("ListServers()=%#v err=%v", servers, err)
	}
	created.DisplayName = "Main updated"
	if err := dashboard.UpdateServer(ctx, created); err != nil {
		t.Fatal(err)
	}
	if err := dashboard.SetServerEnabled(ctx, created.ID, false); err != nil {
		t.Fatal(err)
	}
	servers, err = dashboard.ListServers(ctx)
	if err != nil || len(servers) != 2 || servers[0].Enabled || servers[0].DisplayName != "Main updated" {
		t.Fatalf("updated servers=%#v err=%v", servers, err)
	}
	if err := dashboard.MoveServer(ctx, second.ID, "up"); err != nil {
		t.Fatal(err)
	}
	servers, err = dashboard.ListServers(ctx)
	if err != nil || servers[0].ID != second.ID || servers[1].ID != created.ID {
		t.Fatalf("moved servers=%#v err=%v", servers, err)
	}
}

func TestDashboardSingleAdministrator(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	configured, err := dashboard.AdminConfigured(ctx)
	if err != nil || configured {
		t.Fatalf("configured=%v err=%v", configured, err)
	}
	if err := dashboard.CreateAdmin(ctx, "admin", "hash", "secret"); err != nil {
		t.Fatal(err)
	}
	if err := dashboard.CreateAdmin(ctx, "other", "hash", "secret"); err == nil {
		t.Fatal("second administrator should fail")
	}
	admin, err := dashboard.Admin(ctx)
	if err != nil || admin == nil || admin.Username != "admin" || admin.TokenVersion != 1 {
		t.Fatalf("Admin()=%#v err=%v", admin, err)
	}
	if err := dashboard.UpdateAdminPassword(ctx, "new-hash"); err != nil {
		t.Fatal(err)
	}
	admin, _ = dashboard.Admin(ctx)
	if admin.TokenVersion != 2 {
		t.Fatalf("token version=%d", admin.TokenVersion)
	}
}

func TestDashboardAnnouncementLifecycle(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	created, err := dashboard.CreateAnnouncement(ctx, Announcement{Title: "Update", ContentMarkdown: "**Ready**"})
	if err != nil || created.ID == "" || created.CreatedAt == 0 {
		t.Fatalf("created=%#v err=%v", created, err)
	}
	page, err := dashboard.ListAnnouncements(ctx, AnnouncementFilter{Limit: 20})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != created.ID {
		t.Fatalf("page=%#v err=%v", page, err)
	}
	page, err = dashboard.ListAnnouncements(ctx, AnnouncementFilter{Title: "date", Year: time.Now().Year(), Limit: 20})
	if err != nil || page.Total != 1 {
		t.Fatalf("filtered page=%#v err=%v", page, err)
	}
	years, err := dashboard.ListAnnouncementYears(ctx)
	if err != nil || len(years) != 1 || years[0] != time.Now().Year() {
		t.Fatalf("years=%v err=%v", years, err)
	}
	created.Title = "Updated"
	updated, err := dashboard.UpdateAnnouncement(ctx, created)
	if err != nil || updated.Title != "Updated" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	if err := dashboard.DeleteAnnouncement(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	page, err = dashboard.ListAnnouncements(ctx, AnnouncementFilter{Limit: 20})
	if err != nil || page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("empty page=%#v err=%v", page, err)
	}
}

func TestDashboardAggregateFiltersAreAppliedByStore(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	change := AggregateChangeSet{
		Full: true, SourceRows: 3, SourceWatermark: 100,
		Rows: []AggregateRow{
			{Version: AggregateContractVersion, Kind: "activity", Day: 10, ServerKey: "one", SteamID: "a", Metrics: map[string]int64{"active_play_seconds": 10}},
			{Version: AggregateContractVersion, Kind: "activity", Day: 20, ServerKey: "two", SteamID: "a", Metrics: map[string]int64{"active_play_seconds": 20}},
			{Version: AggregateContractVersion, Kind: "pve_combat", Day: 20, ServerKey: "one", SteamID: "b", Mode: "coop", Metrics: map[string]int64{"common_kills": 30}},
		},
	}
	if err := dashboard.ApplyAggregateChanges(ctx, change); err != nil {
		t.Fatal(err)
	}
	status, err := dashboard.AggregateStatus(ctx)
	if err != nil || status.AggregateVersion != AggregateContractVersion {
		t.Fatalf("aggregate status=%+v err=%v", status, err)
	}
	rows, err := dashboard.ListAggregateRows(ctx, AggregateFilter{Kinds: []string{"activity"}, SteamID: "a", ServerKey: "two", CutoffDay: 15})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Day != 20 || rows[0].ServerKey != "two" || rows[0].Kind != "activity" {
		t.Fatalf("filtered aggregate rows=%#v", rows)
	}
	lifetime, err := dashboard.ListAggregateRows(ctx, AggregateFilter{Grain: AggregateGrainLifetime, Kinds: []string{"activity"}, SteamID: "a", ServerKey: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lifetime) != 1 || lifetime[0].Day != 0 || lifetime[0].Metrics["active_play_seconds"] != 20 {
		t.Fatalf("lifetime aggregate rows=%#v", lifetime)
	}
	monthly, err := dashboard.ListAggregateRows(ctx, AggregateFilter{Grain: AggregateGrainMonthly, Kinds: []string{"activity"}, SteamID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(monthly) != 2 || monthly[0].Metrics["active_play_seconds"]+monthly[1].Metrics["active_play_seconds"] != 30 {
		t.Fatalf("monthly aggregate rows=%#v", monthly)
	}
	if err := dashboard.ApplyAggregateChanges(ctx, AggregateChangeSet{
		Days: []int64{20}, SourceRows: 3, SourceWatermark: 110,
		Rows: []AggregateRow{{Version: AggregateContractVersion, Kind: "activity", Day: 20, ServerKey: "two", SteamID: "a", Metrics: map[string]int64{"active_play_seconds": 25}}},
	}); err != nil {
		t.Fatal(err)
	}
	lifetime, err = dashboard.ListAggregateRows(ctx, AggregateFilter{Grain: AggregateGrainLifetime, Kinds: []string{"activity"}, SteamID: "a", ServerKey: "two"})
	if err != nil || len(lifetime) != 1 || lifetime[0].Metrics["active_play_seconds"] != 25 {
		t.Fatalf("adjusted lifetime aggregate rows=%#v err=%v", lifetime, err)
	}
}

func TestDashboardRejectsUnknownAggregateContractVersions(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	store := dashboard.(*dashboardStore)
	if _, err := store.db.ExecContext(ctx, `UPDATE aggregate_state SET aggregate_version=2 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := dashboard.AggregateStatus(ctx); err == nil {
		t.Fatal("AggregateStatus accepted aggregate contract version 2")
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE aggregate_state SET aggregate_version=1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := dashboard.ApplyAggregateChanges(ctx, AggregateChangeSet{Full: true, Rows: []AggregateRow{{Version: 2, Kind: "activity", Metrics: map[string]int64{}}}}); err == nil {
		t.Fatal("ApplyAggregateChanges accepted aggregate contract version 2")
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO aggregate_rows (kind,day,server_key,steam_id,mode,dimension,metrics_json,aggregate_version) VALUES ('activity',1,'','','','','{}',2)`); err != nil {
		t.Fatal(err)
	}
	if _, err := dashboard.ListAggregateRows(ctx, AggregateFilter{}); err == nil {
		t.Fatal("ListAggregateRows accepted aggregate contract version 2")
	}
	if err := dashboard.ApplyAggregateChanges(ctx, AggregateChangeSet{Full: true}); err == nil {
		t.Fatal("ApplyAggregateChanges replaced an unknown stored aggregate contract version")
	}
}

func TestAggregateContractMigrationEightToNine(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migration.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 8); err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`INSERT INTO aggregate_rows (kind,day,server_key,steam_id,mode,dimension,metrics_json) VALUES ('activity',1,'one','1','','','{}')`,
		`INSERT INTO aggregate_monthly_rows (kind,month,server_key,steam_id,mode,dimension,metrics_json) VALUES ('activity',1,'one','1','','','{}')`,
		`INSERT INTO aggregate_lifetime_rows (kind,server_key,steam_id,mode,dimension,metrics_json) VALUES ('activity','one','1','','','{}')`,
		`INSERT INTO retention_runs (id,executed_at,source_watermark,detail_cutoff,session_cutoff,result_cutoff,equipment_rows,versus_class_rows,session_rows,versus_round_result_rows,versus_run_result_rows) VALUES ('run',1,1,1,1,1,0,0,0,0,0)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := goose.UpToContext(ctx, db, "migrations", 9); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"aggregate_rows", "aggregate_monthly_rows", "aggregate_lifetime_rows", "aggregate_state", "retention_runs"} {
		var version int64
		if err := db.QueryRowContext(ctx, `SELECT aggregate_version FROM `+table+` LIMIT 1`).Scan(&version); err != nil || version != AggregateContractVersion {
			t.Fatalf("%s aggregate_version=%d err=%v", table, version, err)
		}
	}
	if err := goose.DownToContext(ctx, db, "migrations", 8); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"aggregate_rows", "aggregate_monthly_rows", "aggregate_lifetime_rows", "aggregate_state", "retention_runs"} {
		if _, err := db.ExecContext(ctx, `SELECT aggregate_version FROM `+table+` LIMIT 1`); err == nil {
			t.Fatalf("%s retained aggregate_version after Down migration", table)
		}
	}
}
