package server

import (
	"errors"
	"io/fs"
	"mime"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func staticHandler(dist fs.FS, dashboard store.DashboardStore) fiber.Handler {
	return func(c fiber.Ctx) error {
		requested := strings.TrimPrefix(c.Path(), "/")
		requested = path.Clean(requested)
		if requested == "." || requested == "" {
			requested = "index.html"
		}
		if strings.HasPrefix(requested, "../") || requested == ".." {
			return fiber.ErrNotFound
		}
		body, err := fs.ReadFile(dist, requested)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) || path.Ext(requested) != "" {
				return fiber.ErrNotFound
			}
			requested = "index.html"
			body, err = fs.ReadFile(dist, requested)
			if err != nil {
				return err
			}
		}
		if strings.HasPrefix(requested, "assets/") {
			c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		} else {
			c.Set(fiber.HeaderCacheControl, "no-cache")
		}
		if contentType := mime.TypeByExtension(path.Ext(requested)); contentType != "" {
			c.Set(fiber.HeaderContentType, contentType)
		}
		if requested == "index.html" {
			if settings, settingsErr := dashboard.SiteSettings(c.Context()); settingsErr == nil {
				body = applySEOMetadata(body, settings, c.Path())
			}
		}
		return c.Send(body)
	}
}
