package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/gofurry/monitor"
)

const adminMonitorPath = "/api/v1/admin/monitor"

type runtimeMonitor struct {
	monitor *monitor.Monitor
	handler fiber.Handler
}

func newRuntimeMonitor(cfg config.MonitorConfig, dashboard store.DashboardStore) *runtimeMonitor {
	if !cfg.Enabled {
		return nil
	}

	language := "zh-CN"
	if settings, err := dashboard.SiteSettings(context.Background()); err == nil && settings.Language == "en" {
		language = "en"
	}
	title := "运行监控"
	description := "Dashboard 进程、宿主机资源与 HTTP 请求状态"
	if language == "en" {
		title = "Runtime monitor"
		description = "Dashboard process, host resources, and HTTP request status"
	}

	instance := monitor.NewMonitor(http.NotFoundHandler(), monitor.Config{
		Path:                adminMonitorPath,
		Title:               title,
		Description:         description,
		Footer:              "L4D2 Stats",
		DefaultLanguage:     language,
		DefaultTheme:        "light",
		Background:          "solid",
		DefaultSampleWindow: 60,
		DiskPaths:           cfg.DiskPaths,
		Refresh:             cfg.Refresh.Value(),
		APIOnly:             false,
	})
	return &runtimeMonitor{monitor: instance, handler: adaptor.HTTPHandler(instance)}
}

func (m *runtimeMonitor) stop() error {
	if m != nil {
		m.monitor.Stop()
	}
	return nil
}

func (m *runtimeMonitor) observe(c fiber.Ctx) error {
	if m == nil || shouldIgnoreMonitorRequest(c.Path()) {
		return c.Next()
	}

	started := time.Now()
	m.monitor.RequestStarted()
	err := c.Next()
	status := c.Response().StatusCode()
	if err != nil {
		status = fiber.StatusInternalServerError
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			status = fiberErr.Code
		}
	}
	m.monitor.RequestFinished(status, time.Since(started))
	return err
}

func (m *runtimeMonitor) serve(c fiber.Ctx) error {
	// monitor embeds its trusted JavaScript and CSS into one self-contained
	// document. Override the global policy only for this authenticated endpoint.
	c.Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; img-src data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	return m.handler(c)
}

func shouldIgnoreMonitorRequest(path string) bool {
	return path == adminMonitorPath || strings.HasPrefix(path, "/api/v1/health/")
}
