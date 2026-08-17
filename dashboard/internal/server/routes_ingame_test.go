package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	ingameassets "github.com/gofurry/l4d2-plugin-stats/dashboard/ingame"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type ingameMIMEStore struct{}

func (ingameMIMEStore) IngameSettings(context.Context) (store.IngameSettings, error) {
	return store.IngameSettings{
		Enabled: true, HighlightMetrics: [3]string{"active_play_seconds", "special_kills", "rescues"},
		HomeCacheSeconds: 30, PlayerCacheSeconds: 60, RankingCacheSeconds: 120, ContentCacheSeconds: 300,
	}, nil
}

func (ingameMIMEStore) IngameServerSettings(context.Context, string) (store.IngameServerSettings, error) {
	return store.IngameServerSettings{
		TitleMode: "inherit", DescriptionMode: "inherit", BannerMode: "inherit",
		WebsiteMode: "inherit", HighlightMode: "inherit",
	}, nil
}

func (ingameMIMEStore) ListServers(context.Context) ([]store.GameServer, error) {
	return []store.GameServer{{ID: "server-id", DisplayName: "MIME Test", Enabled: true}}, nil
}

func (ingameMIMEStore) ListSiteDocuments(context.Context, bool) ([]store.SiteDocument, error) {
	return []store.SiteDocument{}, nil
}

func (ingameMIMEStore) GetSiteDocument(context.Context, string, bool) (store.SiteDocument, error) {
	return store.SiteDocument{}, nil
}

func (ingameMIMEStore) ListServerDocuments(context.Context, string) ([]store.ServerDocument, error) {
	return []store.ServerDocument{}, nil
}

func (ingameMIMEStore) GetServerDocument(context.Context, string, string) (store.ServerDocument, error) {
	return store.ServerDocument{Mode: "inherit"}, nil
}

func (ingameMIMEStore) PlayerProfileVisibility(context.Context, string) (store.PlayerProfileVisibility, error) {
	return store.PlayerProfileVisibility{}, nil
}

func (ingameMIMEStore) ListAnnouncements(context.Context, store.AnnouncementFilter) (store.AnnouncementPage, error) {
	return store.AnnouncementPage{}, nil
}

func (ingameMIMEStore) GetAnnouncement(context.Context, string) (store.Announcement, error) {
	return store.Announcement{}, nil
}

func (ingameMIMEStore) SiteSettings(context.Context) (store.SiteSettings, error) {
	return store.SiteSettings{A2SRefreshSeconds: 30}, nil
}

type ingameMIMEStatuses struct{}

func (ingameMIMEStatuses) CachedStatuses(context.Context) ([]store.ServerStatus, error) {
	return []store.ServerStatus{{
		ServerID: "server-id", ServerKey: "valid-server-key", DisplayName: "MIME Test",
		Online: true, Map: "c1m1_hotel", Players: 1, MaxPlayers: 8, LastSuccessAt: time.Now(),
	}}, nil
}

func TestIngameRoutesSetExplicitContentTypes(t *testing.T) {
	renderer, err := ingameassets.NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	portal := service.NewIngameService(ingameMIMEStore{}, ingameMIMEStatuses{}, nil, nil, nil)
	app := fiber.New()
	registerIngameRoutes(app, portal, renderer)
	defer app.Shutdown()

	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
		body        string
	}{
		{name: "home HTML", path: "/ingame?server=valid-server-key", status: http.StatusOK, contentType: fiber.MIMETextHTMLCharsetUTF8, body: "<!doctype html>"},
		{name: "error HTML", path: "/ingame?server=invalid", status: http.StatusNotFound, contentType: fiber.MIMETextHTMLCharsetUTF8, body: "无法识别服务器"},
		{name: "CSS", path: "/ingame/assets/v1.3.4/ingame.css", status: http.StatusOK, contentType: fiber.MIMETextCSSCharsetUTF8},
		{name: "PNG", path: "/ingame/assets/v1.3.4/achievements.png", status: http.StatusOK, contentType: "image/png"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, requestErr := app.Test(httptest.NewRequest(http.MethodGet, test.path, nil))
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			defer response.Body.Close()
			if response.StatusCode != test.status {
				t.Fatalf("GET %s status=%d, want %d", test.path, response.StatusCode, test.status)
			}
			contentType := response.Header.Get(fiber.HeaderContentType)
			if contentType != test.contentType {
				t.Fatalf("GET %s Content-Type=%q, want %q", test.path, contentType, test.contentType)
			}
			if contentType == fiber.MIMEOctetStream || strings.HasPrefix(contentType, fiber.MIMEOctetStream+";") {
				t.Fatalf("GET %s regressed to %q", test.path, contentType)
			}
			if test.body != "" {
				buffer := make([]byte, 8192)
				count, _ := response.Body.Read(buffer)
				if !strings.Contains(string(buffer[:count]), test.body) {
					t.Fatalf("GET %s body does not contain %q", test.path, test.body)
				}
			}
		})
	}
}
