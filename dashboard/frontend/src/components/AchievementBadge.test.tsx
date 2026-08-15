import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { achievementAtlas } from '../assets/achievements/generated/achievement-atlas.generated'
import { AchievementBadge } from './AchievementBadge'

describe('Achievement badge atlas', () => {
  it('keeps all 38 artwork tiles unique and inside the fixed 6 by 7 atlas', () => {
    const items = Object.values(achievementAtlas.items)
    expect(items).toHaveLength(38)
    expect(new Set(items.map(item => `${item.x}:${item.y}`))).toHaveLength(38)
    for (const item of items) {
      expect(item.w).toBe(128)
      expect(item.h).toBe(128)
      expect(item.x + item.w).toBeLessThanOrEqual(768)
      expect(item.y + item.h).toBeLessThanOrEqual(896)
    }
  })

  it('uses the sprite for known artwork, provides a tooltip, and hides mystery artwork', async () => {
    const { rerender } = render(<AchievementBadge artworkKey="career.veteran" tier={4} label="老兵" />)
    const badge = screen.getByRole('img', { name: '老兵' })
    expect(badge.querySelector('span')).toHaveStyle({ backgroundPosition: '0px 0px' })
    fireEvent.mouseEnter(badge)
    expect(await screen.findByText('老兵')).toBeInTheDocument()
    rerender(<AchievementBadge artworkKey="career.veteran" mystery label="隐藏成就" />)
    expect(screen.getByRole('img', { name: '隐藏成就' })).toHaveTextContent('?')
    expect(screen.getByRole('img', { name: '隐藏成就' }).querySelector('span')).toBeNull()
  })
})
