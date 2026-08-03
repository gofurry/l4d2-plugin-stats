export interface FooterLink { label: string; url: string; open_new_tab: boolean }
export interface Site { title: string; footer_text: string; footer_links: FooterLink[] }
export interface CoreOverview { total_players: number; active_players_7d: number; total_active_play_seconds: number; completed_pve_runs: number; completed_versus_runs: number }
export interface PVEOverview { common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; rescues: number }
export interface VersusOverview { completed_matches: number; completed_halves: number; human_controlled_infected_kills: number; human_survivor_controls: number }
export interface Overview { core: CoreOverview; pve: PVEOverview; versus: VersusOverview; generated_at: string }
export interface ServerStatus {
  configured_name: string; connect_address: string; online: boolean; stale: boolean; name?: string; map?: string;
  players: number; max_players: number; bots: number; latency_ms?: number; last_success_at?: string; checked_at: string;
}

interface Envelope<T> { data: T; request_id: string }
interface ErrorEnvelope { error?: { code?: string; message?: string } }

async function get<T>(path: string): Promise<T> {
  const response = await fetch(path, { headers: { Accept: 'application/json' } })
  const payload = await response.json() as Envelope<T> & ErrorEnvelope
  if (!response.ok) throw new Error(payload.error?.message ?? `HTTP ${response.status}`)
  return payload.data
}

export const api = {
  site: () => get<Site>('/api/v1/site'),
  overview: () => get<Overview>('/api/v1/dashboard/overview'),
  primaryServer: () => get<ServerStatus | null>('/api/v1/servers/primary/status'),
}
