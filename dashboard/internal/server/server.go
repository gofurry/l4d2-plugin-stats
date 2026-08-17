package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	ingameassets "github.com/gofurry/l4d2-plugin-stats/dashboard/ingame"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

type Dependencies struct {
	Dashboard    store.DashboardStore
	Profiles     store.DashboardProfileStore
	Stats        store.StatsStore
	Overview     *service.OverviewService
	Status       store.ServerStatusProvider
	Players      *service.PlayerService
	Analysis     *service.AnalysisService
	Rankings     *service.RankingService
	Achievements *service.AchievementService
	Data         *service.DataMaintenanceService
	Auth         *auth.Service
	Logger       *zap.Logger
	Assets       fs.FS
	Ingame       *service.IngameService
	IngameRender *ingameassets.Renderer
}

func New(cfg *config.Config, deps Dependencies) *fiber.App {
	profiles := deps.Profiles
	if profiles == nil {
		profiles, _ = deps.Dashboard.(store.DashboardProfileStore)
	}
	app := fiber.New(fiber.Config{
		AppName: "L4D2 Stats", BodyLimit: 1024 * 1024,
		ReadTimeout: cfg.Server.ReadTimeout.Value(), WriteTimeout: cfg.Server.WriteTimeout.Value(),
		IdleTimeout: cfg.Server.IdleTimeout.Value(),
		// The documented deployment keeps Fiber on loopback behind a local Nginx
		// process. Trust X-Real-IP only from loopback so login/setup throttling sees
		// the real client without allowing direct clients to spoof their address.
		TrustProxy:       true,
		ProxyHeader:      "X-Real-IP",
		TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			if fiberErr, ok := errors.AsType[*fiber.Error](err); ok {
				status = fiberErr.Code
			}
			logRequestFailure(deps.Logger, status, zap.String("request_id", c.RequestID()), zap.String("path", c.Path()), zap.Error(err))
			return sendError(c, status, "request_failed", http.StatusText(status))
		},
	})
	app.Use(requestid.New())
	runtimeMonitor := newRuntimeMonitor(cfg.Monitor, deps.Dashboard)
	if runtimeMonitor != nil {
		app.Use(runtimeMonitor.observe)
		app.Hooks().OnPreShutdown(runtimeMonitor.stop)
	}
	app.Use(recoverer.New())
	app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy:     "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: http: https:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'",
		CrossOriginEmbedderPolicy: "unsafe-none",
		CrossOriginResourcePolicy: "cross-origin",
	}))
	app.Use(compress.New())
	app.Use(func(c fiber.Ctx) error {
		if c.Path() == adminMonitorPath {
			return c.Next()
		}
		started := time.Now()
		err := c.Next()
		deps.Logger.Info("http request", zap.String("request_id", c.RequestID()), zap.String("method", c.Method()), zap.String("path", c.Path()), zap.Int("status", c.Response().StatusCode()), zap.Duration("duration", time.Since(started)))
		return err
	})
	app.Use(siteRateLimiter())

	api := app.Group("/api/v1")
	api.Get("/health/live", func(c fiber.Ctx) error {
		return sendData(c, fiber.StatusOK, fiber.Map{"status": "ok"})
	})
	api.Get("/health/ready", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		if err := deps.Dashboard.Ping(ctx); err != nil {
			return sendError(c, fiber.StatusServiceUnavailable, "dashboard_database_unavailable", "dashboard database is unavailable")
		}
		if err := deps.Stats.Ping(ctx); err != nil {
			return sendError(c, fiber.StatusServiceUnavailable, "stats_database_unavailable", "stats database is unavailable")
		}
		version, err := deps.Stats.SchemaVersion(ctx)
		if err != nil || version != store.StatsSchemaVersion {
			return sendError(c, fiber.StatusServiceUnavailable, "stats_schema_incompatible", "stats schema is incompatible")
		}
		return sendData(c, fiber.StatusOK, fiber.Map{"status": "ready", "stats_schema_version": version})
	})
	api.Get("/site", func(c fiber.Ctx) error {
		site, err := deps.Dashboard.Site(c.Context())
		if err != nil {
			return sendError(c, fiber.StatusServiceUnavailable, "site_unavailable", "site configuration is unavailable")
		}
		return sendData(c, fiber.StatusOK, site)
	})
	api.Get("/dashboard/overview", func(c fiber.Ctx) error {
		overview, err := deps.Overview.Get(c.Context())
		if err != nil {
			return sendError(c, fiber.StatusServiceUnavailable, "stats_unavailable", "statistics are temporarily unavailable")
		}
		return sendData(c, fiber.StatusOK, overview)
	})
	api.Get("/servers/status", func(c fiber.Ctx) error {
		statuses, err := deps.Status.Statuses(c.Context())
		if err != nil {
			return sendError(c, fiber.StatusServiceUnavailable, "server_status_unavailable", "server status is temporarily unavailable")
		}
		return sendData(c, fiber.StatusOK, statuses)
	})
	registerPlayerProfileRoutes(api, deps.Players, profiles, deps.Dashboard, deps.Auth)
	registerPlayerRoutes(api, deps.Players, deps.Analysis, deps.Achievements, profiles, deps.Auth)
	registerAchievementRoutes(api, deps.Achievements, deps.Auth, deps.Dashboard, profiles)
	registerAnalysisRoutes(api, deps.Analysis)
	registerRankingRoutes(api, deps.Rankings)
	registerAnnouncementRoutes(api, deps.Dashboard)
	registerSiteDocumentRoutes(api, deps.Dashboard)
	registerSteamRoutes(api, deps.Dashboard, deps.Auth, deps.Logger)
	registerAdminRoutes(api, deps.Dashboard, deps.Status, deps.Auth, deps.Data, deps.Achievements, deps.Logger, runtimeMonitor)
	api.All("/*", func(c fiber.Ctx) error {
		return sendError(c, fiber.StatusNotFound, "not_found", "API route not found")
	})

	registerSEORoutes(app, deps.Dashboard)
	registerIngameRoutes(app, deps.Ingame, deps.IngameRender)
	assets := staticHandler(deps.Assets, deps.Dashboard)
	app.Get("/", assets)
	app.Get("/*", assets)
	return app
}

func logRequestFailure(logger *zap.Logger, status int, fields ...zap.Field) {
	// Browsers may speculatively open a connection and never send an HTTP
	// request. Fiber reports the resulting first-byte timeout as 408 and may
	// retain a stale path in the pooled context; this is connection lifecycle
	// noise, not an application request failure.
	if status == fiber.StatusRequestTimeout {
		logger.Debug("http connection timed out", append(fields, zap.Int("status", status))...)
		return
	}
	logger.Error("http request failed", append(fields, zap.Int("status", status))...)
}
