import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import i18n from '../i18n'
import { AnalysisPage } from './AnalysisPage'

function response(data: unknown): Response {
  return new Response(JSON.stringify({ data, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

const map = { map_name: 'c1m1_hotel', eligible_rounds: 4, completed_rounds: 3, failed_rounds: 1, average_completed_attempt: 1.3, average_duration_seconds: 420, complete_incident_rounds: 0, controls: 0, incaps: 0, deaths: 0 }

describe('AnalysisPage legacy map detail', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('zh-CN')
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('keeps the detail drawer visible when incident collections are null', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.includes('/analysis/options')) return Promise.resolve(response({ servers: [], campaigns: [] }))
      if (path.includes('/analysis/map-detail')) return Promise.resolve(response({
        summary: map,
        incident_composition: { controls: 0, incaps: 0, deaths: 0, revives: 0, ledge_rescues: 0, defib_revives: 0, car_alarms: 0, witch_startles: 0, medkit_heals: 0, objective_completes: 0 },
        timeline: null,
        tank: { spawn_count: 0, death_count: 0, matched_pairs: 0 },
        witch: { spawn_count: 0, death_count: 0, matched_pairs: 0 },
        recent_incidents: null,
      }))
      return Promise.resolve(response({ incident_version: 1, eligible_rounds: 4, completion_rate: 0.75, average_completed_attempt: 1.3, complete_incident_coverage: 0, earliest_incident_at: 0, latest_incident_at: 0, page: 1, page_size: 20, total: 1, maps: [map] }))
    }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<MemoryRouter><QueryClientProvider client={client}><AnalysisPage /></QueryClientProvider></MemoryRouter>)
    fireEvent.click(await screen.findByRole('button', { name: 'c1m1_hotel' }))
    expect(await screen.findByText('暂无完整战局明细')).toBeInTheDocument()
    expect(screen.getByText('该地图只有基础对局统计，旧版本数据或不完整采集不会生成事件图表。')).toBeInTheDocument()
    expect(screen.getAllByText('暂无符合当前采集契约的战局明细').length).toBeGreaterThan(0)
  })
})
