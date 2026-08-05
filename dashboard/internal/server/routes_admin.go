package server

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	csrfmw "github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const adminCookie = "l4d2_stats_admin"
const csrfCookie = "l4d2_stats_csrf"

type adminRoutes struct {
	dashboard                  store.DashboardStore
	status                     store.ServerStatusProvider
	auth                       *auth.Service
	logger                     *zap.Logger
	loginLimiter, setupLimiter *auth.Limiter
	monitor                    *runtimeMonitor
}

func registerAdminRoutes(api fiber.Router, dashboard store.DashboardStore, status store.ServerStatusProvider, authService *auth.Service, logger *zap.Logger, runtimeMonitor *runtimeMonitor) {
	r := &adminRoutes{dashboard: dashboard, status: status, auth: authService, logger: logger, monitor: runtimeMonitor, loginLimiter: auth.NewLimiter(5, 15*time.Minute, 1024), setupLimiter: auth.NewLimiter(10, 15*time.Minute, 256)}
	api.Get("/setup/status", r.setupStatus)
	api.Post("/setup/admin", r.setupAdmin)
	api.Post("/admin/auth/login", r.login)
	if runtimeMonitor != nil {
		api.All("/admin/monitor", r.requireAdmin, runtimeMonitor.serve)
	}
	settings, _ := dashboard.SiteSettings(context.Background())
	csrfMiddleware := csrfmw.New(csrfmw.Config{
		CookieName: csrfCookie, CookiePath: "/api/v1/admin", CookieSameSite: "Strict",
		CookieSecure: isHTTPS(settings.PublicOrigin), CookieHTTPOnly: false, CookieSessionOnly: true,
		IdleTimeout: 30 * time.Minute,
		ErrorHandler: func(c fiber.Ctx, _ error) error {
			return sendError(c, 403, "csrf_invalid", "CSRF token is missing or invalid")
		},
	})
	admin := api.Group("/admin", r.requireAdmin, csrfMiddleware)
	admin.Get("/auth/me", r.me)
	admin.Get("/auth/csrf", r.csrf)
	admin.Post("/auth/logout", r.logout)
	admin.Get("/site", r.getSite)
	admin.Put("/site", r.putSite)
	admin.Get("/servers", r.listServers)
	admin.Post("/servers", r.createServer)
	admin.Put("/servers/:id", r.updateServer)
	admin.Patch("/servers/:id/enabled", r.setServerEnabled)
	admin.Post("/servers/:id/move", r.moveServer)
	admin.Get("/servers/:id/a2s", r.lastServerA2S)
	admin.Post("/servers/:id/a2s", r.refreshServerA2S)
	admin.Delete("/servers/:id", r.deleteServer)
	admin.Put("/account", r.updateAccount)
	admin.Put("/account/password", r.updatePassword)
	admin.Get("/announcements", r.listAnnouncements)
	admin.Post("/announcements", r.createAnnouncement)
	admin.Put("/announcements/:id", r.updateAnnouncement)
	admin.Delete("/announcements/:id", r.deleteAnnouncement)
	admin.Get("/site-documents", r.listSiteDocuments)
	admin.Put("/site-documents/:key", r.updateSiteDocument)
}

func (r *adminRoutes) setupStatus(c fiber.Ctx) error {
	required, expires, err := r.auth.SetupStatus(c.Context())
	if err != nil {
		return sendError(c, 503, "setup_unavailable", "setup status is unavailable")
	}
	data := fiber.Map{"required": required}
	if required && !expires.IsZero() {
		data["expires_at"] = expires.UTC()
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return sendData(c, 200, data)
}

func (r *adminRoutes) setupAdmin(c fiber.Ctx) error {
	if !r.setupLimiter.Allow(c.IP()) {
		return sendError(c, 429, "setup_rate_limited", "too many setup attempts")
	}
	var body struct {
		SetupToken string `json:"setup_token"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if err := validateUsername(body.Username); err != nil {
		return sendError(c, 400, "invalid_username", err.Error())
	}
	if err := validatePassword(body.Password); err != nil {
		return sendError(c, 400, "invalid_password", err.Error())
	}
	if err := r.auth.Setup(c.Context(), body.SetupToken, strings.TrimSpace(body.Username), body.Password); err != nil {
		r.logger.Warn("administrator setup failed", zap.String("request_id", c.RequestID()), zap.String("remote_ip", c.IP()))
		if errors.Is(err, auth.ErrSetupToken) {
			return sendError(c, 401, "setup_token_invalid", err.Error())
		}
		return sendError(c, 409, "setup_failed", "administrator could not be created")
	}
	r.logger.Info("administrator configured", zap.String("request_id", c.RequestID()), zap.String("remote_ip", c.IP()))
	return sendData(c, 201, fiber.Map{"configured": true})
}

func (r *adminRoutes) login(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	if !r.loginLimiter.Allow(c.IP()) {
		return sendError(c, 429, "login_rate_limited", "too many login attempts")
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	token, err := r.auth.Login(c.Context(), body.Username, body.Password)
	if err != nil {
		r.logger.Warn("administrator login failed", zap.String("request_id", c.RequestID()), zap.String("remote_ip", c.IP()))
		return sendError(c, 401, "invalid_credentials", "username or password is incorrect")
	}
	r.setAdminCookie(c, token)
	r.logger.Info("administrator login succeeded", zap.String("request_id", c.RequestID()), zap.String("remote_ip", c.IP()))
	return sendData(c, 200, fiber.Map{"authenticated": true})
}

func (r *adminRoutes) requireAdmin(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	raw := c.Cookies(adminCookie)
	if raw == "" {
		return sendError(c, 401, "admin_unauthorized", "administrator login is required")
	}
	admin, err := r.auth.ValidateAdmin(c.Context(), raw)
	if err != nil {
		return sendError(c, 401, "admin_unauthorized", "administrator login is invalid or expired")
	}
	c.Locals("admin", admin)
	return c.Next()
}

func (r *adminRoutes) me(c fiber.Ctx) error {
	admin := c.Locals("admin").(*store.AdminAccount)
	return sendData(c, 200, struct {
		*store.AdminAccount
		MonitorEnabled bool `json:"monitor_enabled"`
	}{AdminAccount: admin, MonitorEnabled: r.monitor != nil})
}
func (r *adminRoutes) csrf(c fiber.Ctx) error {
	return sendData(c, 200, fiber.Map{"token": csrfmw.TokenFromContext(c)})
}
func (r *adminRoutes) logout(c fiber.Ctx) error {
	if handler := csrfmw.HandlerFromContext(c); handler != nil {
		_ = handler.DeleteToken(c)
	}
	r.clearCookie(c, adminCookie, "/api/v1/admin")
	r.logger.Info("administrator logged out", zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"authenticated": false})
}

func (r *adminRoutes) getSite(c fiber.Ctx) error {
	settings, err := r.dashboard.SiteSettings(c.Context())
	if err != nil {
		return sendError(c, 503, "site_unavailable", "site settings are unavailable")
	}
	return sendData(c, 200, settings)
}
func (r *adminRoutes) putSite(c fiber.Ctx) error {
	var settings store.SiteSettings
	if err := c.Bind().Body(&settings); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if err := validateSite(&settings); err != nil {
		return sendError(c, 400, "invalid_site", err.Error())
	}
	if err := r.dashboard.UpdateSite(c.Context(), settings); err != nil {
		return sendError(c, 500, "site_update_failed", "site settings could not be saved")
	}
	r.logger.Info("site settings updated", zap.String("request_id", c.RequestID()))
	return sendData(c, 200, settings)
}

func (r *adminRoutes) listServers(c fiber.Ctx) error {
	servers, err := r.dashboard.ListServers(c.Context())
	if err != nil {
		return sendError(c, 503, "servers_unavailable", "server directory is unavailable")
	}
	return sendData(c, 200, servers)
}
func (r *adminRoutes) createServer(c fiber.Ctx) error {
	var body struct {
		DisplayName string `json:"display_name"`
		Address     string `json:"address"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	server := store.GameServer{DisplayName: body.DisplayName, Address: body.Address}
	if err := validateGameServer(&server); err != nil {
		return sendError(c, 400, "invalid_server", err.Error())
	}
	created, err := r.dashboard.CreateServer(c.Context(), server)
	if err != nil {
		return sendError(c, 409, "server_create_failed", "server address may already exist")
	}
	r.logger.Info("game server created", zap.String("server_id", created.ID), zap.String("request_id", c.RequestID()))
	return sendData(c, 201, created)
}
func (r *adminRoutes) updateServer(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_server_id", "server id is invalid")
	}
	var body struct {
		DisplayName string `json:"display_name"`
		Address     string `json:"address"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	server := store.GameServer{ID: id, DisplayName: body.DisplayName, Address: body.Address}
	if err := validateGameServer(&server); err != nil {
		return sendError(c, 400, "invalid_server", err.Error())
	}
	if err := r.dashboard.UpdateServer(c.Context(), server); errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "server_not_found", "server was not found")
	} else if err != nil {
		return sendError(c, 409, "server_update_failed", "server could not be updated")
	}
	r.status.InvalidateServer(id)
	r.logger.Info("game server updated", zap.String("server_id", server.ID), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, server)
}
func (r *adminRoutes) setServerEnabled(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_server_id", "server id is invalid")
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.Bind().Body(&body); err != nil || body.Enabled == nil {
		return sendError(c, 400, "invalid_body", "enabled is required")
	}
	if err := r.dashboard.SetServerEnabled(c.Context(), id, *body.Enabled); errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "server_not_found", "server was not found")
	} else if err != nil {
		return sendError(c, 500, "server_update_failed", "server could not be updated")
	}
	r.logger.Info("game server availability updated", zap.String("server_id", id), zap.Bool("enabled", *body.Enabled), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"enabled": *body.Enabled})
}
func (r *adminRoutes) moveServer(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_server_id", "server id is invalid")
	}
	var body struct {
		Direction string `json:"direction"`
	}
	if err := c.Bind().Body(&body); err != nil || (body.Direction != "up" && body.Direction != "down") {
		return sendError(c, 400, "invalid_direction", "direction must be up or down")
	}
	if err := r.dashboard.MoveServer(c.Context(), id, body.Direction); errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "server_not_found", "server was not found")
	} else if err != nil {
		return sendError(c, 500, "server_move_failed", "server could not be moved")
	}
	r.logger.Info("game server moved", zap.String("server_id", id), zap.String("direction", body.Direction), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"moved": true})
}

func (r *adminRoutes) lastServerA2S(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_server_id", "server id is invalid")
	}
	status, available, err := r.status.LastStatus(c.Context(), id)
	if errors.Is(err, store.ErrServerNotFound) {
		return sendError(c, 404, "server_not_found", "server was not found")
	}
	if err != nil {
		return sendError(c, 503, "server_status_unavailable", "server status is unavailable")
	}
	if !available {
		return sendData(c, 200, fiber.Map{"available": false})
	}
	return sendData(c, 200, fiber.Map{"available": true, "status": status})
}

func (r *adminRoutes) refreshServerA2S(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_server_id", "server id is invalid")
	}
	status, err := r.status.RefreshStatus(c.Context(), id)
	if errors.Is(err, store.ErrServerNotFound) {
		return sendError(c, 404, "server_not_found", "server was not found")
	}
	if err != nil {
		return sendError(c, 503, "a2s_query_failed", "A2S query could not be completed")
	}
	r.logger.Info("game server A2S query refreshed", zap.String("server_id", id), zap.Bool("online", status.Online), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, status)
}
func (r *adminRoutes) deleteServer(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_server_id", "server id is invalid")
	}
	if err := r.dashboard.DeleteServer(c.Context(), id); errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "server_not_found", "server was not found")
	} else if err != nil {
		return sendError(c, 500, "server_delete_failed", "server could not be deleted")
	}
	r.status.InvalidateServer(id)
	r.logger.Info("game server deleted", zap.String("server_id", id), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"deleted": true})
}

func (r *adminRoutes) updateAccount(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if err := validateUsername(body.Username); err != nil {
		return sendError(c, 400, "invalid_username", err.Error())
	}
	token, err := r.auth.ChangeUsername(c.Context(), strings.TrimSpace(body.Username))
	if err != nil {
		return sendError(c, 500, "account_update_failed", "administrator account could not be updated")
	}
	r.setAdminCookie(c, token)
	r.logger.Info("administrator username updated", zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"username": strings.TrimSpace(body.Username)})
}
func (r *adminRoutes) updatePassword(c fiber.Ctx) error {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if err := validatePassword(body.NewPassword); err != nil {
		return sendError(c, 400, "invalid_password", err.Error())
	}
	token, err := r.auth.ChangePassword(c.Context(), body.CurrentPassword, body.NewPassword)
	if err != nil {
		return sendError(c, 401, "current_password_invalid", "current password is incorrect")
	}
	r.setAdminCookie(c, token)
	r.logger.Info("administrator password updated", zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"updated": true})
}

func (r *adminRoutes) setAdminCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{Name: adminCookie, Value: token, Path: "/api/v1/admin", HTTPOnly: true, Secure: r.secureCookie(c.Context()), SameSite: "Strict", MaxAge: int(auth.AdminTokenTTL.Seconds()), Expires: time.Now().Add(auth.AdminTokenTTL)})
}
func (r *adminRoutes) clearCookie(c fiber.Ctx, name, path string) {
	c.Cookie(&fiber.Cookie{Name: name, Value: "", Path: path, HTTPOnly: name == adminCookie, Secure: r.secureCookie(c.Context()), SameSite: "Strict", MaxAge: -1, Expires: time.Unix(1, 0)})
}
func (r *adminRoutes) secureCookie(ctx context.Context) bool {
	settings, err := r.dashboard.SiteSettings(ctx)
	return err == nil && isHTTPS(settings.PublicOrigin)
}

func originOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
