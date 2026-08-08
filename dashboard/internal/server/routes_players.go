package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func registerPlayerRoutes(api fiber.Router, players *service.PlayerService) {
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
		return sendData(c, 200, result)
	})
	group.Get("/summary", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
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
		return sendData(c, 200, result)
	})
	group.Get("/versus", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
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
		return sendData(c, 200, result)
	})
	group.Get("/activity", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
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
	group.Get("/sessions", func(c fiber.Ctx) error {
		steamID, ok := playerID(c)
		if !ok {
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
