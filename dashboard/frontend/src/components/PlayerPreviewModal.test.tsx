import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../i18n'
import { PlayerPreviewModal } from './PlayerPreviewModal'

const preview = {
  steam_id: '76561198000000001', player_name: '测试玩家', active_play_seconds: 3660,
  session_count: 9, last_seen_at: 1786665600,
  main_badge: { achievement_key: 'career.veteran.1', title: '初入尸潮', artwork_key: 'career.veteran', tier: 1 },
  pve: { available: true, common_kills: 1234, special_kills: 88, boss_kills: 12, headshot_kills: 321, rescues: 19, campaign_completions: 6 },
  companions: [{ player_name: '队友甲', shared_seconds: 5400, shared_rounds: 4 }],
  versus: { available: true, human_si_kills: 27, infected_damage: 3456, survivor_controls: 13, survivor_incapacitations: 8 },
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

  it('renders PvE, top companions, and Versus sections with localized headshots', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<QueryClientProvider client={client}><PlayerPreviewModal open steamID={preview.steam_id} onClose={() => undefined} /></QueryClientProvider>)
    expect(await screen.findByText('测试玩家')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: '初入尸潮' })).toBeInTheDocument()
    expect(screen.getByText('合作 / 写实')).toBeInTheDocument()
    expect(screen.getByText(/并肩作战 Top 3/)).toBeInTheDocument()
    expect(screen.getByText('队友甲')).toBeInTheDocument()
    expect(screen.getByText(/4 场对局/)).toBeInTheDocument()
    expect(screen.getByText('对抗')).toBeInTheDocument()
    expect(screen.getByText('爆头击杀')).toBeInTheDocument()
    expect(screen.getByText('1,234')).toBeInTheDocument()
    expect(screen.getByText('321')).toBeInTheDocument()
  })
})
