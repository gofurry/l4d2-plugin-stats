package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeoIPSettingsMaskAndStableSecret(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "dashboard.db")
	dashboard, err := OpenDashboard(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := dashboard.UpdateGeoIPSettings(ctx, true, "baidu-private-key", false); err != nil {
		t.Fatal(err)
	}
	settings, err := dashboard.GeoIPSettings(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.APIKeySet || settings.APIKeyMasked != "****-key" || strings.Contains(settings.APIKeyMasked, "baidu-private") || settings.PendingCount != 3 {
		t.Fatalf("masked settings=%+v", settings)
	}
	runtime, err := dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.APIKey != "baidu-private-key" || runtime.CacheSecret == "" {
		t.Fatalf("runtime=%+v err=%v", runtime, err)
	}
	secret := runtime.CacheSecret
	if err := dashboard.UpdateGeoIPSettings(ctx, true, "", false); err != nil {
		t.Fatal(err)
	}
	runtime, err = dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || runtime.APIKey != "baidu-private-key" || runtime.CacheSecret != secret {
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
}
