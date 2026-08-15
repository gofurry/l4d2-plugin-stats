import { achievementAtlas, type AchievementArtworkKey } from './generated/achievement-atlas.generated'

export function isAchievementArtworkKey(value: string): value is AchievementArtworkKey {
  return Object.prototype.hasOwnProperty.call(achievementAtlas.items, value)
}
