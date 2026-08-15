import { useQuery } from '@tanstack/react-query'
import { Alert, Button, Drawer, Empty, Layout, Segmented, Select, Spin, Table, Tabs } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type AnalysisContextRow, type AnalysisIncident, type AnalysisMapRow } from '../api'
import { EChart } from '../components/EChart'
import { FloatingNav } from '../components/FloatingNav'
import styles from './AnalysisPage.module.scss'

const integer = new Intl.NumberFormat()
const percent = (value?: number) => value === undefined ? '—' : `${(value * 100).toFixed(1)}%`
const duration = (value?: number) => value === undefined ? '—' : value < 60 ? `${value.toFixed(0)}s` : `${(value / 60).toFixed(1)}m`
const rate = (value: number, sample: number) => sample > 0 ? (value / sample).toFixed(2) : '—'
const shortDate = (value: number) => {
  if (!value) return '—'
  const date = new Date(value * 1000)
  return `${String(date.getFullYear()).slice(-2)}/${date.getMonth() + 1}/${date.getDate()}`
}
const difficulty = (value: string, zh: boolean) => ({ Easy: zh ? '简单' : 'Easy', Normal: zh ? '普通' : 'Normal', Hard: zh ? '高级' : 'Advanced', Impossible: zh ? '专家' : 'Expert' }[value] ?? value) || (zh ? '未知' : 'Unknown')
const incidentLabels: Record<string, [string, string]> = {
  controls: ['被特感控制', 'Controls'], incaps: ['倒地', 'Incapacitations'], deaths: ['死亡', 'Deaths'], revives: ['倒地救起', 'Revives'], ledge_rescues: ['挂边救援', 'Ledge rescues'], defib_revives: ['电击复活', 'Defibrillator revives'], car_alarms: ['触发警报车', 'Car alarms'],
  witch_startles: ['惊扰 Witch', 'Witch startles'], medkit_heals: ['医疗包治疗', 'Medkit heals'], objective_completes: ['完成目标互动', 'Objectives completed'],
}
const oneDecimal = (value: number) => Number(value.toFixed(1))
const pageSize = 20
type SortState = { field: string; order: 'asc' | 'desc' }

export function AnalysisPage() {
  const { i18n } = useTranslation()
  const zh = i18n.language !== 'en'
  const [range, setRange] = useState('90d')
  const [mode, setMode] = useState('pve')
  const [server, setServer] = useState<string>()
  const [campaign, setCampaign] = useState<string>()
  const [tab, setTab] = useState('maps')
  const [selectedMap, setSelectedMap] = useState('')
  const [mapPage, setMapPage] = useState(1)
  const [contextPage, setContextPage] = useState(1)
  const [mapSort, setMapSort] = useState<SortState>({ field: 'map_name', order: 'asc' })
  const [contextSort, setContextSort] = useState<SortState>({ field: 'round_count', order: 'desc' })
  const filters = { range, mode, server, campaign }
  const mapFilters = { ...filters, page: mapPage, page_size: pageSize, sort: mapSort.field, order: mapSort.order }
  const contextFilters = { ...filters, page: contextPage, page_size: pageSize, sort: contextSort.field, order: contextSort.order }
  const options = useQuery({ queryKey: ['analysis-options', range, mode], queryFn: () => api.analysisOptions({ range, mode }) })
  const maps = useQuery({ queryKey: ['analysis-maps', mapFilters], queryFn: () => api.analysisMaps(mapFilters), enabled: tab === 'maps', placeholderData: previous => previous })
  const contexts = useQuery({ queryKey: ['analysis-contexts', contextFilters], queryFn: () => api.analysisContexts(contextFilters), enabled: tab === 'contexts', placeholderData: previous => previous })
  const detail = useQuery({ queryKey: ['analysis-map-detail', filters, selectedMap], queryFn: () => api.analysisMapDetail(filters, selectedMap), enabled: selectedMap !== '' })
  const timelineItems = useMemo(() => detail.data?.timeline ?? [], [detail.data?.timeline])
  const recentIncidents = detail.data?.recent_incidents ?? []
  const hasCompleteIncidentData = (detail.data?.summary.complete_incident_rounds ?? 0) > 0

  const timeline = useMemo<EChartsCoreOption>(() => ({
    tooltip: { trigger: 'axis', backgroundColor: 'rgba(39,39,38,.95)', borderWidth: 0, textStyle: { color: '#f5efe7' } },
    legend: { type: 'scroll', top: 12, left: 104, right: 16, data: [zh ? '控制' : 'Controls', zh ? '倒地' : 'Incaps', zh ? '死亡' : 'Deaths', zh ? '惊扰 Witch' : 'Witch startles', zh ? '医疗包' : 'Medkit heals', zh ? '目标互动' : 'Objectives'], textStyle: { color: '#d8d0c7' }, pageTextStyle: { color: '#d8d0c7' } },
    graphic: [{ type: 'text', left: 16, top: 15, silent: true, style: { text: zh ? '事件频率' : 'Event frequency', fill: '#c7beb5', fontSize: 12 } }],
    grid: { left: 16, right: 18, top: 54, bottom: 18, containLabel: true },
    xAxis: { type: 'category', data: timelineItems.map(item => `${item.bucket_seconds / 60}m`), axisLabel: { color: '#c7beb5' } },
    yAxis: { type: 'value', axisLabel: { color: '#c7beb5' }, splitLine: { lineStyle: { color: 'rgba(255,255,255,.1)' } } },
    series: [
      { name: zh ? '控制' : 'Controls', type: 'line', data: timelineItems.map(item => oneDecimal(item.controls_per_100_rounds)), smooth: true },
      { name: zh ? '倒地' : 'Incaps', type: 'line', data: timelineItems.map(item => oneDecimal(item.incaps_per_100_rounds)), smooth: true },
      { name: zh ? '死亡' : 'Deaths', type: 'line', data: timelineItems.map(item => oneDecimal(item.deaths_per_100_rounds)), smooth: true },
      { name: zh ? '惊扰 Witch' : 'Witch startles', type: 'line', data: timelineItems.map(item => oneDecimal(item.witch_startles_per_100_rounds)), smooth: true },
      { name: zh ? '医疗包' : 'Medkit heals', type: 'line', data: timelineItems.map(item => oneDecimal(item.medkit_heals_per_100_rounds)), smooth: true },
      { name: zh ? '目标互动' : 'Objectives', type: 'line', data: timelineItems.map(item => oneDecimal(item.objective_completes_per_100_rounds)), smooth: true },
    ],
  }), [timelineItems, zh])

  const sortable = (field: string) => ({ key: field, sorter: true, sortOrder: mapSort.field === field ? (mapSort.order === 'asc' ? 'ascend' as const : 'descend' as const) : null })
  const mapColumns = [
    { title: zh ? '地图' : 'Map', dataIndex: 'map_name', ...sortable('map_name'), render: (value: string, row: AnalysisMapRow) => <Button type="link" onClick={() => setSelectedMap(row.map_name)}>{value}</Button> },
    { title: zh ? '对局数' : 'Matches', dataIndex: 'eligible_rounds', ...sortable('eligible_rounds') },
    { title: zh ? '通关率' : 'Completion rate', ...sortable('completion_rate'), render: (_: unknown, row: AnalysisMapRow) => mode === 'pve' && row.completed_rounds + row.failed_rounds ? percent(row.completed_rounds / (row.completed_rounds + row.failed_rounds)) : '—' },
    { title: zh ? '平均通关所需尝试' : 'Average attempts to complete', ...sortable('average_completed_attempt'), render: (_: unknown, row: AnalysisMapRow) => mode === 'pve' ? row.average_completed_attempt?.toFixed(2) ?? '—' : '—' },
    { title: zh ? '平均对局时长' : 'Average match duration', ...sortable('average_duration_seconds'), render: (_: unknown, row: AnalysisMapRow) => duration(row.average_duration_seconds) },
    { title: zh ? '每场完整对局倒地' : 'Incaps per complete match', ...sortable('incaps_per_complete_round'), render: (_: unknown, row: AnalysisMapRow) => rate(row.incaps, row.complete_incident_rounds) },
    { title: zh ? '每场完整对局死亡' : 'Deaths per complete match', ...sortable('deaths_per_complete_round'), render: (_: unknown, row: AnalysisMapRow) => rate(row.deaths, row.complete_incident_rounds) },
    { title: zh ? '每场完整对局被控' : 'Controls per complete match', ...sortable('controls_per_complete_round'), render: (_: unknown, row: AnalysisMapRow) => rate(row.controls, row.complete_incident_rounds) },
  ]
  const contextSortable = (field: string) => ({ key: field, sorter: true, sortOrder: contextSort.field === field ? (contextSort.order === 'asc' ? 'ascend' as const : 'descend' as const) : null })
  const contextColumns = [
    { title: zh ? '规则配置' : 'Rules', ...contextSortable('ruleset_name'), render: (_: unknown, row: AnalysisContextRow) => <strong>{row.ruleset_name || (zh ? '默认规则' : 'Default rules')}</strong> },
    { title: zh ? '对局数' : 'Matches', dataIndex: 'round_count', ...contextSortable('round_count') },
    { title: zh ? '通关率' : 'Completion rate', ...contextSortable('completion_rate'), render: (_: unknown, row: AnalysisContextRow) => mode === 'pve' && row.completed_rounds + row.failed_rounds ? percent(row.completed_rounds / (row.completed_rounds + row.failed_rounds)) : '—' },
    { title: zh ? '平均对局时长' : 'Average match duration', ...contextSortable('average_duration_seconds'), render: (_: unknown, row: AnalysisContextRow) => duration(row.average_duration_seconds) },
    { title: zh ? '战局明细完整率' : 'Complete battle-detail rate', ...contextSortable('complete_incident_coverage'), render: (_: unknown, row: AnalysisContextRow) => percent(row.complete_incident_rounds / Math.max(1, row.round_count)) },
    { title: zh ? '游戏参数' : 'Game settings', render: (_: unknown, row: AnalysisContextRow) => zh ? `${difficulty(row.difficulty, true)}难度 · 幸存者 ${row.survivor_limit} · 特感玩家上限 ${row.max_player_zombies} · 普通感染者上限 ${row.common_limit} · Tank 生命 ${row.tank_health} · Witch 生命 ${row.witch_health}` : `${difficulty(row.difficulty, false)} · ${row.survivor_limit} survivors · ${row.max_player_zombies} player SI · ${row.common_limit} common limit · Tank ${row.tank_health} HP · Witch ${row.witch_health} HP` },
  ]

  return <Layout className={styles.layout}><FloatingNav/><Layout.Content className={styles.content}>
    <section className={styles.controlCard}><div className={styles.filterBar}><Segmented value={range} onChange={value => { setRange(String(value)); setMapPage(1); setContextPage(1); setSelectedMap('') }} options={[['30d', zh ? '近 30 天' : '30d'], ['90d', zh ? '近 90 天' : '90d'], ['180d', zh ? '近 180 天' : '180d'], ['all', zh ? '全部' : 'All']].map(([value,label])=>({value,label}))}/><div className={styles.filters}><Select value={mode} onChange={value => { setMode(value); setServer(undefined); setCampaign(undefined); setMapPage(1); setContextPage(1); setSelectedMap('') }} options={[{value:'pve',label:'PvE'},{value:'versus',label:zh?'对抗':'Versus'}]}/><Select allowClear showSearch value={server} onChange={value => { setServer(value); setMapPage(1); setContextPage(1); setSelectedMap('') }} placeholder={zh?'全部服务器':'All servers'} options={(options.data?.servers ?? []).map(value=>({value,label:value}))}/><Select allowClear showSearch value={campaign} onChange={value => { setCampaign(value); setMapPage(1); setContextPage(1); setSelectedMap('') }} placeholder={zh?'全部战役':'All campaigns'} options={(options.data?.campaigns ?? []).map(value=>({value,label:value}))}/></div></div><Tabs className={styles.tabs} activeKey={tab} onChange={setTab} items={[{key:'maps',label:zh?'地图与战役':'Maps & campaigns'},{key:'contexts',label:zh?'规则环境':'Rule contexts'}]}/></section>
    {tab === 'maps' && <>{maps.isLoading ? <State loading/> : maps.data ? <>
      <section className={styles.cards}><Metric label={zh?'有效对局':'Eligible matches'} value={integer.format(maps.data.eligible_rounds)}/><Metric label={zh?'通关率':'Completion rate'} value={mode==='pve'?percent(maps.data.completion_rate):'—'}/><Metric label={zh?'平均通关所需尝试':'Average attempts to complete'} value={mode==='pve'?maps.data.average_completed_attempt?.toFixed(2)??'—':'—'}/><Metric label={zh?'战局明细完整率':'Complete battle-detail rate'} value={percent(maps.data.complete_incident_coverage)}/><Metric label={zh?'统计日期':'Statistics period'} value={`${shortDate(maps.data.earliest_incident_at)} - ${shortDate(maps.data.latest_incident_at)}`}/></section>
      <section className={styles.table}><Table dataSource={maps.data.maps} columns={mapColumns} rowKey="map_name" loading={maps.isFetching} sortDirections={['ascend','descend','ascend']} pagination={{current:mapPage,pageSize,total:maps.data.total,showSizeChanger:false,showTotal:total=>zh?`共 ${integer.format(total)} 张地图`:`${integer.format(total)} maps`}} onChange={(pagination, _filters, sorter, extra) => { if (extra.action === 'sort') { const current = Array.isArray(sorter) ? sorter[0] : sorter; if (current?.order) { setMapSort({field:String(current.columnKey ?? current.field ?? 'map_name'),order:current.order==='ascend'?'asc':'desc'}); setMapPage(1) } } else setMapPage(pagination.current ?? 1) }} scroll={{x:1050}}/></section>
    </> : <State/>}</>}
    {tab === 'contexts' && <>{contexts.isLoading ? <State loading/> : contexts.data ? <><section className={`${styles.cards} ${styles.contextCards}`}><Metric label={zh?'全程规则未变化':'Rules unchanged throughout'} value={`${integer.format(contexts.data.stable_context_rounds)} (${percent(contexts.data.stable_context_rounds/Math.max(1,contexts.data.eligible_rounds))})`}/><Metric label={zh?'中途修改规则':'Rules changed mid-match'} value={`${integer.format(contexts.data.changed_rule_rounds)} (${percent(contexts.data.changed_rule_rounds/Math.max(1,contexts.data.eligible_rounds))})`}/><Metric label={zh?'缺少规则记录':'Missing rule records'} value={`${integer.format(contexts.data.no_context_rounds)} (${percent(contexts.data.no_context_rounds/Math.max(1,contexts.data.eligible_rounds))})`}/></section><section className={styles.table}><Table dataSource={contexts.data.contexts} columns={contextColumns} rowKey="fingerprint" loading={contexts.isFetching} sortDirections={['ascend','descend','ascend']} pagination={{current:contextPage,pageSize,total:contexts.data.total,showSizeChanger:false,showTotal:total=>zh?`共 ${integer.format(total)} 种规则`:`${integer.format(total)} rule contexts`}} onChange={(pagination, _filters, sorter, extra) => { if (extra.action === 'sort') { const current = Array.isArray(sorter) ? sorter[0] : sorter; if (current?.order) { setContextSort({field:String(current.columnKey ?? current.field ?? 'round_count'),order:current.order==='ascend'?'asc':'desc'}); setContextPage(1) } } else setContextPage(pagination.current ?? 1) }} scroll={{x:1100}}/></section></> : <State/>}</>}
  </Layout.Content><Drawer width={720} title={selectedMap} open={selectedMap!==''} onClose={()=>setSelectedMap('')}>
    {detail.isLoading && <div className={styles.detailState}><Spin/></div>}
    {detail.isError && <div className={styles.detailState}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={zh?'当前筛选条件下无法读取该地图详情。':'Map details are unavailable under the current filters.'}><Button onClick={() => void detail.refetch()}>{zh?'重新加载':'Retry'}</Button></Empty></div>}
    {detail.data && <div className={styles.detail}>
      <section className={styles.detailCards}><Metric label={zh?'有效对局':'Eligible matches'} value={String(detail.data.summary.eligible_rounds)}/><Metric label={zh?'战局明细完整的对局':'Matches with complete battle details'} value={String(detail.data.summary.complete_incident_rounds)}/><Metric label={zh?'平均对局时长':'Average match duration'} value={duration(detail.data.summary.average_duration_seconds)}/></section>
      {!hasCompleteIncidentData && (
        <Alert type="info" showIcon title={zh?'暂无完整战局明细':'No complete battle details'} description={zh?'该地图只有基础对局统计，旧版本数据或不完整采集不会生成事件图表。':'Only basic match statistics are available. Legacy or incomplete captures do not produce event charts.'}/>
      )}
      <h3>{zh?'战局事件统计':'Battle events'}</h3>
      {hasCompleteIncidentData ? (
        <div className={styles.composition}>{Object.entries(detail.data.incident_composition ?? {}).map(([key,value])=><span key={key}><small>{incidentLabels[key]?.[zh?0:1] ?? key.replaceAll('_',' ')}</small><strong>{integer.format(value)}</strong></span>)}</div>
      ) : <DetailEmpty zh={zh}/>}
      <h3>{zh?'对局时间线（每 100 场）':'Match timeline (per 100 matches)'}</h3>
      {timelineItems.length ? (
        <EChart className={styles.chart} option={timeline} ariaLabel={zh?'每百场对局事件时间线':'Battle events per 100 matches'}/>
      ) : <DetailEmpty zh={zh}/>}
      <h3>Boss</h3>
      {hasCompleteIncidentData ? (
        <div className={styles.boss}><BossCard title="Tank" data={detail.data.tank} zh={zh}/><BossCard title="Witch" data={detail.data.witch} zh={zh}/></div>
      ) : <DetailEmpty zh={zh}/>}
      <h3>{zh?'最近事件':'Recent events'}</h3>
      <div className={styles.recentIncidents}>{recentIncidents.length ? recentIncidents.map((incident,index)=><div key={`${incident.occurred_at}-${index}`}><time>{new Date(incident.occurred_at*1000).toLocaleTimeString(zh?'zh-CN':'en-US',{hour:'2-digit',minute:'2-digit',second:'2-digit',hour12:false})}</time><span>{incidentText(incident,zh)}</span></div>) : <DetailEmpty zh={zh}/>}</div>
    </div>}
  </Drawer></Layout>
}

function Metric({label,value}:{label:string;value:string}) { return <div><span>{label}</span><strong>{value}</strong></div> }
function State({loading=false}:{loading?:boolean}) { return <section className={styles.state}>{loading?<Spin/>:<Empty/>}</section> }
function DetailEmpty({zh}:{zh:boolean}) { return <Empty className={styles.detailEmpty} image={Empty.PRESENTED_IMAGE_SIMPLE} description={zh?'暂无符合当前采集契约的战局明细':'No battle details match the current capture contract'}/> }
function BossCard({title,data,zh}:{title:string;data:{spawn_count:number;death_count:number;matched_pairs:number;average_lifetime_seconds?:number;maximum_lifetime_seconds?:number;one_shot_deaths?:number;startle_count?:number};zh:boolean}) { return <div><h4>{title}</h4><span>{zh?'出现':'Spawn'} <strong>{data.spawn_count}</strong></span><span>{zh?'死亡':'Death'} <strong>{data.death_count}</strong></span><span>{zh?'完整存活时间记录':'Matched lifetime'} <strong>{data.matched_pairs}</strong></span><span>{zh?'平均 / 最长存活':'Average / maximum'} <strong>{duration(data.average_lifetime_seconds)} / {duration(data.maximum_lifetime_seconds)}</strong></span>{title==='Witch'&&<><span>{zh?'惊扰':'Startles'} <strong>{data.startle_count??0}</strong></span><span>{zh?'秒杀':'One-shot'} <strong>{data.one_shot_deaths??0}</strong></span></>}</div> }

function incidentText(incident: AnalysisIncident, zh: boolean) {
  const actor = incident.actor_name || incident.actor_steam_id || (zh ? '未知玩家' : 'Unknown player')
  const target = incident.target_name || incident.target_steam_id || (zh ? '队友' : 'a teammate')
  if (incident.incident_type === 12) return zh ? `${actor} 惊扰了 Witch` : `${actor} startled the Witch`
  if (incident.incident_type === 13) return zh ? `${actor} 为 ${target} 使用医疗包` : `${actor} used a medkit on ${target}`
  if (incident.incident_type === 14) return zh ? `${actor} 完成目标互动` : `${actor} completed an objective interaction`
  return zh ? '未知事件' : 'Unknown event'
}
