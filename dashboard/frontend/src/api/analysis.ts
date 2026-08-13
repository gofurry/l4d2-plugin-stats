import { queryString, request } from './client'

export interface AnalysisMapRow { map_name: string; eligible_rounds: number; completed_rounds: number; failed_rounds: number; average_completed_attempt?: number; average_duration_seconds?: number; complete_incident_rounds: number; controls: number; incaps: number; deaths: number }
export interface AnalysisOptions { servers: string[]; campaigns: string[] }
export interface AnalysisMaps { incident_version: number; eligible_rounds: number; completion_rate?: number; average_completed_attempt?: number; complete_incident_coverage: number; earliest_incident_at: number; latest_incident_at: number; maps: AnalysisMapRow[] }
export interface IncidentComposition { controls: number; incaps: number; deaths: number; revives: number; ledge_rescues: number; defib_revives: number; car_alarms: number }
export interface AnalysisTimelinePoint { bucket_seconds: number; rounds_reached: number; controls_per_100_rounds: number; incaps_per_100_rounds: number; deaths_per_100_rounds: number }
export interface BossAnalysis { spawn_count: number; death_count: number; matched_pairs: number; average_lifetime_seconds?: number; maximum_lifetime_seconds?: number; one_shot_deaths?: number }
export interface AnalysisMapDetail { summary: AnalysisMapRow; incident_composition: IncidentComposition; timeline: AnalysisTimelinePoint[]; tank: BossAnalysis; witch: BossAnalysis }
export interface AnalysisContextRow { fingerprint: string; ruleset_name: string; difficulty: string; survivor_limit: number; max_player_zombies: number; common_limit: number; tank_health: number; witch_health: number; round_count: number; completed_rounds: number; failed_rounds: number; average_duration_seconds?: number; complete_incident_rounds: number }
export interface AnalysisContexts { eligible_rounds: number; stable_context_rounds: number; changed_rule_rounds: number; no_context_rounds: number; contexts: AnalysisContextRow[] }
export interface PlayerIncidentClass { infected_class: number; controls: number; average_duration_seconds: number }
export interface PlayerRescuer { player_name: string; rescues: number }
interface AnalysisFilters extends Record<string, string | number | undefined> { range: string; server?: string; mode?: string; campaign?: string }

export const analysisAPI = {
  analysisOptions: (filters: Pick<AnalysisFilters, 'range' | 'mode'>) => request<AnalysisOptions>(`/api/v1/analysis/options?${queryString(filters)}`),
  analysisMaps: (filters: AnalysisFilters) => request<AnalysisMaps>(`/api/v1/analysis/maps?${queryString(filters)}`),
  analysisMapDetail: (filters: AnalysisFilters, map: string) => request<AnalysisMapDetail>(`/api/v1/analysis/map-detail?${queryString({ ...filters, map })}`),
  analysisContexts: (filters: AnalysisFilters) => request<AnalysisContexts>(`/api/v1/analysis/contexts?${queryString(filters)}`),
}
