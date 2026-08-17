import { describe, expect, it } from 'vitest'
import { completeIngameSettings, defaultIngameSettings, fallbackIngameMetrics } from './AdminIngameDefaults'

describe('in-game admin defaults', () => {
  it('fills missing metrics and cache presets without overwriting valid values', () => {
    const completed = completeIngameSettings({
      title: 'Portal',
      show_players: false,
      highlight_metrics: ['', '', ''],
      home_cache_seconds: 0,
      player_cache_seconds: 300,
      ranking_cache_seconds: 0,
      content_cache_seconds: 600,
    })
    expect(completed.title).toBe('Portal')
    expect(completed.show_players).toBe(false)
    expect(completed.show_server_intro).toBe(true)
    expect(completed.show_server_status).toBe(true)
    expect(completed.highlight_metrics).toEqual(defaultIngameSettings.highlight_metrics)
    expect(completed.home_cache_seconds).toBe(30)
    expect(completed.player_cache_seconds).toBe(300)
    expect(completed.ranking_cache_seconds).toBe(120)
    expect(completed.content_cache_seconds).toBe(600)
  })

  it('keeps fallback catalog entries for all default highlight metrics', () => {
    const keys = fallbackIngameMetrics.map(metric => metric.key)
    for (const key of defaultIngameSettings.highlight_metrics) expect(keys).toContain(key)
  })
})
