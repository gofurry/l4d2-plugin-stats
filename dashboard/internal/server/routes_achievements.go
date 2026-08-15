package server

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func registerAchievementRoutes(api fiber.Router, achievements *service.AchievementService, authService *auth.Service, dashboard store.DashboardStore) {
	if achievements == nil {
		return
	}
	api.Get("/players/:steam_id/achievements", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		viewer := steamIdentity(c, authService)
		result, err := achievements.Player(c.Context(), steamID, viewer == steamID)
		if err != nil {
			return sendError(c, 503, "achievements_unavailable", "achievements are temporarily unavailable")
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		return sendData(c, 200, result)
	})
	api.Get("/players/:steam_id/badges", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		result, err := achievements.Badges(c.Context(), steamID)
		if err != nil {
			return sendError(c, 503, "badges_unavailable", "player badges are temporarily unavailable")
		}
		return sendData(c, 200, result)
	})
	api.Put("/me/badge-showcase", func(c fiber.Ctx) error {
		steamID := steamIdentity(c, authService)
		if steamID == "" {
			return sendError(c, 401, "steam_unauthorized", "Steam login is required")
		}
		if err := authService.ValidateSteamBadgeEdit(c.Context(), c.Cookies(steamBadgeEditCookie), steamID); err != nil {
			return sendError(c, 403, "steam_reauthentication_required", "Recent Steam verification is required")
		}
		if !samePublicOrigin(c, dashboard) {
			return sendError(c, 403, "steam_badge_origin_invalid", "Badge showcase request origin is invalid")
		}
		var body struct {
			Items []store.BadgeShowcaseSlot `json:"items"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return sendError(c, 400, "invalid_body", "request body is invalid")
		}
		for index := range body.Items {
			body.Items[index].AchievementKey = strings.TrimSpace(body.Items[index].AchievementKey)
			if body.Items[index].AchievementKey == "" {
				return sendError(c, 400, "invalid_badge_showcase", "achievement_key is required")
			}
		}
		result, err := achievements.SetBadges(c.Context(), steamID, body.Items)
		if err != nil {
			return sendError(c, 400, "invalid_badge_showcase", err.Error())
		}
		return sendData(c, 200, result)
	})
}

func samePublicOrigin(c fiber.Ctx, dashboard store.DashboardStore) bool {
	if dashboard == nil {
		return false
	}
	settings, err := dashboard.SiteSettings(c.Context())
	if err != nil {
		return false
	}
	want, err := url.Parse(strings.TrimSpace(settings.PublicOrigin))
	if err != nil || want.Scheme == "" || want.Host == "" {
		return false
	}
	got, err := url.Parse(strings.TrimSpace(c.Get("Origin")))
	if err != nil || got.Scheme == "" || got.Host == "" || got.User != nil || got.RawQuery != "" || got.Fragment != "" {
		return false
	}
	return strings.EqualFold(got.Scheme, want.Scheme) && strings.EqualFold(got.Host, want.Host)
}

func steamIdentity(c fiber.Ctx, authService *auth.Service) string {
	if authService == nil {
		return ""
	}
	raw := c.Cookies(steamIdentityCookie)
	if raw == "" {
		return ""
	}
	steamID, err := authService.ValidateSteamIdentity(c.Context(), raw)
	if err != nil {
		return ""
	}
	return steamID
}
