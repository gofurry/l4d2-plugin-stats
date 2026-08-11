-- name: CountSiteSettings :one
SELECT COUNT(*) FROM site_settings;

-- name: CountGameServers :one
SELECT COUNT(*) FROM game_servers;

-- name: GetMetadata :one
SELECT value FROM dashboard_metadata WHERE key = ?1;

-- name: UpsertMetadata :exec
INSERT INTO dashboard_metadata (key, value)
VALUES (?1, ?2)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: GetSiteSettings :one
SELECT language, footer_enabled, background_image_url, public_origin, steam_openid_enabled, steam_openid_proxy_port,
       browser_title, theme, a2s_refresh_seconds, a2s_jitter_seconds, a2s_retry_count, updated_at
FROM site_settings
WHERE id = 1;

-- name: UpsertSiteSettings :exec
INSERT INTO site_settings (
  id, language, footer_enabled, background_image_url, public_origin, steam_openid_enabled, steam_openid_proxy_port,
  browser_title, theme, a2s_refresh_seconds, a2s_jitter_seconds, a2s_retry_count, updated_at
)
VALUES (1, ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
ON CONFLICT(id) DO UPDATE SET
  language = excluded.language,
  footer_enabled = excluded.footer_enabled,
  background_image_url = excluded.background_image_url,
  public_origin = excluded.public_origin,
  steam_openid_enabled = excluded.steam_openid_enabled,
  steam_openid_proxy_port = excluded.steam_openid_proxy_port,
  browser_title = excluded.browser_title,
  theme = excluded.theme,
  a2s_refresh_seconds = excluded.a2s_refresh_seconds,
  a2s_jitter_seconds = excluded.a2s_jitter_seconds,
  a2s_retry_count = excluded.a2s_retry_count,
  updated_at = excluded.updated_at;

-- name: DeleteFooterLinks :exec
DELETE FROM footer_links;

-- name: GetSEOSettings :one
SELECT enabled, description, image_url, updated_at
FROM site_seo_settings
WHERE id = 1;

-- name: UpsertSEOSettings :exec
INSERT INTO site_seo_settings (id, enabled, description, image_url, updated_at)
VALUES (1, ?1, ?2, ?3, ?4)
ON CONFLICT(id) DO UPDATE SET
  enabled = excluded.enabled,
  description = excluded.description,
  image_url = excluded.image_url,
  updated_at = excluded.updated_at;

-- name: ListPublicSiteDocuments :many
SELECT key
FROM site_documents
WHERE enabled = 1 AND content_markdown <> ''
ORDER BY CASE key WHEN 'introduction' THEN 1 WHEN 'commands' THEN 2 ELSE 3 END;

-- name: ListSiteDocuments :many
SELECT key, enabled, content_markdown, updated_at
FROM site_documents
ORDER BY CASE key WHEN 'introduction' THEN 1 WHEN 'commands' THEN 2 ELSE 3 END;

-- name: GetPublicSiteDocument :one
SELECT key, enabled, content_markdown, updated_at
FROM site_documents
WHERE key = ?1 AND enabled = 1 AND content_markdown <> '';

-- name: GetSiteDocument :one
SELECT key, enabled, content_markdown, updated_at
FROM site_documents
WHERE key = ?1;

-- name: UpdateSiteDocument :execrows
UPDATE site_documents SET enabled = ?2, content_markdown = ?3, updated_at = ?4
WHERE key = ?1;

-- name: CreateFooterLink :exec
INSERT INTO footer_links (
  id, label, url, sort_order, created_at, updated_at
) VALUES (?1, ?2, ?3, ?4, ?5, ?6);

-- name: ListPublicFooterLinks :many
SELECT label, url
FROM footer_links
ORDER BY sort_order, id;

-- name: ListFooterLinks :many
SELECT id, label, url
FROM footer_links
ORDER BY sort_order, id;

-- name: DeleteGameServers :exec
DELETE FROM game_servers;

-- name: CreateGameServer :exec
INSERT INTO game_servers (
  id, display_name, address, enabled, sort_order, created_at, updated_at
) VALUES (?1, ?2, ?3, 1, ?4, ?5, ?5);

-- name: ListGameServers :many
SELECT
  id, display_name, address, enabled, sort_order
FROM game_servers
ORDER BY sort_order, id;

-- name: GetGameServer :one
SELECT
  id, display_name, address, enabled, sort_order
FROM game_servers
WHERE id = ?1;

-- name: NextGameServerSortOrder :one
SELECT COALESCE(MAX(sort_order), -1) + 1 FROM game_servers;

-- name: UpdateGameServer :execrows
UPDATE game_servers SET
  display_name = ?2,
  address = ?3,
  updated_at = ?4
WHERE id = ?1;

-- name: SetGameServerEnabled :execrows
UPDATE game_servers SET enabled = ?2, updated_at = ?3 WHERE id = ?1;

-- name: SetGameServerSortOrder :execrows
UPDATE game_servers SET sort_order = ?2, updated_at = ?3 WHERE id = ?1;

-- name: DeleteGameServer :execrows
DELETE FROM game_servers WHERE id = ?1;

-- name: CountAdminAccounts :one
SELECT COUNT(*) FROM admin_account;

-- name: CreateAdminAccount :exec
INSERT INTO admin_account (
  id, username, password_hash, jwt_secret, token_version,
  created_at, updated_at, password_changed_at
) VALUES (1, ?1, ?2, ?3, 1, ?4, ?4, ?4);

-- name: GetAdminAccount :one
SELECT
  username, password_hash, jwt_secret, token_version,
  created_at, updated_at, password_changed_at
FROM admin_account
WHERE id = 1;

-- name: UpdateAdminUsername :exec
UPDATE admin_account SET username = ?1, updated_at = ?2 WHERE id = 1;

-- name: UpdateAdminPassword :exec
UPDATE admin_account SET
  password_hash = ?1,
  token_version = token_version + 1,
  updated_at = ?2,
  password_changed_at = ?2
WHERE id = 1;

-- name: CountAnnouncements :one
SELECT COUNT(*) FROM announcements
WHERE (sqlc.arg(title_filter) = '' OR instr(lower(title), lower(sqlc.arg(title_filter))) > 0)
  AND (sqlc.arg(year_filter) = 0 OR CAST(strftime('%Y', updated_at, 'unixepoch') AS INTEGER) = sqlc.arg(year_filter));

-- name: ListAnnouncements :many
SELECT id, title, content_markdown, created_at, updated_at
FROM announcements
WHERE (sqlc.arg(title_filter) = '' OR instr(lower(title), lower(sqlc.arg(title_filter))) > 0)
  AND (sqlc.arg(year_filter) = 0 OR CAST(strftime('%Y', updated_at, 'unixepoch') AS INTEGER) = sqlc.arg(year_filter))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ListAnnouncementYears :many
SELECT DISTINCT CAST(strftime('%Y', updated_at, 'unixepoch') AS INTEGER) AS year
FROM announcements
ORDER BY year DESC;

-- name: GetAnnouncement :one
SELECT id, title, content_markdown, created_at, updated_at
FROM announcements
WHERE id = ?1;

-- name: CreateAnnouncement :exec
INSERT INTO announcements (
  id, title, content_markdown, created_at, updated_at
) VALUES (?1, ?2, ?3, ?4, ?4);

-- name: UpdateAnnouncement :execrows
UPDATE announcements SET
  title = ?2,
  content_markdown = ?3,
  updated_at = ?4
WHERE id = ?1;

-- name: DeleteAnnouncement :execrows
DELETE FROM announcements WHERE id = ?1;

-- name: GetAggregateStatus :one
SELECT state, last_started_at, last_finished_at, source_rows, aggregate_rows, last_error,
       source_watermark, last_duration_ms, last_changed_days, last_build_mode, aggregate_version
FROM aggregate_state WHERE id = 1;

-- name: MarkAggregateStarted :exec
UPDATE aggregate_state SET state = 'building', last_started_at = ?1, last_error = '' WHERE id = 1;

-- name: MarkAggregateFailed :exec
UPDATE aggregate_state SET state = 'failed', last_error = ?1 WHERE id = 1;

-- name: CompleteAggregateBuild :exec
UPDATE aggregate_state SET state = 'ready', last_finished_at = ?1,
  source_rows = ?2, aggregate_rows = ?3, source_watermark = ?4,
  last_duration_ms = ?5, last_changed_days = ?6, last_build_mode = ?7,
  aggregate_version = ?8, last_error = '' WHERE id = 1;

-- name: DeleteAggregateRows :exec
DELETE FROM aggregate_rows;

-- name: DeleteAggregateRowsForDay :exec
DELETE FROM aggregate_rows WHERE day = ?1;

-- name: CountAggregateRows :one
SELECT
  (SELECT COUNT(*) FROM aggregate_rows) +
  (SELECT COUNT(*) FROM aggregate_monthly_rows) +
  (SELECT COUNT(*) FROM aggregate_lifetime_rows);

-- name: InsertAggregateRow :exec
INSERT INTO aggregate_rows (
  kind, day, server_key, steam_id, mode, dimension, metrics_json, aggregate_version
) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8);

-- name: GetDataMaintenanceSettings :one
SELECT aggregate_interval_minutes, detail_retention_days,
       session_retention_days, result_retention_days, updated_at
FROM data_maintenance_settings WHERE id = 1;

-- name: UpdateDataMaintenanceSettings :exec
UPDATE data_maintenance_settings SET
  aggregate_interval_minutes = ?1,
  detail_retention_days = ?2,
  session_retention_days = ?3,
  result_retention_days = ?4,
  updated_at = ?5
WHERE id = 1;

-- name: CreateRetentionRun :exec
INSERT INTO retention_runs (
  id, executed_at, source_watermark, detail_cutoff, session_cutoff, result_cutoff,
  equipment_rows, versus_class_rows, session_rows,
  versus_round_result_rows, versus_run_result_rows, aggregate_version
) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12);

-- name: CountRetentionRuns :one
SELECT COUNT(*) FROM retention_runs;
