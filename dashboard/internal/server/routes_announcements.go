package server

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func registerAnnouncementRoutes(api fiber.Router, dashboard store.DashboardStore) {
	api.Get("/announcements", func(c fiber.Ctx) error {
		filter, _, err := announcementFilter(c)
		if err != nil {
			return sendError(c, 400, "invalid_page", err.Error())
		}
		result, err := dashboard.ListAnnouncements(c.Context(), filter)
		if err != nil {
			return sendError(c, 503, "announcements_unavailable", "announcements are temporarily unavailable")
		}
		return sendData(c, 200, result)
	})
	api.Get("/announcements/years", func(c fiber.Ctx) error {
		years, err := dashboard.ListAnnouncementYears(c.Context())
		if err != nil {
			return sendError(c, 503, "announcement_years_unavailable", "announcement years are temporarily unavailable")
		}
		return sendData(c, 200, years)
	})
}

func announcementPage(c fiber.Ctx) (int, int32, error) {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 || page > 10000 {
		return 0, 0, errors.New("page must be between 1 and 10000")
	}
	limit, err := pageLimit(c.Query("limit"))
	if err != nil {
		return 0, 0, err
	}
	return page, limit, nil
}

func announcementFilter(c fiber.Ctx) (store.AnnouncementFilter, int, error) {
	page, limit, err := announcementPage(c)
	if err != nil {
		return store.AnnouncementFilter{}, 0, err
	}
	title := strings.TrimSpace(c.Query("title"))
	if len([]rune(title)) > 120 {
		return store.AnnouncementFilter{}, 0, errors.New("title must not exceed 120 characters")
	}
	year := 0
	if rawYear := strings.TrimSpace(c.Query("year")); rawYear != "" {
		year, err = strconv.Atoi(rawYear)
		if err != nil || year < 1970 || year > 3000 {
			return store.AnnouncementFilter{}, 0, errors.New("year is invalid")
		}
	}
	return store.AnnouncementFilter{Title: title, Year: year, Limit: limit, Offset: int32(page-1) * limit}, page, nil
}

func (r *adminRoutes) listAnnouncements(c fiber.Ctx) error {
	filter, _, err := announcementFilter(c)
	if err != nil {
		return sendError(c, 400, "invalid_page", err.Error())
	}
	result, err := r.dashboard.ListAnnouncements(c.Context(), filter)
	if err != nil {
		return sendError(c, 503, "announcements_unavailable", "announcements are temporarily unavailable")
	}
	return sendData(c, 200, result)
}

func (r *adminRoutes) createAnnouncement(c fiber.Ctx) error {
	value, err := announcementBody(c)
	if err != nil {
		return sendError(c, 400, "invalid_announcement", err.Error())
	}
	created, err := r.dashboard.CreateAnnouncement(c.Context(), value)
	if err != nil {
		return sendError(c, 500, "announcement_create_failed", "announcement could not be created")
	}
	r.invalidateIngameAll()
	r.logger.Info("announcement created", zap.String("announcement_id", created.ID), zap.String("request_id", c.RequestID()))
	return sendData(c, 201, created)
}

func (r *adminRoutes) updateAnnouncement(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_announcement_id", "announcement id is invalid")
	}
	value, err := announcementBody(c)
	if err != nil {
		return sendError(c, 400, "invalid_announcement", err.Error())
	}
	value.ID = id
	updated, err := r.dashboard.UpdateAnnouncement(c.Context(), value)
	if errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "announcement_not_found", "announcement was not found")
	}
	if err != nil {
		return sendError(c, 500, "announcement_update_failed", "announcement could not be updated")
	}
	r.invalidateIngameAll()
	r.logger.Info("announcement updated", zap.String("announcement_id", id), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, updated)
}

func (r *adminRoutes) deleteAnnouncement(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return sendError(c, 400, "invalid_announcement_id", "announcement id is invalid")
	}
	if err := r.dashboard.DeleteAnnouncement(c.Context(), id); errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "announcement_not_found", "announcement was not found")
	} else if err != nil {
		return sendError(c, 500, "announcement_delete_failed", "announcement could not be deleted")
	}
	r.invalidateIngameAll()
	r.logger.Info("announcement deleted", zap.String("announcement_id", id), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, fiber.Map{"deleted": true})
}

func announcementBody(c fiber.Ctx) (store.Announcement, error) {
	var value store.Announcement
	if err := c.Bind().Body(&value); err != nil {
		return store.Announcement{}, errors.New("request body is invalid")
	}
	value.Title = strings.TrimSpace(value.Title)
	value.ContentMarkdown = strings.TrimSpace(value.ContentMarkdown)
	if err := validateAnnouncement(value); err != nil {
		return store.Announcement{}, err
	}
	return value, nil
}
