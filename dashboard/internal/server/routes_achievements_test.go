package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type originDashboardStore struct{ testDashboardStore }

func (originDashboardStore) SiteSettings(context.Context) (store.SiteSettings, error) {
	return store.SiteSettings{PublicOrigin: "https://stats.example.com"}, nil
}

func TestBadgeShowcaseOriginMustMatchPublicOrigin(t *testing.T) {
	app := fiber.New()
	app.Put("/", func(c fiber.Ctx) error {
		if !samePublicOrigin(c, originDashboardStore{}) {
			return c.SendStatus(http.StatusForbidden)
		}
		return c.SendStatus(http.StatusNoContent)
	})
	for _, tc := range []struct {
		origin string
		status int
	}{
		{"https://stats.example.com", http.StatusNoContent},
		{"https://STATS.EXAMPLE.COM", http.StatusNoContent},
		{"http://stats.example.com", http.StatusForbidden},
		{"https://evil.example", http.StatusForbidden},
		{"", http.StatusForbidden},
	} {
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		response, err := app.Test(req)
		if err != nil || response.StatusCode != tc.status {
			t.Fatalf("origin %q status=%d err=%v want=%d", tc.origin, response.StatusCode, err, tc.status)
		}
	}
}
