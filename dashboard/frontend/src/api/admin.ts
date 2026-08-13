import { adminWrite, queryString, request } from './client'
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
export interface DataGrowthStatus { aggregate: AggregateStatus; settings: DataMaintenanceSettings; stats_database: DatabaseUsage; dashboard_database: DatabaseUsage; log_bytes: number; retention_runs: number; retention_plan: RetentionPlan; analysis: AnalysisStatus; incident_retention_plan: IncidentRetentionPlan }

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
}
