import { adminWrite, request } from './client'

export type IngameMode = 'inherit' | 'override' | 'hidden'
export type IngameDocumentKey = 'introduction' | 'commands' | 'resources'

export interface IngameMetricDefinition {
  key: string
  label: string
  mode: string
  ranking_metric: string
  format: string
}

export interface IngameSettings {
  enabled: boolean
  title: string
  description: string
  banner_url: string
  background_url: string
  website_url: string
  show_announcements: boolean
  show_players: boolean
  show_highlights: boolean
  show_server_intro: boolean
  show_server_status: boolean
  highlight_metrics: [string, string, string]
  home_cache_seconds: number
  player_cache_seconds: number
  ranking_cache_seconds: number
  content_cache_seconds: number
  updated_at: number
}

export interface IngameServerSettings {
  server_key: string
  title_mode: Exclude<IngameMode, 'hidden'>
  title: string
  description_mode: IngameMode
  description: string
  banner_mode: IngameMode
  banner_url: string
  background_mode: IngameMode
  background_url: string
  website_mode: IngameMode
  website_url: string
  highlight_mode: Exclude<IngameMode, 'hidden'>
  highlight_metrics: [string, string, string]
  updated_at: number
}

export interface IngameServerDocument {
  server_key: string
  key: IngameDocumentKey
  mode: IngameMode
  content_markdown: string
  updated_at: number
}

export interface IngameAdminConfig {
  settings: IngameSettings
  metric_catalog: IngameMetricDefinition[]
  public_origin: string
}

export interface IngameGroupInstance {
  server_id: string
  name: string
  address: string
  sort_order: number
  online: boolean
  stale: boolean
}

export interface IngameGroup {
  server_key: string
  title: string
  instances: IngameGroupInstance[]
}

export interface IngameGroupConfig {
  settings: IngameServerSettings
  documents: IngameServerDocument[]
  metric_catalog: IngameMetricDefinition[]
  server_key: string
  title: string
  instances: IngameGroupInstance[]
  public_origin: string
}

export const ingameAPI = {
  ingameSettings: () => request<IngameAdminConfig>('/api/v1/admin/ingame'),
  saveIngameSettings: (settings: IngameSettings) => adminWrite<IngameAdminConfig>('/api/v1/admin/ingame', 'PUT', settings),
  ingameGroups: () => request<IngameGroup[]>('/api/v1/admin/ingame/groups'),
  ingameGroup: (serverKey: string) => request<IngameGroupConfig>(`/api/v1/admin/ingame/groups/${encodeURIComponent(serverKey)}`),
  saveIngameGroup: (serverKey: string, settings: IngameServerSettings) => adminWrite<IngameServerSettings>(`/api/v1/admin/ingame/groups/${encodeURIComponent(serverKey)}`, 'PUT', settings),
  saveIngameGroupDocument: (serverKey: string, document: IngameServerDocument) => adminWrite<IngameServerDocument>(`/api/v1/admin/ingame/groups/${encodeURIComponent(serverKey)}/documents/${document.key}`, 'PUT', document),
}
