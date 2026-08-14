import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../i18n'
import { PlayerPreviewModal } from './PlayerPreviewModal'

const preview = {
  steam_id: '76561198000000001', player_name: '测试玩家', active_play_seconds: 3660,
  campaign_completions: 12, tank_kills: 7, witch_kills: 5, common_kills: 1234,
  special_kills: 88, headshot_kills: 321, incap_revives: 19, incapacitations: 6,
  last_seen_at: 1786665600,
}

describe('PlayerPreviewModal', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('zh-CN')
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response(JSON.stringify({ data: preview, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders the frozen eight metric slots without relationship details', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<QueryClientProvider client={client}><PlayerPreviewModal open steamID={preview.steam_id} onClose={() => undefined} /></QueryClientProvider>)
    expect(await screen.findByText('测试玩家')).toBeInTheDocument()
    expect(screen.getByText('Tank 7 / Witch 5')).toBeInTheDocument()
    expect(screen.getByText('1,234')).toBeInTheDocument()
    expect(screen.getByText('321')).toBeInTheDocument()
    expect(screen.queryByText('并肩作战')).not.toBeInTheDocument()
  })
})
