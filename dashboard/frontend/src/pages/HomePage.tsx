import { GlobalOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { Alert, Button, Card, Col, Layout, Row, Skeleton, Space, Statistic, Tag, Typography } from 'antd'
import { useTranslation } from 'react-i18next'
import { api, type Overview, type ServerStatus } from '../api'
import { toggleLanguage } from '../i18n'
import { SiteFooter } from '../components/SiteFooter'
import styles from './HomePage.module.scss'

const { Header, Content } = Layout

const integer = new Intl.NumberFormat()

function MetricCard({ title, value, suffix }: { title: string; value: number | string; suffix?: string }) {
  return <Card className={styles.metricCard}><Statistic title={title} value={value} suffix={suffix} /></Card>
}

function ServerCard({ status, loading, failed }: { status: ServerStatus | null | undefined; loading: boolean; failed: boolean }) {
  const { t } = useTranslation()
  if (loading) return <Card className={styles.serverCard}><Skeleton active paragraph={{ rows: 3 }} /></Card>
  if (failed) return <Card className={styles.serverCard}><Alert type="warning" showIcon title={t('serverUnavailable')} /></Card>
  if (!status) return <Card className={styles.serverCard}><Typography.Text type="secondary">{t('notConfigured')}</Typography.Text></Card>
  const tag = status.online ? <Tag color="green">{t('online')}</Tag> : <Tag color={status.stale ? 'orange' : 'red'}>{status.stale ? t('stale') : t('offline')}</Tag>
  return <Card className={styles.serverCard}>
    <div className={`${styles.serverTop} flex items-start justify-between`}>
      <div><Typography.Text className={styles.eyebrow}>{t('serverStatus')}</Typography.Text><Typography.Title level={2}>{status.name || status.configured_name}</Typography.Title></div>
      {tag}
    </div>
    <Row gutter={[16, 16]} className={styles.serverDetails}>
      <Col xs={12} md={6}><span>{t('map')}</span><strong>{status.map || '—'}</strong></Col>
      <Col xs={12} md={6}><span>{t('players')}</span><strong>{status.players} / {status.max_players}</strong></Col>
      <Col xs={12} md={6}><span>{t('bots')}</span><strong>{status.bots}</strong></Col>
      <Col xs={12} md={6}><span>{t('latency')}</span><strong>{status.latency_ms == null ? '—' : `${status.latency_ms} ms`}</strong></Col>
    </Row>
    <Space wrap>
      <Button type="primary" icon={<PlayCircleOutlined />} href={`steam://connect/${status.connect_address}`}>{t('join')}</Button>
      {status.last_success_at && <Typography.Text type="secondary">{t('lastSuccess')}: {new Date(status.last_success_at).toLocaleString()}</Typography.Text>}
    </Space>
  </Card>
}

function OverviewContent({ data }: { data: Overview }) {
  const { t } = useTranslation()
  const completed = data.core.completed_pve_runs + data.core.completed_versus_runs
  const empty = data.core.total_players === 0 && data.core.total_active_play_seconds === 0
  return <>
    <Typography.Title level={2} className={styles.sectionTitle}>{t('overview')}</Typography.Title>
    {empty && <Alert className={styles.alert} type="info" showIcon title={t('noData')} />}
    <Row gutter={[16, 16]}>
      <Col xs={12} lg={6}><MetricCard title={t('totalPlayers')} value={integer.format(data.core.total_players)} /></Col>
      <Col xs={12} lg={6}><MetricCard title={t('active7d')} value={integer.format(data.core.active_players_7d)} /></Col>
      <Col xs={12} lg={6}><MetricCard title={t('playTime')} value={integer.format(Math.round(data.core.total_active_play_seconds / 3600))} suffix={t('hours')} /></Col>
      <Col xs={12} lg={6}><MetricCard title={t('completedRuns')} value={integer.format(completed)} suffix={`${t('pveRuns')} ${data.core.completed_pve_runs} / ${t('versusRuns')} ${data.core.completed_versus_runs}`} /></Col>
    </Row>
    <Row gutter={[20, 20]} className={styles.modeGrid}>
      <Col xs={24} xl={12}><Card title={t('pve')} className={styles.modeCard}><Row gutter={[12, 20]}>
        <Col span={12}><Statistic title={t('commonKills')} value={data.pve.common_kills} /></Col>
        <Col span={12}><Statistic title={t('specialKills')} value={data.pve.special_kills} /></Col>
        <Col span={12}><Statistic title={t('bossKills')} value={data.pve.tank_kills + data.pve.witch_kills} /></Col>
        <Col span={12}><Statistic title={t('rescues')} value={data.pve.rescues} /></Col>
      </Row></Card></Col>
      <Col xs={24} xl={12}><Card title={t('versus')} className={styles.modeCard}><Row gutter={[12, 20]}>
        <Col span={12}><Statistic title={t('matches')} value={data.versus.completed_matches} /></Col>
        <Col span={12}><Statistic title={t('halves')} value={data.versus.completed_halves} /></Col>
        <Col span={12}><Statistic title={t('humanSIKills')} value={data.versus.human_controlled_infected_kills} /></Col>
        <Col span={12}><Statistic title={t('controls')} value={data.versus.human_survivor_controls} /></Col>
      </Row></Card></Col>
    </Row>
  </>
}

export function HomePage() {
  const { t } = useTranslation()
  const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 5 * 60_000 })
  const overview = useQuery({ queryKey: ['overview'], queryFn: api.overview, staleTime: 60_000, retry: 1 })
  const status = useQuery({ queryKey: ['primary-server'], queryFn: api.primaryServer, refetchInterval: 30_000, retry: 1 })
  return <Layout className={styles.layout}>
    <Header className={`${styles.header} flex items-center justify-between`}>
      <div><Typography.Title level={3}>{site.data?.title ?? 'L4D2 Stats'}</Typography.Title><Typography.Text>{t('subtitle')}</Typography.Text></div>
      <Button type="text" icon={<GlobalOutlined />} onClick={toggleLanguage}>{t('language')}</Button>
    </Header>
    <Content className={`${styles.content} mx-auto`}>
      {site.isError && <Alert className={styles.alert} type="warning" showIcon title={t('siteUnavailable')} />}
      <ServerCard status={status.data} loading={status.isLoading} failed={status.isError} />
      {overview.isLoading && <div className={styles.loading}><Skeleton active paragraph={{ rows: 7 }} /></div>}
      {overview.isError && <Alert className={styles.alert} type="warning" showIcon title={t('statsUnavailable')}
        action={<Button icon={<ReloadOutlined />} onClick={() => void overview.refetch()}>{t('retry')}</Button>} />}
      {overview.data && <OverviewContent data={overview.data} />}
      <SiteFooter site={site.data} />
    </Content>
  </Layout>
}
