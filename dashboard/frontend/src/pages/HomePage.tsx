import { BookOutlined, CodeOutlined, InfoCircleOutlined, PlayCircleOutlined, ReloadOutlined, RightOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { Alert, Button, Empty, Layout, Modal, Skeleton, Spin } from 'antd'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type Overview, type ServerPlayer, type ServerStatus, type SiteDocumentKey } from '../api'
import { FloatingNav } from '../components/FloatingNav'
import { FloatingToolbar } from '../components/FloatingToolbar'
import { MarkdownContent } from '../components/MarkdownContent'
import { PlayerPreviewModal } from '../components/PlayerPreviewModal'
import { SiteFooter } from '../components/SiteFooter'
import styles from './HomePage.module.scss'

const { Content } = Layout
const integer = new Intl.NumberFormat()

function Metric({ title, value, suffix, details }: { title: string; value: number | string; suffix?: string; details?: string }) {
  return <div className={styles.metric}>
    <span>{title}</span>
    <div className={styles.metricValue}><strong>{value}</strong>{suffix && <small>{suffix}</small>}{details && <small>{details}</small>}</div>
  </div>
}

function ServerRow({ status }: { status: ServerStatus }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
	const [selectedPlayer, setSelectedPlayer] = useState<ServerPlayer | null>(null)
  const state = status.stale ? 'stale' : status.checking ? 'checking' : status.online ? 'online' : 'offline'
  const players = status.player_list ?? []
  const toggle = () => setExpanded(value => !value)

  return <article className={`${styles.serverItem} ${expanded ? styles.expanded : ''}`}>
    <div className={styles.serverRow} role="button" tabIndex={0} aria-expanded={expanded} onClick={toggle}
      onKeyDown={event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); toggle() } }}>
      <div className={styles.serverIdentity}>
        <div className={styles.serverNameLine}>
          <i className={`${styles.statusDot} ${styles[`statusDot_${state}`]}`} aria-label={t(state)} title={t(state)} />
          <strong>{status.name || status.display_name}</strong>
        </div>
        {status.name && status.name !== status.display_name && <span>{status.display_name}</span>}
      </div>
      <div className={styles.serverFacts}>
        <div><span>{t('map')}</span><strong>{status.map || (status.checking ? t('checking') : '—')}</strong></div>
        <div><span>{t('players')}</span><strong>{status.players} / {status.max_players}</strong></div>
        <div><span>{t('latency')}</span><strong>{status.latency_ms == null ? '—' : `${status.latency_ms} ms`}</strong></div>
      </div>
      <Button type="primary" icon={<PlayCircleOutlined />} href={`steam://connect/${status.address}`}
        onClick={event => event.stopPropagation()}>{t('join')}</Button>
      <RightOutlined className={styles.expandIcon} aria-hidden="true" />
    </div>
    <div className={`${styles.playerExpandRegion} ${expanded ? styles.open : ''}`}><div className={styles.playerExpandInner}>
      {expanded && <div className={styles.playerPanel}>
        {status.checking && !status.stale && <span className={styles.playerNotice}>{t('serverChecking')}</span>}
        {!status.checking && !status.online && <span className={styles.playerNotice}>{t('playerDetailsUnavailable')}</span>}
        {status.online && players.length === 0 && <span className={styles.playerNotice}>{status.players === 0 ? t('noOnlinePlayers') : t('noPlayerDetails')}</span>}
        {players.length > 0 && <div className={styles.playerList}>
          <div className={styles.playerListHeader}><span>{t('playerName')}</span><span>{t('score')}</span><span>{t('onlineDuration')}</span></div>
          {players.map((player, index) => <div className={`${styles.playerEntry} ${player.steam_id ? styles.linkedPlayer : ''}`} key={player.steam_id ?? `${player.name}-${index}`}
			role={player.steam_id ? 'button' : undefined} tabIndex={player.steam_id ? 0 : undefined}
			title={player.steam_id ? t('clickForPlayerStats') : undefined}
			onClick={() => { if (player.steam_id) setSelectedPlayer(player) }}
			onKeyDown={event => { if (player.steam_id && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); setSelectedPlayer(player) } }}>
            <strong>{player.name || t('unnamedPlayer')}</strong><span>{player.score}</span><span>{formatDuration(player.duration_seconds)}</span>
          </div>)}
        </div>}
      </div>}
    </div></div>
	<PlayerPreviewModal open={selectedPlayer !== null} steamID={selectedPlayer?.steam_id ?? ''} playerName={selectedPlayer?.name}
		contextLabel={status.name || status.display_name} onClose={() => setSelectedPlayer(null)} />
  </article>
}

function formatDuration(seconds: number) {
  if (seconds < 60) return `${Math.max(0, seconds)}s`
  const minutes = Math.floor(seconds / 60)
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h ${minutes % 60}m`
}

function ServerList({ data, loading, failed }: { data?: ServerStatus[]; loading: boolean; failed: boolean }) {
  const { t } = useTranslation()
  return <section className={styles.section}>
    {loading && <div className={styles.loadingRow}><Skeleton active paragraph={{ rows: 2 }} /></div>}
    {failed && <Alert className={styles.serverAlert} type="warning" showIcon title={t('serverUnavailable')} />}
    {!loading && !failed && data?.length === 0 && <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('notConfigured')} /></div>}
    <div className={styles.serverList}>{data?.map(status => <ServerRow key={status.server_id} status={status} />)}</div>
  </section>
}

function OverviewContent({ data }: { data: Overview }) {
  const { t } = useTranslation()
  const completed = data.core.completed_pve_runs + data.core.completed_versus_runs
  const empty = data.core.total_players === 0 && data.core.total_active_play_seconds === 0
  return <section className={`${styles.section} ${styles.overviewSection}`}>
    {empty && <Alert className={styles.alert} type="info" showIcon title={t('noData')} />}
    <div className={styles.metrics}>
      <Metric title={t('totalPlayers')} value={integer.format(data.core.total_players)} />
      <Metric title={t('active7d')} value={integer.format(data.core.active_players_7d)} />
      <Metric title={t('playTime')} value={integer.format(Math.round(data.core.total_active_play_seconds / 3600))} suffix={t('hours')} />
      <Metric title={t('completedRuns')} value={integer.format(completed)}
        details={`${t('pveRuns')} ${integer.format(data.core.completed_pve_runs)} · ${t('versusRuns')} ${integer.format(data.core.completed_versus_runs)}`} />
    </div>
  </section>
}

export function HomePage() {
  const { t } = useTranslation()
  const [selectedDocument, setSelectedDocument] = useState<SiteDocumentKey | null>(null)
  const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 5 * 60_000 })
  const overview = useQuery({ queryKey: ['overview'], queryFn: api.overview, staleTime: 60_000, retry: 1 })
  const statusRefreshMS = (site.data?.a2s_refresh_seconds ?? 30) * 1000
  const statuses = useQuery({
    queryKey: ['server-statuses'],
    queryFn: api.serverStatuses,
    refetchInterval: query => query.state.data?.some(status => status.checking) ? 1000 : statusRefreshMS,
    retry: 1,
  })
  const document = useQuery({
    queryKey: ['site-document', selectedDocument],
    queryFn: () => api.siteDocument(selectedDocument!),
    enabled: selectedDocument !== null,
    staleTime: 5 * 60_000,
    retry: 1,
  })
  const documentLabels: Record<SiteDocumentKey, string> = {
    introduction: t('serverIntroduction'),
    commands: t('serverCommands'),
    resources: t('serverResources'),
  }
  const documentIcons = { introduction: <InfoCircleOutlined />, commands: <CodeOutlined />, resources: <BookOutlined /> }
  const documentItems = (site.data?.site_documents ?? []).map(key => ({
    key,
    label: documentLabels[key],
    icon: documentIcons[key],
    onClick: () => setSelectedDocument(key),
  }))

  return <Layout className={styles.layout}>
    <FloatingNav />
    {documentItems.length > 0 && <FloatingToolbar ariaLabel={t('homeTools')} items={documentItems} />}
    <Content className={styles.content}>
      {site.isError && <Alert className={styles.alert} type="warning" showIcon title={t('siteUnavailable')} />}
      {overview.isLoading && <div className={styles.loading}><Skeleton active paragraph={{ rows: 7 }} /></div>}
      {overview.isError && <Alert className={styles.alert} type="warning" showIcon title={t('statsUnavailable')}
        action={<Button icon={<ReloadOutlined />} onClick={() => void overview.refetch()}>{t('retry')}</Button>} />}
      {overview.data && <OverviewContent data={overview.data} />}
      <ServerList data={statuses.data} loading={statuses.isLoading} failed={statuses.isError} />
      <SiteFooter site={site.data} />
    </Content>
    <Modal className={styles.siteDocumentModal} open={selectedDocument !== null} title={selectedDocument ? documentLabels[selectedDocument] : ''}
      footer={null} onCancel={() => setSelectedDocument(null)} destroyOnHidden>
      <div className={styles.siteDocumentBody}>
        {document.isLoading && <Spin />}
        {document.isError && <Alert type="warning" showIcon title={t('contentUnavailable')} />}
        {document.data && <MarkdownContent source={document.data.content_markdown} />}
      </div>
    </Modal>
  </Layout>
}
