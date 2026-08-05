package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func registerRankingRoutes(api fiber.Router, rankings *service.RankingService) {
	api.Get("/rankings/servers", func(c fiber.Ctx) error {
		servers, err := rankings.Servers(c.Context())
		if err != nil {
			return sendError(c, 503, "ranking_unavailable", "ranking data is temporarily unavailable")
		}
		return sendData(c, 200, servers)
	})
	api.Get("/rankings/players", func(c fiber.Ctx) error {
		query := strings.TrimSpace(c.Query("q"))
		if len(query) > 64 {
			return sendError(c, 400, "invalid_player_query", "player query must not exceed 64 characters")
		}
		players, err := rankings.SearchPlayers(c.Context(), query)
		if err != nil {
			return sendError(c, 503, "ranking_unavailable", "ranking data is temporarily unavailable")
		}
		return sendData(c, 200, players)
	})
	api.Get("/rankings", func(c fiber.Ctx) error {
		mode := c.Query("mode", "activity")
		metric := c.Query("metric", "active_time")
		if !service.ValidRanking(mode, metric) {
			return sendError(c, 400, "invalid_ranking", "unsupported ranking mode or metric")
		}
		cutoff, err := rangeCutoff(c.Query("range"))
		if err != nil {
			return sendError(c, 400, "invalid_range", err.Error())
		}
		limit, err := pageLimit(c.Query("limit"))
		if err != nil {
			return sendError(c, 400, "invalid_limit", err.Error())
		}
		page, err := strconv.Atoi(c.Query("page", "1"))
		if err != nil || page < 1 || page > 10000 {
			return sendError(c, 400, "invalid_page", "page must be between 1 and 10000")
		}
		minimumHours, err := strconv.Atoi(c.Query("min_hours", "0"))
		if err != nil || minimumHours < 0 || minimumHours > 10000 {
			return sendError(c, 400, "invalid_min_hours", "min_hours must be between 0 and 10000")
		}
		steamIDs, err := rankingSteamIDs(c.Query("players"))
		if err != nil {
			return sendError(c, 400, "invalid_players", err.Error())
		}
		subject := strings.TrimSpace(c.Query("subject"))
		if subject != "" && !validSteamID64(subject) {
			return sendError(c, 400, "invalid_subject", "subject must be a 17-digit SteamID64")
		}
		serverKey := strings.TrimSpace(c.Query("server"))
		if len(serverKey) > 64 {
			return sendError(c, 400, "invalid_server", "server must not exceed 64 characters")
		}
		result, err := rankings.List(c.Context(), store.RankingQuery{
			Mode: mode, Metric: metric, ServerKey: serverKey, Cutoff: cutoff,
			MinimumActiveSec: int64(minimumHours) * 3600, SteamIDs: steamIDs, SubjectSteamID: subject,
			Limit: int(limit), Offset: (page - 1) * int(limit),
		})
		if err != nil {
			return sendError(c, 503, "ranking_unavailable", "ranking data is temporarily unavailable")
		}
		return sendData(c, 200, result)
	})
}

func rankingSteamIDs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 20 {
		return nil, fmt.Errorf("players accepts at most 20 SteamID64 values")
	}
	seen := make(map[string]struct{}, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		steamID := strings.TrimSpace(part)
		if !validSteamID64(steamID) {
			return nil, fmt.Errorf("players contains an invalid SteamID64")
		}
		if _, ok := seen[steamID]; ok {
			continue
		}
		seen[steamID] = struct{}{}
		result = append(result, steamID)
	}
	return result, nil
}
