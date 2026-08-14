import type { CSSProperties } from 'react'
import { achievementAtlas, achievementAtlasImage, type AchievementArtworkKey } from '../assets/achievements/generated/achievement-atlas.generated'
import styles from './AchievementBadge.module.scss'

type AchievementBadgeProps = {
  artworkKey?: AchievementArtworkKey
  tier?: 1 | 2 | 3 | 4 | number
  size?: number
  locked?: boolean
  mystery?: boolean
  className?: string
  label?: string
}

export function AchievementBadge({ artworkKey, tier, size = 48, locked = false, mystery = false, className = '', label }: AchievementBadgeProps) {
  if (mystery || !artworkKey) {
    return <span className={`${styles.badge} ${styles.mystery} ${className}`} style={{ width: size, height: size, fontSize: Math.max(14, size * .42) }} role="img" aria-label={label ?? '隐藏成就'}>?</span>
  }
  const item = achievementAtlas.items[artworkKey]
  const scale = size / achievementAtlas.tileWidth
  const spriteStyle: CSSProperties = {
    backgroundImage: `url(${achievementAtlasImage})`,
    backgroundSize: `${achievementAtlas.imageWidth * scale}px ${achievementAtlas.imageHeight * scale}px`,
    backgroundPosition: `${-item.x * scale}px ${-item.y * scale}px`,
  }
  const tierClass = tier && tier >= 1 && tier <= 4 ? styles[`tier${tier}` as keyof typeof styles] : ''
  return <span className={`${styles.badge} ${tierClass} ${locked ? styles.locked : ''} ${className}`} style={{ width: size, height: size }} role="img" aria-label={label ?? artworkKey}>
    <span className={styles.sprite} style={spriteStyle} />
  </span>
}
