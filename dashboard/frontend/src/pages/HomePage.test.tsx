import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import i18n from '../i18n'
import { HomePage } from './HomePage'

const site = { language: 'zh-CN' as const, browser_title: 'L4D2 Stats', theme: 'light' as const, footer_enabled: true, background_image_url: '', footer_links: [{ label: '项目主页', url: 'https://example.com' }], steam_openid_enabled: false, a2s_refresh_seconds: 30, configured: true }
const overview = {
  core: { total_players: 42, active_players_7d: 8, total_active_play_seconds: 7200, completed_pve_runs: 3, completed_versus_runs: 1 },
  pve: { common_kills: 1000, special_kills: 60, tank_kills: 4, witch_kills: 2, rescues: 12 },
  versus: { completed_matches: 1, completed_halves: 2, human_controlled_infected_kills: 20, human_survivor_controls: 16 },
  generated_at: '2026-08-03T00:00:00Z',
}
const status = {
  server_id: '199d4525-a1af-472d-a1f0-4b35592caf1b', display_name: '一号服务器', address: '127.0.0.1:27015', online: true, stale: false, checking: false,
  name: '真实服名', map: 'c5m1_waterfront', players: 5, max_players: 8, bots: 1, latency_ms: 18,
  player_list: [{ name: 'Coach', score: 12, duration_seconds: 90 }],
  checked_at: '2026-08-03T00:00:00Z', last_success_at: '2026-08-03T00:00:00Z',
}

function response(data: unknown, ok = true): Response {
  return new Response(JSON.stringify(ok ? { data, request_id: 'test' } : { error: { message: 'offline' }, request_id: 'test' }), {
    status: ok ? 200 : 503,
    headers: { 'Content-Type': 'application/json' },
  })
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return render(<MemoryRouter><QueryClientProvider client={client}><HomePage /></QueryClientProvider></MemoryRouter>)
}

describe('HomePage', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('zh-CN')
  })

  afterEach(() => cleanup())

  it('renders the server, historical overview, and custom footer', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.endsWith('/site')) return Promise.resolve(response(site))
      if (path.endsWith('/overview')) return Promise.resolve(response(overview))
      return Promise.resolve(response([status]))
    }))
    renderPage()
    expect(await screen.findByText('真实服名')).toBeInTheDocument()
    expect(await screen.findByText('42')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '项目主页' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /加入服务器/ })).toHaveAttribute('href', 'steam://connect/127.0.0.1:27015')
  })

  it('keeps the server card usable when historical statistics fail', async () => {
    let overviewCalls = 0
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.endsWith('/site')) return Promise.resolve(response(site))
      if (path.endsWith('/overview')) {
        overviewCalls++
        return Promise.resolve(response(null, false))
      }
      return Promise.resolve(response([status]))
    }))
    renderPage()
    expect(await screen.findByText('真实服名')).toBeInTheDocument()
    expect(await screen.findByText(/统计数据暂时不可用/, {}, { timeout: 3_000 })).toBeInTheDocument()
    await waitFor(() => expect(overviewCalls).toBeGreaterThanOrEqual(1))
  })

  it('shows a server card immediately while the first A2S query runs', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.endsWith('/site')) return Promise.resolve(response(site))
      if (path.endsWith('/overview')) return Promise.resolve(response(overview))
      return Promise.resolve(response([{ ...status, name: undefined, map: undefined, online: false, checking: true, checked_at: '0001-01-01T00:00:00Z' }]))
    }))
    renderPage()
    expect((await screen.findAllByText('一号服务器')).length).toBeGreaterThan(0)
    expect(await screen.findByText('正在查询')).toBeInTheDocument()
  })

  it('explains an empty statistics database without hiding zero metrics', async () => {
    const empty = {
      ...overview,
      core: { total_players: 0, active_players_7d: 0, total_active_play_seconds: 0, completed_pve_runs: 0, completed_versus_runs: 0 },
      pve: { common_kills: 0, special_kills: 0, tank_kills: 0, witch_kills: 0, rescues: 0 },
      versus: { completed_matches: 0, completed_halves: 0, human_controlled_infected_kills: 0, human_survivor_controls: 0 },
    }
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.endsWith('/site')) return Promise.resolve(response(site))
      if (path.endsWith('/overview')) return Promise.resolve(response(empty))
      return Promise.resolve(response([]))
    }))
    renderPage()
    expect(await screen.findByText(/暂时没有可展示的数据/)).toBeInTheDocument()
    expect(screen.getByText('尚未配置可展示的服务器')).toBeInTheDocument()
  })
})
