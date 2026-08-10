import { Button, Tag } from 'antd'
import { useTranslation } from 'react-i18next'
import type { PlayerChapter, PlayerSession } from '../../api'
import { Section } from './PlayerShared'
import { date, duration, type PlayerCopy } from './playerFormat'
import styles from './PlayerPage.module.scss'

interface HistoryPageState {
  hasNextPage?: boolean
  isFetchingNextPage: boolean
  fetchNextPage: () => unknown
}

export function PlayerHistory({ sessions, chapters, sessionPage, chapterPage, copy }: {
  sessions: PlayerSession[]
  chapters: PlayerChapter[]
  sessionPage: HistoryPageState
  chapterPage: HistoryPageState
  copy: PlayerCopy
}) {
  const { t } = useTranslation()
  return <div className={styles.historyGrid}>
    <Section title={copy.sessionHistory}>
      <div className={styles.recordList}>{sessions.map((item, index) => <div key={`${item.started_at}-${index}`}>
        <strong className={styles.recordPrimary}>{item.player_name || item.server_key}</strong>
        <span className={styles.recordSecondary}>{item.server_key}</span>
        <span className={styles.recordDate}>{date(item.started_at)}</span>
        <span className={styles.recordDuration}>{duration(item.active_play_seconds)}</span>
        <span className={styles.recordStatus}><Tag>{item.status}</Tag></span>
      </div>)}</div>
      {sessionPage.hasNextPage && <Button loading={sessionPage.isFetchingNextPage} onClick={() => void sessionPage.fetchNextPage()}>{t('loadMore')}</Button>}
    </Section>
    <Section title={copy.chapterHistory}>
      <div className={styles.recordList}>{chapters.map((item, index) => <div key={`${item.started_at}-${index}`}>
        <strong className={styles.recordPrimary}>{item.map_name}</strong>
        <span className={styles.recordSecondary}>{item.game_mode} · {item.side || '—'}</span>
        <span className={styles.recordDate}>{date(item.started_at)}</span>
        <span className={styles.recordDuration}>{duration(item.active_play_seconds)}</span>
        <span className={styles.recordStatus}><Tag>{item.status}</Tag></span>
      </div>)}</div>
      {chapterPage.hasNextPage && <Button loading={chapterPage.isFetchingNextPage} onClick={() => void chapterPage.fetchNextPage()}>{t('loadMore')}</Button>}
    </Section>
  </div>
}
