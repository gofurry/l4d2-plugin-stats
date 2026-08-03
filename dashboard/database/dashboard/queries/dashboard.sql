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
SELECT title, footer_text, updated_at
FROM site_settings
WHERE id = 1;

-- name: UpsertSiteSettings :exec
INSERT INTO site_settings (id, title, footer_text, updated_at)
VALUES (1, ?1, ?2, ?3)
ON CONFLICT(id) DO UPDATE SET
  title = excluded.title,
  footer_text = excluded.footer_text,
  updated_at = excluded.updated_at;

-- name: DeleteFooterLinks :exec
DELETE FROM footer_links;

-- name: CreateFooterLink :exec
INSERT INTO footer_links (
  label, url, sort_order, open_new_tab, enabled, created_at, updated_at
) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7);

-- name: ListEnabledFooterLinks :many
SELECT label, url, open_new_tab
FROM footer_links
WHERE enabled = 1
ORDER BY sort_order, id;

-- name: DeleteGameServers :exec
DELETE FROM game_servers;

-- name: CreateGameServer :exec
INSERT INTO game_servers (
  server_key, display_name, connect_address, query_address,
  is_primary, enabled, sort_order, created_at, updated_at
) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9);

-- name: GetPrimaryServer :one
SELECT
  id, server_key, display_name, connect_address, query_address,
  is_primary, enabled, sort_order
FROM game_servers
WHERE is_primary = 1 AND enabled = 1
LIMIT 1;
