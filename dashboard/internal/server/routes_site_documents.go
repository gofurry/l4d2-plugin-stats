package server

import (
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

func registerSiteDocumentRoutes(api fiber.Router, dashboard store.DashboardStore) {
	api.Get("/site-documents/:key", func(c fiber.Ctx) error {
		key := c.Params("key")
		if !validSiteDocumentKey(key) {
			return sendError(c, 400, "invalid_site_document", "site document key is invalid")
		}
		document, err := dashboard.GetSiteDocument(c.Context(), key, true)
		if errors.Is(err, sql.ErrNoRows) {
			return sendError(c, 404, "site_document_not_found", "site document is unavailable")
		}
		if err != nil {
			return sendError(c, 503, "site_document_unavailable", "site document is temporarily unavailable")
		}
		c.Set(fiber.HeaderCacheControl, "public, max-age=300")
		return sendData(c, 200, document)
	})
}

func (r *adminRoutes) listSiteDocuments(c fiber.Ctx) error {
	documents, err := r.dashboard.ListSiteDocuments(c.Context(), false)
	if err != nil {
		return sendError(c, 503, "site_documents_unavailable", "site documents are unavailable")
	}
	return sendData(c, 200, documents)
}

func (r *adminRoutes) updateSiteDocument(c fiber.Ctx) error {
	document := store.SiteDocument{Key: c.Params("key")}
	if err := c.Bind().Body(&document); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	document.Key = c.Params("key")
	if err := validateSiteDocument(&document); err != nil {
		return sendError(c, 400, "invalid_site_document", err.Error())
	}
	updated, err := r.dashboard.UpdateSiteDocument(c.Context(), document)
	if errors.Is(err, sql.ErrNoRows) {
		return sendError(c, 404, "site_document_not_found", "site document was not found")
	}
	if err != nil {
		return sendError(c, 500, "site_document_update_failed", "site document could not be saved")
	}
	r.logger.Info("site document updated", zap.String("site_document", document.Key), zap.String("request_id", c.RequestID()))
	return sendData(c, 200, updated)
}

func validSiteDocumentKey(key string) bool {
	return key == "introduction" || key == "commands" || key == "resources"
}
