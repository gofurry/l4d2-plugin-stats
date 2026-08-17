package server

import (
	"bytes"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	ingameassets "github.com/gofurry/l4d2-plugin-stats/dashboard/ingame"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
)

type ingameErrorView struct {
	Title     string
	Message   string
	ServerKey string
}

func registerIngameRoutes(app *fiber.App, ingame *service.IngameService, renderer *ingameassets.Renderer) {
	if ingame == nil || renderer == nil {
		return
	}
	app.Get("/ingame/assets/"+ingameassets.AssetVersion+"/ingame.css", func(c fiber.Ctx) error {
		content, err := ingameassets.CSS()
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		c.Set(fiber.HeaderContentType, fiber.MIMETextCSSCharsetUTF8)
		return c.Send(content)
	})
	app.Get("/ingame/assets/"+ingameassets.AssetVersion+"/achievements.png", func(c fiber.Ctx) error {
		content, err := ingameassets.AchievementAtlas()
		if err != nil {
			return err
		}
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		c.Set(fiber.HeaderContentType, "image/png")
		return c.Send(content)
	})
	app.Get("/ingame", func(c fiber.Ctx) error {
		view, err := ingame.Home(c.Context(), strings.TrimSpace(c.Query("server")))
		if err != nil {
			return renderIngameError(c, renderer, err, c.Query("server"))
		}
		return renderIngame(c, renderer, "home.html", view, fiber.StatusOK)
	})
	app.Get("/ingame/player/:steamid", func(c fiber.Ctx) error {
		view, err := ingame.Player(c.Context(), strings.TrimSpace(c.Query("server")), c.Params("steamid"))
		if err != nil {
			return renderIngameError(c, renderer, err, c.Query("server"))
		}
		return renderIngame(c, renderer, "player.html", view, fiber.StatusOK)
	})
	app.Get("/ingame/rankings", func(c fiber.Ctx) error {
		page, err := strconv.Atoi(c.Query("page", "1"))
		if err != nil || page < 1 || page > 10000 {
			return renderIngameError(c, renderer, errors.New("invalid ranking page"), c.Query("server"))
		}
		view, err := ingame.Rankings(c.Context(), strings.TrimSpace(c.Query("server")), strings.TrimSpace(c.Query("metric")), page)
		if err != nil {
			return renderIngameError(c, renderer, err, c.Query("server"))
		}
		return renderIngame(c, renderer, "rankings.html", view, fiber.StatusOK)
	})
	app.Get("/ingame/info/:key", func(c fiber.Ctx) error {
		view, err := ingame.Info(c.Context(), strings.TrimSpace(c.Query("server")), c.Params("key"))
		if err != nil {
			return renderIngameError(c, renderer, err, c.Query("server"))
		}
		return renderIngame(c, renderer, "info.html", view, fiber.StatusOK)
	})
	app.Get("/ingame/announcement/:id", func(c fiber.Ctx) error {
		view, err := ingame.Announcement(c.Context(), strings.TrimSpace(c.Query("server")), c.Params("id"))
		if err != nil {
			return renderIngameError(c, renderer, err, c.Query("server"))
		}
		return renderIngame(c, renderer, "announcement.html", view, fiber.StatusOK)
	})
	app.Get("/ingame/*", func(c fiber.Ctx) error {
		return renderIngameError(c, renderer, service.ErrIngameContentUnavailable, c.Query("server"))
	})
}

func renderIngame(c fiber.Ctx, renderer *ingameassets.Renderer, templateName string, value any, status int) error {
	var output bytes.Buffer
	if err := renderer.Render(&output, templateName, value); err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.Status(status).Send(output.Bytes())
}

func renderIngameError(c fiber.Ctx, renderer *ingameassets.Renderer, err error, serverKey string) error {
	status := fiber.StatusServiceUnavailable
	view := ingameErrorView{Title: "暂时无法显示", Message: "游戏内页面暂时不可用，请稍后重试。", ServerKey: strings.TrimSpace(serverKey)}
	switch {
	case errors.Is(err, service.ErrIngameDisabled):
		view.Title, view.Message = "游戏内页面未启用", "管理员尚未启用游戏内页面。"
	case errors.Is(err, service.ErrIngameUnknownServer):
		status = fiber.StatusNotFound
		view.Title, view.Message = "无法识别服务器", "当前链接中的 server_key 尚未出现在已保存的服务器快照中。"
	case errors.Is(err, service.ErrIngameContentUnavailable):
		status = fiber.StatusNotFound
		view.Title, view.Message = "内容不可用", "该内容未启用、已隐藏或不存在。"
	case errors.Is(err, service.ErrIngamePlayerNotFound):
		status = fiber.StatusNotFound
		view.Title, view.Message = "找不到玩家", "该玩家不存在或标识无效。"
	case strings.Contains(err.Error(), "invalid ranking") || strings.Contains(err.Error(), "unsupported ranking"):
		status = fiber.StatusBadRequest
		view.Title, view.Message = "排行榜参数无效", "请选择有效的指标和页码。"
	}
	return renderIngame(c, renderer, "error.html", view, status)
}
