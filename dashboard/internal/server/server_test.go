package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

type testDashboardStore struct{}

func (testDashboardStore) Ping(context.Context) error                      { return nil }
func (testDashboardStore) MigrationVersion(context.Context) (int64, error) { return 5, nil }
func (testDashboardStore) Site(context.Context) (store.Site, error) {
	return store.Site{Language: "zh-CN", Links: []store.FooterLink{}}, nil
}

func TestAdminJWTAndFiberCSRFProtectWrites(t *testing.T) {
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	authService, setupToken, err := auth.New(ctx, dashboard)
	if err != nil {
		t.Fatal(err)
	}
	if err := authService.Setup(ctx, setupToken, "admin", "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Monitor.Enabled = true
	stats := testStatsStore{}
	app := New(&cfg, Dependencies{
		Dashboard: dashboard, Stats: stats, Overview: service.NewOverviewService(stats, time.Minute),
		Status: testStatusProvider{}, Players: service.NewPlayerService(stats), Auth: authService,
		Logger: zap.NewNop(), Assets: fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("dashboard")}},
	})
	defer app.Shutdown()

	unauthorizedMonitor, _ := app.Test(httptest.NewRequest(http.MethodGet, adminMonitorPath, nil))
	if unauthorizedMonitor.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized monitor status=%d", unauthorizedMonitor.StatusCode)
	}

	unauthorized, _ := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/site", nil))
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.StatusCode)
	}
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"username":"admin","password":"correct horse battery staple"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse, err := app.Test(loginRequest)
	if err != nil || loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("login status=%d err=%v", loginResponse.StatusCode, err)
	}
	var adminAuthCookie *http.Cookie
	for _, cookie := range loginResponse.Cookies() {
		if cookie.Name == adminCookie {
			adminAuthCookie = cookie
		}
	}
	if adminAuthCookie == nil || !adminAuthCookie.HttpOnly || adminAuthCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("admin cookie=%#v", adminAuthCookie)
	}

	monitorRequest := httptest.NewRequest(http.MethodGet, adminMonitorPath, nil)
	monitorRequest.AddCookie(adminAuthCookie)
	monitorResponse, err := app.Test(monitorRequest)
	if err != nil || monitorResponse.StatusCode != http.StatusOK {
		t.Fatalf("monitor status=%d err=%v", monitorResponse.StatusCode, err)
	}
	if contentType := monitorResponse.Header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("monitor content type=%q", contentType)
	}
	if policy := monitorResponse.Header.Get("Content-Security-Policy"); !strings.Contains(policy, "script-src 'unsafe-inline'") {
		t.Fatalf("monitor content security policy=%q", policy)
	}

	csrfRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth/csrf", nil)
	csrfRequest.AddCookie(adminAuthCookie)
	csrfResponse, err := app.Test(csrfRequest)
	if err != nil || csrfResponse.StatusCode != http.StatusOK {
		t.Fatalf("csrf status=%d err=%v", csrfResponse.StatusCode, err)
	}
	var csrfCookieValue string
	for _, cookie := range csrfResponse.Cookies() {
		if cookie.Name == csrfCookie {
			csrfCookieValue = cookie.Value
		}
	}
	var payload struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(csrfResponse.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if csrfCookieValue == "" || payload.Data.Token != csrfCookieValue {
		t.Fatalf("csrf token response=%q cookie=%q", payload.Data.Token, csrfCookieValue)
	}

	settingsBody := `{"language":"zh-CN","browser_title":"L4D2 Stats","theme":"dark","footer_enabled":false,"background_image_url":"","public_origin":"","steam_openid_enabled":false,"a2s_refresh_seconds":30,"a2s_jitter_seconds":2,"a2s_retry_count":1,"footer_links":[]}`
	withoutCSRF := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site", strings.NewReader(settingsBody))
	withoutCSRF.Header.Set("Content-Type", "application/json")
	withoutCSRF.AddCookie(adminAuthCookie)
	response, _ := app.Test(withoutCSRF)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("write without CSRF status=%d", response.StatusCode)
	}
	withCSRF := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site", strings.NewReader(settingsBody))
	withCSRF.Header.Set("Content-Type", "application/json")
	withCSRF.Header.Set("X-Csrf-Token", csrfCookieValue)
	withCSRF.AddCookie(adminAuthCookie)
	withCSRF.AddCookie(&http.Cookie{Name: csrfCookie, Value: csrfCookieValue})
	response, err = app.Test(withCSRF)
	if err != nil || response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("write with CSRF status=%d err=%v body=%s", response.StatusCode, err, body)
	}

	createAnnouncement := httptest.NewRequest(http.MethodPost, "/api/v1/admin/announcements", strings.NewReader(`{"title":"Update","content_markdown":"**Ready**"}`))
	createAnnouncement.Header.Set("Content-Type", "application/json")
	createAnnouncement.Header.Set("X-Csrf-Token", csrfCookieValue)
	createAnnouncement.AddCookie(adminAuthCookie)
	createAnnouncement.AddCookie(&http.Cookie{Name: csrfCookie, Value: csrfCookieValue})
	response, err = app.Test(createAnnouncement)
	if err != nil || response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("create announcement status=%d err=%v body=%s", response.StatusCode, err, body)
	}
	publicAnnouncements, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/announcements", nil))
	if err != nil || publicAnnouncements.StatusCode != http.StatusOK {
		t.Fatalf("public announcements status=%d err=%v", publicAnnouncements.StatusCode, err)
	}
}
func (testDashboardStore) SiteSettings(context.Context) (store.SiteSettings, error) {
	return store.SiteSettings{Language: "zh-CN", BrowserTitle: "L4D2 Stats", Theme: "light", A2SRefreshSeconds: 30, A2SJitterSeconds: 2, A2SRetryCount: 1}, nil
}
func (testDashboardStore) UpdateSite(context.Context, store.SiteSettings) error { return nil }
func (testDashboardStore) ListSiteDocuments(context.Context, bool) ([]store.SiteDocument, error) {
	return []store.SiteDocument{}, nil
}
func (testDashboardStore) GetSiteDocument(context.Context, string, bool) (store.SiteDocument, error) {
	return store.SiteDocument{}, sql.ErrNoRows
}
func (testDashboardStore) UpdateSiteDocument(_ context.Context, document store.SiteDocument) (store.SiteDocument, error) {
	return document, nil
}
func (testDashboardStore) ListServers(context.Context) ([]store.GameServer, error) {
	return []store.GameServer{}, nil
}
func (testDashboardStore) CreateServer(_ context.Context, s store.GameServer) (store.GameServer, error) {
	return s, nil
}
func (testDashboardStore) UpdateServer(context.Context, store.GameServer) error      { return nil }
func (testDashboardStore) SetServerEnabled(context.Context, string, bool) error      { return nil }
func (testDashboardStore) MoveServer(context.Context, string, string) error          { return nil }
func (testDashboardStore) DeleteServer(context.Context, string) error                { return nil }
func (testDashboardStore) AdminConfigured(context.Context) (bool, error)             { return false, nil }
func (testDashboardStore) CreateAdmin(context.Context, string, string, string) error { return nil }
func (testDashboardStore) Admin(context.Context) (*store.AdminAccount, error)        { return nil, nil }
func (testDashboardStore) UpdateAdminUsername(context.Context, string) error         { return nil }
func (testDashboardStore) UpdateAdminPassword(context.Context, string) error         { return nil }
func (testDashboardStore) ListAnnouncements(context.Context, store.AnnouncementFilter) (store.AnnouncementPage, error) {
	return store.AnnouncementPage{Items: []store.Announcement{}}, nil
}
func (testDashboardStore) ListAnnouncementYears(context.Context) ([]int, error) { return []int{}, nil }
func (testDashboardStore) GetAnnouncement(context.Context, string) (store.Announcement, error) {
	return store.Announcement{}, nil
}
func (testDashboardStore) CreateAnnouncement(_ context.Context, value store.Announcement) (store.Announcement, error) {
	return value, nil
}
func (testDashboardStore) UpdateAnnouncement(_ context.Context, value store.Announcement) (store.Announcement, error) {
	return value, nil
}
func (testDashboardStore) DeleteAnnouncement(context.Context, string) error { return nil }
func (testDashboardStore) Close() error                                     { return nil }

type testStatsStore struct{ overview store.Overview }

func (testStatsStore) Ping(context.Context) error                   { return nil }
func (testStatsStore) SchemaVersion(context.Context) (int64, error) { return 1, nil }
func (s testStatsStore) Overview(context.Context, time.Time) (store.Overview, error) {
	return s.overview, nil
}
func (testStatsStore) PlayerSummary(context.Context, string) (*store.PlayerSummary, error) {
	return nil, nil
}
func (testStatsStore) SearchPlayers(context.Context, string, int32) ([]store.PlayerIdentity, error) {
	return nil, nil
}
func (testStatsStore) PlayerPVE(context.Context, string, int64) (store.PlayerPVE, error) {
	return store.PlayerPVE{}, nil
}
func (testStatsStore) PlayerVersus(context.Context, string, int64) (store.PlayerVersus, error) {
	return store.PlayerVersus{}, nil
}
func (testStatsStore) PlayerActivity(context.Context, string, int64) (store.PlayerActivity, error) {
	return store.PlayerActivity{}, nil
}
func (testStatsStore) PlayerSessions(context.Context, string, int64, string, int32) ([]store.PlayerSession, error) {
	return nil, nil
}
func (testStatsStore) PlayerChapters(context.Context, string, int64, string, int32) ([]store.PlayerChapter, error) {
	return nil, nil
}
func (testStatsStore) Close() error { return nil }

type testStatusProvider struct{}

func (testStatusProvider) Statuses(context.Context) ([]store.ServerStatus, error) {
	return []store.ServerStatus{{ServerID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Online: true}}, nil
}
func (testStatusProvider) LastStatus(context.Context, string) (store.ServerStatus, bool, error) {
	return store.ServerStatus{ServerID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Online: true}, true, nil
}
func (testStatusProvider) RefreshStatus(context.Context, string) (store.ServerStatus, error) {
	return store.ServerStatus{ServerID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Online: true}, nil
}
func (testStatusProvider) InvalidateServer(string) {}

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

	for _, path := range []string{"/api/v1/health/live", "/api/v1/health/ready", "/api/v1/dashboard/overview", "/api/v1/servers/status"} {
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
