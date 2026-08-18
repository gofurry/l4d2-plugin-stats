package server

import (
	"context"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

type adminIngameResponse struct {
	Settings      store.IngameSettings             `json:"settings"`
	MetricCatalog []service.IngameMetricDefinition `json:"metric_catalog"`
	PublicOrigin  string                           `json:"public_origin"`
}

type adminIngameGroupInstance struct {
	ServerID  string `json:"server_id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	SortOrder int64  `json:"sort_order"`
	Online    bool   `json:"online"`
	Stale     bool   `json:"stale"`
}

type adminIngameGroup struct {
	ServerKey string                     `json:"server_key"`
	Title     string                     `json:"title"`
	Instances []adminIngameGroupInstance `json:"instances"`
}

type adminIngameGroupResponse struct {
	Settings      store.IngameServerSettings       `json:"settings"`
	Documents     []store.ServerDocument           `json:"documents"`
	QuickLinks    []store.IngameQuickLink          `json:"quick_links"`
	MetricCatalog []service.IngameMetricDefinition `json:"metric_catalog"`
	ServerKey     string                           `json:"server_key"`
	Title         string                           `json:"title"`
	Instances     []adminIngameGroupInstance       `json:"instances"`
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

func (r *adminRoutes) listIngameMapNames(c fiber.Ctx) error {
	if r.ingameStore == nil {
		return sendError(c, 503, "ingame_map_names_unavailable", "custom map names are unavailable")
	}
	values, err := r.ingameStore.ListIngameMapNames(c.Context())
	if err != nil {
		return sendError(c, 503, "ingame_map_names_unavailable", "custom map names are unavailable")
	}
	return sendData(c, 200, values)
}

func (r *adminRoutes) putIngameMapNames(c fiber.Ctx) error {
	if r.ingameStore == nil {
		return sendError(c, 503, "ingame_map_names_unavailable", "custom map names are unavailable")
	}
	var body struct {
		MapNames []store.IngameMapName `json:"map_names"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	for index := range body.MapNames {
		body.MapNames[index].MapName = strings.ToLower(strings.TrimSpace(body.MapNames[index].MapName))
		body.MapNames[index].DisplayName = strings.TrimSpace(body.MapNames[index].DisplayName)
	}
	if err := service.ValidateIngameMapNames(body.MapNames); err != nil {
		return sendError(c, 400, "invalid_ingame_map_names", err.Error())
	}
	values, err := r.ingameStore.ReplaceIngameMapNames(c.Context(), body.MapNames)
	if err != nil {
		return sendError(c, 500, "ingame_map_names_update_failed", "custom map names could not be saved")
	}
	r.invalidateIngameAll()
	r.logger.Info("in-game custom map names updated", zap.Int("count", len(values)), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, values)
}

func (r *adminRoutes) listIngameGroups(c fiber.Ctx) error {
	groups, err := r.ingameGroups(c)
	if err != nil {
		return sendError(c, 503, "ingame_groups_unavailable", "in-game server groups are unavailable")
	}
	return sendData(c, 200, groups)
}

func (r *adminRoutes) getIngameGroup(c fiber.Ctx) error {
	group, ok := r.requireIngameGroup(c)
	if !ok {
		return nil
	}
	response, err := r.ingameGroupResponse(c, group)
	if err != nil {
		return sendError(c, 503, "ingame_group_settings_unavailable", "server-group in-game settings are unavailable")
	}
	return sendData(c, 200, response)
}

func (r *adminRoutes) putIngameGroup(c fiber.Ctx) error {
	group, ok := r.requireIngameGroup(c)
	if !ok {
		return nil
	}
	var settings store.IngameServerSettings
	if err := c.Bind().Body(&settings); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	settings.ServerKey = group.ServerKey
	normalizeIngameServerSettings(&settings)
	if err := service.ValidateIngameServerSettings(settings); err != nil {
		return sendError(c, 400, "invalid_ingame_group_settings", err.Error())
	}
	updated, err := r.ingameStore.UpdateIngameServerSettings(c.Context(), settings)
	if err != nil {
		return sendError(c, 500, "ingame_group_settings_update_failed", "server-group in-game settings could not be saved")
	}
	r.invalidateIngameGroup(group.ServerKey)
	r.logger.Info("server-group in-game portal settings updated", zap.String("server_key", group.ServerKey), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, updated)
}

func (r *adminRoutes) putIngameGroupQuickLinks(c fiber.Ctx) error {
	group, ok := r.requireIngameGroup(c)
	if !ok {
		return nil
	}
	var body struct {
		Links []store.IngameQuickLink `json:"links"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	for index := range body.Links {
		body.Links[index].ServerKey = group.ServerKey
		body.Links[index].Label = strings.TrimSpace(body.Links[index].Label)
		body.Links[index].URL = strings.TrimSpace(body.Links[index].URL)
		body.Links[index].SortOrder = int64(index)
	}
	if err := service.ValidateServerQuickLinks(body.Links); err != nil {
		return sendError(c, 400, "invalid_ingame_quick_links", err.Error())
	}
	links, err := r.ingameStore.ReplaceServerQuickLinks(c.Context(), group.ServerKey, body.Links)
	if err != nil {
		return sendError(c, 500, "ingame_quick_links_update_failed", "server-group quick links could not be saved")
	}
	r.invalidateIngameGroup(group.ServerKey)
	r.logger.Info("server-group in-game quick links updated", zap.String("server_key", group.ServerKey), zap.Int("count", len(links)), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, links)
}

func (r *adminRoutes) listIngameGroupDocuments(c fiber.Ctx) error {
	group, ok := r.requireIngameGroup(c)
	if !ok {
		return nil
	}
	documents, err := r.completeServerDocuments(c, group.ServerKey)
	if err != nil {
		return sendError(c, 503, "ingame_group_documents_unavailable", "server-group in-game documents are unavailable")
	}
	return sendData(c, 200, documents)
}

func (r *adminRoutes) putIngameGroupDocument(c fiber.Ctx) error {
	group, ok := r.requireIngameGroup(c)
	if !ok {
		return nil
	}
	document := store.ServerDocument{ServerKey: group.ServerKey, Key: c.Params("key")}
	if err := c.Bind().Body(&document); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	document.ServerKey = group.ServerKey
	document.Key = c.Params("key")
	document.Mode = strings.TrimSpace(document.Mode)
	if err := service.ValidateServerDocument(document); err != nil {
		return sendError(c, 400, "invalid_ingame_group_document", err.Error())
	}
	updated, err := r.ingameStore.UpdateServerDocument(c.Context(), document)
	if err != nil {
		return sendError(c, 500, "ingame_group_document_update_failed", "server-group in-game document could not be saved")
	}
	r.invalidateIngameGroup(group.ServerKey)
	r.logger.Info("server-group in-game document updated", zap.String("server_key", group.ServerKey), zap.String("document", document.Key), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, updated)
}

func (r *adminRoutes) ingameResponse(c fiber.Ctx, settings store.IngameSettings) adminIngameResponse {
	site, _ := r.dashboard.SiteSettings(c.Context())
	return adminIngameResponse{Settings: settings, MetricCatalog: service.IngameMetricCatalog(), PublicOrigin: site.PublicOrigin}
}

func (r *adminRoutes) ingameGroupResponse(c fiber.Ctx, group adminIngameGroup) (adminIngameGroupResponse, error) {
	settings, err := r.ingameStore.IngameServerSettings(c.Context(), group.ServerKey)
	if err != nil {
		return adminIngameGroupResponse{}, err
	}
	quickLinks, err := r.ingameStore.ListServerQuickLinks(c.Context(), group.ServerKey)
	if err != nil {
		return adminIngameGroupResponse{}, err
	}
	documents, err := r.completeServerDocuments(c, group.ServerKey)
	if err != nil {
		return adminIngameGroupResponse{}, err
	}
	site, err := r.dashboard.SiteSettings(c.Context())
	if err != nil {
		return adminIngameGroupResponse{}, err
	}
	return adminIngameGroupResponse{
		Settings: settings, Documents: documents, QuickLinks: quickLinks, MetricCatalog: service.IngameMetricCatalog(),
		ServerKey: group.ServerKey, Title: group.Title, Instances: group.Instances, PublicOrigin: site.PublicOrigin,
	}, nil
}

func (r *adminRoutes) completeServerDocuments(c fiber.Ctx, serverKey string) ([]store.ServerDocument, error) {
	documents, err := r.ingameStore.ListServerDocuments(c.Context(), serverKey)
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
			document = store.ServerDocument{ServerKey: serverKey, Key: key, Mode: "inherit"}
		}
		result = append(result, document)
	}
	return result, nil
}

func (r *adminRoutes) ingameGroups(c fiber.Ctx) ([]adminIngameGroup, error) {
	if r.ingameStore == nil {
		return nil, store.ErrServerNotFound
	}
	servers, err := r.dashboard.ListServers(c.Context())
	if err != nil {
		return nil, err
	}
	statuses := make([]store.ServerStatus, 0, len(servers))
	if cached, ok := r.status.(interface {
		CachedStatuses(context.Context) ([]store.ServerStatus, error)
	}); ok {
		statuses, err = cached.CachedStatuses(c.Context())
		if err != nil {
			return nil, err
		}
	} else if r.status != nil {
		for _, server := range servers {
			status, available, statusErr := r.status.LastStatus(c.Context(), server.ID)
			if statusErr == nil && available {
				statuses = append(statuses, status)
			}
		}
	}
	statusByID := make(map[string]store.ServerStatus, len(statuses))
	for _, status := range statuses {
		statusByID[status.ServerID] = status
	}
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].SortOrder != servers[j].SortOrder {
			return servers[i].SortOrder < servers[j].SortOrder
		}
		if servers[i].DisplayName != servers[j].DisplayName {
			return servers[i].DisplayName < servers[j].DisplayName
		}
		return servers[i].Address < servers[j].Address
	})
	global, err := r.ingameStore.IngameSettings(c.Context())
	if err != nil {
		return nil, err
	}
	overrides, err := r.ingameStore.ListIngameServerSettings(c.Context())
	if err != nil {
		return nil, err
	}
	overrideByKey := make(map[string]store.IngameServerSettings, len(overrides))
	for _, override := range overrides {
		overrideByKey[override.ServerKey] = override
	}
	groups := make([]adminIngameGroup, 0)
	indexByKey := make(map[string]int)
	for _, server := range servers {
		status, exists := statusByID[server.ID]
		if !server.Enabled || !exists || service.ValidateIngameServerKey(status.ServerKey) != nil {
			continue
		}
		index, exists := indexByKey[status.ServerKey]
		if !exists {
			config := service.ResolveIngameConfig(global, overrideByKey[status.ServerKey], server.DisplayName)
			groups = append(groups, adminIngameGroup{ServerKey: status.ServerKey, Title: config.Appearance.Title})
			index = len(groups) - 1
			indexByKey[status.ServerKey] = index
		}
		groups[index].Instances = append(groups[index].Instances, adminIngameGroupInstance{
			ServerID: server.ID, Name: server.DisplayName, Address: server.Address,
			SortOrder: server.SortOrder, Online: status.Online, Stale: status.Stale,
		})
	}
	return groups, nil
}

func (r *adminRoutes) requireIngameGroup(c fiber.Ctx) (adminIngameGroup, bool) {
	key := strings.TrimSpace(c.Params("server_key"))
	if service.ValidateIngameServerKey(key) != nil {
		_ = sendError(c, 400, "invalid_server_key", "server key is invalid")
		return adminIngameGroup{}, false
	}
	groups, err := r.ingameGroups(c)
	if err != nil {
		_ = sendError(c, 503, "ingame_groups_unavailable", "in-game server groups are unavailable")
		return adminIngameGroup{}, false
	}
	for _, group := range groups {
		if group.ServerKey == key {
			return group, true
		}
	}
	_ = sendError(c, 404, "ingame_group_not_found", "server group was not found")
	return adminIngameGroup{}, false
}

func (r *adminRoutes) invalidateIngameAll() {
	if r.ingame != nil {
		r.ingame.InvalidateAll()
	}
}

func (r *adminRoutes) invalidateIngameGroup(serverKey string) {
	if r.ingame != nil {
		r.ingame.InvalidateServer(serverKey)
	}
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
