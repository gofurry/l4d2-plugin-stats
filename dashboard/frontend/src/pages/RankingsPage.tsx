import { FilterOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { Button, Empty, Layout, Modal, Select, Spin, Table, Tabs } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useDeferredValue, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { api, type RankingEntry } from '../api'
import { EChart } from '../components/EChart'
import { FloatingNav } from '../components/FloatingNav'
import { FloatingToolbar } from '../components/FloatingToolbar'
import styles from './RankingsPage.module.scss'

const metricOptions = {
  activity: ['active_time', 'sessions'],
  pve: ['common_kills', 'headshot_kills', 'special_kills', 'boss_kills', 'special_damage', 'rescues', 'teammate_protections', 'hunter_skeets', 'charger_levels', 'healing', 'campaign_completions', 'tongue_self_cuts', 'rocks_destroyed', 'car_alarms_triggered', 'common_kills_per_hour', 'special_kills_per_hour', 'rescues_per_hour', 'incaps_per_hour', 'deaths_per_hour', 'friendly_fire_per_hour', 'tank_participation_rate', 'witch_participation_rate'],
  versus_survivor: ['human_si_kills', 'damage', 'rescues', 'teammate_protections', 'hunter_skeets', 'charger_levels', 'car_alarms_triggered', 'human_si_kills_per_hour', 'rescues_per_hour', 'incaps_per_hour'],
  versus_infected: ['damage', 'incaps', 'kills', 'controls', 'damage_per_hour', 'incaps_per_spawn', 'controls_per_spawn', 'kills_per_spawn'],
} as const

const metricLabels: Record<string, [string, string]> = {
  active_time: ['实际操作时长', 'Active play time'], sessions: ['会话数', 'Sessions'], common_kills: ['普通感染者击杀', 'Common kills'], headshot_kills: ['爆头击杀', 'Headshot kills'], special_kills: ['特殊感染者击杀', 'Special kills'], boss_kills: ['Boss 击杀', 'Boss kills'], special_damage: ['特感与 Boss 伤害', 'SI and Boss damage'], rescues: ['团队救援', 'Team rescues'], healing: ['治疗量', 'Healing'], campaign_completions: ['完成战役', 'Campaign completions'], tongue_self_cuts: ['断舌自救', 'Self tongue cuts'], rocks_destroyed: ['击碎 Tank 石块', 'Tank rocks destroyed'], car_alarms_triggered: ['触发警报车', 'Car alarms triggered'], common_kills_per_hour: ['每小时普通感染者击杀', 'Common kills per hour'], special_kills_per_hour: ['每小时特感击杀', 'Special kills per hour'], human_si_kills: ['真人特感 / Tank 击杀', 'Human SI / Tank kills'], damage: ['伤害', 'Damage'], human_si_kills_per_hour: ['每小时真人特感击杀', 'Human SI kills per hour'], incaps: ['击倒真人幸存者', 'Human survivor incaps'], kills: ['击杀真人幸存者', 'Human survivor kills'], controls: ['控制真人幸存者', 'Human survivor controls'], damage_per_hour: ['每小时伤害', 'Damage per hour'],
  rescues_per_hour: ['每小时团队救援', 'Team rescues per hour'], incaps_per_hour: ['每小时倒地（越低越好）', 'Incaps per hour (lower is better)'], deaths_per_hour: ['每小时死亡（越低越好）', 'Deaths per hour (lower is better)'], friendly_fire_per_hour: ['每小时友伤（越低越好）', 'Friendly fire per hour (lower is better)'], tank_participation_rate: ['Tank 击杀参与率', 'Tank participation rate'], witch_participation_rate: ['Witch 击杀参与率', 'Witch participation rate'], incaps_per_spawn: ['每次复活击倒', 'Incaps per spawn'], controls_per_spawn: ['每次复活控制', 'Controls per spawn'], kills_per_spawn: ['每次复活击杀', 'Kills per spawn'],
  teammate_protections: ['保护队友', 'Teammate protections'], hunter_skeets: ['Hunter 空中击杀', 'Hunter Skeets'], charger_levels: ['近战截停 Charger', 'Charger Levels'],
}

const modeLabels: Record<string, [string, string]> = {
  activity: ['活跃', 'Activity'], pve: ['PvE', 'PvE'], versus_survivor: ['对抗幸存者', 'Versus survivor'], versus_infected: ['对抗感染者', 'Versus infected'],
}

const number = new Intl.NumberFormat()
const playerStorageKey = 'l4d2-stats.player.steam-id.v1'
const validSteamID = (value: string) => /^7656119\d{10}$/.test(value)
const valueLabel = (metric: string, value: number) => metric === 'active_time' ? `${(value / 3600).toFixed(1)} h` : metric.endsWith('_rate') ? `${(value * 100).toFixed(1)}%` : metric.includes('_per_') ? value.toFixed(2) : number.format(Math.round(value))

export function RankingsPage() {
  const { i18n } = useTranslation()
  const zh = i18n.language !== 'en'
  const navigate = useNavigate()
  const [mode, setMode] = useState<keyof typeof metricOptions>('pve')
  const [metric, setMetric] = useState<string>('common_kills')
  const [range, setRange] = useState('all')
  const [server, setServer] = useState('')
  const [page, setPage] = useState(1)
  const [filterOpen, setFilterOpen] = useState(false)
  const [playerSearch, setPlayerSearch] = useState('')
  const [selectedPlayers, setSelectedPlayers] = useState<string[]>([])
  const [playerLabels, setPlayerLabels] = useState<Record<string, string>>({})
  const [mySteamID] = useState(() => {
    const value = localStorage.getItem(playerStorageKey) ?? ''
    return validSteamID(value) ? value : ''
  })
  const deferredPlayerSearch = useDeferredValue(playerSearch.trim())
  const servers = useQuery({ queryKey: ['ranking-servers'], queryFn: api.rankingServers, staleTime: 300_000 })
  const players = useQuery({ queryKey: ['ranking-players', deferredPlayerSearch], queryFn: () => api.rankingPlayers(deferredPlayerSearch), enabled: filterOpen, staleTime: 30_000 })
  const playerFilterKey = selectedPlayers.join(',')
  const query = useQuery({ queryKey: ['rankings', mode, metric, range, server, playerFilterKey, mySteamID, page], queryFn: () => api.rankings({ mode, metric, range, server, players: selectedPlayers, subject: mySteamID, page, limit: 20 }) })
  const top = useQuery({ queryKey: ['rankings-top', mode, metric, range, server, playerFilterKey], queryFn: () => api.rankings({ mode, metric, range, server, players: selectedPlayers, page: 1, limit: 10 }), enabled: page !== 1 })
  const topPage = page === 1 ? query.data : top.data
  const chooseMode = (value: string | number) => {
    const next = String(value) as keyof typeof metricOptions
    setMode(next)
    setMetric(metricOptions[next][0])
    setPage(1)
  }
  const label = (key: string) => metricLabels[key]?.[zh ? 0 : 1] ?? key
  const playerOptions = useMemo(() => {
    const options = new Map<string, string>()
    for (const [steamID, text] of Object.entries(playerLabels)) options.set(steamID, text)
    for (const player of players.data ?? []) options.set(player.steam_id, `${player.name || player.steam_id} · ${player.steam_id}`)
    return [...options].map(([value, optionLabel]) => ({ value, label: optionLabel }))
  }, [playerLabels, players.data])

  const chartOption = useMemo<EChartsCoreOption>(() => {
    const items = [...(topPage?.items ?? [])].slice(0, 10).reverse()
    return {
      animationDuration: 450,
      grid: { left: 16, right: 70, top: 12, bottom: 22, containLabel: true },
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, backgroundColor: 'rgba(67,48,38,.94)', borderWidth: 0, textStyle: { color: '#fff7ed' } },
      xAxis: { type: 'value', axisLabel: { color: '#806d62' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
      yAxis: { type: 'category', data: items.map(item => item.player_name || item.steam_id), axisLabel: { color: '#5f4a3f', width: 150, overflow: 'truncate' }, axisLine: { show: false }, axisTick: { show: false } },
      series: [{ type: 'bar', data: items.map(item => item.value), barMaxWidth: 24, label: { show: true, position: 'right', color: '#6d5548', formatter: (params: { value?: unknown }) => valueLabel(metric, Number(params.value ?? 0)) }, itemStyle: { color: '#c8753f', borderRadius: [0, 7, 7, 0] } }],
    }
  }, [topPage, metric])

  const columns = [
    { title: '#', dataIndex: 'rank', width: 70 },
    { title: zh ? '玩家' : 'Player', key: 'player', render: (_: unknown, item: RankingEntry) => <button className={styles.playerLink} onClick={() => navigate(`/player?steam_id=${item.steam_id}`)}>{item.player_name || item.steam_id}<small>{item.steam_id}</small></button> },
    { title: label(metric), dataIndex: 'value', align: 'right' as const, render: (value: number) => <strong>{valueLabel(metric, value)}</strong> },
    { title: zh ? '有效时长' : 'Active time', dataIndex: 'active_play_seconds', align: 'right' as const, render: (value: number) => `${(value / 3600).toFixed(1)} h` },
  ]

  const toolbarItems = [{ key: 'filter', label: zh ? '筛选排行榜' : 'Filter rankings', icon: <FilterOutlined />, onClick: () => setFilterOpen(true) }]
  const rankingError = query.isError || (page !== 1 && top.isError)
  const rankingLoading = query.isLoading || (page !== 1 && top.isLoading)
  const noRankingData = !rankingLoading && !rankingError && (query.data?.total ?? 0) === 0

  return <Layout className={styles.layout}><FloatingNav /><FloatingToolbar ariaLabel={zh ? '排行榜工具' : 'Ranking tools'} items={toolbarItems} /><Layout.Content className={styles.content}>
    <Tabs className={styles.modeTabs} activeKey={mode} onChange={chooseMode} items={Object.keys(metricOptions).map(value => ({ key: value, label: modeLabels[value][zh ? 0 : 1] }))} />

    {rankingError && <section className={styles.statePanel}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={zh ? '排行榜暂时无法读取，请稍后重试。' : 'Rankings are temporarily unavailable.'}><div className={styles.stateActions}><Button onClick={() => void Promise.all([query.refetch(), top.refetch()])}>{zh ? '重新加载' : 'Retry'}</Button><Button type="primary" onClick={() => setFilterOpen(true)}>{zh ? '调整筛选' : 'Change filters'}</Button></div></Empty></section>}
    {noRankingData && <section className={styles.statePanel}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={zh ? '当前玩法或筛选条件下还没有排行榜数据。' : 'There is no ranking data for this mode or filter yet.'}><div className={styles.stateActions}>{mode !== 'pve' && <Button type="primary" onClick={() => chooseMode('pve')}>{zh ? '查看 PvE 排行榜' : 'View PvE rankings'}</Button>}<Button onClick={() => setFilterOpen(true)}>{zh ? '调整筛选' : 'Change filters'}</Button></div></Empty></section>}
    {!rankingError && !noRankingData && <><section className={styles.chartPanel}>
      <div className={styles.panelHeading}><h3>{label(metric)} · Top 10</h3><span>{topPage ? new Date(topPage.generated_at).toLocaleString() : ''}</span></div>
      {rankingLoading ? <Spin /> : (topPage?.items ?? []).length ? <EChart className={styles.chart} option={chartOption} ariaLabel={`${label(metric)} Top 10`} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />}
    </section>

    <section className={styles.tablePanel}>
      <Table<RankingEntry> className={styles.rankingList} columns={columns} dataSource={query.data?.items ?? []} rowKey="steam_id" loading={query.isLoading} pagination={{ current: page, pageSize: 20, total: query.data?.total ?? 0, showSizeChanger: false, showTotal: total => zh ? `共 ${number.format(total)} 名` : `${number.format(total)} players`, onChange: setPage }} scroll={{ x: 680 }} />
    </section>
    {mySteamID && <section className={styles.myRankCard}>
      <span className={styles.myRankNumber}>#{query.data?.self?.rank ?? '—'}</span>
      <button className={styles.myRankIdentity} onClick={() => navigate(`/player?steam_id=${mySteamID}`)} type="button"><strong>{zh ? '我的排名' : 'My rank'}</strong><small>{query.data?.self?.player_name || mySteamID}</small></button>
      <div><span>{label(metric)}</span><strong>{query.data?.self ? valueLabel(metric, query.data.self.value) : '—'}</strong></div>
      <div><span>{zh ? '有效时长' : 'Active time'}</span><strong>{query.data?.self ? `${(query.data.self.active_play_seconds / 3600).toFixed(1)} h` : '—'}</strong></div>
    </section>}</>}
  </Layout.Content><Modal title={zh ? '筛选排行榜' : 'Filter rankings'} open={filterOpen} onCancel={() => setFilterOpen(false)} onOk={() => setFilterOpen(false)} okText={zh ? '完成' : 'Done'} cancelButtonProps={{ style: { display: 'none' } }}>
    <div className={styles.filterModal}>
      <label><span>{zh ? '指标' : 'Metric'}</span><Select value={metric} onChange={value => { setMetric(value); setPage(1) }} options={metricOptions[mode].map(value => ({ value, label: label(value) }))} /></label>
      <label><span>{zh ? '时间范围' : 'Range'}</span><Select value={range} onChange={value => { setRange(value); setPage(1) }} options={[['all', zh ? '全部' : 'All time'], ['30d', zh ? '近 30 天' : '30 days'], ['90d', zh ? '近 90 天' : '90 days'], ['365d', zh ? '近一年' : '1 year']].map(([value, text]) => ({ value, label: text }))} /></label>
      <label><span>{zh ? '服务器' : 'Server'}</span><Select value={server} onChange={value => { setServer(value); setPage(1) }} options={[{ value: '', label: zh ? '全部服务器' : 'All servers' }, ...(servers.data ?? []).map(value => ({ value, label: value }))]} /></label>
      <label><span>{zh ? '玩家' : 'Players'}</span><Select mode="multiple" allowClear filterOption={false} maxTagCount="responsive" value={selectedPlayers} searchValue={playerSearch} onSearch={setPlayerSearch} onSelect={(value, option) => setPlayerLabels(current => ({ ...current, [value]: String(option.label ?? value) }))} onChange={values => { setSelectedPlayers(values.slice(0, 20)); setPage(1) }} options={playerOptions} notFoundContent={players.isFetching ? <Spin size="small" /> : null} placeholder={zh ? '按名称或 SteamID 搜索，可多选' : 'Search name or SteamID; multiple allowed'} /></label>
    </div>
  </Modal></Layout>
}
