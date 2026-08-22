package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	chatauditdb "github.com/gofurry/l4d2-plugin-stats/dashboard/database/chataudit"
	"github.com/pressly/goose/v3"
)

type chatAuditStore struct {
	db   *sql.DB
	path string
}

func OpenChatAudit(ctx context.Context, path string) (ChatAuditStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create chat audit database directory: %w", err)
	}
	_, statErr := os.Stat(path)
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open chat audit database: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping chat audit database: %w", err)
	}
	if errors.Is(statErr, os.ErrNotExist) {
		_ = os.Chmod(path, 0o600)
	}
	goose.SetBaseFS(chatauditdb.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set chat audit migration dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate chat audit database: %w", err)
	}
	return &chatAuditStore{db: db, path: path}, nil
}

func (s *chatAuditStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *chatAuditStore) Close() error                   { return s.db.Close() }

func (s *chatAuditStore) SchemaVersion(ctx context.Context) (int64, error) {
	var version int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM chat_schema_migrations`).Scan(&version)
	return version, err
}

func (s *chatAuditStore) Cursor(ctx context.Context, bootID string) (ChatIngestCursor, error) {
	var result ChatIngestCursor
	err := s.db.QueryRowContext(ctx, `SELECT boot_id, server_key, last_chat_seq, gap_count, last_gap_from, last_gap_to, updated_at FROM chat_ingest_cursors WHERE boot_id = ?`, bootID).Scan(
		&result.BootID, &result.ServerKey, &result.LastChatSeq, &result.GapCount, &result.LastGapFrom, &result.LastGapTo, &result.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ChatIngestCursor{BootID: bootID}, nil
	}
	return result, err
}

func (s *chatAuditStore) Ingest(ctx context.Context, state ChatCaptureState, messages []ChatMessage, gapFrom, gapTo int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	lastSeq := int64(0)
	for _, message := range messages {
		_, err = tx.ExecContext(ctx, `INSERT INTO chat_messages (
message_id, server_key, boot_id, chat_seq, session_id, steam_id, source_user_id,
player_name, occurred_at, map_name, game_mode, team, channel, alive, command_like, content
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(message_id) DO NOTHING`,
			message.MessageID, message.ServerKey, message.BootID, message.ChatSeq, nullableText(message.SessionID), nullableText(message.SteamID), message.SourceUserID,
			message.PlayerName, message.OccurredAt, message.MapName, message.GameMode, message.Team, message.Channel, boolInt(message.Alive), boolInt(message.CommandLike), message.Content,
		)
		if err != nil {
			return fmt.Errorf("insert chat message: %w", err)
		}
		if message.ChatSeq > lastSeq {
			lastSeq = message.ChatSeq
		}
	}
	if gapTo > lastSeq {
		lastSeq = gapTo
	}
	if lastSeq == 0 {
		return tx.Commit()
	}
	now := time.Now().Unix()
	gapIncrement := int64(0)
	if gapTo >= gapFrom && gapFrom > 0 {
		gapIncrement = gapTo - gapFrom + 1
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO chat_ingest_cursors (
boot_id, server_key, last_chat_seq, gap_count, last_gap_from, last_gap_to, updated_at
) VALUES (?,?,?,?,?,?,?) ON CONFLICT(boot_id) DO UPDATE SET
server_key=excluded.server_key,
last_chat_seq=CASE WHEN excluded.last_chat_seq > chat_ingest_cursors.last_chat_seq THEN excluded.last_chat_seq ELSE chat_ingest_cursors.last_chat_seq END,
gap_count=chat_ingest_cursors.gap_count + excluded.gap_count,
last_gap_from=CASE WHEN excluded.gap_count > 0 THEN excluded.last_gap_from ELSE chat_ingest_cursors.last_gap_from END,
last_gap_to=CASE WHEN excluded.gap_count > 0 THEN excluded.last_gap_to ELSE chat_ingest_cursors.last_gap_to END,
updated_at=excluded.updated_at`, state.BootID, state.ServerKey, lastSeq, gapIncrement, gapFrom, gapTo, now)
	if err != nil {
		return fmt.Errorf("advance chat ingest cursor: %w", err)
	}
	return tx.Commit()
}

func (s *chatAuditStore) Search(ctx context.Context, filter ChatSearchFilter) (ChatSearchPage, error) {
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 100
	}
	query := `SELECT message_id, server_key, boot_id, chat_seq, COALESCE(session_id,''), COALESCE(steam_id,''), source_user_id, player_name, occurred_at, map_name, game_mode, team, channel, alive, command_like, content FROM chat_messages WHERE 1=1`
	args := make([]any, 0, 16)
	add := func(clause string, value any) { query += clause; args = append(args, value) }
	if filter.From > 0 {
		add(" AND occurred_at >= ?", filter.From)
	}
	if filter.To > 0 {
		add(" AND occurred_at <= ?", filter.To)
	}
	if filter.ServerKey != "" {
		add(" AND server_key = ?", filter.ServerKey)
	}
	if filter.SteamID != "" {
		add(" AND steam_id = ?", filter.SteamID)
	}
	if filter.Nickname != "" {
		add(" AND instr(lower(player_name), lower(?)) > 0", filter.Nickname)
	}
	if filter.MapName != "" {
		add(" AND map_name = ?", filter.MapName)
	}
	if filter.GameMode != "" {
		add(" AND game_mode = ?", filter.GameMode)
	}
	if filter.Team != "" {
		add(" AND team = ?", filter.Team)
	}
	if filter.Channel != "" {
		add(" AND channel = ?", filter.Channel)
	}
	if filter.MessageKind == "command" {
		query += " AND command_like = 1"
	}
	if filter.MessageKind == "normal" {
		query += " AND command_like = 0"
	}
	if filter.Keyword != "" {
		add(" AND instr(content, ?) > 0", filter.Keyword)
	}
	if filter.BootID != "" {
		add(" AND boot_id = ?", filter.BootID)
	}
	if filter.CursorAt > 0 && filter.CursorID != "" {
		query += " AND (occurred_at < ? OR (occurred_at = ? AND message_id < ?))"
		args = append(args, filter.CursorAt, filter.CursorAt, filter.CursorID)
	}
	query += " ORDER BY occurred_at DESC, message_id DESC LIMIT ?"
	args = append(args, filter.Limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ChatSearchPage{}, err
	}
	defer rows.Close()
	items := make([]ChatMessage, 0, filter.Limit+1)
	for rows.Next() {
		var item ChatMessage
		var alive, commandLike int64
		if err := rows.Scan(&item.MessageID, &item.ServerKey, &item.BootID, &item.ChatSeq, &item.SessionID, &item.SteamID, &item.SourceUserID, &item.PlayerName, &item.OccurredAt, &item.MapName, &item.GameMode, &item.Team, &item.Channel, &alive, &commandLike, &item.Content); err != nil {
			return ChatSearchPage{}, err
		}
		item.Alive, item.CommandLike = alive != 0, commandLike != 0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ChatSearchPage{}, err
	}
	page := ChatSearchPage{Items: items}
	if len(items) > filter.Limit {
		last := items[filter.Limit-1]
		page.Items = items[:filter.Limit]
		page.NextCursorAt, page.NextCursorID = last.OccurredAt, last.MessageID
	}
	return page, nil
}

func (s *chatAuditStore) RetentionPlan(ctx context.Context, retentionDays, now int64) (ChatRetentionPlan, error) {
	plan := ChatRetentionPlan{RetentionDays: retentionDays}
	if retentionDays == 0 {
		return plan, nil
	}
	plan.Cutoff = now - retentionDays*86400
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages WHERE occurred_at < ?`, plan.Cutoff).Scan(&plan.DeleteCount)
	return plan, err
}

func (s *chatAuditStore) ApplyRetention(ctx context.Context, retentionDays, now int64) (int64, error) {
	if retentionDays == 0 {
		return 0, nil
	}
	cutoff := now - retentionDays*86400
	var total int64
	for {
		result, err := s.db.ExecContext(ctx, `DELETE FROM chat_messages WHERE message_id IN (SELECT message_id FROM chat_messages WHERE occurred_at < ? ORDER BY occurred_at LIMIT 500)`, cutoff)
		if err != nil {
			return total, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return total, err
		}
		total += count
		if count < 500 {
			return total, nil
		}
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
	}
}

func (s *chatAuditStore) Status(ctx context.Context, settings ChatAuditSettings, ingestionLag, dropped int64) (ChatAuditStatus, error) {
	status := ChatAuditStatus{RetentionDays: settings.RetentionDays, LastCleanupAt: settings.LastCleanupAt, IngestionLag: ingestionLag, DroppedCount: dropped}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MIN(occurred_at),0), COALESCE(MAX(occurred_at),0) FROM chat_messages`).Scan(&status.MessageCount, &status.OldestMessageAt, &status.NewestMessageAt); err != nil {
		return status, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(gap_count),0), COALESCE(MAX(updated_at),0) FROM chat_ingest_cursors`).Scan(&status.KnownGapCount, &status.LastIngestAt); err != nil {
		return status, err
	}
	usage, err := s.DatabaseUsage(ctx)
	if err != nil {
		return status, err
	}
	status.Database = usage
	return status, nil
}

func (s *chatAuditStore) DatabaseUsage(ctx context.Context) (DatabaseUsage, error) {
	var pageCount, pageSize int64
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return DatabaseUsage{}, err
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return DatabaseUsage{}, err
	}
	var walBytes int64
	if info, err := os.Stat(s.path + "-wal"); err == nil {
		walBytes = info.Size()
	}
	return DatabaseUsage{Driver: "sqlite", Bytes: pageCount * pageSize, WALBytes: walBytes}, nil
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
