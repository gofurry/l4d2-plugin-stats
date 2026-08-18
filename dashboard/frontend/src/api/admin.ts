import { adminDownload, adminWrite, queryString, request } from './client'
import type { Announcement, AnnouncementInput, AnnouncementPage } from './site'

export interface AdminIdentity { username: string; created_at: number; updated_at: number; password_changed_at: number; monitor_enabled: boolean }
export interface AggregateStatus { aggregate_version: number; state: string; last_started_at: number; last_finished_at: number; source_rows: number; aggregate_rows: number; source_watermark: number; last_duration_ms: number; last_changed_days: number; last_build_mode: string; last_error?: string }
export interface DataMaintenanceSettings { aggregate_interval_minutes: number; detail_retention_days: number; session_retention_days: number; result_retention_days: number; incident_retention_days: number; updated_at: number }
export interface DatabaseUsage { driver: string; bytes: number; wal_bytes?: number }
export interface RetentionPlan { aggregate_version: number; generated_at: number; detail_cutoff: number; session_cutoff: number; result_cutoff: number; equipment_rows_eligible: number; versus_class_rows_eligible: number; session_rows_eligible: number; versus_round_results_eligible: number; versus_run_results_eligible: number; source_watermark: number; plan_id: string; deletion_enabled: boolean; aggregate_coverage_ready: boolean }
export interface RetentionResult { run_id: string; executed_at: number; equipment_rows: number; versus_class_rows: number; session_rows: number; versus_round_result_rows: number; versus_run_result_rows: number }
export interface AnalysisStatus { incident_version: number; incident_rows: number; capture_enabled_rounds: number; complete_rounds: number; complete_ratio: number; rows_last_30d: number; earliest_incident_at: number; latest_incident_at: number; projected_rows_for_retention: number; retention_days: number; cleanup_runs: number }
export interface IncidentRetentionPlan { incident_version: number; generated_at: number; cutoff: number; incident_rows_eligible: number; unknown_version_rows: number; candidate_watermark: number; plan_id: string; deletion_enabled: boolean }
export interface IncidentRetentionResult { run_id: string; executed_at: number; incident_rows: number }
export interface DataGrowthStatus { aggregate: AggregateStatus; settings: DataMaintenanceSettings; stats_database: DatabaseUsage; dashboard_database: DatabaseUsage; log_bytes: number; retention_runs: number; retention_plan: RetentionPlan; analysis: AnalysisStatus; incident_retention_plan: IncidentRetentionPlan; chat_audit?: ChatAuditStatus; geoip?: GeoIPSettings }
export interface AchievementEngineState { achievement_contract_version: number; catalog_items: number; evaluated_players: number; pending_backfill: number; global_source_watermark: number; dirty_cursor_watermark: number; dirty_cursor_steam_id: string; backfill_cursor: string; backfill_complete: boolean; last_run_at: number; last_success_at: number; last_error?: string; updated_at: number }
export interface ChatAuditSettings { enabled: boolean; retention_days: number; last_cleanup_at: number; updated_at: number }
export interface ChatRetentionPlan { plan_id: string; retention_days: number; cutoff: number; delete_count: number }
export interface ChatAuditStatus { database: DatabaseUsage; message_count: number; oldest_message_at: number; newest_message_at: number; retention_days: number; last_cleanup_at: number; ingestion_lag: number; dropped_count: number; known_gap_count: number; last_ingest_at: number }
export interface ChatSearchFilter { from?: number; to?: number; server_key?: string; steam_id?: string; nickname?: string; map_name?: string; game_mode?: string; team?: string; channel?: string; message_kind?: string; keyword?: string; boot_id?: string; cursor_at?: number; cursor_id?: string; limit?: number }
export interface ChatMessage { message_id: string; server_key: string; boot_id: string; chat_seq: number; session_id?: string; steam_id?: string; source_user_id: number; player_name: string; occurred_at: number; map_name: string; game_mode: string; team: string; channel: string; alive: boolean; command_like: boolean; content: string }
export interface ChatSearchPage { items: ChatMessage[]; next_cursor_at?: number; next_cursor_id?: string }
export interface GeoIPEntry { provider: string; country: string; country_code: string; province: string; city: string; district: string; adcode: string; longitude?: number; latitude?: number; coordinate_system: string; precision: string; status: string; error_code?: string; resolved_at: number; expires_at: number }
export interface GeoIPSettings { enabled: boolean; provider: string; api_key_configured: boolean; api_key_masked?: string; last_success_at: number; last_error_at: number; last_error_code?: string; ipv4_status: string; ipv6_status: string; cache_count: number; pending_count: number; updated_at: number }
export interface ConnectionAuditFilter { from?: number; to?: number; server_key?: string; steam_id?: string; nickname?: string; ip_address?: string; location?: string; cursor_at?: number; cursor_id?: string; limit?: number }
export interface ConnectionAuditRow { session_id: string; server_key: string; steam_id: string; player_name: string; ip_address: string; started_at: number; ended_at?: number; connected_seconds: number; status: string; disconnect_reason: string; geoip?: GeoIPEntry }
export interface ConnectionAuditPage { items: ConnectionAuditRow[]; next_cursor_at?: number; next_cursor_id?: string }

export const adminAPI = {
  setupStatus: () => request<{ required: boolean; expires_at?: string }>('/api/v1/setup/status'),
  setupAdmin: (body: { setup_token: string; username: string; password: string }) => request('/api/v1/setup/admin', { method: 'POST', body: JSON.stringify(body) }),
  login: (username: string, password: string) => request('/api/v1/admin/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  logout: () => adminWrite('/api/v1/admin/auth/logout', 'POST'),
  adminMe: () => request<AdminIdentity>('/api/v1/admin/auth/me'),
  adminAnnouncements: (page = 1, limit = 50) => request<AnnouncementPage>(`/api/v1/admin/announcements?${queryString({ page, limit })}`),
  createAnnouncement: (value: AnnouncementInput) => adminWrite<Announcement>('/api/v1/admin/announcements', 'POST', value),
  updateAnnouncement: (id: string, value: AnnouncementInput) => adminWrite<Announcement>(`/api/v1/admin/announcements/${id}`, 'PUT', value),
  deleteAnnouncement: (id: string) => adminWrite(`/api/v1/admin/announcements/${id}`, 'DELETE'),
  updateAccount: (username: string) => adminWrite('/api/v1/admin/account', 'PUT', { username }),
  updatePassword: (current_password: string, new_password: string) => adminWrite('/api/v1/admin/account/password', 'PUT', { current_password, new_password }),
  dataStatus: () => request<DataGrowthStatus>('/api/v1/admin/data/status'),
  dataSettings: () => request<DataMaintenanceSettings>('/api/v1/admin/data/settings'),
  saveDataSettings: (settings: DataMaintenanceSettings) => adminWrite<DataMaintenanceSettings>('/api/v1/admin/data/settings', 'PUT', settings),
  aggregateNow: () => adminWrite<DataGrowthStatus>('/api/v1/admin/data/aggregate', 'POST'),
  retentionPlan: () => request<RetentionPlan>('/api/v1/admin/data/retention/plan'),
  applyRetention: (plan_id: string) => adminWrite<RetentionResult>('/api/v1/admin/data/retention/apply', 'POST', { plan_id }),
  applyIncidentRetention: (plan_id: string) => adminWrite<IncidentRetentionResult>('/api/v1/admin/data/incidents/retention/apply', 'POST', { plan_id }),
  achievementEngineState: () => request<AchievementEngineState>('/api/v1/admin/data/achievement-engine'),
  chatAuditSettings: () => request<ChatAuditSettings>('/api/v1/admin/audit/chat/settings'),
  saveChatAuditSettings: (settings: ChatAuditSettings) => adminWrite<ChatRetentionPlan | ChatAuditSettings>('/api/v1/admin/audit/chat/settings', 'PUT', settings),
  confirmChatAuditSettings: (plan_id: string, settings: ChatAuditSettings) => adminWrite<{ deleted: number; settings: ChatAuditSettings }>('/api/v1/admin/audit/chat/settings/confirm', 'POST', { plan_id, settings }),
  chatAuditStatus: () => request<ChatAuditStatus>('/api/v1/admin/audit/chat/status'),
  searchChatAudit: (filter: ChatSearchFilter) => adminWrite<ChatSearchPage>('/api/v1/admin/audit/chat/search', 'POST', filter),
  exportChatAudit: (format: 'csv' | 'jsonl', filter: ChatSearchFilter) => adminDownload('/api/v1/admin/audit/chat/export', { format, filter }),
  geoIPSettings: () => request<GeoIPSettings>('/api/v1/admin/audit/geoip/settings'),
  saveGeoIPSettings: (body: { enabled: boolean; api_key?: string; clear_api_key?: boolean }) => adminWrite<GeoIPSettings>('/api/v1/admin/audit/geoip/settings', 'PUT', body),
  testGeoIP: (ip: string) => adminWrite<GeoIPEntry>('/api/v1/admin/audit/geoip/test', 'POST', { ip }),
  searchConnections: (filter: ConnectionAuditFilter) => adminWrite<ConnectionAuditPage>('/api/v1/admin/audit/connections/search', 'POST', filter),
}
