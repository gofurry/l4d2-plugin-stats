package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (s *statsStore) placeholder(index int) string {
	if s.driver == "postgres" {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func (s *statsStore) ListChatCaptureStates(ctx context.Context) ([]ChatCaptureState, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	rows, err := s.db.QueryContext(queryCtx, `SELECT boot_id, server_key, capture_version, capture_enabled, started_at, ended_at, last_saved_at, observed_count, persisted_count, dropped_count, last_chat_seq, oldest_retained_seq, revision FROM lps_chat_capture_state ORDER BY last_saved_at DESC LIMIT 512`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ChatCaptureState, 0)
	for rows.Next() {
		var item ChatCaptureState
		var enabled int64
		var ended sql.NullInt64
		if err := rows.Scan(&item.BootID, &item.ServerKey, &item.CaptureVersion, &enabled, &item.StartedAt, &ended, &item.LastSavedAt, &item.ObservedCount, &item.PersistedCount, &item.DroppedCount, &item.LastChatSeq, &item.OldestRetainedSeq, &item.Revision); err != nil {
			return nil, err
		}
		item.CaptureEnabled = enabled != 0
		if ended.Valid {
			item.EndedAt = &ended.Int64
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *statsStore) ListChatOutbox(ctx context.Context, bootID string, afterSeq int64, limit int) ([]ChatMessage, error) {
	if limit < 1 || limit > 256 {
		limit = 128
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	query := `SELECT message_id, server_key, boot_id, chat_seq, COALESCE(session_id,''), COALESCE(steam_id,''), source_user_id, player_name, occurred_at, map_name, game_mode, team, channel, alive, command_like, content FROM lps_chat_outbox WHERE boot_id = ` + s.placeholder(1) + ` AND chat_seq > ` + s.placeholder(2) + ` ORDER BY chat_seq LIMIT ` + s.placeholder(3)
	rows, err := s.db.QueryContext(queryCtx, query, bootID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ChatMessage, 0, limit)
	for rows.Next() {
		var item ChatMessage
		var alive, commandLike int64
		if err := rows.Scan(&item.MessageID, &item.ServerKey, &item.BootID, &item.ChatSeq, &item.SessionID, &item.SteamID, &item.SourceUserID, &item.PlayerName, &item.OccurredAt, &item.MapName, &item.GameMode, &item.Team, &item.Channel, &alive, &commandLike, &item.Content); err != nil {
			return nil, err
		}
		item.Alive, item.CommandLike = alive != 0, commandLike != 0
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *statsStore) OldestChatOutboxSeq(ctx context.Context, bootID string) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	query := `SELECT COALESCE(MIN(chat_seq),0) FROM lps_chat_outbox WHERE boot_id = ` + s.placeholder(1)
	var value int64
	err := s.db.QueryRowContext(queryCtx, query, bootID).Scan(&value)
	return value, err
}

func (s *statsStore) ConnectionAudit(ctx context.Context, filter ConnectionAuditFilter) (ConnectionAuditPage, error) {
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 100
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	query := `SELECT session_id, server_key, steam_id, player_name, ip_address, started_at, ended_at, connected_seconds, status, disconnect_reason FROM lps_sessions WHERE 1=1`
	args := make([]any, 0, 12)
	add := func(clause string, value any) {
		args = append(args, value)
		query += strings.ReplaceAll(clause, "?", s.placeholder(len(args)))
	}
	if filter.From > 0 {
		add(" AND started_at >= ?", filter.From)
	}
	if filter.To > 0 {
		add(" AND started_at <= ?", filter.To)
	}
	if filter.ServerKey != "" {
		add(" AND server_key = ?", filter.ServerKey)
	}
	if filter.SteamID != "" {
		add(" AND steam_id = ?", filter.SteamID)
	}
	if filter.Nickname != "" {
		add(" AND LOWER(player_name) LIKE ?", "%"+strings.ToLower(filter.Nickname)+"%")
	}
	if filter.IPAddress != "" {
		add(" AND ip_address = ?", filter.IPAddress)
	}
	if filter.CursorAt > 0 && filter.CursorID != "" {
		args = append(args, filter.CursorAt)
		p1 := s.placeholder(len(args))
		args = append(args, filter.CursorAt)
		p2 := s.placeholder(len(args))
		args = append(args, filter.CursorID)
		p3 := s.placeholder(len(args))
		query += " AND (started_at < " + p1 + " OR (started_at = " + p2 + " AND session_id < " + p3 + "))"
	}
	args = append(args, filter.Limit+1)
	query += " ORDER BY started_at DESC, session_id DESC LIMIT " + s.placeholder(len(args))
	rows, err := s.db.QueryContext(queryCtx, query, args...)
	if err != nil {
		return ConnectionAuditPage{}, err
	}
	defer rows.Close()
	items := make([]ConnectionAuditRow, 0, filter.Limit+1)
	for rows.Next() {
		var item ConnectionAuditRow
		var ended sql.NullInt64
		if err := rows.Scan(&item.SessionID, &item.ServerKey, &item.SteamID, &item.PlayerName, &item.IPAddress, &item.StartedAt, &ended, &item.ConnectedSeconds, &item.Status, &item.DisconnectReason); err != nil {
			return ConnectionAuditPage{}, err
		}
		if ended.Valid {
			item.EndedAt = &ended.Int64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ConnectionAuditPage{}, err
	}
	page := ConnectionAuditPage{Items: items}
	if len(items) > filter.Limit {
		last := items[filter.Limit-1]
		page.Items = items[:filter.Limit]
		page.NextCursorAt, page.NextCursorID = last.StartedAt, last.SessionID
	}
	return page, nil
}
