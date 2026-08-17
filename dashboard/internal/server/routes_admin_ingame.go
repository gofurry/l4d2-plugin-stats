package server

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type adminIngameResponse struct {
	Settings      store.IngameSettings             `json:"settings"`
	MetricCatalog []service.IngameMetricDefinition `json:"metric_catalog"`
	PublicOrigin  string                           `json:"public_origin"`
}

type adminServerIngameResponse struct {
	Settings      store.IngameServerSettings       `json:"settings"`
	Documents     []store.ServerDocument           `json:"documents"`
	MetricCatalog []service.IngameMetricDefinition `json:"metric_catalog"`
	ServerKey     string                           `json:"server_key"`
	PublicOrigin  string                           `json:"public_origin"`
}

func (r *adminRoutes) getIngameSettings(c fiber.Ctx) error {
	if r.ingameStore == nil {
		return sendError(c, 503, "ingame_settings_unavailable", "in-game portal settings are unavailable")
	}
	settings, err := r.ingameStore.IngameSettings(c.Context())
	if err != nil {
		return sendError(c, 503, "ingame_settings_unavailable", "in-game portal settings are unavailable")
	}
	return sendData(c, 200, r.ingameResponse(c, settings))
}

func (r *adminRoutes) putIngameSettings(c fiber.Ctx) error {
	if r.ingameStore == nil {
		return sendError(c, 503, "ingame_settings_unavailable", "in-game portal settings are unavailable")
	}
	var settings store.IngameSettings
	if err := c.Bind().Body(&settings); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	normalizeIngameSettings(&settings)
	if err := service.ValidateIngameSettings(settings); err != nil {
		return sendError(c, 400, "invalid_ingame_settings", err.Error())
	}
	updated, err := r.ingameStore.UpdateIngameSettings(c.Context(), settings)
	if err != nil {
		return sendError(c, 500, "ingame_settings_update_failed", "in-game portal settings could not be saved")
	}
	r.invalidateIngameAll()
	r.logger.Info("in-game portal settings updated", zap.String("request_id", c.RequestID()))
	return sendData(c, 200, r.ingameResponse(c, updated))
}

func (r *adminRoutes) getServerIngameSettings(c fiber.Ctx) error {
	id, ok := r.ingameServerID(c)
	if !ok {
		return nil
	}
	response, err := r.serverIngameResponse(c, id)
	if err != nil {
		return sendError(c, 503, "server_ingame_settings_unavailable", "server in-game settings are unavailable")
	}
	return sendData(c, 200, response)
}

func (r *adminRoutes) putServerIngameSettings(c fiber.Ctx) error {
	id, ok := r.ingameServerID(c)
	if !ok {
		return nil
	}
	var settings store.IngameServerSettings
	if err := c.Bind().Body(&settings); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	settings.ServerID = id
	normalizeIngameServerSettings(&settings)
	if err := service.ValidateIngameServerSettings(settings); err != nil {
		return sendError(c, 400, "invalid_server_ingame_settings", err.Error())
	}
	updated, err := r.ingameStore.UpdateIngameServerSettings(c.Context(), settings)
	if err != nil {
		return sendError(c, 500, "server_ingame_settings_update_failed", "server in-game settings could not be saved")
	}
	r.invalidateIngameServer(c, id)
	r.logger.Info("server in-game portal settings updated", zap.String("server_id", id), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, updated)
}

func (r *adminRoutes) listServerIngameDocuments(c fiber.Ctx) error {
	id, ok := r.ingameServerID(c)
	if !ok {
		return nil
	}
	documents, err := r.completeServerDocuments(c, id)
	if err != nil {
		return sendError(c, 503, "server_ingame_documents_unavailable", "server in-game documents are unavailable")
	}
	return sendData(c, 200, documents)
}

func (r *adminRoutes) putServerIngameDocument(c fiber.Ctx) error {
	id, ok := r.ingameServerID(c)
	if !ok {
		return nil
	}
	document := store.ServerDocument{ServerID: id, Key: c.Params("key")}
	if err := c.Bind().Body(&document); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	document.ServerID = id
	document.Key = c.Params("key")
	document.Mode = strings.TrimSpace(document.Mode)
	if err := service.ValidateServerDocument(document); err != nil {
		return sendError(c, 400, "invalid_server_ingame_document", err.Error())
	}
	updated, err := r.ingameStore.UpdateServerDocument(c.Context(), document)
	if err != nil {
		return sendError(c, 500, "server_ingame_document_update_failed", "server in-game document could not be saved")
	}
	r.invalidateIngameServer(c, id)
	r.logger.Info("server in-game document updated", zap.String("server_id", id), zap.String("document", document.Key), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, updated)
}

func (r *adminRoutes) ingameResponse(c fiber.Ctx, settings store.IngameSettings) adminIngameResponse {
	site, _ := r.dashboard.SiteSettings(c.Context())
	return adminIngameResponse{Settings: settings, MetricCatalog: service.IngameMetricCatalog(), PublicOrigin: site.PublicOrigin}
}

func (r *adminRoutes) serverIngameResponse(c fiber.Ctx, serverID string) (adminServerIngameResponse, error) {
	settings, err := r.ingameStore.IngameServerSettings(c.Context(), serverID)
	if err != nil {
		return adminServerIngameResponse{}, err
	}
	documents, err := r.completeServerDocuments(c, serverID)
	if err != nil {
		return adminServerIngameResponse{}, err
	}
	site, err := r.dashboard.SiteSettings(c.Context())
	if err != nil {
		return adminServerIngameResponse{}, err
	}
	serverKey := ""
	if r.status != nil {
		status, available, statusErr := r.status.LastStatus(c.Context(), serverID)
		if statusErr == nil && available {
			serverKey = status.ServerKey
		}
	}
	return adminServerIngameResponse{
		Settings: settings, Documents: documents, MetricCatalog: service.IngameMetricCatalog(),
		ServerKey: serverKey, PublicOrigin: site.PublicOrigin,
	}, nil
}

func (r *adminRoutes) completeServerDocuments(c fiber.Ctx, serverID string) ([]store.ServerDocument, error) {
	documents, err := r.ingameStore.ListServerDocuments(c.Context(), serverID)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]store.ServerDocument, len(documents))
	for _, document := range documents {
		byKey[document.Key] = document
	}
	result := make([]store.ServerDocument, 0, 3)
	for _, key := range []string{store.IngameDocumentIntroduction, store.IngameDocumentCommands, store.IngameDocumentResources} {
		document, exists := byKey[key]
		if !exists {
			document = store.ServerDocument{ServerID: serverID, Key: key, Mode: "inherit"}
		}
		result = append(result, document)
	}
	return result, nil
}

func (r *adminRoutes) ingameServerID(c fiber.Ctx) (string, bool) {
	if r.ingameStore == nil {
		_ = sendError(c, 503, "ingame_settings_unavailable", "in-game portal settings are unavailable")
		return "", false
	}
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		_ = sendError(c, 400, "invalid_server_id", "server id is invalid")
		return "", false
	}
	servers, err := r.dashboard.ListServers(c.Context())
	if err != nil {
		_ = sendError(c, 503, "servers_unavailable", "server directory is unavailable")
		return "", false
	}
	for _, server := range servers {
		if server.ID == id {
			return id, true
		}
	}
	_ = sendError(c, 404, "server_not_found", "server was not found")
	return "", false
}

func (r *adminRoutes) invalidateIngameAll() {
	if r.ingame != nil {
		r.ingame.InvalidateAll()
	}
}

func (r *adminRoutes) invalidateIngameServer(c fiber.Ctx, serverID string) {
	if r.ingame == nil {
		return
	}
	if r.status != nil {
		status, available, err := r.status.LastStatus(c.Context(), serverID)
		if err == nil && available && status.ServerKey != "" {
			r.ingame.InvalidateServer(status.ServerKey)
			return
		}
	}
	r.ingame.InvalidateAll()
}

func normalizeIngameSettings(settings *store.IngameSettings) {
	settings.Title = strings.TrimSpace(settings.Title)
	settings.Description = strings.TrimSpace(settings.Description)
	settings.BannerURL = strings.TrimSpace(settings.BannerURL)
	settings.BackgroundURL = strings.TrimSpace(settings.BackgroundURL)
	settings.WebsiteURL = strings.TrimSpace(settings.WebsiteURL)
}

func normalizeIngameServerSettings(settings *store.IngameServerSettings) {
	settings.TitleMode = strings.TrimSpace(settings.TitleMode)
	settings.Title = strings.TrimSpace(settings.Title)
	settings.DescriptionMode = strings.TrimSpace(settings.DescriptionMode)
	settings.Description = strings.TrimSpace(settings.Description)
	settings.BannerMode = strings.TrimSpace(settings.BannerMode)
	settings.BannerURL = strings.TrimSpace(settings.BannerURL)
	settings.BackgroundMode = strings.TrimSpace(settings.BackgroundMode)
	settings.BackgroundURL = strings.TrimSpace(settings.BackgroundURL)
	settings.WebsiteMode = strings.TrimSpace(settings.WebsiteMode)
	settings.WebsiteURL = strings.TrimSpace(settings.WebsiteURL)
	settings.HighlightMode = strings.TrimSpace(settings.HighlightMode)
}
