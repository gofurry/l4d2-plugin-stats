import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { achievementAtlas } from '../assets/achievements/generated/achievement-atlas.generated'
import { AchievementBadge } from './AchievementBadge'

describe('Achievement badge atlas', () => {
  it('keeps all 26 artwork tiles unique and inside the fixed 6 by 5 atlas', () => {
    const items = Object.values(achievementAtlas.items)
    expect(items).toHaveLength(26)
    expect(new Set(items.map(item => `${item.x}:${item.y}`))).toHaveLength(26)
    for (const item of items) {
      expect(item.w).toBe(128)
      expect(item.h).toBe(128)
      expect(item.x + item.w).toBeLessThanOrEqual(768)
      expect(item.y + item.h).toBeLessThanOrEqual(640)
    }
  })

  it('uses the sprite for known artwork and hides it for mystery placeholders', () => {
    const { rerender } = render(<AchievementBadge artworkKey="career.veteran" tier={4} label="老兵" />)
    expect(screen.getByRole('img', { name: '老兵' }).querySelector('span')).toHaveStyle({ backgroundPosition: '0px 0px' })
    rerender(<AchievementBadge artworkKey="career.veteran" mystery label="隐藏成就" />)
    expect(screen.getByRole('img', { name: '隐藏成就' })).toHaveTextContent('?')
    expect(screen.getByRole('img', { name: '隐藏成就' }).querySelector('span')).toBeNull()
  })
})
