import { adminWrite, queryString, request } from './client'

export interface FooterLink { id?: string; label: string; url: string }
export type SiteLanguage = 'zh-CN' | 'en'
export type SiteTheme = 'light' | 'dark'
export type SiteDocumentKey = 'introduction' | 'commands' | 'resources'
export interface SiteDocument { key: SiteDocumentKey; enabled: boolean; content_markdown: string; updated_at: number }
export interface Site { language: SiteLanguage; browser_title: string; theme: SiteTheme; footer_enabled: boolean; background_image_url: string; footer_links: FooterLink[]; steam_openid_enabled: boolean; a2s_refresh_seconds: number; site_documents: SiteDocumentKey[]; configured: boolean }
export interface SiteSettings { language: SiteLanguage; browser_title: string; theme: SiteTheme; footer_enabled: boolean; background_image_url: string; public_origin: string; steam_openid_enabled: boolean; steam_openid_proxy_url: string; a2s_refresh_seconds: number; a2s_jitter_seconds: number; a2s_retry_count: number; seo_enabled: boolean; seo_description: string; seo_image_url: string; footer_links: FooterLink[] }
export interface CoreOverview { total_players: number; active_players_7d: number; total_active_play_seconds: number; completed_pve_runs: number; completed_versus_runs: number }
export interface PVEOverview { common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; rescues: number }
export interface VersusOverview { completed_matches: number; completed_halves: number; human_controlled_infected_kills: number; human_survivor_controls: number }
export interface Overview { core: CoreOverview; pve: PVEOverview; versus: VersusOverview; generated_at: string }
export interface Announcement { id: string; title: string; content_markdown: string; created_at: number; updated_at: number }
export interface AnnouncementPage { items: Announcement[]; total: number; page: number; limit: number }
export interface AnnouncementInput { title: string; content_markdown: string }

export const siteAPI = {
  site: () => request<Site>('/api/v1/site'),
  overview: () => request<Overview>('/api/v1/dashboard/overview'),
  siteDocument: (key: SiteDocumentKey) => request<SiteDocument>(`/api/v1/site-documents/${key}`),
  announcements: (page = 1, limit = 20, title = '', year?: number) => request<AnnouncementPage>(`/api/v1/announcements?${queryString({ page, limit, title, year })}`),
  announcementYears: () => request<number[]>('/api/v1/announcements/years'),
  adminSite: () => request<SiteSettings>('/api/v1/admin/site'),
  saveSite: (site: SiteSettings) => adminWrite<SiteSettings>('/api/v1/admin/site', 'PUT', site),
  adminSiteDocuments: () => request<SiteDocument[]>('/api/v1/admin/site-documents'),
  saveSiteDocument: (document: SiteDocument) => adminWrite<SiteDocument>(`/api/v1/admin/site-documents/${document.key}`, 'PUT', document),
}
