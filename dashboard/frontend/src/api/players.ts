import { queryString, request } from './client'

export interface PlayerSummary { steam_id: string; last_name: string; first_seen_at: number; last_seen_at: number; session_count: number; connected_seconds: number; active_play_seconds: number }
export type PlayerProfileSection = 'overview' | 'achievements' | 'analysis' | 'pve' | 'pve-details' | 'versus-survivor' | 'versus-survivor-details' | 'versus-infected' | 'versus-infected-details' | 'relationships' | 'history'
export interface PlayerProfile { steam_id: string; player_name: string; visible_sections: PlayerProfileSection[]; self: boolean }
export interface PlayerProfileVisibility { visible_sections: PlayerProfileSection[]; updated_at?: number }
export interface PlayerPreview {
  steam_id: string; player_name: string; session_count: number; active_play_seconds: number; last_seen_at: number
  pve: { available: boolean; common_kills: number; special_kills: number; boss_kills: number; headshot_kills: number; rescues: number; campaign_completions: number }
  versus: { available: boolean; human_si_kills: number; infected_damage: number; survivor_controls: number; survivor_incapacitations: number }
  companions: { player_name: string; shared_seconds: number; shared_rounds: number }[]
  badges: AchievementBadge[]
  /** Compatibility alias for the first showcase slot. */
  main_badge?: AchievementBadge
}
export interface AchievementBadge { slot?: number; achievement_key: string; title: string; artwork_key: string; tier?: number }
export interface AchievementCard {
  achievement_key: string; group_key: string; title: string; description: string; category: string; metric_id?: string
  threshold?: number; tier?: number; visibility: 'public' | 'mystery' | 'secret'; counts_toward_completion: boolean; artwork_key?: string
  unlocked: boolean; current_value?: number; unlocked_at?: number; grant_kind?: 'live' | 'backfill'; value_at_unlock?: number; evidence_steam_id?: string; global_unlock_rate: number
}
export interface PlayerAchievements {
  achievement_contract_version: number
  overview: { unlocked: number; total: number; completion_percent: number; easter_eggs: number; recent_unlock?: { achievement_key: string; unlocked_at: number }; badges: AchievementBadge[] }
  items: AchievementCard[]; unseen_live?: AchievementCard[]; unseen_backfill_count?: number
}
export interface PlayerBadges { achievement_contract_version: number; items: AchievementBadge[] }
export interface SteamIdentity { steam_id: string; badge_edit_authorized: boolean }
export interface PlayerAnalysis { view: string; active_play_seconds: number; metrics: Record<string, number | null>; samples: Record<string, number>; recent_incidents: { earliest_incident_at: number; latest_incident_at: number; controls_received: number; average_control_seconds?: number; incaps: number; deaths: number; teammates_rescued: number; rescued_by_teammates: number; control_classes: { infected_class: number; controls: number; average_duration_seconds: number }[]; top_rescuers: { player_name: string; rescues: number }[]; two_cap_episodes: number; three_cap_episodes: number; four_cap_episodes: number } }
export interface PlayerActivityPoint { day: number; session_count: number; connected_seconds: number; active_play_seconds: number }
export interface PlayerServerActivity { server_key: string; session_count: number; active_play_seconds: number }
export interface PlayerActivity { timeline: PlayerActivityPoint[]; servers: PlayerServerActivity[] }
export interface CollectionCoverage { collected_segments: number; total_segments: number; complete: boolean }
export interface PVEInfectedClass { class_id: number; kills: number; assists: number | null; damage: number; controls_received: number; controlled_seconds: number; saves: number }
export interface PVEEquipment { equipment_id: number; actions: number; common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; headshot_kills: number; damage_to_special: number; damage_to_tank: number; damage_to_witch: number }
export interface PlayerPVEOverview {
  common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; damage_to_special: number; damage_to_tank: number; damage_to_witch: number; damage_taken_infected: number; friendly_fire: number; friendly_fire_taken: number
  incapacitations: number; deaths: number; revives: number; incap_revives: number; ledge_rescues: number; defib_revives: number; rescues_received: number
  medkits_used: number; medkits_used_self: number; medkits_used_on_others: number; healing: number; medkit_healing_self: number; medkit_healing_others: number; pills_used: number; adrenaline_used: number; temporary_health_received: number
  chapter_participations: number; chapter_completions: number; chapter_completions_alive: number; chapter_completions_dead: number; campaign_completions: number
  melee_tongue_self_cuts: number; tank_rocks_destroyed: number; witch_oneshots: number; witch_solo_kills: number; tank_encounters: number; tank_kill_participations: number; witch_encounters: number; witch_kill_participations: number
  incendiary_packs_deployed: number; explosive_packs_deployed: number; objective_interactions: number; ammo_pile_uses: number; incapacitated_seconds: number; ledge_hanging_seconds: number; black_white_teammates_restored: number; car_alarms_triggered: number
  teammate_protections: number | null; ledge_grabs: number | null; tank_rock_hits_received: number | null; hunter_skeets: number | null; charger_levels: number | null; telemetry_coverage: CollectionCoverage
  special_assists: number | null; tank_assists: number; witch_assists: number; assist_coverage: CollectionCoverage
}
export interface PlayerPVEDetails { infected_classes: PVEInfectedClass[]; equipment: PVEEquipment[] }
export interface VersusSurvivorClass { class_id: number; human_controller_kills: number; bot_controller_kills: number; human_controller_assists: number | null; bot_controller_assists: number | null; damage_to_human_controllers: number; damage_to_bot_controllers: number }
export interface VersusInfectedClass { class_id: number; spawns: number; damage_to_human_survivors: number; damage_to_bot_survivors: number; human_survivor_incaps: number; bot_survivor_incaps: number; human_survivor_kills: number; bot_survivor_kills: number; human_survivor_controls: number; bot_survivor_controls: number; human_survivor_control_seconds: number; bot_survivor_control_seconds: number; human_survivor_ability_hits: number; bot_survivor_ability_hits: number; human_survivor_ability_damage: number; bot_survivor_ability_damage: number }
export interface PlayerVersusSurvivor {
  survivor_common_kills: number; human_special_kills: number; bot_special_kills: number; human_tank_kills: number; bot_tank_kills: number; survivor_damage: number; survivor_damage_taken: number; survivor_friendly_fire: number; survivor_friendly_fire_taken: number; survivor_incapacitations: number; survivor_deaths: number; survivor_revives: number; survivor_incap_revives: number; survivor_ledge_rescues: number; survivor_defib_revives: number; survivor_rescues_received: number
  survivor_medkits_self: number; survivor_medkits_others: number; survivor_healing_self: number; survivor_healing_others: number; survivor_pills: number; survivor_adrenaline: number; survivor_temporary_health: number; survivor_witch_kills: number; survivor_witch_damage: number
  molotovs_thrown: number; pipe_bombs_thrown: number; vomit_jars_thrown: number; survivor_incendiary_packs: number; survivor_explosive_packs: number; survivor_tongue_self_cuts: number; survivor_tank_rocks_destroyed: number; survivor_witch_oneshots: number; survivor_witch_solo_kills: number; survivor_objective_interactions: number; survivor_car_alarms_triggered: number
  survivor_teammate_protections: number | null; survivor_ledge_grabs: number | null; survivor_tank_rock_hits_received: number | null; survivor_hunter_skeets: number | null; survivor_charger_levels: number | null; telemetry_coverage: CollectionCoverage
  human_special_assists: number | null; bot_special_assists: number | null; human_tank_assists: number | null; bot_tank_assists: number | null; survivor_witch_encounters: number | null; survivor_witch_kill_participations: number | null; survivor_witch_assists: number | null; survivor_black_white_teammates_restored: number | null; assist_coverage: CollectionCoverage
}
export interface PlayerVersusSurvivorDetails { survivor_classes: VersusSurvivorClass[] }
export interface PlayerVersusInfected {
  infected_spawns: number; damage_to_human_survivors: number; damage_to_bot_survivors: number; human_survivor_incaps: number; bot_survivor_incaps: number; human_survivor_kills: number; bot_survivor_kills: number; human_survivor_controls: number; human_survivor_control_seconds: number
}
export interface PlayerVersusInfectedDetails { infected_classes: VersusInfectedClass[] }
export interface PlayerRelationshipDirection { incap_revives: number; ledge_rescues: number; defib_revives: number; smoker_rescues: number; hunter_rescues: number; jockey_rescues: number; charger_rescues: number; special_rescues: number; support_actions: number; control_rescue_duration_ms: number; average_control_rescue_ms?: number; medkits_used: number; medkit_healing: number; black_white_restores: number; friendly_fire_damage: number }
export interface PlayerRelationship { peer_steam_id: string; peer_name: string; shared_rounds: number; shared_seconds: number; outgoing: PlayerRelationshipDirection; incoming: PlayerRelationshipDirection; mutual_support: number }
export interface PlayerRelationshipSummary { peer_steam_id: string; peer_name: string; shared_rounds: number; shared_seconds: number; support_actions: number }
export interface PlayerRelationshipPage { relationship_version: number; page: number; page_size: number; total: number; summaries: { most_companion?: PlayerRelationshipSummary; most_supported?: PlayerRelationshipSummary; most_supported_by?: PlayerRelationshipSummary; most_mutual?: PlayerRelationshipSummary }; items: PlayerRelationship[] }
export interface PlayerSession { server_key: string; player_name: string; started_at: number; ended_at?: number; connected_seconds: number; active_play_seconds: number; status: string; disconnect_reason: string }
export interface PlayerChapter { server_key: string; mode_family: string; game_mode: string; map_name: string; side: string; started_at: number; ended_at?: number; active_play_seconds: number; status: string }
export interface Page<T> { items: T[]; next_cursor?: string }

export const playersAPI = {
  steamIdentity: () => request<SteamIdentity | null>('/api/v1/steam/identity'),
  playerProfile: (id: string) => request<PlayerProfile>(`/api/v1/players/${id}/profile`),
  savePlayerProfileVisibility: (visibleSections: PlayerProfileSection[]) => request<PlayerProfileVisibility>('/api/v1/me/profile-visibility', { method: 'PUT', body: JSON.stringify({ visible_sections: visibleSections }) }),
  playerSummary: (id: string) => request<PlayerSummary>(`/api/v1/players/${id}/summary`),
  playerPreview: (id: string) => request<PlayerPreview>(`/api/v1/players/${id}/preview`),
  playerActivity: (id: string, range: string, server = '') => request<PlayerActivity>(`/api/v1/players/${id}/activity?${queryString({ range, server })}`),
  playerPVE: (id: string, range: string, server = '', mode = '') => request<PlayerPVEOverview>(`/api/v1/players/${id}/pve?${queryString({ range, server, mode, view: 'pve' })}`),
  playerPVEDetails: (id: string, range: string, server = '', mode = '') => request<PlayerPVEDetails>(`/api/v1/players/${id}/pve?${queryString({ range, server, mode, view: 'pve-details' })}`),
  playerVersusSurvivor: (id: string, range: string, server = '') => request<PlayerVersusSurvivor>(`/api/v1/players/${id}/versus?${queryString({ range, server, view: 'versus-survivor' })}`),
  playerVersusSurvivorDetails: (id: string, range: string, server = '') => request<PlayerVersusSurvivorDetails>(`/api/v1/players/${id}/versus?${queryString({ range, server, view: 'versus-survivor-details' })}`),
  playerVersusInfected: (id: string, range: string, server = '') => request<PlayerVersusInfected>(`/api/v1/players/${id}/versus?${queryString({ range, server, view: 'versus-infected' })}`),
  playerVersusInfectedDetails: (id: string, range: string, server = '') => request<PlayerVersusInfectedDetails>(`/api/v1/players/${id}/versus?${queryString({ range, server, view: 'versus-infected-details' })}`),
  playerAnalysis: (id: string, range: string, server = '', view = 'pve') => request<PlayerAnalysis>(`/api/v1/players/${id}/analysis?${queryString({ range, server, view })}`),
  playerRelationships: (id: string, range: string, server = '', mode = 'all', page = 1, pageSize = 20, sort = 'shared_rounds', order: 'asc' | 'desc' = 'desc') => request<PlayerRelationshipPage>(`/api/v1/players/${id}/relationships?${queryString({ range, server, mode, page, page_size: pageSize, sort, order })}`),
  playerAchievements: (id: string) => request<PlayerAchievements>(`/api/v1/players/${id}/achievements`),
  playerBadges: (id: string) => request<PlayerBadges>(`/api/v1/players/${id}/badges`),
  saveBadgeShowcase: (items: { slot: number; achievement_key: string }[]) => request<PlayerBadges>('/api/v1/me/badge-showcase', { method: 'PUT', body: JSON.stringify({ items }) }),
  playerSessions: (id: string, cursor = '') => request<Page<PlayerSession>>(`/api/v1/players/${id}/sessions?${queryString({ limit: 20, cursor })}`),
  playerChapters: (id: string, cursor = '') => request<Page<PlayerChapter>>(`/api/v1/players/${id}/chapters?${queryString({ limit: 20, cursor })}`),
}
