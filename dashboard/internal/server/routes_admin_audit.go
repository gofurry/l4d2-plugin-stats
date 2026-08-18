package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func (r *adminRoutes) chatAuditSettings(c fiber.Ctx) error {
	settings, err := r.chatAudit.Settings(c.Context())
	if err != nil {
		return sendError(c, 503, "chat_audit_unavailable", "chat audit settings are unavailable")
	}
	return sendData(c, 200, settings)
}

func (r *adminRoutes) updateChatAuditSettings(c fiber.Ctx) error {
	var settings store.ChatAuditSettings
	if err := c.Bind().Body(&settings); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	plan, err := r.chatAudit.UpdateSettings(c.Context(), settings)
	if err != nil {
		return sendError(c, 400, "invalid_chat_audit_settings", err.Error())
	}
	if plan.PlanID != "" {
		return sendData(c, 200, plan)
	}
	return sendData(c, 200, settings)
}

func (r *adminRoutes) confirmChatAuditSettings(c fiber.Ctx) error {
	var body struct {
		PlanID   string                  `json:"plan_id"`
		Settings store.ChatAuditSettings `json:"settings"`
	}
	if err := c.Bind().Body(&body); err != nil || body.PlanID == "" {
		return sendError(c, 400, "invalid_body", "plan_id and settings are required")
	}
	deleted, err := r.chatAudit.ConfirmSettings(c.Context(), body.PlanID, body.Settings)
	if err != nil {
		return sendError(c, 409, "chat_retention_confirmation_failed", err.Error())
	}
	return sendData(c, 200, fiber.Map{"deleted": deleted, "settings": body.Settings})
}

func (r *adminRoutes) chatAuditStatus(c fiber.Ctx) error {
	status, err := r.chatAudit.Status(c.Context())
	if err != nil {
		return sendError(c, 503, "chat_audit_unavailable", "chat audit status is unavailable")
	}
	return sendData(c, 200, status)
}

func (r *adminRoutes) searchChatAudit(c fiber.Ctx) error {
	var filter store.ChatSearchFilter
	if err := c.Bind().Body(&filter); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if err := validateChatFilter(&filter); err != nil {
		return sendError(c, 400, "invalid_chat_filter", err.Error())
	}
	page, err := r.chatAudit.Search(c.Context(), filter)
	if err != nil {
		return sendError(c, 503, "chat_search_unavailable", "chat search is unavailable")
	}
	return sendData(c, 200, page)
}

func validateChatFilter(filter *store.ChatSearchFilter) error {
	if filter.From == 0 && filter.To == 0 {
		filter.From = time.Now().Add(-24 * time.Hour).Unix()
	}
	if filter.From > 0 && filter.To > 0 && filter.From > filter.To {
		return fmt.Errorf("from must not be after to")
	}
	if len(filter.Keyword) > 256 || len(filter.Nickname) > 128 || len(filter.MapName) > 128 || len(filter.BootID) > 128 {
		return fmt.Errorf("one or more filters are too long")
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 100
	}
	return nil
}

func (r *adminRoutes) exportChatAudit(c fiber.Ctx) error {
	var body struct {
		Format string                 `json:"format"`
		Filter store.ChatSearchFilter `json:"filter"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	body.Format = strings.ToLower(strings.TrimSpace(body.Format))
	if body.Format != "csv" && body.Format != "jsonl" {
		return sendError(c, 400, "invalid_export_format", "format must be csv or jsonl")
	}
	if err := validateChatFilter(&body.Filter); err != nil {
		return sendError(c, 400, "invalid_chat_filter", err.Error())
	}
	admin := c.Locals("admin").(*store.AdminAccount)
	reader, writer := io.Pipe()
	format, filter, adminName := body.Format, body.Filter, admin.Username
	go r.streamChatExport(writer, format, filter, adminName)
	c.Set(fiber.HeaderContentType, exportContentType(format))
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="chat-audit-`+time.Now().UTC().Format("20060102T150405Z")+`.`+format+`"`)
	return c.SendStream(reader)
}

func (r *adminRoutes) streamChatExport(writer *io.PipeWriter, format string, filter store.ChatSearchFilter, admin string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer writer.Close()
	count := int64(0)
	completed := false
	defer func() {
		auditCtx, auditCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer auditCancel()
		_ = r.chatAudit.RecordExport(auditCtx, admin, format, filter, &count, completed)
	}()
	var csvWriter *csv.Writer
	var jsonWriter *json.Encoder
	if format == "csv" {
		csvWriter = csv.NewWriter(writer)
		if err := csvWriter.Write(chatCSVHeader()); err != nil {
			return
		}
	} else {
		jsonWriter = json.NewEncoder(writer)
		jsonWriter.SetEscapeHTML(false)
	}
	filter.Limit = 200
	for {
		page, err := r.chatAudit.Search(ctx, filter)
		if err != nil {
			return
		}
		for _, message := range page.Items {
			if format == "csv" {
				if err := csvWriter.Write(chatCSVRow(message)); err != nil {
					return
				}
			} else if err := jsonWriter.Encode(message); err != nil {
				return
			}
			count++
		}
		if csvWriter != nil {
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return
			}
		}
		if page.NextCursorID == "" {
			completed = true
			return
		}
		filter.CursorAt, filter.CursorID = page.NextCursorAt, page.NextCursorID
	}
}

func exportContentType(format string) string {
	if format == "csv" {
		return "text/csv; charset=utf-8"
	}
	return "application/x-ndjson; charset=utf-8"
}

func chatCSVHeader() []string {
	return []string{"message_id", "server_key", "boot_id", "chat_seq", "session_id", "steam_id", "source_user_id", "player_name", "occurred_at", "map_name", "game_mode", "team", "channel", "alive", "command_like", "content"}
}

func chatCSVRow(message store.ChatMessage) []string {
	return []string{
		csvSafe(message.MessageID), csvSafe(message.ServerKey), csvSafe(message.BootID), strconv.FormatInt(message.ChatSeq, 10),
		csvSafe(message.SessionID), csvSafe(message.SteamID), strconv.FormatInt(message.SourceUserID, 10), csvSafe(message.PlayerName),
		strconv.FormatInt(message.OccurredAt, 10), csvSafe(message.MapName), csvSafe(message.GameMode), csvSafe(message.Team),
		csvSafe(message.Channel), strconv.FormatBool(message.Alive), strconv.FormatBool(message.CommandLike), csvSafe(message.Content),
	}
}

func csvSafe(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func (r *adminRoutes) geoIPSettings(c fiber.Ctx) error {
	settings, err := r.geoIP.Settings(c.Context())
	if err != nil {
		return sendError(c, 503, "geoip_unavailable", "GeoIP settings are unavailable")
	}
	return sendData(c, 200, settings)
}

func (r *adminRoutes) updateGeoIPSettings(c fiber.Ctx) error {
	var body struct {
		Enabled  bool   `json:"enabled"`
		APIKey   string `json:"api_key"`
		ClearKey bool   `json:"clear_api_key"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if err := r.geoIP.UpdateSettings(c.Context(), body.Enabled, body.APIKey, body.ClearKey); err != nil {
		return sendError(c, 400, "invalid_geoip_settings", err.Error())
	}
	settings, _ := r.geoIP.Settings(c.Context())
	return sendData(c, 200, settings)
}

func (r *adminRoutes) testGeoIP(c fiber.Ctx) error {
	var body struct {
		IP string `json:"ip"`
	}
	if err := c.Bind().Body(&body); err != nil || strings.TrimSpace(body.IP) == "" {
		return sendError(c, 400, "invalid_body", "a public test IP is required")
	}
	entry, err := r.geoIP.Test(c.Context(), body.IP)
	if err != nil {
		return sendError(c, 409, "geoip_test_failed", err.Error())
	}
	return sendData(c, 200, entry)
}

func (r *adminRoutes) searchConnections(c fiber.Ctx) error {
	var filter store.ConnectionAuditFilter
	if err := c.Bind().Body(&filter); err != nil {
		return sendError(c, 400, "invalid_body", "request body is invalid")
	}
	if filter.From == 0 && filter.To == 0 {
		filter.From = time.Now().Add(-24 * time.Hour).Unix()
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 100
	}
	page, err := r.geoIP.Connections(c.Context(), filter)
	if err != nil {
		return sendError(c, 503, "connection_audit_unavailable", "connection audit is unavailable")
	}
	return sendData(c, 200, page)
}
