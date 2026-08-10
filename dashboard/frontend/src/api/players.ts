import { queryString, request } from './client'

export interface PlayerSummary { steam_id: string; last_name: string; first_seen_at: number; last_seen_at: number; session_count: number; connected_seconds: number; active_play_seconds: number }
export interface PlayerPreview {
  steam_id: string; player_name: string; session_count: number; active_play_seconds: number; last_seen_at: number
  pve: { available: boolean; special_kills: number; boss_kills: number; rescues: number; campaign_completions: number }
  versus: { available: boolean; human_si_kills: number; infected_damage: number; survivor_controls: number; survivor_incapacitations: number }
}
export interface PlayerActivityPoint { day: number; session_count: number; connected_seconds: number; active_play_seconds: number }
export interface PlayerServerActivity { server_key: string; session_count: number; active_play_seconds: number }
export interface PlayerActivity { timeline: PlayerActivityPoint[]; servers: PlayerServerActivity[] }
export interface PVEInfectedClass { class_id: number; kills: number; damage: number; controls_received: number; controlled_seconds: number; saves: number }
export interface PVEEquipment { equipment_id: number; actions: number; common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; headshot_kills: number; damage_to_special: number; damage_to_tank: number; damage_to_witch: number }
export interface PlayerPVE {
  common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; damage_to_special: number; damage_to_tank: number; damage_to_witch: number; damage_taken_infected: number; friendly_fire: number; friendly_fire_taken: number
  incapacitations: number; deaths: number; revives: number; incap_revives: number; ledge_rescues: number; defib_revives: number; rescues_received: number
  medkits_used: number; medkits_used_self: number; medkits_used_on_others: number; healing: number; medkit_healing_self: number; medkit_healing_others: number; pills_used: number; adrenaline_used: number; temporary_health_received: number
  chapter_participations: number; chapter_completions: number; chapter_completions_alive: number; chapter_completions_dead: number; campaign_completions: number
  melee_tongue_self_cuts: number; tank_rocks_destroyed: number; witch_oneshots: number; witch_solo_kills: number; tank_encounters: number; tank_kill_participations: number; witch_encounters: number; witch_kill_participations: number
  incendiary_packs_deployed: number; explosive_packs_deployed: number; objective_interactions: number; ammo_pile_uses: number; incapacitated_seconds: number; ledge_hanging_seconds: number; black_white_teammates_restored: number; car_alarms_triggered: number
  infected_classes: PVEInfectedClass[]; equipment: PVEEquipment[]
}
export interface VersusSurvivorClass { class_id: number; human_controller_kills: number; bot_controller_kills: number; damage_to_human_controllers: number; damage_to_bot_controllers: number }
export interface VersusInfectedClass { class_id: number; spawns: number; damage_to_human_survivors: number; damage_to_bot_survivors: number; human_survivor_incaps: number; bot_survivor_incaps: number; human_survivor_kills: number; bot_survivor_kills: number; human_survivor_controls: number; bot_survivor_controls: number; human_survivor_control_seconds: number; bot_survivor_control_seconds: number; human_survivor_ability_hits: number; bot_survivor_ability_hits: number; human_survivor_ability_damage: number; bot_survivor_ability_damage: number }
export interface PlayerVersus {
  survivor_common_kills: number; human_special_kills: number; bot_special_kills: number; human_tank_kills: number; bot_tank_kills: number; survivor_damage: number; survivor_damage_taken: number; survivor_friendly_fire: number; survivor_friendly_fire_taken: number; survivor_incapacitations: number; survivor_deaths: number; survivor_revives: number; survivor_incap_revives: number; survivor_ledge_rescues: number; survivor_defib_revives: number; survivor_rescues_received: number
  survivor_medkits_self: number; survivor_medkits_others: number; survivor_healing_self: number; survivor_healing_others: number; survivor_pills: number; survivor_adrenaline: number; survivor_temporary_health: number; survivor_witch_kills: number; survivor_witch_damage: number
  molotovs_thrown: number; pipe_bombs_thrown: number; vomit_jars_thrown: number; survivor_incendiary_packs: number; survivor_explosive_packs: number; survivor_tongue_self_cuts: number; survivor_tank_rocks_destroyed: number; survivor_witch_oneshots: number; survivor_witch_solo_kills: number; survivor_car_alarms_triggered: number
  infected_spawns: number; damage_to_human_survivors: number; damage_to_bot_survivors: number; human_survivor_incaps: number; bot_survivor_incaps: number; human_survivor_kills: number; bot_survivor_kills: number; human_survivor_controls: number; human_survivor_control_seconds: number
  survivor_classes: VersusSurvivorClass[]; infected_classes: VersusInfectedClass[]
}
export interface PlayerSession { server_key: string; player_name: string; started_at: number; ended_at?: number; connected_seconds: number; active_play_seconds: number; status: string; disconnect_reason: string }
export interface PlayerChapter { server_key: string; mode_family: string; game_mode: string; map_name: string; side: string; started_at: number; ended_at?: number; active_play_seconds: number; status: string }
export interface Page<T> { items: T[]; next_cursor?: string }

export const playersAPI = {
  steamIdentity: () => request<{ steam_id: string } | null>('/api/v1/steam/identity'),
  playerSummary: (id: string) => request<PlayerSummary>(`/api/v1/players/${id}/summary`),
  playerPreview: (id: string) => request<PlayerPreview>(`/api/v1/players/${id}/preview`),
  playerActivity: (id: string, range: string, server = '') => request<PlayerActivity>(`/api/v1/players/${id}/activity?${queryString({ range, server })}`),
  playerPVE: (id: string, range: string, server = '', mode = '') => request<PlayerPVE>(`/api/v1/players/${id}/pve?${queryString({ range, server, mode })}`),
  playerVersus: (id: string, range: string, server = '') => request<PlayerVersus>(`/api/v1/players/${id}/versus?${queryString({ range, server })}`),
  playerSessions: (id: string, cursor = '') => request<Page<PlayerSession>>(`/api/v1/players/${id}/sessions?${queryString({ limit: 20, cursor })}`),
  playerChapters: (id: string, cursor = '') => request<Page<PlayerChapter>>(`/api/v1/players/${id}/chapters?${queryString({ limit: 20, cursor })}`),
}
