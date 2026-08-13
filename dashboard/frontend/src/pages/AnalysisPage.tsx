import { useQuery } from '@tanstack/react-query'
import { Button, Drawer, Empty, Layout, Segmented, Select, Spin, Table, Tabs, Tag } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type AnalysisContextRow, type AnalysisMapRow } from '../api'
import { EChart } from '../components/EChart'
import { FloatingNav } from '../components/FloatingNav'
import styles from './AnalysisPage.module.scss'

const integer = new Intl.NumberFormat()
const percent = (value?: number) => value === undefined ? '—' : `${(value * 100).toFixed(1)}%`
const duration = (value?: number) => value === undefined ? '—' : value < 60 ? `${value.toFixed(0)}s` : `${(value / 60).toFixed(1)}m`
const rate = (value: number, sample: number) => sample > 0 ? (value / sample).toFixed(2) : '—'

export function AnalysisPage() {
  const { i18n } = useTranslation()
  const zh = i18n.language !== 'en'
  const [range, setRange] = useState('90d')
  const [mode, setMode] = useState('pve')
  const [tab, setTab] = useState('maps')
  const [selectedMap, setSelectedMap] = useState('')
  const filters = { range, mode }
  const maps = useQuery({ queryKey: ['analysis-maps', filters], queryFn: () => api.analysisMaps(filters), enabled: tab === 'maps' })
  const contexts = useQuery({ queryKey: ['analysis-contexts', filters], queryFn: () => api.analysisContexts(filters), enabled: tab === 'contexts' })
  const detail = useQuery({ queryKey: ['analysis-map-detail', filters, selectedMap], queryFn: () => api.analysisMapDetail(filters, selectedMap), enabled: selectedMap !== '' })

  const timeline = useMemo<EChartsCoreOption>(() => ({
    tooltip: { trigger: 'axis', backgroundColor: 'rgba(39,39,38,.95)', borderWidth: 0, textStyle: { color: '#f5efe7' } },
    legend: { data: [zh ? '控制' : 'Controls', zh ? '倒地' : 'Incaps', zh ? '死亡' : 'Deaths'], textStyle: { color: '#d8d0c7' } },
    grid: { left: 12, right: 18, top: 44, bottom: 18, containLabel: true },
    xAxis: { type: 'category', data: detail.data?.timeline.map(item => `${item.bucket_seconds / 60}m`) ?? [], axisLabel: { color: '#c7beb5' } },
    yAxis: { type: 'value', name: zh ? '每 100 个到达该时段的 Round' : 'Per 100 rounds reaching bucket', nameTextStyle: { color: '#c7beb5' }, axisLabel: { color: '#c7beb5' }, splitLine: { lineStyle: { color: 'rgba(255,255,255,.1)' } } },
    series: [
      { name: zh ? '控制' : 'Controls', type: 'line', data: detail.data?.timeline.map(item => item.controls_per_100_rounds) ?? [], smooth: true },
      { name: zh ? '倒地' : 'Incaps', type: 'line', data: detail.data?.timeline.map(item => item.incaps_per_100_rounds) ?? [], smooth: true },
      { name: zh ? '死亡' : 'Deaths', type: 'line', data: detail.data?.timeline.map(item => item.deaths_per_100_rounds) ?? [], smooth: true },
    ],
  }), [detail.data, zh])

  const mapColumns = [
    { title: zh ? '地图' : 'Map', dataIndex: 'map_name', render: (value: string, row: AnalysisMapRow) => <Button type="link" onClick={() => setSelectedMap(row.map_name)}>{value}</Button> },
    { title: zh ? '样本 Round' : 'Rounds', dataIndex: 'eligible_rounds' },
    { title: zh ? '完成率' : 'Completion', render: (_: unknown, row: AnalysisMapRow) => row.completed_rounds + row.failed_rounds ? percent(row.completed_rounds / (row.completed_rounds + row.failed_rounds)) : '—', sorter: (a: AnalysisMapRow, b: AnalysisMapRow) => (a.completed_rounds / Math.max(1, a.completed_rounds + a.failed_rounds)) - (b.completed_rounds / Math.max(1, b.completed_rounds + b.failed_rounds)), defaultSortOrder: mode === 'pve' ? 'ascend' as const : undefined },
    { title: zh ? '完成时平均尝试' : 'Avg completed attempt', render: (_: unknown, row: AnalysisMapRow) => row.average_completed_attempt?.toFixed(2) ?? '—' },
    { title: zh ? '平均时长' : 'Avg duration', render: (_: unknown, row: AnalysisMapRow) => duration(row.average_duration_seconds) },
    { title: zh ? '倒地 / 完整 Round' : 'Incaps / complete round', render: (_: unknown, row: AnalysisMapRow) => rate(row.incaps, row.complete_incident_rounds) },
    { title: zh ? '死亡 / 完整 Round' : 'Deaths / complete round', render: (_: unknown, row: AnalysisMapRow) => rate(row.deaths, row.complete_incident_rounds) },
    { title: zh ? '控制 / 完整 Round' : 'Controls / complete round', render: (_: unknown, row: AnalysisMapRow) => rate(row.controls, row.complete_incident_rounds) },
  ]
  const contextColumns = [
    { title: zh ? '规则环境' : 'Context', render: (_: unknown, row: AnalysisContextRow) => <div><strong>{row.ruleset_name || row.difficulty || (zh ? '未命名规则' : 'Unnamed rules')}</strong><br/><Tag>{row.fingerprint}</Tag></div> },
    { title: zh ? 'Round 数' : 'Rounds', dataIndex: 'round_count' },
    { title: zh ? '完成率' : 'Completion', render: (_: unknown, row: AnalysisContextRow) => row.completed_rounds + row.failed_rounds ? percent(row.completed_rounds / (row.completed_rounds + row.failed_rounds)) : '—' },
    { title: zh ? '平均时长' : 'Average duration', render: (_: unknown, row: AnalysisContextRow) => duration(row.average_duration_seconds) },
    { title: zh ? 'Incident 覆盖' : 'Incident coverage', render: (_: unknown, row: AnalysisContextRow) => percent(row.complete_incident_rounds / Math.max(1, row.round_count)) },
    { title: zh ? '关键参数' : 'Tracked values', render: (_: unknown, row: AnalysisContextRow) => `${row.difficulty || '—'} · ${row.survivor_limit}P · SI ${row.max_player_zombies} · CI ${row.common_limit} · Tank ${row.tank_health} · Witch ${row.witch_health}` },
  ]

  return <Layout className={styles.layout}><FloatingNav/><Layout.Content className={styles.content}>
    <header className={styles.header}><div><h1>{zh ? '战局分析' : 'Analysis'}</h1><p>{zh ? '基于已验证的 Round Context 与低频 Incident，按地图和规则环境观察战局。' : 'Inspect verified round context and low-frequency incidents by map and rule environment.'}</p></div><div className={styles.filters}><Segmented value={range} onChange={value => setRange(String(value))} options={[['30d', zh ? '近 30 天' : '30d'], ['90d', zh ? '近 90 天' : '90d'], ['180d', zh ? '近 180 天' : '180d'], ['all', zh ? '全部可用' : 'All available']].map(([value,label])=>({value,label}))}/><Select value={mode} onChange={setMode} options={[{value:'pve',label:'PvE'},{value:'versus',label:zh?'对抗':'Versus'}]}/></div></header>
    <Tabs className={styles.tabs} activeKey={tab} onChange={setTab} items={[{key:'maps',label:zh?'地图与战役':'Maps & campaigns'},{key:'contexts',label:zh?'规则环境':'Rule contexts'}]}/>
    {tab === 'maps' && <>{maps.isLoading ? <State loading/> : maps.data ? <>
      <section className={styles.cards}><Metric label={zh?'有效 Round':'Eligible rounds'} value={integer.format(maps.data.eligible_rounds)}/><Metric label={zh?'完成率':'Completion rate'} value={mode==='pve'?percent(maps.data.completion_rate):'—'}/><Metric label={zh?'完成时平均尝试':'Avg completed attempt'} value={maps.data.average_completed_attempt?.toFixed(2)??'—'}/><Metric label={zh?'完整 Incident 覆盖':'Complete incident coverage'} value={percent(maps.data.complete_incident_coverage)}/></section>
      <p className={styles.availability}>{zh?'Incident 可用窗口':'Incident availability'}: {maps.data.earliest_incident_at ? new Date(maps.data.earliest_incident_at*1000).toLocaleDateString() : '—'} — {maps.data.latest_incident_at ? new Date(maps.data.latest_incident_at*1000).toLocaleDateString() : '—'} · v{maps.data.incident_version}</p>
      <section className={styles.table}><Table dataSource={maps.data.maps} columns={mapColumns} rowKey="map_name" pagination={false} scroll={{x:1050}}/></section>
    </> : <State/>}</>}
    {tab === 'contexts' && <>{contexts.isLoading ? <State loading/> : contexts.data ? <><section className={styles.cards}><Metric label={zh?'稳定规则 Round':'Stable-context rounds'} value={percent(contexts.data.stable_context_rounds/Math.max(1,contexts.data.eligible_rounds))}/><Metric label={zh?'中途改规则 Round':'Changed-rule rounds'} value={percent(contexts.data.changed_rule_rounds/Math.max(1,contexts.data.eligible_rounds))}/><Metric label={zh?'无 Context 历史 Round':'No-context historical rounds'} value={percent(contexts.data.no_context_rounds/Math.max(1,contexts.data.eligible_rounds))}/></section><section className={styles.table}><Table dataSource={contexts.data.contexts} columns={contextColumns} rowKey="fingerprint" pagination={false} scroll={{x:1000}}/></section></> : <State/>}</>}
  </Layout.Content><Drawer width={720} title={selectedMap} open={selectedMap!==''} onClose={()=>setSelectedMap('')}>
    {detail.isLoading ? <Spin/> : detail.data && <div className={styles.detail}><section className={styles.detailCards}><Metric label={zh?'有效 Round':'Eligible rounds'} value={String(detail.data.summary.eligible_rounds)}/><Metric label={zh?'完整 Incident Round':'Complete incident rounds'} value={String(detail.data.summary.complete_incident_rounds)}/><Metric label={zh?'平均时长':'Average duration'} value={duration(detail.data.summary.average_duration_seconds)}/></section><h3>{zh?'事件构成':'Incident composition'}</h3><div className={styles.composition}>{Object.entries(detail.data.incident_composition).map(([key,value])=><span key={key}><small>{key.replaceAll('_',' ')}</small><strong>{integer.format(value)}</strong></span>)}</div><h3>{zh?'标准化 Round 时间线':'Normalized round timeline'}</h3><EChart className={styles.chart} option={timeline} ariaLabel={zh?'标准化事件时间线':'Normalized incident timeline'}/><h3>Boss</h3><div className={styles.boss}><BossCard title="Tank" data={detail.data.tank}/><BossCard title="Witch" data={detail.data.witch}/></div></div>}
  </Drawer></Layout>
}

function Metric({label,value}:{label:string;value:string}) { return <div><span>{label}</span><strong>{value}</strong></div> }
function State({loading=false}:{loading?:boolean}) { return <section className={styles.state}>{loading?<Spin/>:<Empty/>}</section> }
function BossCard({title,data}:{title:string;data:{spawn_count:number;death_count:number;matched_pairs:number;average_lifetime_seconds?:number;maximum_lifetime_seconds?:number;one_shot_deaths?:number}}) { return <div><h4>{title}</h4><span>Spawn <strong>{data.spawn_count}</strong></span><span>Death <strong>{data.death_count}</strong></span><span>Matched <strong>{data.matched_pairs}</strong></span><span>Avg / max <strong>{duration(data.average_lifetime_seconds)} / {duration(data.maximum_lifetime_seconds)}</strong></span>{title==='Witch'&&<span>One-shot <strong>{data.one_shot_deaths??0}</strong></span>}</div> }
