import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { PlayerAchievements as AchievementData } from '../../api'
import { PlayerAchievements } from './PlayerAchievements'

const mysteryData: AchievementData = {
  achievement_contract_version: 1,
  overview: { unlocked: 0, total: 58, completion_percent: 0, easter_eggs: 0, badges: [] },
  items: [{
    achievement_key: 'mystery.50', group_key: 'mystery.50', title: '???', description: '条件尚未发现', category: 'special',
    visibility: 'mystery', counts_toward_completion: true, unlocked: false, global_unlock_rate: 0,
  }],
}

describe('PlayerAchievements', () => {
  it('renders the frozen anonymous mystery copy without progress or showcase controls', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<QueryClientProvider client={client}><PlayerAchievements steamID="76561198000000001" data={mysteryData} loading={false} self={false} canEdit={false} onRequireAuth={() => undefined} zh /></QueryClientProvider>)
    expect(screen.getByText('???')).toBeInTheDocument()
    expect(screen.getByText('隐藏成就')).toBeInTheDocument()
    expect(screen.getByText('条件尚未发现')).toBeInTheDocument()
    expect(screen.queryByText('展示徽章')).not.toBeInTheDocument()
  })

  it('uses full numeric progress and requires Steam verification before editing', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    const requireAuth = vi.fn()
    const data: AchievementData = {
      ...mysteryData,
      overview: { unlocked: 1, total: 58, completion_percent: 1.7, easter_eggs: 1, badges: [] },
      items: [{
        achievement_key: 'combat.marksman.1', group_key: 'combat.marksman', title: '神射手', description: '累计爆头。', category: 'combat', metric_id: 'pve.headshots',
        threshold: 1000, tier: 1, visibility: 'public', counts_toward_completion: true, artwork_key: 'combat.marksman', unlocked: true, current_value: 10000, global_unlock_rate: 10,
      }, {
        achievement_key: 'combat.marksman.2', group_key: 'combat.marksman', title: '神射手', description: '累计爆头。', category: 'combat', metric_id: 'pve.headshots',
        threshold: 10000, tier: 2, visibility: 'public', counts_toward_completion: true, artwork_key: 'combat.marksman', unlocked: false, current_value: 10000, global_unlock_rate: 2,
      }],
    }
    render(<QueryClientProvider client={client}><PlayerAchievements steamID="76561198000000001" data={data} loading={false} self={false} canEdit={false} onRequireAuth={requireAuth} zh /></QueryClientProvider>)
    expect(screen.getByText('10,000')).toBeInTheDocument()
    expect(screen.getByText('/ 10,000')).toBeInTheDocument()
    expect(screen.queryByText(/10K|1万/)).not.toBeInTheDocument()
    expect(screen.getByText('发现彩蛋 1')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /展示徽章/ }))
    expect(requireAuth).toHaveBeenCalledOnce()
  })
})
