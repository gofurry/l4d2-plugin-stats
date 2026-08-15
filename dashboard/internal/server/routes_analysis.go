package server

import (
	"errors"
	"strconv"
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
		filter, ok = analysisPageFilter(c, filter, mapAnalysisSorts, "map_name", "asc")
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
		if result.Summary.MapName == "" {
			return sendError(c, 404, "analysis_map_not_found", "map has no data under the current filters")
		}
		return sendData(c, 200, result)
	})
	group.Get("/contexts", func(c fiber.Ctx) error {
		filter, ok := analysisFilter(c)
		if !ok {
			return nil
		}
		filter, ok = analysisPageFilter(c, filter, contextAnalysisSorts, "round_count", "desc")
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

var mapAnalysisSorts = map[string]struct{}{
	"map_name": {}, "eligible_rounds": {}, "completion_rate": {}, "average_completed_attempt": {},
	"average_duration_seconds": {}, "incaps_per_complete_round": {}, "deaths_per_complete_round": {}, "controls_per_complete_round": {},
}

var contextAnalysisSorts = map[string]struct{}{
	"ruleset_name": {}, "round_count": {}, "completion_rate": {}, "average_duration_seconds": {}, "complete_incident_coverage": {},
}

func analysisPageFilter(c fiber.Ctx, filter store.AnalysisFilter, allowed map[string]struct{}, defaultSort, defaultOrder string) (store.AnalysisFilter, bool) {
	page, err := strconv.ParseInt(c.Query("page", "1"), 10, 64)
	if err != nil || page < 1 {
		_ = sendError(c, 400, "invalid_page", "page must be a positive integer")
		return filter, false
	}
	pageSize, err := strconv.ParseInt(c.Query("page_size", "20"), 10, 64)
	if err != nil || pageSize < 1 || pageSize > 100 {
		_ = sendError(c, 400, "invalid_page_size", "page_size must be between 1 and 100")
		return filter, false
	}
	sortName := c.Query("sort", defaultSort)
	if _, exists := allowed[sortName]; !exists {
		_ = sendError(c, 400, "invalid_sort", "unsupported analysis sort field")
		return filter, false
	}
	order := c.Query("order", defaultOrder)
	if order != "asc" && order != "desc" {
		_ = sendError(c, 400, "invalid_order", "order must be asc or desc")
		return filter, false
	}
	filter.Page, filter.PageSize, filter.Sort, filter.Order = page, pageSize, sortName, order
	return filter, true
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
