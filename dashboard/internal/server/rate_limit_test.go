package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestSiteRateLimiterCoversAPIWithJSONResponse(t *testing.T) {
	app := fiber.New()
	app.Use(siteRateLimiter())
	app.Get("/api/v1/test", func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })
	for request := 0; request < siteRateLimitPerMinute; request++ {
		response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))
		if err != nil || response.StatusCode != http.StatusNoContent {
			t.Fatalf("request %d: status=%v err=%v", request, response.StatusCode, err)
		}
	}
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusTooManyRequests || !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("limited response: status=%d content-type=%q", response.StatusCode, response.Header.Get("Content-Type"))
	}
}
