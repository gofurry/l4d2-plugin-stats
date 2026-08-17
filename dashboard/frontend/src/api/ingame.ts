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
  website_url: string
  show_announcements: boolean
  show_players: boolean
  show_highlights: boolean
  highlight_metrics: [string, string, string]
  home_cache_seconds: number
  player_cache_seconds: number
  ranking_cache_seconds: number
  content_cache_seconds: number
  updated_at: number
}

export interface IngameServerSettings {
  server_id: string
  title_mode: Exclude<IngameMode, 'hidden'>
  title: string
  description_mode: IngameMode
  description: string
  banner_mode: IngameMode
  banner_url: string
  website_mode: IngameMode
  website_url: string
  highlight_mode: Exclude<IngameMode, 'hidden'>
  highlight_metrics: [string, string, string]
  updated_at: number
}

export interface IngameServerDocument {
  server_id: string
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

export interface IngameServerConfig {
  settings: IngameServerSettings
  documents: IngameServerDocument[]
  metric_catalog: IngameMetricDefinition[]
  server_key: string
  public_origin: string
}

export const ingameAPI = {
  ingameSettings: () => request<IngameAdminConfig>('/api/v1/admin/ingame'),
  saveIngameSettings: (settings: IngameSettings) => adminWrite<IngameAdminConfig>('/api/v1/admin/ingame', 'PUT', settings),
  serverIngameSettings: (id: string) => request<IngameServerConfig>(`/api/v1/admin/servers/${id}/ingame`),
  saveServerIngameSettings: (id: string, settings: IngameServerSettings) => adminWrite<IngameServerSettings>(`/api/v1/admin/servers/${id}/ingame`, 'PUT', settings),
  serverIngameDocuments: (id: string) => request<IngameServerDocument[]>(`/api/v1/admin/servers/${id}/ingame/documents`),
  saveServerIngameDocument: (id: string, document: IngameServerDocument) => adminWrite<IngameServerDocument>(`/api/v1/admin/servers/${id}/ingame/documents/${document.key}`, 'PUT', document),
}
