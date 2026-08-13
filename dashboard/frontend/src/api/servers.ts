import { adminWrite, request } from './client'

export interface ServerPlayer { name: string; steam_id?: string; score: number; duration_seconds: number }
export interface ServerRule { name: string; value: string }
export interface ServerStatus { server_id: string; display_name: string; address: string; online: boolean; stale: boolean; checking: boolean; name?: string; map?: string; players: number; max_players: number; bots: number; latency_ms?: number; last_success_at?: string; checked_at: string; player_list?: ServerPlayer[]; rules?: ServerRule[] }
export interface ServerA2SState { available: boolean; status?: ServerStatus }
export interface GameServer { id?: string; display_name: string; address: string; enabled: boolean; sort_order: number }
export interface GameServerInput { display_name: string; address: string }

export const serversAPI = {
  serverStatuses: () => request<ServerStatus[]>('/api/v1/servers/status'),
  servers: () => request<GameServer[]>('/api/v1/admin/servers'),
  createServer: (server: GameServerInput) => adminWrite<GameServer>('/api/v1/admin/servers', 'POST', server),
  updateServer: (id: string, server: GameServerInput) => adminWrite<GameServer>(`/api/v1/admin/servers/${id}`, 'PUT', server),
  setServerEnabled: (id: string, enabled: boolean) => adminWrite<{ enabled: boolean }>(`/api/v1/admin/servers/${id}/enabled`, 'PATCH', { enabled }),
  moveServer: (id: string, direction: 'up' | 'down') => adminWrite<{ moved: boolean }>(`/api/v1/admin/servers/${id}/move`, 'POST', { direction }),
  serverA2S: (id: string) => request<ServerA2SState>(`/api/v1/admin/servers/${id}/a2s`),
  refreshServerA2S: (id: string) => adminWrite<ServerStatus>(`/api/v1/admin/servers/${id}/a2s`, 'POST'),
  deleteServer: (id: string) => adminWrite(`/api/v1/admin/servers/${id}`, 'DELETE'),
}
