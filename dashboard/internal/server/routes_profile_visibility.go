package server

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type playerPublicProfile struct {
	SteamID         string                       `json:"steam_id"`
	PlayerName      string                       `json:"player_name"`
	VisibleSections []store.PlayerProfileSection `json:"visible_sections"`
	Self            bool                         `json:"self"`
}

func registerPlayerProfileRoutes(api fiber.Router, players *service.PlayerService, profiles store.DashboardProfileStore, dashboard store.DashboardStore, authService *auth.Service) {
	api.Get("/players/:steam_id/profile", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		summary, err := players.Summary(c.Context(), steamID)
		if err != nil {
			return statsError(c, err)
		}
		if summary == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		if profiles == nil {
			return sendError(c, 503, "profile_visibility_unavailable", "player profile visibility is temporarily unavailable")
		}
		visibility, err := profiles.PlayerProfileVisibility(c.Context(), steamID)
		if err != nil {
			return sendError(c, 503, "profile_visibility_unavailable", "player profile visibility is temporarily unavailable")
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		return sendData(c, 200, playerPublicProfile{
			SteamID: steamID, PlayerName: summary.LastName, VisibleSections: visibility.VisibleSections,
			Self: steamIdentity(c, authService) == steamID,
		})
	})

	api.Put("/me/profile-visibility", func(c fiber.Ctx) error {
		steamID := steamIdentity(c, authService)
		if steamID == "" {
			return sendError(c, 401, "steam_unauthorized", "Steam login is required")
		}
		if !samePublicOrigin(c, dashboard) {
			return sendError(c, 403, "profile_visibility_origin_invalid", "Profile visibility request origin is invalid")
		}
		var body struct {
			VisibleSections []store.PlayerProfileSection `json:"visible_sections"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return sendError(c, 400, "invalid_body", "request body is invalid")
		}
		if len(body.VisibleSections) > len(store.PlayerProfileSections) {
			return sendError(c, 400, "invalid_profile_visibility", "too many profile sections")
		}
		if profiles == nil {
			return sendError(c, 503, "profile_visibility_unavailable", "player profile visibility is temporarily unavailable")
		}
		visibility, err := profiles.ReplacePlayerProfileVisibility(c.Context(), steamID, body.VisibleSections, time.Now().Unix())
		if err != nil {
			return sendError(c, 400, "invalid_profile_visibility", err.Error())
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		return sendData(c, 200, visibility)
	})
}

func playerSectionAllowed(c fiber.Ctx, profiles store.DashboardProfileStore, authService *auth.Service, steamID string, section store.PlayerProfileSection) (bool, error) {
	if steamIdentity(c, authService) == steamID {
		return true, nil
	}
	if profiles == nil {
		return false, fiber.ErrServiceUnavailable
	}
	visibility, err := profiles.PlayerProfileVisibility(c.Context(), steamID)
	if err != nil {
		return false, err
	}
	return visibility.Visible(section), nil
}

func requirePlayerSection(c fiber.Ctx, profiles store.DashboardProfileStore, authService *auth.Service, steamID string, section store.PlayerProfileSection) bool {
	allowed, err := playerSectionAllowed(c, profiles, authService, steamID, section)
	if err != nil {
		_ = sendError(c, 503, "profile_visibility_unavailable", "player profile visibility is temporarily unavailable")
		return false
	}
	if !allowed {
		_ = sendError(c, 403, "profile_section_private", "This player profile section is private")
		return false
	}
	return true
}
