package server

import (
	"encoding/xml"
	"html"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

const seoMarker = "<!--SEO_META-->"

func registerSEORoutes(app *fiber.App, dashboard store.DashboardStore) {
	app.Get("/robots.txt", func(c fiber.Ctx) error {
		settings, err := dashboard.SiteSettings(c.Context())
		if err != nil || !settings.SEOEnabled {
			c.Type("text/plain", "utf-8")
			return c.SendString("User-agent: *\nDisallow: /\n")
		}
		body := "User-agent: *\nAllow: /\nDisallow: /admin/\nDisallow: /player\nDisallow: /monitor\n"
		if settings.PublicOrigin != "" {
			body += "Sitemap: " + strings.TrimRight(settings.PublicOrigin, "/") + "/sitemap.xml\n"
		}
		c.Type("text/plain", "utf-8")
		return c.SendString(body)
	})
	app.Get("/sitemap.xml", func(c fiber.Ctx) error {
		settings, err := dashboard.SiteSettings(c.Context())
		if err != nil || !settings.SEOEnabled || settings.PublicOrigin == "" {
			return fiber.ErrNotFound
		}
		origin := strings.TrimRight(settings.PublicOrigin, "/")
		type sitemapURL struct {
			Loc string `xml:"loc"`
		}
		payload, err := xml.Marshal(struct {
			XMLName xml.Name     `xml:"urlset"`
			Xmlns   string       `xml:"xmlns,attr"`
			URLs    []sitemapURL `xml:"url"`
		}{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: []sitemapURL{{origin + "/"}, {origin + "/rankings"}, {origin + "/announcements"}}})
		if err != nil {
			return err
		}
		c.Type("application/xml", "utf-8")
		return c.Send(append([]byte(xml.Header), payload...))
	})
}

func applySEOMetadata(body []byte, settings store.SiteSettings, requestPath string) []byte {
	indexable := requestPath == "/" || requestPath == "/rankings" || requestPath == "/announcements"
	enabled := settings.SEOEnabled && indexable
	robots := "noindex,nofollow"
	if enabled {
		robots = "index,follow"
	}
	parts := []string{`<meta name="robots" content="` + robots + `" />`}
	if enabled {
		title := html.EscapeString(settings.BrowserTitle)
		description := html.EscapeString(settings.SEODescription)
		parts = append(parts,
			`<meta name="description" content="`+description+`" />`,
			`<meta property="og:type" content="website" />`,
			`<meta property="og:title" content="`+title+`" />`,
			`<meta property="og:description" content="`+description+`" />`,
			`<meta name="twitter:card" content="summary_large_image" />`,
		)
		if settings.PublicOrigin != "" {
			canonical := strings.TrimRight(settings.PublicOrigin, "/") + requestPath
			parts = append(parts, `<link rel="canonical" href="`+html.EscapeString(canonical)+`" />`, `<meta property="og:url" content="`+html.EscapeString(canonical)+`" />`)
		}
		if settings.SEOImageURL != "" {
			parts = append(parts, `<meta property="og:image" content="`+html.EscapeString(settings.SEOImageURL)+`" />`)
		}
	}
	replaced := strings.Replace(string(body), seoMarker, strings.Join(parts, "\n    "), 1)
	if settings.BrowserTitle != "" {
		start, end := strings.Index(replaced, "<title>"), strings.Index(replaced, "</title>")
		if start >= 0 && end > start {
			replaced = replaced[:start+7] + html.EscapeString(settings.BrowserTitle) + replaced[end:]
		}
	}
	return []byte(replaced)
}
