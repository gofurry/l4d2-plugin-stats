import type { IngameSettings } from '../../api'

export const cachePresets = {
  home_cache_seconds: [10, 30, 60, 120],
  player_cache_seconds: [30, 60, 120, 300],
  ranking_cache_seconds: [60, 120, 300, 600],
  content_cache_seconds: [60, 300, 600, 1800],
} as const

export const fallbackIngameMetrics = [
  { key: 'active_play_seconds', cn: '实际游戏时间', en: 'Active play time' },
  { key: 'sessions', cn: '会话次数', en: 'Sessions' },
  { key: 'common_kills', cn: '普通感染者击杀', en: 'Common infected kills' },
  { key: 'special_kills', cn: '特殊感染者击杀', en: 'Special infected kills' },
  { key: 'boss_kills', cn: 'Boss 击杀', en: 'Boss kills' },
  { key: 'campaign_completions', cn: '战役完成', en: 'Campaign completions' },
  { key: 'rescues', cn: '队友救援', en: 'Teammate rescues' },
  { key: 'human_si_kills', cn: '真人特感击杀', en: 'Human SI kills' },
  { key: 'infected_damage', cn: '感染者伤害', en: 'Infected damage' },
  { key: 'survivor_controls', cn: '控制幸存者', en: 'Survivor controls' },
  { key: 'survivor_incaps', cn: '击倒幸存者', en: 'Survivor incapacitations' },
] as const

export const defaultIngameSettings: IngameSettings = {
  enabled: true,
  title: '',
  description: '',
  banner_url: '',
  background_url: '',
  website_url: '',
  show_announcements: true,
  show_players: true,
  show_highlights: true,
  highlight_metrics: ['active_play_seconds', 'special_kills', 'rescues'],
  home_cache_seconds: 30,
  player_cache_seconds: 60,
  ranking_cache_seconds: 120,
  content_cache_seconds: 300,
  updated_at: 0,
}

export function completeIngameSettings(settings?: Partial<IngameSettings>): IngameSettings {
  const highlights = settings?.highlight_metrics
  const validHighlights = highlights?.length === 3 && highlights.every(Boolean)
  return {
    ...defaultIngameSettings,
    ...settings,
    highlight_metrics: validHighlights ? highlights : [...defaultIngameSettings.highlight_metrics],
    home_cache_seconds: approvedPreset(settings?.home_cache_seconds, cachePresets.home_cache_seconds, defaultIngameSettings.home_cache_seconds),
    player_cache_seconds: approvedPreset(settings?.player_cache_seconds, cachePresets.player_cache_seconds, defaultIngameSettings.player_cache_seconds),
    ranking_cache_seconds: approvedPreset(settings?.ranking_cache_seconds, cachePresets.ranking_cache_seconds, defaultIngameSettings.ranking_cache_seconds),
    content_cache_seconds: approvedPreset(settings?.content_cache_seconds, cachePresets.content_cache_seconds, defaultIngameSettings.content_cache_seconds),
  }
}

function approvedPreset(value: number | undefined, presets: readonly number[], fallback: number) {
  return value !== undefined && presets.includes(value) ? value : fallback
}
