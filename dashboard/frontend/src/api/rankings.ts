import { queryString, request } from './client'

export interface RankingEntry { rank: number; steam_id: string; player_name: string; value: number; active_play_seconds: number }
export interface PlayerIdentity { steam_id: string; name: string }
export interface RankingPage { metric: string; mode: string; higher_is_better: boolean; lower_is_better: boolean; items: RankingEntry[]; total: number; self?: RankingEntry; generated_at: string }

export const rankingsAPI = {
  rankings: (params: { mode: string; metric: string; range: string; server?: string; players?: string[]; subject?: string; page: number; limit?: number }) => request<RankingPage>(`/api/v1/rankings?${queryString({ mode: params.mode, metric: params.metric, range: params.range, page: params.page, limit: params.limit ?? 20, server: params.server, players: params.players?.join(','), subject: params.subject })}`),
  rankingServers: () => request<string[]>('/api/v1/rankings/servers'),
  rankingPlayers: (query = '') => request<PlayerIdentity[]>(`/api/v1/rankings/players?${queryString({ q: query })}`),
}
