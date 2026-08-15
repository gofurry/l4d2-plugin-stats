package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

const profileTestSteamID = "76561198000000001"

type profileStatsStore struct{ testStatsStore }

func (profileStatsStore) PlayerSummary(context.Context, string) (*store.PlayerSummary, error) {
	return &store.PlayerSummary{SteamID: profileTestSteamID, LastName: "Player", SessionCount: 9}, nil
}

func (profileStatsStore) PlayerPVE(context.Context, string, int64) (store.PlayerPVE, error) {
	return store.PlayerPVE{CommonKills: 123, Classes: []store.PVEInfectedClass{{ClassID: 1, Kills: 7}}, Equipment: []store.PVEEquipment{{EquipmentID: 1, Actions: 8}}}, nil
}

func TestPlayerProfileVisibilityRoutesAndSectionEnforcement(t *testing.T) {
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	if err := dashboard.CreateAdmin(ctx, "admin", "unused", "profile-test-signing-secret"); err != nil {
		t.Fatal(err)
	}
	if err := dashboard.UpdateSite(ctx, store.SiteSettings{
		Language: "zh-CN", BrowserTitle: "L4D2 Stats", Theme: "light", PublicOrigin: "https://stats.example.com",
		A2SRefreshSeconds: 30, A2SJitterSeconds: 2, A2SRetryCount: 1,
	}); err != nil {
		t.Fatal(err)
	}
	authService, _, err := auth.New(ctx, dashboard)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := authService.SignSteamIdentity(ctx, profileTestSteamID)
	if err != nil {
		t.Fatal(err)
	}
	players := service.NewPlayerService(profileStatsStore{})
	app := fiber.New()
	api := app.Group("/api/v1")
	registerPlayerProfileRoutes(api, players, dashboard, dashboard, authService)
	registerPlayerRoutes(api, players, nil, nil, dashboard, authService)

	profile := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/profile", "", "")
	if profile.StatusCode != http.StatusOK {
		t.Fatalf("default profile status=%d", profile.StatusCode)
	}
	var profilePayload struct {
		Data playerPublicProfile `json:"data"`
	}
	if err := json.NewDecoder(profile.Body).Decode(&profilePayload); err != nil {
		t.Fatal(err)
	}
	if profilePayload.Data.Self || len(profilePayload.Data.VisibleSections) != 3 || profilePayload.Data.PlayerName != "Player" {
		t.Fatalf("default profile=%#v", profilePayload.Data)
	}

	if response := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/summary", "", ""); response.StatusCode != http.StatusOK {
		t.Fatalf("default overview status=%d", response.StatusCode)
	}
	if response := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/pve", "", ""); response.StatusCode != http.StatusForbidden {
		t.Fatalf("private PvE status=%d", response.StatusCode)
	}
	if response := performProfileRequest(t, app, http.MethodPut, "/api/v1/me/profile-visibility", `{"visible_sections":["pve-details"]}`, ""); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous update status=%d", response.StatusCode)
	}
	crossOrigin := httptest.NewRequest(http.MethodPut, "/api/v1/me/profile-visibility", strings.NewReader(`{"visible_sections":["overview"]}`))
	crossOrigin.Header.Set("Content-Type", "application/json")
	crossOrigin.Header.Set("Origin", "https://evil.example")
	crossOrigin.AddCookie(&http.Cookie{Name: steamIdentityCookie, Value: identity})
	crossResponse, err := app.Test(crossOrigin)
	if err != nil || crossResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin update status=%d err=%v", crossResponse.StatusCode, err)
	}
	if response := performProfileRequest(t, app, http.MethodPut, "/api/v1/me/profile-visibility", `{"visible_sections":["unknown"]}`, identity); response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown section update status=%d", response.StatusCode)
	}

	update := performProfileRequest(t, app, http.MethodPut, "/api/v1/me/profile-visibility", `{"visible_sections":["pve-details"]}`, identity)
	if update.StatusCode != http.StatusOK {
		t.Fatalf("authenticated update status=%d", update.StatusCode)
	}
	ownerProfile := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/profile", "", identity)
	if err := json.NewDecoder(ownerProfile.Body).Decode(&profilePayload); err != nil || !profilePayload.Data.Self || len(profilePayload.Data.VisibleSections) != 1 {
		t.Fatalf("owner profile=%#v err=%v", profilePayload.Data, err)
	}
	if response := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/pve?view=pve", "", ""); response.StatusCode != http.StatusForbidden {
		t.Fatalf("hidden PvE overview status=%d", response.StatusCode)
	}
	details := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/pve?view=pve-details", "", "")
	if details.StatusCode != http.StatusOK {
		t.Fatalf("public PvE details status=%d", details.StatusCode)
	}
	var detailsPayload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(details.Body).Decode(&detailsPayload); err != nil {
		t.Fatal(err)
	}
	if _, leaked := detailsPayload.Data["common_kills"]; leaked || detailsPayload.Data["infected_classes"] == nil || detailsPayload.Data["equipment"] == nil {
		t.Fatalf("details payload=%#v", detailsPayload.Data)
	}
	if response := performProfileRequest(t, app, http.MethodGet, "/api/v1/players/"+profileTestSteamID+"/pve", "", identity); response.StatusCode != http.StatusOK {
		t.Fatalf("owner private PvE status=%d", response.StatusCode)
	}
}

func TestVersusPayloadIsScopedToRequestedTab(t *testing.T) {
	value := store.PlayerVersus{
		SurvivorCommonKills: 11, HumanSpecialKills: 12, InfectedSpawns: 13, DamageToHumanSurvivors: 14,
		SurvivorClasses: []store.VersusSurvivorClass{{ClassID: 1}}, InfectedClasses: []store.VersusInfectedClass{{ClassID: 2}},
	}
	survivor, err := versusPayload(value, store.PlayerProfileVersusSurvivor)
	if err != nil || survivor["survivor_common_kills"] == nil || survivor["infected_spawns"] != nil || survivor["survivor_classes"] != nil {
		t.Fatalf("survivor payload=%#v err=%v", survivor, err)
	}
	infected, err := versusPayload(value, store.PlayerProfileVersusInfected)
	if err != nil || infected["infected_spawns"] == nil || infected["survivor_common_kills"] != nil || infected["infected_classes"] != nil {
		t.Fatalf("infected payload=%#v err=%v", infected, err)
	}
}

func performProfileRequest(t *testing.T, app *fiber.App, method, path, body, identity string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://stats.example.com")
	}
	if identity != "" {
		request.AddCookie(&http.Cookie{Name: steamIdentityCookie, Value: identity})
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
