package server

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func registerAnalysisRoutes(api fiber.Router, analysis *service.AnalysisService) {
	if analysis == nil {
		return
	}
	group := api.Group("/analysis")
	group.Get("/options", func(c fiber.Ctx) error {
		filter, ok := analysisFilter(c)
		if !ok {
			return nil
		}
		result, err := analysis.Options(c.Context(), filter)
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
	group.Get("/maps", func(c fiber.Ctx) error {
		filter, ok := analysisFilter(c)
		if !ok {
			return nil
		}
		result, err := analysis.Maps(c.Context(), filter)
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
	group.Get("/map-detail", func(c fiber.Ctx) error {
		filter, ok := analysisFilter(c)
		if !ok {
			return nil
		}
		mapName := strings.TrimSpace(c.Query("map"))
		if mapName == "" || len(mapName) > 128 {
			return sendError(c, 400, "invalid_map", "map must be between 1 and 128 characters")
		}
		result, err := analysis.MapDetail(c.Context(), filter, mapName)
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
	group.Get("/contexts", func(c fiber.Ctx) error {
		filter, ok := analysisFilter(c)
		if !ok {
			return nil
		}
		result, err := analysis.Contexts(c.Context(), filter)
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
}

func analysisFilter(c fiber.Ctx) (store.AnalysisFilter, bool) {
	cutoff, err := analysisRangeCutoff(c.Query("range"))
	if err != nil {
		_ = sendError(c, 400, "invalid_range", err.Error())
		return store.AnalysisFilter{}, false
	}
	filter := store.AnalysisFilter{Cutoff: cutoff, ServerKey: strings.TrimSpace(c.Query("server")), Mode: c.Query("mode", "pve"), CampaignKey: strings.TrimSpace(c.Query("campaign"))}
	if len(filter.ServerKey) > 64 || len(filter.CampaignKey) > 128 {
		_ = sendError(c, 400, "invalid_filter", "server or campaign filter is too long")
		return store.AnalysisFilter{}, false
	}
	if filter.Mode != "pve" && filter.Mode != "versus" {
		_ = sendError(c, 400, "invalid_mode", "mode must be pve or versus")
		return store.AnalysisFilter{}, false
	}
	return filter, true
}

func analysisRangeCutoff(value string) (int64, error) {
	if value == "" {
		value = "90d"
	}
	switch value {
	case "all":
		return 0, nil
	case "30d":
		return time.Now().AddDate(0, 0, -30).Unix(), nil
	case "90d":
		return time.Now().AddDate(0, 0, -90).Unix(), nil
	case "180d":
		return time.Now().AddDate(0, 0, -180).Unix(), nil
	case "365d":
		return time.Now().AddDate(-1, 0, 0).Unix(), nil
	default:
		return 0, errors.New("range must be 30d, 90d, 180d, 365d or all")
	}
}
