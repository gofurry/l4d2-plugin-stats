import { DeleteOutlined, LoginOutlined, SearchOutlined } from '@ant-design/icons'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Alert, Button, Empty, Input, Layout, Modal, Segmented, Select, Spin, Table, Tabs, Tag, Typography } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, APIError, type PVEEquipment, type PVEInfectedClass, type VersusInfectedClass, type VersusSurvivorClass } from '../api'
import { EChart } from '../components/EChart'
import { FloatingNav } from '../components/FloatingNav'
import { FloatingToolbar } from '../components/FloatingToolbar'
import styles from './PlayerPage.module.scss'

const storageKey = 'l4d2-stats.player.steam-id.v1'
const valid = (value: string) => /^7656119\d{10}$/.test(value)
const sharedSteamID = () => new URLSearchParams(window.location.search).get('steam_id') ?? ''
const number = new Intl.NumberFormat()
const hours = (seconds: number) => `${(seconds / 3600).toFixed(seconds >= 36000 ? 0 : 1)} h`
const duration = (seconds: number) => seconds >= 3600 ? hours(seconds) : `${Math.round(seconds / 60)} min`
const date = (unix?: number) => unix ? new Date(unix * 1000).toLocaleString() : '—'

const infectedNames = ['Smoker', 'Boomer', 'Hunter', 'Spitter', 'Jockey', 'Charger', 'Unknown', 'Tank']
const equipmentNames = [
  '', 'Other Firearm', 'Pistol', 'Dual Pistols', 'Magnum', 'Uzi', 'Silenced SMG', 'MP5',
  'Pump Shotgun', 'Chrome Shotgun', 'Auto Shotgun', 'SPAS', 'M16', 'AK-47', 'SCAR', 'SG552',
  'Hunting Rifle', 'Military Sniper', 'Scout', 'AWP', 'Grenade Launcher', 'M60', 'Chainsaw',
  'Mounted Gun', 'Minigun', 'Baseball Bat', 'Cricket Bat', 'Crowbar', 'Electric Guitar', 'Fire Axe',
  'Frying Pan', 'Golf Club', 'Katana', 'Knife', 'Machete', 'Pitchfork', 'Shovel', 'Tonfa',
  'Molotov', 'Pipe Bomb', 'Vomit Jar',
]

const palette = ['#c66f3b', '#899a67', '#d3a65f', '#9a7564', '#b65d50', '#6f8e90']
const chartBase = {
  animationDuration: 450,
  textStyle: { color: '#5b473c', fontFamily: 'Inter, Segoe UI, Microsoft YaHei, sans-serif' },
  tooltip: { trigger: 'axis', backgroundColor: 'rgba(67,48,38,.94)', borderWidth: 0, textStyle: { color: '#fff7ed' } },
  grid: { left: 42, right: 18, top: 36, bottom: 34, containLabel: true },
}

type Metric = [string, number | string]

function MetricList({ items }: { items: Metric[] }) {
  return <dl className={styles.metricList}>{items.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{typeof value === 'number' ? number.format(value) : value}</dd></div>)}</dl>
}

function Section({ title, children, wide = false }: { title: string; children: React.ReactNode; wide?: boolean }) {
  return <section className={`${styles.dataSection}${wide ? ` ${styles.wide}` : ''}`}><h3>{title}</h3>{children}</section>
}

export function PlayerPage() {
  const { t, i18n } = useTranslation()
  const zh = i18n.language !== 'en'
  const copy = zh ? {
    activity: '活跃趋势', serverShare: '服务器分布', overview: '生涯概览', combat: '战斗', survival: '生存与承伤', teamwork: '团队与补给', progress: '章节成绩', skills: '技巧与互动', classes: '特殊感染者明细', equipment: '武器与装备',
    pveTab: 'PvE', versusSurvivor: '对抗 · 幸存者', versusInfected: '对抗 · 感染者', history: '最近记录', activeRatio: '实际操作占比', firstSeen: '首次出现', activeTime: '实际操作',
    connected: '连接', sessions: '会话', kills: '击杀', damage: '伤害', controlled: '受控次数', controlTime: '受控时长', saves: '解救', actions: '使用/动作', headshots: '爆头击杀', human: '真人', bot: 'Bot', received: '获救次数', supplies: '治疗与补给', encounters: '遭遇与参与', abilityHits: '技能命中', abilityDamage: '技能伤害',
    noActivity: '该时间范围内没有活跃记录', sessionHistory: 'Session', chapterHistory: '章节参与', mode: '模式', side: '阵营', status: '状态', time: '时间', server: '服务器', bossDamage: 'Boss 伤害', infectedDamage: '感染者承伤', friendlyFireDealt: '造成友伤', friendlyFireTaken: '受到友伤', downEdge: '倒地 / 挂边', ledgeRescue: '挂边救援', defib: '电击复活', medkitOthers: '对队友打包', healingOthers: '对队友治疗量', blackWhite: '恢复黑白队友', survived: '存活通关', deadCompletion: '死亡通关', ammo: '补充弹药', objective: '目标互动', tongue: '断舌自救', rocks: '击碎 Tank 石块', witchOneShot: '秒杀 Witch', witchSolo: '单杀 Witch', tankParticipation: 'Tank 击杀参与', witchParticipation: 'Witch 击杀参与', humanSI: '真人特感击杀', botSI: 'Bot 特感击杀', humanTank: '真人 Tank 击杀', botTank: 'Bot Tank 击杀', throwables: '投掷物', spawns: '出生',
  } : {
    activity: 'Activity trend', serverShare: 'Server distribution', overview: 'Career overview', combat: 'Combat', survival: 'Survival and damage', teamwork: 'Teamwork and supplies', progress: 'Chapter results', skills: 'Skills and interactions', classes: 'Special infected detail', equipment: 'Weapons and equipment',
    pveTab: 'PvE', versusSurvivor: 'Versus · Survivor', versusInfected: 'Versus · Infected', history: 'Recent records', activeRatio: 'Active ratio', firstSeen: 'First seen', activeTime: 'Active time',
    connected: 'Connected', sessions: 'Sessions', kills: 'Kills', damage: 'Damage', controlled: 'Controls received', controlTime: 'Control time', saves: 'Saves', actions: 'Actions', headshots: 'Headshot kills', human: 'Human', bot: 'Bot', received: 'Rescues received', supplies: 'Healing and supplies', encounters: 'Encounters and participation', abilityHits: 'Ability hits', abilityDamage: 'Ability damage',
    noActivity: 'No activity in this range', sessionHistory: 'Sessions', chapterHistory: 'Chapters', mode: 'Mode', side: 'Side', status: 'Status', time: 'Time', server: 'Server', bossDamage: 'Boss damage', infectedDamage: 'Damage from infected', friendlyFireDealt: 'Friendly fire dealt', friendlyFireTaken: 'Friendly fire taken', downEdge: 'Incapacitated / ledge', ledgeRescue: 'Ledge rescues', defib: 'Defibrillator revives', medkitOthers: 'Medkits used on teammates', healingOthers: 'Healing teammates', blackWhite: 'Black-and-white restored', survived: 'Completed alive', deadCompletion: 'Completed dead', ammo: 'Ammo pile uses', objective: 'Objective interactions', tongue: 'Self tongue cuts', rocks: 'Tank rocks destroyed', witchOneShot: 'Witch one-shots', witchSolo: 'Witch solo kills', tankParticipation: 'Tank kill participations', witchParticipation: 'Witch kill participations', humanSI: 'Human SI kills', botSI: 'Bot SI kills', humanTank: 'Human Tank kills', botTank: 'Bot Tank kills', throwables: 'Throwables', spawns: 'Spawns',
  }

  const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 300_000 })
	const [steamID, setSteamID] = useState(() => {
		const shared = sharedSteamID()
		return valid(shared) ? shared : localStorage.getItem(storageKey) ?? ''
	})
  const [input, setInput] = useState(steamID)
  const [range, setRange] = useState('all')
  const [server, setServer] = useState('')
  const [gameMode, setGameMode] = useState('')
  const [queryOpen, setQueryOpen] = useState(false)
  useEffect(() => { if (valid(sharedSteamID())) return; void api.steamIdentity().then(identity => { if (identity?.steam_id) { localStorage.setItem(storageKey, identity.steam_id); setSteamID(identity.steam_id); setInput(identity.steam_id) } }).catch(() => undefined) }, [])
  const enabled = valid(steamID)
  const summary = useQuery({ queryKey: ['player-summary', steamID], queryFn: () => api.playerSummary(steamID), enabled })
  const activity = useQuery({ queryKey: ['player-activity', steamID, range, server], queryFn: () => api.playerActivity(steamID, range, server), enabled })
  const pve = useQuery({ queryKey: ['player-pve', steamID, range, server, gameMode], queryFn: () => api.playerPVE(steamID, range, server, gameMode), enabled })
  const versus = useQuery({ queryKey: ['player-versus', steamID, range, server], queryFn: () => api.playerVersus(steamID, range, server), enabled })
  const sessionPage = useInfiniteQuery({ queryKey: ['player-sessions', steamID], queryFn: ({ pageParam }) => api.playerSessions(steamID, pageParam), initialPageParam: '', getNextPageParam: last => last.next_cursor, enabled })
  const chapterPage = useInfiniteQuery({ queryKey: ['player-chapters', steamID], queryFn: ({ pageParam }) => api.playerChapters(steamID, pageParam), initialPageParam: '', getNextPageParam: last => last.next_cursor, enabled })
  const sessions = sessionPage.data?.pages.flatMap(page => page.items) ?? []
  const chapters = chapterPage.data?.pages.flatMap(page => page.items) ?? []
  const search = () => { const value = input.trim(); if (!valid(value)) return; localStorage.setItem(storageKey, value); setSteamID(value); setQueryOpen(false) }
  const clear = () => { localStorage.removeItem(storageKey); setSteamID(''); setInput('') }
  const notFound = summary.error instanceof APIError && summary.error.code === 'player_not_found'

  const activityOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: '#6f5b50' } },
    xAxis: { type: 'category', data: activity.data?.timeline.map(point => new Date(point.day * 86400_000).toLocaleDateString()) ?? [], axisLabel: { color: '#806d62', hideOverlap: true }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
    yAxis: { type: 'value', axisLabel: { color: '#806d62', formatter: '{value}h' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
    series: [
      { name: copy.activeTime, type: 'line', smooth: true, showSymbol: false, data: activity.data?.timeline.map(point => +(point.active_play_seconds / 3600).toFixed(2)) ?? [], lineStyle: { width: 3, color: palette[0] }, areaStyle: { color: 'rgba(198,111,59,.12)' } },
      { name: copy.connected, type: 'line', smooth: true, showSymbol: false, data: activity.data?.timeline.map(point => +(point.connected_seconds / 3600).toFixed(2)) ?? [], lineStyle: { width: 2, color: palette[1] } },
    ],
  }), [activity.data, copy.activeTime, copy.connected])

  const serverOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    tooltip: { trigger: 'item', backgroundColor: 'rgba(67,48,38,.94)', borderWidth: 0, textStyle: { color: '#fff7ed' } },
    legend: { bottom: 0, type: 'scroll', textStyle: { color: '#6f5b50' } },
    series: [{ type: 'pie', radius: ['45%', '70%'], center: ['50%', '44%'], padAngle: 2, itemStyle: { borderRadius: 5 }, data: activity.data?.servers.slice(0, 10).map((item, index) => ({ name: item.server_key, value: +(item.active_play_seconds / 3600).toFixed(2), itemStyle: { color: palette[index % palette.length] } })) ?? [] }],
  }), [activity.data])

  const pveClassOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: '#6f5b50' } },
    xAxis: { type: 'category', data: pve.data?.infected_classes.map(item => infectedNames[item.class_id - 1]) ?? [], axisLabel: { color: '#806d62' }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
    yAxis: { type: 'value', axisLabel: { color: '#806d62' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
    series: [
      { name: copy.kills, type: 'bar', data: pve.data?.infected_classes.map(item => item.kills) ?? [], itemStyle: { color: palette[0], borderRadius: [5, 5, 0, 0] } },
      { name: copy.saves, type: 'bar', data: pve.data?.infected_classes.map(item => item.saves) ?? [], itemStyle: { color: palette[1], borderRadius: [5, 5, 0, 0] } },
    ],
  }), [pve.data, copy.kills, copy.saves])

  const versusClassOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: '#6f5b50' } },
    xAxis: { type: 'category', data: versus.data?.infected_classes.map(item => infectedNames[item.class_id - 1] ?? `#${item.class_id}`) ?? [], axisLabel: { color: '#806d62' }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
    yAxis: { type: 'value', axisLabel: { color: '#806d62' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
    series: [
      { name: copy.damage, type: 'bar', data: versus.data?.infected_classes.map(item => item.damage_to_human_survivors) ?? [], itemStyle: { color: palette[0] } },
      { name: t('incaps'), type: 'bar', data: versus.data?.infected_classes.map(item => item.human_survivor_incaps) ?? [], itemStyle: { color: palette[2] } },
      { name: t('controls'), type: 'bar', data: versus.data?.infected_classes.map(item => item.human_survivor_controls) ?? [], itemStyle: { color: palette[1] } },
    ],
  }), [versus.data, copy.damage, t])

  const activeRatio = summary.data?.connected_seconds ? `${Math.round(summary.data.active_play_seconds / summary.data.connected_seconds * 100)}%` : '—'
  const equipmentColumns = [
    { title: copy.equipment, dataIndex: 'equipment_id', key: 'equipment', render: (id: number) => equipmentNames[id] ?? `#${id}` },
    { title: copy.actions, dataIndex: 'actions', key: 'actions', sorter: (a: PVEEquipment, b: PVEEquipment) => a.actions - b.actions },
    { title: t('commonKills'), dataIndex: 'common_kills', key: 'common', sorter: (a: PVEEquipment, b: PVEEquipment) => a.common_kills - b.common_kills },
    { title: t('specialKills'), dataIndex: 'special_kills', key: 'special', sorter: (a: PVEEquipment, b: PVEEquipment) => a.special_kills - b.special_kills },
    { title: copy.headshots, dataIndex: 'headshot_kills', key: 'headshots', sorter: (a: PVEEquipment, b: PVEEquipment) => a.headshot_kills - b.headshot_kills },
    { title: 'Tank', key: 'tank', render: (_: unknown, item: PVEEquipment) => `${number.format(item.tank_kills)} / ${number.format(item.damage_to_tank)}` },
    { title: 'Witch', key: 'witch', render: (_: unknown, item: PVEEquipment) => `${number.format(item.witch_kills)} / ${number.format(item.damage_to_witch)}` },
  ]
  const pveClassColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: PVEInfectedClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: copy.kills, dataIndex: 'kills', key: 'kills' },
    { title: copy.damage, dataIndex: 'damage', key: 'damage' },
    { title: copy.controlled, dataIndex: 'controls_received', key: 'controls' },
    { title: copy.controlTime, dataIndex: 'controlled_seconds', key: 'seconds', render: (value: number) => duration(value) },
    { title: copy.saves, dataIndex: 'saves', key: 'saves' },
  ]
  const survivorClassColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: VersusSurvivorClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: `${copy.human} ${copy.kills}`, dataIndex: 'human_controller_kills', key: 'humanKills' },
    { title: `${copy.bot} ${copy.kills}`, dataIndex: 'bot_controller_kills', key: 'botKills' },
    { title: `${copy.human} ${copy.damage}`, dataIndex: 'damage_to_human_controllers', key: 'humanDamage' },
    { title: `${copy.bot} ${copy.damage}`, dataIndex: 'damage_to_bot_controllers', key: 'botDamage' },
  ]
  const infectedClassColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: VersusInfectedClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: copy.spawns, dataIndex: 'spawns', key: 'spawns' },
    { title: `${copy.human} ${copy.damage}`, dataIndex: 'damage_to_human_survivors', key: 'humanDamage' },
    { title: `${copy.bot} ${copy.damage}`, dataIndex: 'damage_to_bot_survivors', key: 'botDamage' },
    { title: `${copy.human} ${t('incaps')}`, dataIndex: 'human_survivor_incaps', key: 'humanIncaps' },
    { title: `${copy.bot} ${t('incaps')}`, dataIndex: 'bot_survivor_incaps', key: 'botIncaps' },
    { title: `${copy.human} ${t('kills')}`, dataIndex: 'human_survivor_kills', key: 'humanKills' },
    { title: `${copy.bot} ${t('kills')}`, dataIndex: 'bot_survivor_kills', key: 'botKills' },
    { title: `${copy.human} ${t('controls')}`, dataIndex: 'human_survivor_controls', key: 'humanControls' },
    { title: `${copy.bot} ${t('controls')}`, dataIndex: 'bot_survivor_controls', key: 'botControls' },
    { title: copy.abilityHits, key: 'abilityHits', render: (_: unknown, item: VersusInfectedClass) => number.format(item.human_survivor_ability_hits + item.bot_survivor_ability_hits) },
    { title: copy.abilityDamage, key: 'abilityDamage', render: (_: unknown, item: VersusInfectedClass) => number.format(item.human_survivor_ability_damage + item.bot_survivor_ability_damage) },
  ]
  const toolbarItems = [
    { key: 'query', label: t('query'), icon: <SearchOutlined />, onClick: () => { setInput(steamID); setQueryOpen(true) } },
    ...(site.data?.steam_openid_enabled ? [{ key: 'steam', label: t('steamLogin'), icon: <LoginOutlined />, onClick: () => { window.location.href = '/api/v1/steam/login' } }] : []),
    { key: 'clear', label: t('clearIdentity'), icon: <DeleteOutlined />, onClick: clear, disabled: !steamID, danger: true },
  ]

  return <Layout className={styles.layout}><FloatingNav /><FloatingToolbar ariaLabel={zh ? '个人查询工具' : 'Player query tools'} items={toolbarItems} /><Layout.Content className={styles.content}>
    {!steamID && <div className={styles.empty}><Empty description={t('enterSteamID')} /></div>}
    {notFound && <Alert type="info" showIcon title={t('playerNotFound')} />}
    {summary.isError && !notFound && <Alert type="warning" showIcon title={t('statsUnavailable')} />}
    {summary.isLoading && <div className={styles.loading}><Spin /></div>}

    {summary.data && <>
      <section className={styles.identityPanel}>
        <div className={styles.identity}><div><Typography.Title level={2}>{summary.data.last_name || summary.data.steam_id}</Typography.Title><Tag>{summary.data.steam_id}</Tag></div><div className={styles.identityFilters}><Segmented value={range} onChange={value => setRange(String(value))} options={[['all', t('allTime')], ['30d', t('days30')], ['90d', t('days90')], ['365d', t('days365')]].map(([value, label]) => ({ value, label }))} /><Select value={server} onChange={setServer} options={[{ value: '', label: zh ? '全部服务器' : 'All servers' }, ...(activity.data?.servers ?? []).map(item => ({ value: item.server_key, label: item.server_key }))]} /><Select value={gameMode} onChange={setGameMode} options={[{ value: '', label: zh ? '合作 + 写实' : 'Co-op + Realism' }, { value: 'coop', label: zh ? '合作' : 'Co-op' }, { value: 'realism', label: zh ? '写实' : 'Realism' }]} /></div></div>
        <MetricList items={[[copy.sessions, summary.data.session_count], [copy.connected, hours(summary.data.connected_seconds)], [copy.activeTime, hours(summary.data.active_play_seconds)], [copy.activeRatio, activeRatio], [copy.firstSeen, date(summary.data.first_seen_at)], [t('lastSeen'), date(summary.data.last_seen_at)]]} />
      </section>

      <Tabs className={styles.tabs} items={[
        { key: 'overview', label: zh ? '概览' : 'Overview', children: <div className={styles.chartGrid}>
          <Section title={copy.activity}>{activity.isLoading ? <Spin /> : activity.data?.timeline.length ? <EChart ariaLabel={copy.activity} className={styles.chart} option={activityOption} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={copy.noActivity} />}</Section>
          <Section title={copy.serverShare}>{activity.isLoading ? <Spin /> : <EChart ariaLabel={copy.serverShare} className={styles.chart} option={serverOption} />}</Section>
        </div> },
        { key: 'pve', label: copy.pveTab, children: pve.isLoading ? <Spin /> : pve.data && <div className={styles.sectionGrid}>
          <Section title={copy.combat}><MetricList items={[[t('commonKills'), pve.data.common_kills], [t('specialKills'), pve.data.special_kills], ['Tank', pve.data.tank_kills], ['Witch', pve.data.witch_kills], [t('damageSpecial'), pve.data.damage_to_special], [zh ? '对 Tank 伤害' : 'Damage to Tank', pve.data.damage_to_tank], [zh ? '对 Witch 伤害' : 'Damage to Witch', pve.data.damage_to_witch]]} /></Section>
          <Section title={copy.survival}><MetricList items={[[t('deaths'), pve.data.deaths], [t('incaps'), pve.data.incapacitations], [copy.infectedDamage, pve.data.damage_taken_infected], [copy.friendlyFireDealt, pve.data.friendly_fire], [copy.friendlyFireTaken, pve.data.friendly_fire_taken], [zh ? '倒地时长' : 'Incapacitated time', duration(pve.data.incapacitated_seconds)], [zh ? '挂边时长' : 'Ledge time', duration(pve.data.ledge_hanging_seconds)]]} /></Section>
          <Section title={copy.teamwork}><MetricList items={[[t('revives'), pve.data.incap_revives], [copy.ledgeRescue, pve.data.ledge_rescues], [copy.defib, pve.data.defib_revives], [copy.received, pve.data.rescues_received], [copy.medkitOthers, pve.data.medkits_used_on_others], [copy.healingOthers, pve.data.medkit_healing_others], [copy.blackWhite, pve.data.black_white_teammates_restored]]} /></Section>
          <Section title={copy.supplies}><MetricList items={[[zh ? '对自己打包' : 'Medkits used on self', pve.data.medkits_used_self], [zh ? '自我治疗量' : 'Self healing', pve.data.medkit_healing_self], [zh ? '止痛药' : 'Pain pills', pve.data.pills_used], [zh ? '肾上腺素' : 'Adrenaline', pve.data.adrenaline_used], [zh ? '获得临时生命' : 'Temporary health received', pve.data.temporary_health_received], [zh ? '燃烧弹药包' : 'Incendiary packs', pve.data.incendiary_packs_deployed], [zh ? '高爆弹药包' : 'Explosive packs', pve.data.explosive_packs_deployed], [copy.ammo, pve.data.ammo_pile_uses]]} /></Section>
          <Section title={copy.progress}><MetricList items={[[t('chapters'), pve.data.chapter_participations], [copy.survived, pve.data.chapter_completions_alive], [copy.deadCompletion, pve.data.chapter_completions_dead], [t('campaigns'), pve.data.campaign_completions], [copy.ammo, pve.data.ammo_pile_uses], [copy.objective, pve.data.objective_interactions]]} /></Section>
          <Section title={copy.skills}><MetricList items={[[copy.tongue, pve.data.melee_tongue_self_cuts], [copy.rocks, pve.data.tank_rocks_destroyed], [copy.witchOneShot, pve.data.witch_oneshots], [copy.witchSolo, pve.data.witch_solo_kills], [zh ? 'Tank 遭遇' : 'Tank encounters', pve.data.tank_encounters], [copy.tankParticipation, pve.data.tank_kill_participations], [zh ? 'Witch 遭遇' : 'Witch encounters', pve.data.witch_encounters], [copy.witchParticipation, pve.data.witch_kill_participations], [copy.objective, pve.data.objective_interactions]]} /></Section>
        </div> },
        { key: 'pve-details', label: zh ? 'PvE 明细' : 'PvE details', children: pve.isLoading ? <Spin /> : pve.data && <div className={styles.sectionGrid}>
          <Section title={copy.classes} wide><EChart ariaLabel={copy.classes} className={styles.chart} option={pveClassOption} /><Table<PVEInfectedClass> className={styles.embeddedTable} columns={pveClassColumns} dataSource={pve.data.infected_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 720 }} /></Section>
          <Section title={copy.equipment} wide><Table<PVEEquipment> className={styles.embeddedTable} columns={equipmentColumns} dataSource={pve.data.equipment} rowKey="equipment_id" pagination={{ pageSize: 10, hideOnSinglePage: true }} size="small" scroll={{ x: 760 }} /></Section>
        </div> },
        { key: 'versus-survivor', label: copy.versusSurvivor, children: versus.isLoading ? <Spin /> : versus.data && <div className={styles.sectionGrid}>
          <Section title={copy.combat}><MetricList items={[[t('commonKills'), versus.data.survivor_common_kills], [copy.humanSI, versus.data.human_special_kills], [copy.botSI, versus.data.bot_special_kills], [copy.humanTank, versus.data.human_tank_kills], [copy.botTank, versus.data.bot_tank_kills], [copy.damage, versus.data.survivor_damage]]} /></Section>
          <Section title={copy.survival}><MetricList items={[[t('deaths'), versus.data.survivor_deaths], [t('incaps'), versus.data.survivor_incapacitations], [copy.infectedDamage, versus.data.survivor_damage_taken], [copy.friendlyFireDealt, versus.data.survivor_friendly_fire], [copy.friendlyFireTaken, versus.data.survivor_friendly_fire_taken], [copy.received, versus.data.survivor_rescues_received]]} /></Section>
          <Section title={copy.teamwork}><MetricList items={[[t('revives'), versus.data.survivor_incap_revives], [copy.ledgeRescue, versus.data.survivor_ledge_rescues], [copy.defib, versus.data.survivor_defib_revives], [copy.medkitOthers, versus.data.survivor_medkits_others], [copy.healingOthers, versus.data.survivor_healing_others], [zh ? '自用医疗包 / 治疗量' : 'Self medkits / healing', `${number.format(versus.data.survivor_medkits_self)} / ${number.format(versus.data.survivor_healing_self)}`], [zh ? '止痛药 / 肾上腺素' : 'Pills / adrenaline', `${number.format(versus.data.survivor_pills)} / ${number.format(versus.data.survivor_adrenaline)}`], [zh ? '获得临时生命' : 'Temporary health received', versus.data.survivor_temporary_health]]} /></Section>
          <Section title={copy.supplies}><MetricList items={[[zh ? '燃烧瓶' : 'Molotovs', versus.data.molotovs_thrown], [zh ? '土制炸弹' : 'Pipe bombs', versus.data.pipe_bombs_thrown], [zh ? '胆汁罐' : 'Vomit jars', versus.data.vomit_jars_thrown], [zh ? '燃烧弹药包' : 'Incendiary packs', versus.data.survivor_incendiary_packs], [zh ? '高爆弹药包' : 'Explosive packs', versus.data.survivor_explosive_packs], [zh ? 'Witch 击杀 / 伤害' : 'Witch kills / damage', `${number.format(versus.data.survivor_witch_kills)} / ${number.format(versus.data.survivor_witch_damage)}`]]} /></Section>
          <Section title={copy.skills}><MetricList items={[[copy.tongue, versus.data.survivor_tongue_self_cuts], [copy.rocks, versus.data.survivor_tank_rocks_destroyed], [copy.witchOneShot, versus.data.survivor_witch_oneshots], [copy.witchSolo, versus.data.survivor_witch_solo_kills]]} /></Section>
        </div> },
        { key: 'versus-survivor-details', label: zh ? '对抗幸存者明细' : 'Versus survivor details', children: versus.isLoading ? <Spin /> : versus.data && <div className={styles.sectionGrid}>
          <Section title={copy.classes} wide><Table<VersusSurvivorClass> className={styles.embeddedTable} columns={survivorClassColumns} dataSource={versus.data.survivor_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 720 }} /></Section>
        </div> },
        { key: 'versus-infected', label: copy.versusInfected, children: versus.isLoading ? <Spin /> : versus.data && <div className={styles.sectionGrid}>
          <Section title={copy.overview}><MetricList items={[[t('infectedSpawns'), versus.data.infected_spawns], [`${copy.human} ${copy.damage}`, versus.data.damage_to_human_survivors], [`${copy.bot} ${copy.damage}`, versus.data.damage_to_bot_survivors], [`${copy.human} ${t('incaps')}`, versus.data.human_survivor_incaps], [`${copy.bot} ${t('incaps')}`, versus.data.bot_survivor_incaps], [`${copy.human} ${t('kills')}`, versus.data.human_survivor_kills], [`${copy.bot} ${t('kills')}`, versus.data.bot_survivor_kills], [t('controls'), versus.data.human_survivor_controls], [t('controlTime'), duration(versus.data.human_survivor_control_seconds)]]} /></Section>
        </div> },
        { key: 'versus-infected-details', label: zh ? '对抗感染者明细' : 'Versus infected details', children: versus.isLoading ? <Spin /> : versus.data && <div className={styles.sectionGrid}>
          <Section title={copy.classes} wide><EChart ariaLabel={copy.classes} className={styles.largeChart} option={versusClassOption} /></Section>
          <Section title={copy.classes} wide><Table<VersusInfectedClass> className={styles.embeddedTable} columns={infectedClassColumns} dataSource={versus.data.infected_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 1200 }} /></Section>
        </div> },
        { key: 'history', label: copy.history, children: <div className={styles.historyGrid}>
          <Section title={copy.sessionHistory}><div className={styles.recordList}>{sessions.map((item, index) => <div key={`${item.started_at}-${index}`}><strong>{item.player_name || item.server_key}</strong><span>{item.server_key}</span><span>{date(item.started_at)}</span><span>{duration(item.active_play_seconds)}</span><Tag>{item.status}</Tag></div>)}</div>{sessionPage.hasNextPage && <Button loading={sessionPage.isFetchingNextPage} onClick={() => void sessionPage.fetchNextPage()}>{t('loadMore')}</Button>}</Section>
          <Section title={copy.chapterHistory}><div className={styles.recordList}>{chapters.map((item, index) => <div key={`${item.started_at}-${index}`}><strong>{item.map_name}</strong><span>{item.game_mode} · {item.side || '—'}</span><span>{date(item.started_at)}</span><span>{duration(item.active_play_seconds)}</span><Tag>{item.status}</Tag></div>)}</div>{chapterPage.hasNextPage && <Button loading={chapterPage.isFetchingNextPage} onClick={() => void chapterPage.fetchNextPage()}>{t('loadMore')}</Button>}</Section>
        </div> },
      ]} />
    </>}
  </Layout.Content><Modal title={zh ? '查询玩家' : 'Query player'} open={queryOpen} onCancel={() => setQueryOpen(false)} onOk={search} okButtonProps={{ disabled: !valid(input.trim()) }} okText={t('query')}>
    <Input autoFocus value={input} maxLength={17} placeholder="SteamID64" onChange={event => setInput(event.target.value)} onPressEnter={search} />
  </Modal></Layout>
}
