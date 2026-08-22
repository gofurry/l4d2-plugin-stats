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

func TestGeoIPSettingsMaskAndStableSecret(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "dashboard.db")
	dashboard, err := OpenDashboard(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := dashboard.UpdateGeoIPSettings(ctx, "baidu-private-key", false, 2); err != nil {
		t.Fatal(err)
	}
	settings, err := dashboard.GeoIPSettings(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.APIKeySet || settings.APIKeyMasked != "****-key" || strings.Contains(settings.APIKeyMasked, "baidu-private") || settings.PendingCount != 3 || settings.QPSLimit != 2 {
		t.Fatalf("masked settings=%+v", settings)
	}
	runtime, err := dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.APIKey != "baidu-private-key" || runtime.CacheSecret == "" {
		t.Fatalf("runtime=%+v err=%v", runtime, err)
	}
	secret := runtime.CacheSecret
	if err := dashboard.UpdateGeoIPSettings(ctx, "", false, 3); err != nil {
		t.Fatal(err)
	}
	runtime, err = dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.APIKey != "baidu-private-key" || runtime.CacheSecret != secret || runtime.QPSLimit != 3 {
		t.Fatalf("empty update erased key or rotated secret: %+v err=%v", runtime, err)
	}
	if err := dashboard.Close(); err != nil {
		t.Fatal(err)
	}
	dashboard, err = OpenDashboard(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	runtime, err = dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.CacheSecret != secret {
		t.Fatalf("cache secret was not stable after reopen: %+v err=%v", runtime, err)
	}
	if err := dashboard.UpdateGeoIPSettings(ctx, "", true, 2); err != nil {
		t.Fatal(err)
	}
	runtime, err = dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.APIKey != "" || runtime.QPSLimit != 2 {
		t.Fatalf("explicit key clear did not disable GeoIP: %+v err=%v", runtime, err)
	}
	if err := dashboard.UpdateGeoIPSettings(ctx, "key", false, 4); err == nil {
		t.Fatal("out-of-range GeoIP QPS was accepted")
	}
	db := dashboard.(*dashboardStore).db
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(geoip_settings)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	foundQPS, foundEnabled := false, false
	for rows.Next() {
		var cid, notNull, primaryKey int64
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		foundQPS = foundQPS || name == "qps_limit"
		foundEnabled = foundEnabled || name == "enabled"
	}
	if !foundQPS || foundEnabled {
		t.Fatalf("schema 23 GeoIP columns: qps_limit=%v enabled=%v", foundQPS, foundEnabled)
	}
}

func TestExpiredGeoIPCacheCleanupIsBounded(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	now := int64(1_800_000_000)
	for index, expiresAt := range []int64{now - 2, now - 1, now + 1} {
		if err := dashboard.UpsertGeoIPCache(ctx, GeoIPCacheEntry{
			IPHash: string(rune('a' + index)), Provider: "baidu", Status: "resolved",
			CoordinateSystem: "bd09ll", Precision: "city", ResolvedAt: now - 10, ExpiresAt: expiresAt,
		}); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := dashboard.DeleteExpiredGeoIPCache(ctx, now, 1)
	if err != nil || deleted != 1 {
		t.Fatalf("first cleanup deleted=%d err=%v", deleted, err)
	}
	deleted, err = dashboard.DeleteExpiredGeoIPCache(ctx, now, 500)
	if err != nil || deleted != 1 {
		t.Fatalf("second cleanup deleted=%d err=%v", deleted, err)
	}
	count, err := dashboard.GeoIPCacheCount(ctx)
	if err != nil || count != 1 {
		t.Fatalf("remaining cache=%d err=%v", count, err)
	}
}

func TestGeoIPSchema23UpgradePreservesKeyAndRemovesEnableFlag(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "dashboard.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(dashboarddb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, "migrations", 22); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE geoip_settings SET enabled=1, api_key='existing-key' WHERE singleton_id=1`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	dashboard, err := OpenDashboard(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	runtime, err := dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.APIKey != "existing-key" || runtime.QPSLimit != 2 {
		t.Fatalf("upgraded runtime=%+v err=%v", runtime, err)
	}
	var enabledColumn int64
	if err := dashboard.(*dashboardStore).db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('geoip_settings') WHERE name='enabled'`).Scan(&enabledColumn); err != nil {
		t.Fatal(err)
	}
	if enabledColumn != 0 {
		t.Fatal("schema 23 retained the obsolete GeoIP enabled column")
	}
}
