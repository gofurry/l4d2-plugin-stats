package server

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

type testDashboardStore struct{}

func (testDashboardStore) Ping(context.Context) error                      { return nil }
func (testDashboardStore) MigrationVersion(context.Context) (int64, error) { return 1, nil }
func (testDashboardStore) Bootstrap(context.Context, config.BootstrapConfig, bool) (bool, error) {
	return false, nil
}
func (testDashboardStore) Site(context.Context) (store.Site, error) {
	return store.Site{Title: "Test", Links: []store.FooterLink{}}, nil
}
func (testDashboardStore) PrimaryServer(context.Context) (*store.GameServer, error) { return nil, nil }
func (testDashboardStore) Close() error                                             { return nil }

type testStatsStore struct{ overview store.Overview }

func (testStatsStore) Ping(context.Context) error                   { return nil }
func (testStatsStore) SchemaVersion(context.Context) (int64, error) { return 1, nil }
func (s testStatsStore) Overview(context.Context, time.Time) (store.Overview, error) {
	return s.overview, nil
}
func (testStatsStore) Close() error { return nil }

type testStatusProvider struct{}

func (testStatusProvider) PrimaryStatus(context.Context) (*store.ServerStatus, error) {
	return &store.ServerStatus{ConfiguredName: "Main", ConnectAddress: "127.0.0.1:27015", Online: true}, nil
}

func TestAPIRoutesStayJSONAndSPAOnlyFallsBackForClientRoutes(t *testing.T) {
	cfg := config.Default()
	stats := testStatsStore{overview: store.Overview{Core: store.CoreOverview{TotalPlayers: 2}}}
	app := New(&cfg, Dependencies{
		Dashboard: testDashboardStore{}, Stats: stats,
		Overview: service.NewOverviewService(stats, time.Minute), Status: testStatusProvider{},
		Logger: zap.NewNop(), Assets: fstest.MapFS{
			"index.html":    &fstest.MapFile{Data: []byte("<html>dashboard</html>")},
			"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
		},
	})

	for _, path := range []string{"/api/v1/health/live", "/api/v1/health/ready", "/api/v1/dashboard/overview"} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != 200 || !strings.Contains(response.Header.Get("Content-Type"), "application/json") || !strings.Contains(string(body), "request_id") {
			t.Fatalf("GET %s = %d %s %s", path, response.StatusCode, response.Header.Get("Content-Type"), body)
		}
	}

	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/not-real", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 404 || !strings.Contains(response.Header.Get("Content-Type"), "application/json") || strings.Contains(string(body), "<html>") {
		t.Fatalf("API 404 = %d %s %s", response.StatusCode, response.Header.Get("Content-Type"), body)
	}

	response, err = app.Test(httptest.NewRequest("GET", "/players/7656119", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	if response.StatusCode != 200 || !strings.Contains(string(body), "dashboard") || response.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("SPA fallback = %d %s, cache=%q", response.StatusCode, body, response.Header.Get("Cache-Control"))
	}

	response, err = app.Test(httptest.NewRequest("GET", "/missing.js", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 404 {
		t.Fatalf("missing asset status = %d", response.StatusCode)
	}
}
