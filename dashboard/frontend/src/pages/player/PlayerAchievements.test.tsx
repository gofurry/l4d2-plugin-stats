import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
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
    render(<QueryClientProvider client={client}><PlayerAchievements steamID="76561198000000001" data={mysteryData} loading={false} self={false} zh /></QueryClientProvider>)
    expect(screen.getByText('???')).toBeInTheDocument()
    expect(screen.getByText('隐藏成就')).toBeInTheDocument()
    expect(screen.getByText('条件尚未发现')).toBeInTheDocument()
    expect(screen.queryByText('展示徽章')).not.toBeInTheDocument()
  })
})
