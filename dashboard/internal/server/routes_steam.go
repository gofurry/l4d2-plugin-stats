package server

import (
	"crypto/subtle"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

const steamStateCookie = "l4d2_stats_steam_state"
const steamIdentityCookie = "l4d2_stats_steam_identity"

func registerSteamRoutes(api fiber.Router, dashboard store.DashboardStore, authService *auth.Service, logger *zap.Logger) {
	api.Get("/steam/login", func(c fiber.Ctx) error {
		settings, err := dashboard.SiteSettings(c.Context())
		if err != nil {
			return sendError(c, 503, "site_unavailable", "site settings are unavailable")
		}
		if !settings.SteamOpenIDEnabled || settings.PublicOrigin == "" {
			return sendError(c, 503, "steam_openid_unavailable", "Steam OpenID is not enabled")
		}
		state, err := auth.RandomToken(24)
		if err != nil {
			return err
		}
		verifier, err := openidVerifier(settings)
		if err != nil {
			return sendError(c, 500, "steam_openid_config_invalid", "Steam OpenID configuration is invalid")
		}
		loginURL, err := verifier.LoginURL(state)
		if err != nil {
			return err
		}
		c.Cookie(&fiber.Cookie{Name: steamStateCookie, Value: state, Path: "/api/v1/steam", HTTPOnly: true, Secure: isHTTPS(settings.PublicOrigin), SameSite: "Lax", MaxAge: 600, Expires: time.Now().Add(10 * time.Minute)})
		return redirect(c, loginURL)
	})
	api.Get("/steam/callback", func(c fiber.Ctx) error {
		settings, err := dashboard.SiteSettings(c.Context())
		if err != nil || !settings.SteamOpenIDEnabled || settings.PublicOrigin == "" {
			return sendError(c, 503, "steam_openid_unavailable", "Steam OpenID is not enabled")
		}
		verifier, err := openidVerifier(settings)
		if err != nil {
			return sendError(c, 500, "steam_openid_config_invalid", "Steam OpenID configuration is invalid")
		}
		callbackURL, parseErr := url.Parse(c.OriginalURL())
		if parseErr != nil {
			return sendError(c, 400, "steam_openid_invalid", "Steam OpenID callback is invalid")
		}
		values, err := url.ParseQuery(callbackURL.RawQuery)
		if err != nil {
			return sendError(c, 400, "steam_openid_invalid", "Steam OpenID callback is invalid")
		}
		identity, err := verifier.VerifyValues(c.Context(), values)
		if err != nil {
			logger.Warn("Steam OpenID verification failed", zap.String("request_id", c.RequestID()), zap.Error(err))
			return sendError(c, 401, "steam_openid_invalid", "Steam identity verification failed")
		}
		state := c.Cookies(steamStateCookie)
		if state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(identity.State)) != 1 {
			return sendError(c, 401, "steam_state_invalid", "Steam login state is invalid or expired")
		}
		token, err := authService.SignSteamIdentity(c.Context(), identity.SteamID)
		if err != nil {
			return sendError(c, 503, "steam_identity_unavailable", "Steam identity could not be issued")
		}
		clearSteamCookie(c, steamStateCookie, isHTTPS(settings.PublicOrigin))
		c.Cookie(&fiber.Cookie{Name: steamIdentityCookie, Value: token, Path: "/api/v1/steam", HTTPOnly: true, Secure: isHTTPS(settings.PublicOrigin), SameSite: "Lax", MaxAge: 300, Expires: time.Now().Add(5 * time.Minute)})
		return redirect(c, "/player")
	})
	api.Get("/steam/identity", func(c fiber.Ctx) error {
		raw := c.Cookies(steamIdentityCookie)
		if raw == "" {
			return sendData(c, 200, nil)
		}
		steamID, err := authService.ValidateSteamIdentity(c.Context(), raw)
		if err != nil {
			secure := false
			if settings, settingsErr := dashboard.SiteSettings(c.Context()); settingsErr == nil {
				secure = isHTTPS(settings.PublicOrigin)
			}
			clearSteamCookie(c, steamIdentityCookie, secure)
			return sendData(c, 200, nil)
		}
		secure := false
		if settings, settingsErr := dashboard.SiteSettings(c.Context()); settingsErr == nil {
			secure = isHTTPS(settings.PublicOrigin)
		}
		clearSteamCookie(c, steamIdentityCookie, secure)
		c.Set(fiber.HeaderCacheControl, "no-store")
		return sendData(c, 200, fiber.Map{"steam_id": steamID})
	})
}

func isHTTPS(origin string) bool { return strings.HasPrefix(strings.ToLower(origin), "https://") }
func redirect(c fiber.Ctx, location string) error {
	c.Set(fiber.HeaderLocation, location)
	return c.SendStatus(fiber.StatusFound)
}
func clearSteamCookie(c fiber.Ctx, name string, secure bool) {
	c.Cookie(&fiber.Cookie{Name: name, Value: "", Path: "/api/v1/steam", HTTPOnly: true, Secure: secure, SameSite: "Lax", MaxAge: -1, Expires: time.Unix(1, 0)})
}
