import { useQuery } from '@tanstack/react-query'
import { Alert, Button, Modal, Skeleton } from 'antd'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import styles from '../pages/HomePage.module.scss'

const integer = new Intl.NumberFormat()

interface PlayerPreviewModalProps {
  open: boolean
  steamID: string
  playerName?: string
  contextLabel?: string
  onClose: () => void
}

export function PlayerPreviewModal({ open, steamID, playerName, contextLabel, onClose }: PlayerPreviewModalProps) {
  const { t } = useTranslation()
  const preview = useQuery({
    queryKey: ['player-preview', steamID],
    queryFn: () => api.playerPreview(steamID),
    enabled: open && steamID !== '',
    staleTime: 60_000,
    retry: 1,
  })
  const data = preview.data

  return <Modal className={styles.playerPreviewModal} open={open} title={null} footer={null} onCancel={onClose} destroyOnHidden>
    <div className={styles.previewHeader}>
      <div><strong>{data?.player_name || playerName || t('unnamedPlayer')}</strong><code>{steamID}</code></div>
      {contextLabel && <span><i className={`${styles.statusDot} ${styles.statusDot_online}`} />{contextLabel}</span>}
    </div>
    {preview.isLoading && <Skeleton active paragraph={{ rows: 5 }} />}
    {preview.isError && <Alert type="warning" showIcon title={t('playerStatsUnavailable')} />}
    {data && <>
      <div className={styles.previewMetrics}>
        <PreviewMetric label={t('activePlayTime')} value={formatDuration(data.active_play_seconds)} />
        <PreviewMetric label={t('campaigns')} value={integer.format(data.campaign_completions)} />
        <PreviewMetric label={t('bossKills')} value={`Tank ${integer.format(data.tank_kills)} / Witch ${integer.format(data.witch_kills)}`} />
        <PreviewMetric label={t('commonKills')} value={integer.format(data.common_kills)} />
        <PreviewMetric label={t('specialKills')} value={integer.format(data.special_kills)} />
        <PreviewMetric label={t('headshotKills')} value={integer.format(data.headshot_kills)} />
        <PreviewMetric label={t('revives')} value={integer.format(data.incap_revives)} />
        <PreviewMetric label={t('incaps')} value={integer.format(data.incapacitations)} />
      </div>
      <div className={styles.previewActions}>
        <Button href={`https://steamcommunity.com/profiles/${steamID}`} target="_blank" rel="noreferrer">{t('viewSteamProfile')}</Button>
        <Button type="primary" href={`/player?steam_id=${steamID}`}>{t('viewFullProfile')}</Button>
      </div>
    </>}
  </Modal>
}

function PreviewMetric({ label, value }: { label: string; value: string }) {
  return <div><span>{label}</span><strong>{value}</strong></div>
}

function formatDuration(seconds: number) {
  if (seconds < 60) return `${Math.max(0, seconds)}s`
  const minutes = Math.floor(seconds / 60)
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h ${minutes % 60}m`
}
