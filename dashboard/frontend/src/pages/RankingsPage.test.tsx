import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import i18n from '../i18n'
import { RankingsPage } from './RankingsPage'

function response(data: unknown): Response {
  return new Response(JSON.stringify({ data, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('RankingsPage empty state', () => {
  beforeEach(async () => {
    localStorage.clear()
    await i18n.changeLanguage('zh-CN')
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders an empty state when a legacy response contains null items', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.includes('/rankings/servers')) return Promise.resolve(response([]))
      return Promise.resolve(response({ metric: 'common_kills', mode: 'pve', higher_is_better: true, lower_is_better: false, items: null, total: 0, generated_at: '2026-08-15T00:00:00Z' }))
    }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<MemoryRouter><QueryClientProvider client={client}><RankingsPage /></QueryClientProvider></MemoryRouter>)
    expect(await screen.findByText('当前玩法或筛选条件下还没有排行榜数据。')).toBeInTheDocument()
  })
})
