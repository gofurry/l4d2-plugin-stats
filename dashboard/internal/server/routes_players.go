package server

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/auth"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func registerPlayerRoutes(api fiber.Router, players *service.PlayerService, analysis *service.AnalysisService, achievements *service.AchievementService, profiles store.DashboardProfileStore, authService *auth.Service) {
	group := api.Group("/players/:steam_id")
	group.Get("/preview", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		result, err := players.Preview(c.Context(), steamID)
		if err != nil {
			return statsError(c, err)
		}
		if result == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		showAchievements, visibilityErr := playerSectionAllowed(c, profiles, authService, steamID, store.PlayerProfileAchievements)
		if visibilityErr != nil {
			return sendError(c, 503, "profile_visibility_unavailable", "player profile visibility is temporarily unavailable")
		}
		if achievements != nil && showAchievements {
			badges, badgeErr := achievements.Badges(c.Context(), steamID)
			if badgeErr == nil && len(badges.Items) > 0 {
				badge := badges.Items[0]
				result.MainBadge = &store.PlayerPreviewBadge{AchievementKey: badge.AchievementKey, Title: badge.Title, ArtworkKey: badge.ArtworkKey, Tier: badge.Tier}
			}
		}
		return sendData(c, 200, result)
	})
	group.Get("/summary", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		if !requirePlayerSection(c, profiles, authService, steamID, store.PlayerProfileOverview) {
			return nil
		}
		result, err := players.Summary(c.Context(), steamID)
		if err != nil {
			return statsError(c, err)
		}
		if result == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		return sendData(c, 200, result)
	})
	group.Get("/pve", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		view := c.Query("view", string(store.PlayerProfilePVE))
		section := store.PlayerProfileSection(view)
		if section != store.PlayerProfilePVE && section != store.PlayerProfilePVEDetails {
			return sendError(c, 400, "invalid_view", "view must be pve or pve-details")
		}
		if !requirePlayerSection(c, profiles, authService, steamID, section) {
			return nil
		}
		if exists, err := players.Summary(c.Context(), steamID); err != nil {
			return statsError(c, err)
		} else if exists == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		cutoff, err := rangeCutoff(c.Query("range"))
		if err != nil {
			return sendError(c, 400, "invalid_range", err.Error())
		}
		mode := c.Query("mode")
		if mode != "" && mode != "coop" && mode != "realism" {
			return sendError(c, 400, "invalid_mode", "mode must be coop or realism")
		}
		serverKey := c.Query("server")
		if len(serverKey) > 64 {
			return sendError(c, 400, "invalid_server", "server is too long")
		}
		result, err := players.PVEFiltered(c.Context(), steamID, store.PlayerFilter{Cutoff: cutoff, ServerKey: serverKey, GameMode: mode})
		if err != nil {
			return statsError(c, err)
		}
		payload, err := playerObject(result)
		if err != nil {
			return sendError(c, 503, "stats_unavailable", "statistics are temporarily unavailable")
		}
		if section == store.PlayerProfilePVEDetails {
			payload = fiber.Map{"infected_classes": payload["infected_classes"], "equipment": payload["equipment"]}
		} else {
			delete(payload, "infected_classes")
			delete(payload, "equipment")
		}
		return sendData(c, 200, payload)
	})
	group.Get("/versus", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		view := c.Query("view", string(store.PlayerProfileVersusSurvivor))
		section := store.PlayerProfileSection(view)
		if section != store.PlayerProfileVersusSurvivor && section != store.PlayerProfileVersusSurvivorDetails && section != store.PlayerProfileVersusInfected && section != store.PlayerProfileVersusInfectedDetails {
			return sendError(c, 400, "invalid_view", "unsupported versus view")
		}
		if !requirePlayerSection(c, profiles, authService, steamID, section) {
			return nil
		}
		if exists, err := players.Summary(c.Context(), steamID); err != nil {
			return statsError(c, err)
		} else if exists == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		cutoff, err := rangeCutoff(c.Query("range"))
		if err != nil {
			return sendError(c, 400, "invalid_range", err.Error())
		}
		serverKey := c.Query("server")
		if len(serverKey) > 64 {
			return sendError(c, 400, "invalid_server", "server is too long")
		}
		result, err := players.VersusFiltered(c.Context(), steamID, store.PlayerFilter{Cutoff: cutoff, ServerKey: serverKey})
		if err != nil {
			return statsError(c, err)
		}
		payload, err := versusPayload(result, section)
		if err != nil {
			return sendError(c, 503, "stats_unavailable", "statistics are temporarily unavailable")
		}
		return sendData(c, 200, payload)
	})
	group.Get("/activity", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		if !requirePlayerSection(c, profiles, authService, steamID, store.PlayerProfileOverview) {
			return nil
		}
		if exists, err := players.Summary(c.Context(), steamID); err != nil {
			return statsError(c, err)
		} else if exists == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		cutoff, err := rangeCutoff(c.Query("range"))
		if err != nil {
			return sendError(c, 400, "invalid_range", err.Error())
		}
		serverKey := c.Query("server")
		if len(serverKey) > 64 {
			return sendError(c, 400, "invalid_server", "server is too long")
		}
		result, err := players.ActivityFiltered(c.Context(), steamID, store.PlayerFilter{Cutoff: cutoff, ServerKey: serverKey})
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
	group.Get("/analysis", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		if !requirePlayerSection(c, profiles, authService, steamID, store.PlayerProfileAnalysis) {
			return nil
		}
		if players == nil || analysis == nil {
			return sendError(c, 503, "analysis_unavailable", "player analysis is temporarily unavailable")
		}
		if exists, err := players.Summary(c.Context(), steamID); err != nil {
			return statsError(c, err)
		} else if exists == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		cutoff, err := analysisRangeCutoff(c.Query("range"))
		if err != nil {
			return sendError(c, 400, "invalid_range", err.Error())
		}
		serverKey := c.Query("server")
		if len(serverKey) > 64 {
			return sendError(c, 400, "invalid_server", "server is too long")
		}
		view := c.Query("view", "pve")
		if view != "pve" && view != "versus_survivor" && view != "versus_infected" {
			return sendError(c, 400, "invalid_view", "view must be pve, versus_survivor or versus_infected")
		}
		result, err := analysis.Player(c.Context(), steamID, store.PlayerFilter{Cutoff: cutoff, ServerKey: serverKey}, view)
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
	group.Get("/relationships", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		if !requirePlayerSection(c, profiles, authService, steamID, store.PlayerProfileRelationships) {
			return nil
		}
		if exists, err := players.Summary(c.Context(), steamID); err != nil {
			return statsError(c, err)
		} else if exists == nil {
			return sendError(c, 404, "player_not_found", "player was not found")
		}
		cutoff, err := rangeCutoff(c.Query("range"))
		if err != nil {
			return sendError(c, 400, "invalid_range", err.Error())
		}
		serverKey := strings.TrimSpace(c.Query("server"))
		if len(serverKey) > 64 {
			return sendError(c, 400, "invalid_server", "server is too long")
		}
		mode := c.Query("mode", "all")
		if mode != "all" && mode != "pve" && mode != "versus" {
			return sendError(c, 400, "invalid_mode", "mode must be all, pve or versus")
		}
		page, err := strconv.ParseInt(c.Query("page", "1"), 10, 64)
		if err != nil || page < 1 {
			return sendError(c, 400, "invalid_page", "page must be a positive integer")
		}
		pageSize, err := strconv.ParseInt(c.Query("page_size", "20"), 10, 64)
		if err != nil || pageSize < 1 || pageSize > 100 {
			return sendError(c, 400, "invalid_page_size", "page_size must be between 1 and 100")
		}
		sortName := c.Query("sort", "shared_rounds")
		allowedSorts := map[string]bool{
			"player_name": true, "shared_rounds": true, "shared_seconds": true,
			"outgoing_support": true, "incoming_support": true, "mutual_support": true,
			"outgoing_healing": true, "incoming_healing": true,
			"outgoing_friendly_fire": true, "incoming_friendly_fire": true,
		}
		if !allowedSorts[sortName] {
			return sendError(c, 400, "invalid_sort", "unsupported relationship sort field")
		}
		order := c.Query("order", "desc")
		if order != "asc" && order != "desc" {
			return sendError(c, 400, "invalid_order", "order must be asc or desc")
		}
		result, err := players.Relationships(c.Context(), steamID, store.PlayerRelationshipQuery{
			PlayerFilter: store.PlayerFilter{Cutoff: cutoff, ServerKey: serverKey, GameMode: mode},
			Page:         page, PageSize: pageSize, Sort: sortName, Order: order,
		})
		if err != nil {
			return statsError(c, err)
		}
		return sendData(c, 200, result)
	})
	group.Get("/sessions", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		if !requirePlayerSection(c, profiles, authService, steamID, store.PlayerProfileHistory) {
			return nil
		}
		at, id, limit, ok := pageArgs(c)
		if !ok {
			return nil
		}
		rows, err := players.Sessions(c.Context(), steamID, at, id, limit+1)
		if err != nil {
			return statsError(c, err)
		}
		page := store.Page[store.PlayerSession]{Items: rows}
		if len(rows) > int(limit) {
			last := rows[int(limit)-1]
			page.Items = rows[:int(limit)]
			page.NextCursor = encodeCursor(last.StartedAt, last.SessionID)
		}
		return sendData(c, 200, page)
	})
	group.Get("/chapters", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
			return nil
		}
		if !requirePlayerSection(c, profiles, authService, steamID, store.PlayerProfileHistory) {
			return nil
		}
		at, id, limit, ok := pageArgs(c)
		if !ok {
			return nil
		}
		rows, err := players.Chapters(c.Context(), steamID, at, id, limit+1)
		if err != nil {
			return statsError(c, err)
		}
		page := store.Page[store.PlayerChapter]{Items: rows}
		if len(rows) > int(limit) {
			last := rows[int(limit)-1]
			page.Items = rows[:int(limit)]
			page.NextCursor = encodeCursor(last.StartedAt, last.SegmentID)
		}
		return sendData(c, 200, page)
	})
}

func playerObject(value any) (fiber.Map, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := fiber.Map{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func versusPayload(value store.PlayerVersus, section store.PlayerProfileSection) (fiber.Map, error) {
	object, err := playerObject(value)
	if err != nil {
		return nil, err
	}
	switch section {
	case store.PlayerProfileVersusSurvivorDetails:
		return fiber.Map{"survivor_classes": object["survivor_classes"]}, nil
	case store.PlayerProfileVersusInfectedDetails:
		return fiber.Map{"infected_classes": object["infected_classes"]}, nil
	case store.PlayerProfileVersusInfected:
		allowed := map[string]bool{
			"infected_spawns": true, "damage_to_human_survivors": true, "damage_to_bot_survivors": true,
			"human_survivor_incaps": true, "bot_survivor_incaps": true, "human_survivor_kills": true,
			"bot_survivor_kills": true, "human_survivor_controls": true, "human_survivor_control_seconds": true,
		}
		for key := range object {
			if !allowed[key] {
				delete(object, key)
			}
		}
		return object, nil
	default:
		for key := range object {
			allowed := strings.HasPrefix(key, "survivor_") || strings.HasPrefix(key, "human_special_") || strings.HasPrefix(key, "bot_special_") || strings.HasPrefix(key, "human_tank_") || strings.HasPrefix(key, "bot_tank_") || key == "molotovs_thrown" || key == "pipe_bombs_thrown" || key == "vomit_jars_thrown" || key == "assist_coverage"
			if !allowed || key == "survivor_classes" {
				delete(object, key)
			}
		}
		return object, nil
	}
}

func playerID(c fiber.Ctx) (string, bool) {
	steamID := c.Params("steam_id")
	if !validSteamID64(steamID) {
		_ = sendError(c, 400, "invalid_steam_id", "steam_id must be a 17-digit SteamID64")
		return "", false
	}
	return steamID, true
}
func pageArgs(c fiber.Ctx) (int64, string, int32, bool) {
	at, id, err := decodeCursor(c.Query("cursor"))
	if err != nil {
		_ = sendError(c, 400, "invalid_cursor", err.Error())
		return 0, "", 0, false
	}
	limit, err := pageLimit(c.Query("limit"))
	if err != nil {
		_ = sendError(c, 400, "invalid_limit", err.Error())
		return 0, "", 0, false
	}
	return at, id, limit, true
}
func statsError(c fiber.Ctx, err error) error {
	return sendError(c, 503, "stats_unavailable", "statistics are temporarily unavailable")
}
