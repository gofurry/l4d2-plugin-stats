import { CheckOutlined, EyeInvisibleOutlined, PushpinOutlined } from '@ant-design/icons'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Empty, Progress, Segmented, Spin, Tag, message } from 'antd'
import { useMemo, useState } from 'react'
import { api, APIError, type AchievementBadge as Badge, type AchievementCard, type PlayerAchievements as AchievementData } from '../../api'
import { isAchievementArtworkKey } from '../../assets/achievements/achievementArtwork'
import { AchievementBadge } from '../../components/AchievementBadge'
import styles from './PlayerPage.module.scss'

const number = new Intl.NumberFormat()

export function PlayerAchievements({ steamID, data, loading, self, canEdit, onRequireAuth, zh }: { steamID: string; data?: AchievementData; loading: boolean; self: boolean; canEdit: boolean; onRequireAuth: () => void; zh: boolean }) {
  const client = useQueryClient()
  const [category, setCategory] = useState('all')
  const [selection, setSelection] = useState<Badge[] | null>(null)
  const badges = selection ?? data?.overview.badges ?? []
  const save = useMutation({
    mutationFn: (items: Badge[]) => api.saveBadgeShowcase(items.map((item, index) => ({ slot: index + 1, achievement_key: item.achievement_key }))),
    onSuccess: result => {
      setSelection(result.items)
      void client.invalidateQueries({ queryKey: ['player-achievements', steamID] })
      void client.invalidateQueries({ queryKey: ['player-preview', steamID] })
      void message.success(zh ? '展示徽章已保存' : 'Badge showcase saved')
    },
    onError: error => {
      setSelection(null)
      if (error instanceof APIError && (error.code === 'steam_reauthentication_required' || error.code === 'steam_unauthorized')) return onRequireAuth()
      void message.error(zh ? '徽章保存失败' : 'Failed to save badges')
    },
  })
  const groups = useMemo(() => groupCards(data?.items ?? [], category), [category, data?.items])
  if (loading) return <div className={styles.achievementLoading}><Spin /></div>
  if (!data) return <Alert type="warning" showIcon title={zh ? '成就暂时不可用' : 'Achievements are temporarily unavailable'} />

  const toggleBadge = (card: AchievementCard) => {
    if (!card.unlocked || !card.artwork_key) return
    if (!canEdit) return onRequireAuth()
    const current = [...badges]
    const index = current.findIndex(item => item.achievement_key === card.achievement_key)
    if (index >= 0) current.splice(index, 1)
    else if (current.length < 3) current.push({ slot: current.length + 1, achievement_key: card.achievement_key, title: card.title, artwork_key: card.artwork_key, tier: card.tier })
    else return void message.info(zh ? '最多展示 3 个徽章' : 'You can showcase up to three badges')
    const normalized = current.map((item, slot) => ({ ...item, slot: slot + 1 }))
    setSelection(normalized)
    save.mutate(normalized)
  }

  return <div className={styles.achievementStack}>
    <section className={styles.achievementOverview}>
      <div className={styles.achievementProgress}><div><span>{zh ? '成就进度' : 'Achievement progress'}<small>{zh ? `发现彩蛋 ${data.overview.easter_eggs}` : `Easter eggs ${data.overview.easter_eggs}`}</small></span><strong>{data.overview.unlocked} / {data.overview.total}</strong></div><Progress percent={Number(data.overview.completion_percent.toFixed(1))} strokeColor="#d4763b" railColor="rgba(127,101,83,.16)" /></div>
      <div className={styles.achievementSummary}><span>{zh ? '完成度' : 'Completion'}<strong>{data.overview.completion_percent.toFixed(1)}%</strong></span><span>{zh ? '最近解锁' : 'Latest unlock'}<strong>{latestTitle(data)}</strong></span></div>
      <div className={styles.badgeShowcase}><div><strong>{zh ? '徽章展示' : 'Badge showcase'}</strong><small>{canEdit ? (zh ? '点击已解锁成就可调整，最多 3 个' : 'Select up to three unlocked achievements') : self ? (zh ? '编辑前需要重新验证 Steam' : 'Steam re-verification is required to edit') : (zh ? '验证 Steam 身份后可编辑自己的徽章' : 'Verify Steam to edit your own badges')}</small></div><div>{badges.map(item => <AchievementBadge key={item.achievement_key} artworkKey={isAchievementArtworkKey(item.artwork_key) ? item.artwork_key : undefined} tier={item.tier} size={46} label={item.title} />)}{badges.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={zh ? '暂无徽章' : 'No badges'} />}</div></div>
      <div className={styles.achievementFilters}><Segmented block value={category} onChange={value => setCategory(String(value))} options={[
        ['all', zh ? '全部' : 'All'], ['career', zh ? '生涯' : 'Career'], ['combat', zh ? '战斗' : 'Combat'], ['support', zh ? '支援' : 'Support'], ['boss', 'Boss'], ['versus', zh ? '对抗' : 'Versus'], ['bond', zh ? '羁绊' : 'Bond'], ['special', zh ? '特殊' : 'Special'],
      ].map(([value, label]) => ({ value, label }))} /></div>
    </section>

    <div className={styles.achievementGrid}>{groups.map(group => {
      const card = group.display
      const equip = group.highestUnlocked
      const artwork = card.artwork_key && isAchievementArtworkKey(card.artwork_key) ? card.artwork_key : undefined
      const selected = equip ? badges.some(item => item.achievement_key === equip.achievement_key) : false
      return <article key={card.group_key || card.achievement_key} className={`${styles.achievementCard} ${card.unlocked ? styles.achievementUnlocked : ''}`}>
        <AchievementBadge artworkKey={artwork} tier={card.tier} size={64} locked={!card.unlocked && card.visibility === 'public'} mystery={card.visibility === 'mystery' && !card.unlocked} label={card.title} />
        <div className={styles.achievementCardBody}><div className={styles.achievementTitleRow}><strong>{card.title}</strong>{card.visibility === 'mystery' && !card.unlocked ? <span className={styles.mysteryLabel}>{zh ? '隐藏成就' : 'Hidden achievement'}</span> : null}{card.tier ? <Tag>{tierName(card.tier)}</Tag> : null}{card.visibility === 'mystery' && !card.unlocked ? <EyeInvisibleOutlined /> : null}</div><p>{card.description}</p>
          {!card.unlocked && card.visibility === 'public' && <><Progress size="small" percent={progress(card)} showInfo={false} strokeColor="#d4763b" /><div className={styles.achievementValue}><span>{formatMetric(card.current_value)}</span><span>/ {formatMetric(card.threshold)}</span></div></>}
          {group.nextTier && <small className={styles.nextTier}>{zh ? '下一等级' : 'Next tier'}：{tierName(group.nextTier.tier ?? 0)} · {formatMetric(group.nextTier.threshold)}</small>}
        </div>
        {equip && <div className={styles.achievementFooter}><div className={styles.achievementMeta}><span><CheckOutlined /> {zh ? '已确认' : 'Confirmed'} {date(equip.unlocked_at)}</span><span>{zh ? '全服' : 'Global'} {equip.global_unlock_rate.toFixed(1)}%</span>{equip.evidence_steam_id && <span>{zh ? '搭档' : 'Partner'} {equip.evidence_steam_id}</span>}</div><Button className={styles.badgeAction} size="small" type={selected ? 'primary' : 'default'} icon={<PushpinOutlined />} loading={save.isPending} onClick={() => toggleBadge(equip)}>{selected ? (zh ? '取消展示' : 'Remove badge') : (zh ? '展示徽章' : 'Show badge')}</Button></div>}
      </article>
    })}</div>
  </div>
}

function groupCards(items: AchievementCard[], category: string) {
  const groups = new Map<string, AchievementCard[]>()
  items.filter(item => category === 'all' || item.category === category).forEach(item => {
    const key = item.group_key || item.achievement_key
    groups.set(key, [...(groups.get(key) ?? []), item])
  })
  return [...groups.values()].map(items => {
    const ordered = [...items].sort((a, b) => (a.tier ?? 0) - (b.tier ?? 0))
    const unlocked = ordered.filter(item => item.unlocked)
    const highestUnlocked = unlocked.at(-1)
    const nextTier = ordered.find(item => !item.unlocked)
    return { display: nextTier ?? highestUnlocked ?? ordered[0], highestUnlocked, nextTier: highestUnlocked ? nextTier : undefined }
  })
}

function progress(card: AchievementCard) { return card.threshold ? Math.min(100, (card.current_value ?? 0) / card.threshold * 100) : 0 }
function tierName(tier: number) { return ['—', 'I', 'II', 'III', 'IV'][tier] ?? String(tier) }
function date(value = 0) { return value ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(value * 1000)) : '—' }
function formatMetric(value = 0) {
  return number.format(value)
}
function latestTitle(data: AchievementData) { const key = data.overview.recent_unlock?.achievement_key; return key ? data.items.find(item => item.achievement_key === key)?.title ?? '—' : '—' }
