package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
)

func TestDashboardMigrateAndBootstrapIsIdempotent(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "dashboard.db")
	dashboard, err := OpenDashboard(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()

	version, err := dashboard.MigrationVersion(ctx)
	if err != nil || version != 1 {
		t.Fatalf("migration version = %d, err = %v", version, err)
	}
	first := config.BootstrapConfig{
		Site: config.BootstrapSiteConfig{
			Title: "First", FooterText: "line one\nline two",
			FooterLinks: []config.BootstrapLinkConfig{{Label: "ICP", URL: "https://example.com", Enabled: true}},
		},
		Servers: []config.BootstrapServerConfig{{
			ServerKey: "main", DisplayName: "Main", ConnectAddress: "127.0.0.1:27015",
			QueryAddress: "127.0.0.1:27015", Primary: true, Enabled: true,
		}},
	}
	applied, err := dashboard.Bootstrap(ctx, first, false)
	if err != nil || !applied {
		t.Fatalf("first Bootstrap() = %v, %v", applied, err)
	}

	second := first
	second.Site.Title = "Second"
	applied, err = dashboard.Bootstrap(ctx, second, false)
	if err != nil || applied {
		t.Fatalf("second Bootstrap() = %v, %v; want skipped", applied, err)
	}
	site, err := dashboard.Site(ctx)
	if err != nil || site.Title != "First" || len(site.Links) != 1 {
		t.Fatalf("Site() = %#v, %v", site, err)
	}

	applied, err = dashboard.Bootstrap(ctx, second, true)
	if err != nil || !applied {
		t.Fatalf("replace Bootstrap() = %v, %v", applied, err)
	}
	site, err = dashboard.Site(ctx)
	if err != nil || site.Title != "Second" {
		t.Fatalf("replaced Site() = %#v, %v", site, err)
	}
	primary, err := dashboard.PrimaryServer(ctx)
	if err != nil || primary == nil || primary.ServerKey != "main" {
		t.Fatalf("PrimaryServer() = %#v, %v", primary, err)
	}

	if err := dashboard.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenDashboard(ctx, databasePath)
	if err != nil {
		t.Fatalf("reopen dashboard: %v", err)
	}
	defer reopened.Close()
	version, err = reopened.MigrationVersion(ctx)
	if err != nil || version != 1 {
		t.Fatalf("reopened migration version = %d, err = %v", version, err)
	}
}

func TestDashboardBootstrapWithNoServersStillRunsOnlyOnce(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	bootstrap := config.BootstrapConfig{Site: config.BootstrapSiteConfig{Title: "Empty directory"}}
	if applied, err := dashboard.Bootstrap(ctx, bootstrap, false); err != nil || !applied {
		t.Fatalf("first Bootstrap() = %v, %v", applied, err)
	}
	bootstrap.Servers = []config.BootstrapServerConfig{{
		ServerKey: "late", DisplayName: "Late", ConnectAddress: "127.0.0.1:27015",
		QueryAddress: "127.0.0.1:27015", Primary: true, Enabled: true,
	}}
	if applied, err := dashboard.Bootstrap(ctx, bootstrap, false); err != nil || applied {
		t.Fatalf("second Bootstrap() = %v, %v; marker should prevent an implicit overwrite", applied, err)
	}
	primary, err := dashboard.PrimaryServer(ctx)
	if err != nil || primary != nil {
		t.Fatalf("PrimaryServer() = %#v, %v", primary, err)
	}
}
