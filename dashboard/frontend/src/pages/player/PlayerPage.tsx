import { DeleteOutlined, FilterOutlined, IdcardOutlined, LoginOutlined, SearchOutlined } from '@ant-design/icons'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Alert, Empty, Input, Layout, Modal, Segmented, Select, Spin, Tabs, Tag, Typography, message } from 'antd'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, APIError } from '../../api'
import { isAchievementArtworkKey } from '../../assets/achievements/achievementArtwork'
import { FloatingNav } from '../../components/FloatingNav'
import { FloatingToolbar } from '../../components/FloatingToolbar'
import { AchievementBadge } from '../../components/AchievementBadge'
import { PlayerPreviewModal } from '../../components/PlayerPreviewModal'
import { PlayerAchievements } from './PlayerAchievements'
import { PlayerActivity } from './PlayerActivity'
import { PlayerHistory } from './PlayerHistory'
import { PlayerAnalysis } from './PlayerAnalysis'
import { PlayerPVE } from './PlayerPVE'
import { PlayerRelationships } from './PlayerRelationships'
import { MetricList } from './PlayerShared'
import { PlayerVersus } from './PlayerVersus'
import { date, hours, playerStorageKey, sharedSteamID, validSteamID } from './playerFormat'
import styles from './PlayerPage.module.scss'

export function PlayerPage() {
  const { t, i18n } = useTranslation()
  const zh = i18n.language !== 'en'
  const copy = zh ? {
    activity: '活跃趋势', serverShare: '服务器分布', overview: '生涯概览', combat: '战斗', survival: '生存与承伤', teamwork: '团队与补给', progress: '章节成绩', skills: '技巧与互动', classes: '特殊感染者明细', equipment: '武器与装备',
    pveTab: 'PvE', versusSurvivor: '对抗 · 幸存者', versusInfected: '对抗 · 感染者', history: '最近记录', activeRatio: '实际操作占比', firstSeen: '首次出现', activeTime: '实际操作',
    connected: '连接', sessions: '会话', kills: '击杀', damage: '伤害', controlled: '受控次数', controlTime: '受控时长', saves: '解救', actions: '使用/动作', headshots: '爆头击杀', received: '获救次数', supplies: '治疗与补给', encounters: '遭遇与参与', abilityHits: '技能命中', abilityDamage: '技能伤害', humanKills: '击杀真人', botKills: '击杀 Bot', humanDamage: '对真人造成伤害', botDamage: '对 Bot 造成伤害', humanIncaps: '击倒真人幸存者', botIncaps: '击倒 Bot 幸存者', humanSurvivorKills: '击杀真人幸存者', botSurvivorKills: '击杀 Bot 幸存者', humanControls: '控制真人幸存者', botControls: '控制 Bot 幸存者',
    noActivity: '该时间范围内没有活跃记录', sessionHistory: '会话连接', chapterHistory: '章节参与', mode: '模式', side: '阵营', status: '状态', time: '时间', server: '服务器', bossDamage: 'Boss 伤害', infectedDamage: '感染者承伤', friendlyFireDealt: '造成友伤', friendlyFireTaken: '受到友伤', downEdge: '倒地 / 挂边', ledgeRescue: '挂边救援', defib: '电击复活', medkitOthers: '对队友打包', healingOthers: '对队友治疗量', blackWhite: '恢复黑白队友', survived: '存活通关', deadCompletion: '死亡通关', ammo: '补充弹药', objective: '机关互动', tongue: '断舌自救', rocks: '击碎 Tank 石块', witchOneShot: '秒杀 Witch', witchSolo: '单杀 Witch', tankParticipation: 'Tank 击杀参与', witchParticipation: 'Witch 击杀参与', humanSI: '击杀真人特感', botSI: '击杀 Bot 特感', humanTank: '击杀真人 Tank', botTank: '击杀 Bot Tank', throwables: '投掷物', spawns: '复活',
  } : {
    activity: 'Activity trend', serverShare: 'Server distribution', overview: 'Career overview', combat: 'Combat', survival: 'Survival and damage', teamwork: 'Teamwork and supplies', progress: 'Chapter results', skills: 'Skills and interactions', classes: 'Special infected detail', equipment: 'Weapons and equipment',
    pveTab: 'PvE', versusSurvivor: 'Versus · Survivor', versusInfected: 'Versus · Infected', history: 'Recent records', activeRatio: 'Active ratio', firstSeen: 'First seen', activeTime: 'Active time',
    connected: 'Connected', sessions: 'Sessions', kills: 'Kills', damage: 'Damage', controlled: 'Controls received', controlTime: 'Control time', saves: 'Saves', actions: 'Actions', headshots: 'Headshot kills', received: 'Rescues received', supplies: 'Healing and supplies', encounters: 'Encounters and participation', abilityHits: 'Ability hits', abilityDamage: 'Ability damage', humanKills: 'Human kills', botKills: 'Bot kills', humanDamage: 'Damage to humans', botDamage: 'Damage to bots', humanIncaps: 'Human survivor incaps', botIncaps: 'Bot survivor incaps', humanSurvivorKills: 'Human survivor kills', botSurvivorKills: 'Bot survivor kills', humanControls: 'Human survivor controls', botControls: 'Bot survivor controls',
    noActivity: 'No activity in this range', sessionHistory: 'Sessions', chapterHistory: 'Chapters', mode: 'Mode', side: 'Side', status: 'Status', time: 'Time', server: 'Server', bossDamage: 'Boss damage', infectedDamage: 'Damage from infected', friendlyFireDealt: 'Friendly fire dealt', friendlyFireTaken: 'Friendly fire taken', downEdge: 'Incapacitated / ledge', ledgeRescue: 'Ledge rescues', defib: 'Defibrillator revives', medkitOthers: 'Medkits used on teammates', healingOthers: 'Healing teammates', blackWhite: 'Black-and-white restored', survived: 'Completed alive', deadCompletion: 'Completed dead', ammo: 'Ammo pile uses', objective: 'Objective interactions', tongue: 'Self tongue cuts', rocks: 'Tank rocks destroyed', witchOneShot: 'Witch one-shots', witchSolo: 'Witch solo kills', tankParticipation: 'Tank kill participations', witchParticipation: 'Witch kill participations', humanSI: 'Human SI kills', botSI: 'Bot SI kills', humanTank: 'Human Tank kills', botTank: 'Bot Tank kills', throwables: 'Throwables', spawns: 'Respawns',
  }

  const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 300_000 })
  const [steamID, setSteamID] = useState(() => {
    const shared = sharedSteamID()
    return validSteamID(shared) ? shared : localStorage.getItem(playerStorageKey) ?? ''
  })
  const [savedSteamID, setSavedSteamID] = useState(() => localStorage.getItem(playerStorageKey) ?? '')
  const [input, setInput] = useState(steamID)
  const [range, setRange] = useState('all')
  const [server, setServer] = useState('')
  const [gameMode, setGameMode] = useState('')
  const [queryOpen, setQueryOpen] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [activeTab, setActiveTab] = useState('overview')
  const [relationshipMode, setRelationshipMode] = useState('all')
  const [relationshipFilterOpen, setRelationshipFilterOpen] = useState(false)
  const [analysisView, setAnalysisView] = useState('pve')
  const [authenticatedSteamID, setAuthenticatedSteamID] = useState('')
  const achievementToastKey = useRef('')
  useEffect(() => { void api.steamIdentity().then(identity => { if (identity?.steam_id) { setAuthenticatedSteamID(identity.steam_id); localStorage.setItem(playerStorageKey, identity.steam_id); setSavedSteamID(identity.steam_id); setSteamID(identity.steam_id); setInput(identity.steam_id) } }).catch(() => undefined) }, [])
  const enabled = validSteamID(steamID)
  const summary = useQuery({ queryKey: ['player-summary', steamID], queryFn: () => api.playerSummary(steamID), enabled })
  const activity = useQuery({ queryKey: ['player-activity', steamID, range, server], queryFn: () => api.playerActivity(steamID, range, server), enabled })
  const pve = useQuery({ queryKey: ['player-pve', steamID, range, server, gameMode], queryFn: () => api.playerPVE(steamID, range, server, gameMode), enabled })
  const versus = useQuery({ queryKey: ['player-versus', steamID, range, server], queryFn: () => api.playerVersus(steamID, range, server), enabled })
  const analysis = useQuery({ queryKey: ['player-analysis', steamID, range, server, analysisView], queryFn: () => api.playerAnalysis(steamID, range, server, analysisView), enabled })
  const achievements = useQuery({ queryKey: ['player-achievements', steamID], queryFn: () => api.playerAchievements(steamID), enabled })
  const sessionPage = useInfiniteQuery({ queryKey: ['player-sessions', steamID], queryFn: ({ pageParam }) => api.playerSessions(steamID, pageParam), initialPageParam: '', getNextPageParam: last => last.next_cursor, enabled })
  const chapterPage = useInfiniteQuery({ queryKey: ['player-chapters', steamID], queryFn: ({ pageParam }) => api.playerChapters(steamID, pageParam), initialPageParam: '', getNextPageParam: last => last.next_cursor, enabled })
  const sessions = sessionPage.data?.pages.flatMap(page => page.items) ?? []
  const chapters = chapterPage.data?.pages.flatMap(page => page.items) ?? []
  const search = () => { const value = input.trim(); if (!validSteamID(value)) return; localStorage.setItem(playerStorageKey, value); setSavedSteamID(value); setSteamID(value); setQueryOpen(false) }
  const clear = () => { localStorage.removeItem(playerStorageKey); setSavedSteamID(''); setSteamID(''); setInput(''); setPreviewOpen(false) }
  const notFound = summary.error instanceof APIError && summary.error.code === 'player_not_found'
  const activeRatio = summary.data?.connected_seconds ? `${Math.round(summary.data.active_play_seconds / summary.data.connected_seconds * 100)}%` : '—'
  useEffect(() => {
    const data = achievements.data
    if (!data || authenticatedSteamID !== steamID) return
    const key = `${steamID}:${data.unseen_live?.map(item => item.achievement_key).join(',') ?? ''}:${data.unseen_backfill_count ?? 0}`
    if (key === achievementToastKey.current || (!data.unseen_live?.length && !data.unseen_backfill_count)) return
    achievementToastKey.current = key
    data.unseen_live?.forEach(item => void message.success(`🏆 ${zh ? '解锁新成就' : 'Achievement unlocked'}：${item.title}`))
    if (data.unseen_backfill_count) void message.success(`🏆 ${zh ? `已确认 ${data.unseen_backfill_count} 项历史成就` : `${data.unseen_backfill_count} historical achievements confirmed`}`)
  }, [achievements.data, authenticatedSteamID, steamID, zh])
  const toolbarItems = [
    { key: 'query', label: t('query'), icon: <SearchOutlined />, onClick: () => { setInput(steamID); setQueryOpen(true) } },
    ...(activeTab === 'relationships' && enabled ? [{ key: 'relationship-filter', label: zh ? '筛选玩家关系' : 'Filter player relationships', icon: <FilterOutlined />, active: relationshipMode !== 'all', onClick: () => setRelationshipFilterOpen(true) }] : []),
    ...(validSteamID(savedSteamID) ? [{ key: 'preview', label: t('previewPlayerCard'), icon: <IdcardOutlined />, onClick: () => setPreviewOpen(true) }] : []),
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
        <div className={styles.identity}><div className={styles.identityPrimary}><div className={styles.identityNameLine}><Typography.Title level={2}>{summary.data.last_name || summary.data.steam_id}</Typography.Title>{achievements.data?.overview.badges.map(item => <AchievementBadge key={item.achievement_key} artworkKey={isAchievementArtworkKey(item.artwork_key) ? item.artwork_key : undefined} tier={item.tier} size={28} label={item.title} />)}</div><div className={styles.identityMeta}><Tag>{summary.data.steam_id}</Tag>{achievements.data && <button type="button" onClick={() => setActiveTab('achievements')}>{zh ? '成就' : 'Achievements'} {achievements.data.overview.unlocked}/{achievements.data.overview.total} · {zh ? '彩蛋' : 'Easter eggs'} {achievements.data.overview.easter_eggs}</button>}</div></div><div className={styles.identityFilters}><Segmented value={range} onChange={value => setRange(String(value))} options={[['all', t('allTime')], ['30d', t('days30')], ['90d', t('days90')], ['365d', t('days365')]].map(([value, label]) => ({ value, label }))} /><Select value={server} onChange={setServer} options={[{ value: '', label: zh ? '全部服务器' : 'All servers' }, ...(activity.data?.servers ?? []).map(item => ({ value: item.server_key, label: item.server_key }))]} /><Select value={gameMode} onChange={setGameMode} options={[{ value: '', label: zh ? '合作 + 写实' : 'Co-op + Realism' }, { value: 'coop', label: zh ? '合作' : 'Co-op' }, { value: 'realism', label: zh ? '写实' : 'Realism' }]} /></div></div>
        <MetricList items={[[copy.sessions, summary.data.session_count], [copy.connected, hours(summary.data.connected_seconds)], [copy.activeTime, hours(summary.data.active_play_seconds)], [copy.activeRatio, activeRatio], [copy.firstSeen, date(summary.data.first_seen_at)], [t('lastSeen'), date(summary.data.last_seen_at)]]} />
      </section>
      <Tabs className={styles.tabs} activeKey={activeTab} onChange={setActiveTab} items={[
        { key: 'overview', label: zh ? '概览' : 'Overview', children: <PlayerActivity data={activity.data} loading={activity.isLoading} copy={copy} /> },
        { key: 'achievements', label: zh ? '成就' : 'Achievements', children: <PlayerAchievements key={steamID} steamID={steamID} data={achievements.data} loading={achievements.isLoading} self={authenticatedSteamID === steamID} zh={zh} /> },
        { key: 'analysis', label: zh ? '分析' : 'Analysis', children: <PlayerAnalysis data={analysis.data} loading={analysis.isLoading} view={analysisView} onView={setAnalysisView} zh={zh} /> },
        { key: 'pve', label: copy.pveTab, children: <PlayerPVE data={pve.data} loading={pve.isLoading} copy={copy} zh={zh} /> },
        { key: 'pve-details', label: zh ? 'PvE 明细' : 'PvE details', children: <PlayerPVE data={pve.data} loading={pve.isLoading} copy={copy} zh={zh} details /> },
        { key: 'versus-survivor', label: copy.versusSurvivor, children: <PlayerVersus data={versus.data} loading={versus.isLoading} view="survivor" copy={copy} zh={zh} /> },
        { key: 'versus-survivor-details', label: zh ? '对抗幸存者明细' : 'Versus survivor details', children: <PlayerVersus data={versus.data} loading={versus.isLoading} view="survivor-details" copy={copy} zh={zh} /> },
        { key: 'versus-infected', label: copy.versusInfected, children: <PlayerVersus data={versus.data} loading={versus.isLoading} view="infected" copy={copy} zh={zh} /> },
        { key: 'versus-infected-details', label: zh ? '对抗感染者明细' : 'Versus infected details', children: <PlayerVersus data={versus.data} loading={versus.isLoading} view="infected-details" copy={copy} zh={zh} /> },
        { key: 'relationships', label: zh ? '玩家关系' : 'Player relationships', children: <PlayerRelationships key={`${range}:${server}:${relationshipMode}`} steamID={steamID} range={range} server={server} mode={relationshipMode} enabled={enabled} zh={zh} /> },
        { key: 'history', label: copy.history, children: <PlayerHistory sessions={sessions} chapters={chapters} sessionPage={sessionPage} chapterPage={chapterPage} copy={copy} /> },
      ]} />
    </>}
  </Layout.Content><PlayerPreviewModal open={previewOpen} steamID={savedSteamID} playerName={savedSteamID === steamID ? summary.data?.last_name : undefined} onClose={() => setPreviewOpen(false)} /><Modal title={zh ? '筛选玩家关系' : 'Filter player relationships'} open={relationshipFilterOpen} onCancel={() => setRelationshipFilterOpen(false)} onOk={() => setRelationshipFilterOpen(false)} okText={zh ? '完成' : 'Done'} cancelButtonProps={{ style: { display: 'none' } }}>
    <Segmented block value={relationshipMode} onChange={value => setRelationshipMode(String(value))} options={[{ value: 'all', label: zh ? 'PvE + 对抗' : 'PvE + Versus' }, { value: 'pve', label: 'PvE' }, { value: 'versus', label: zh ? '对抗' : 'Versus' }]} />
  </Modal><Modal title={zh ? '查询玩家' : 'Query player'} open={queryOpen} onCancel={() => setQueryOpen(false)} onOk={search} okButtonProps={{ disabled: !validSteamID(input.trim()) }} okText={t('query')}>
    <Input autoFocus value={input} maxLength={17} placeholder="SteamID64" onChange={event => setInput(event.target.value)} onPressEnter={search} />
  </Modal></Layout>
}
