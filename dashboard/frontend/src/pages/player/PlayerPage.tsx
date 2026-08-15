import { DeleteOutlined, FilterOutlined, IdcardOutlined, LoginOutlined, SearchOutlined } from '@ant-design/icons'
import { useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Empty, Input, Layout, Modal, Segmented, Select, Spin, Tabs, Tag, Typography, message } from 'antd'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { api, APIError, type PlayerProfileSection } from '../../api'
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
import { MetricList, PlayerTabPanel } from './PlayerShared'
import { PlayerVersus } from './PlayerVersus'
import { PlayerVisibilitySettings } from './PlayerVisibilitySettings'
import { playerProfileSections } from './playerVisibility'
import { date, hours, playerStorageKey, preferredPlayerSteamID, validSteamID } from './playerFormat'
import styles from './PlayerPage.module.scss'

export function PlayerPage() {
  const queryClient = useQueryClient()
  const location = useLocation()
  const navigate = useNavigate()
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
  const routeSteamID = new URLSearchParams(location.search).get('steam_id') ?? ''
  const [savedSteamID, setSavedSteamID] = useState(() => localStorage.getItem(playerStorageKey) ?? '')
  const [selectedSteamID, setSelectedSteamID] = useState(savedSteamID)
  const [manualSelection, setManualSelection] = useState(false)
  const [authenticatedSteamID, setAuthenticatedSteamID] = useState('')
  const [badgeEditAuthorized, setBadgeEditAuthorized] = useState(false)
  const steamID = validSteamID(routeSteamID) ? routeSteamID : manualSelection ? selectedSteamID : preferredPlayerSteamID('', authenticatedSteamID, savedSteamID)
  const [input, setInput] = useState(() => preferredPlayerSteamID(routeSteamID, '', savedSteamID))
  const [range, setRange] = useState('all')
  const [server, setServer] = useState('')
  const [gameMode, setGameMode] = useState('')
  const [statsFilterOpen, setStatsFilterOpen] = useState(false)
  const [draftServer, setDraftServer] = useState('')
  const [draftGameMode, setDraftGameMode] = useState('')
  const [queryOpen, setQueryOpen] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [activeTab, setActiveTab] = useState(() => new URLSearchParams(window.location.search).get('tab') ?? 'overview')
  const [relationshipMode, setRelationshipMode] = useState('all')
  const [relationshipFilterOpen, setRelationshipFilterOpen] = useState(false)
  const [analysisView, setAnalysisView] = useState('pve')
  const achievementToastKey = useRef('')
  useEffect(() => { void api.steamIdentity().then(identity => { if (identity?.steam_id) { setAuthenticatedSteamID(identity.steam_id); setBadgeEditAuthorized(identity.badge_edit_authorized) } else { setAuthenticatedSteamID(''); setBadgeEditAuthorized(false) } }).catch(() => undefined) }, [])
  const enabled = validSteamID(steamID)
  const profile = useQuery({ queryKey: ['player-profile', steamID], queryFn: () => api.playerProfile(steamID), enabled })
  const self = profile.data?.self === true
  const canView = (section: PlayerProfileSection) => self || profile.data?.visible_sections.includes(section) === true
  const availableTabKeys = profile.data ? (self ? [...playerProfileSections, 'settings'] : profile.data.visible_sections) : []
  const currentTab = availableTabKeys.includes(activeTab as PlayerProfileSection | 'settings') ? activeTab : availableTabKeys[0] ?? 'overview'
  const summary = useQuery({ queryKey: ['player-summary', steamID], queryFn: () => api.playerSummary(steamID), enabled: enabled && !!profile.data && canView('overview') })
  const activityServers = useQuery({ queryKey: ['player-activity', steamID, 'all', ''], queryFn: () => api.playerActivity(steamID, 'all'), enabled: enabled && !!profile.data && canView('overview') })
  const activity = useQuery({ queryKey: ['player-activity', steamID, range, server], queryFn: () => api.playerActivity(steamID, range, server), enabled: enabled && !!profile.data && canView('overview') })
  const pve = useQuery({ queryKey: ['player-pve', steamID, range, server, gameMode], queryFn: () => api.playerPVE(steamID, range, server, gameMode), enabled: enabled && !!profile.data && canView('pve') && currentTab === 'pve' })
  const pveDetails = useQuery({ queryKey: ['player-pve-details', steamID, range, server, gameMode], queryFn: () => api.playerPVEDetails(steamID, range, server, gameMode), enabled: enabled && !!profile.data && canView('pve-details') && currentTab === 'pve-details' })
  const versusSurvivor = useQuery({ queryKey: ['player-versus-survivor', steamID, range, server], queryFn: () => api.playerVersusSurvivor(steamID, range, server), enabled: enabled && !!profile.data && canView('versus-survivor') && currentTab === 'versus-survivor' })
  const versusSurvivorDetails = useQuery({ queryKey: ['player-versus-survivor-details', steamID, range, server], queryFn: () => api.playerVersusSurvivorDetails(steamID, range, server), enabled: enabled && !!profile.data && canView('versus-survivor-details') && currentTab === 'versus-survivor-details' })
  const versusInfected = useQuery({ queryKey: ['player-versus-infected', steamID, range, server], queryFn: () => api.playerVersusInfected(steamID, range, server), enabled: enabled && !!profile.data && canView('versus-infected') && currentTab === 'versus-infected' })
  const versusInfectedDetails = useQuery({ queryKey: ['player-versus-infected-details', steamID, range, server], queryFn: () => api.playerVersusInfectedDetails(steamID, range, server), enabled: enabled && !!profile.data && canView('versus-infected-details') && currentTab === 'versus-infected-details' })
  const analysis = useQuery({ queryKey: ['player-analysis', steamID, range, server, analysisView], queryFn: () => api.playerAnalysis(steamID, range, server, analysisView), enabled: enabled && !!profile.data && canView('analysis') && currentTab === 'analysis' })
  const achievements = useQuery({ queryKey: ['player-achievements', steamID], queryFn: () => api.playerAchievements(steamID), enabled: enabled && !!profile.data && canView('achievements') })
  const sessionPage = useInfiniteQuery({ queryKey: ['player-sessions', steamID], queryFn: ({ pageParam }) => api.playerSessions(steamID, pageParam), initialPageParam: '', getNextPageParam: last => last.next_cursor, enabled: enabled && !!profile.data && canView('history') && currentTab === 'history' })
  const chapterPage = useInfiniteQuery({ queryKey: ['player-chapters', steamID], queryFn: ({ pageParam }) => api.playerChapters(steamID, pageParam), initialPageParam: '', getNextPageParam: last => last.next_cursor, enabled: enabled && !!profile.data && canView('history') && currentTab === 'history' })
  const sessions = sessionPage.data?.pages.flatMap(page => page.items) ?? []
  const chapters = chapterPage.data?.pages.flatMap(page => page.items) ?? []
  const search = () => { const value = input.trim(); if (!validSteamID(value)) return; localStorage.setItem(playerStorageKey, value); setSavedSteamID(value); setSelectedSteamID(value); setManualSelection(true); setQueryOpen(false) }
  const clear = () => {
    localStorage.removeItem(playerStorageKey)
    setSavedSteamID('')
    setSelectedSteamID('')
    setManualSelection(true)
    setInput('')
    setPreviewOpen(false)
    const query = new URLSearchParams(location.search)
    query.delete('steam_id')
    navigate({ pathname: location.pathname, search: query.toString() ? `?${query}` : '' }, { replace: true })
  }
  const requireBadgeAuthentication = () => {
    if (!site.data?.steam_openid_enabled) return void message.info(zh ? '站点尚未启用 Steam 登录' : 'Steam login is not enabled')
    const query = new URLSearchParams({ purpose: 'badge_edit', return_to: '/player?tab=achievements' })
    window.location.href = `/api/v1/steam/login?${query}`
  }
  const notFound = profile.error instanceof APIError && profile.error.code === 'player_not_found'
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
  const openStatsFilter = () => {
    setDraftServer(server)
    setDraftGameMode(gameMode)
    setStatsFilterOpen(true)
  }
  const applyStatsFilter = () => {
    setServer(draftServer)
    setGameMode(draftGameMode)
    setStatsFilterOpen(false)
  }
  const toolbarItems = [
    { key: 'query', label: t('query'), icon: <SearchOutlined />, onClick: () => { setInput(steamID); setQueryOpen(true) } },
    ...(enabled ? [{ key: 'stats-filter', label: zh ? '筛选统计范围' : 'Filter statistics', icon: <FilterOutlined />, active: server !== '' || gameMode !== '', onClick: openStatsFilter }] : []),
    ...(currentTab === 'relationships' && enabled ? [{ key: 'relationship-filter', label: zh ? '筛选玩家关系' : 'Filter player relationships', icon: <FilterOutlined />, active: relationshipMode !== 'all', onClick: () => setRelationshipFilterOpen(true) }] : []),
    ...(validSteamID(savedSteamID) ? [{ key: 'preview', label: t('previewPlayerCard'), icon: <IdcardOutlined />, onClick: () => setPreviewOpen(true) }] : []),
    ...(site.data?.steam_openid_enabled ? [{ key: 'steam', label: t('steamLogin'), icon: <LoginOutlined />, onClick: () => { window.location.href = '/api/v1/steam/login' } }] : []),
    { key: 'clear', label: t('clearIdentity'), icon: <DeleteOutlined />, onClick: clear, disabled: !steamID, danger: true },
  ]
  const selectTab = (tab: string) => {
    setActiveTab(tab)
    const query = new URLSearchParams(location.search)
    query.set('tab', tab)
    navigate({ pathname: location.pathname, search: `?${query}` }, { replace: true })
  }
  const panel = (key: string, children: ReactNode, error = false) => <PlayerTabPanel resetKey={`${steamID}:${key}`} error={error} zh={zh}>{children}</PlayerTabPanel>
  const allTabItems = [
    { key: 'overview', label: zh ? '概览' : 'Overview', children: panel('overview', <PlayerActivity data={activity.data} loading={activity.isLoading} copy={copy} />, summary.isError || activity.isError) },
    { key: 'achievements', label: zh ? '成就' : 'Achievements', children: panel('achievements', <PlayerAchievements key={steamID} steamID={steamID} data={achievements.data} loading={achievements.isLoading} self={self} canEdit={self && badgeEditAuthorized} onRequireAuth={requireBadgeAuthentication} zh={zh} />, achievements.isError) },
    { key: 'analysis', label: zh ? '分析' : 'Analysis', children: panel('analysis', <PlayerAnalysis data={analysis.data} loading={analysis.isLoading} view={analysisView} onView={setAnalysisView} zh={zh} />, analysis.isError) },
    { key: 'pve', label: copy.pveTab, children: panel('pve', <PlayerPVE data={pve.data} loading={pve.isLoading} copy={copy} zh={zh} />, pve.isError) },
    { key: 'pve-details', label: zh ? 'PvE 明细' : 'PvE details', children: panel('pve-details', <PlayerPVE data={pveDetails.data} loading={pveDetails.isLoading} copy={copy} zh={zh} details />, pveDetails.isError) },
    { key: 'versus-survivor', label: copy.versusSurvivor, children: panel('versus-survivor', <PlayerVersus data={versusSurvivor.data} loading={versusSurvivor.isLoading} view="survivor" copy={copy} zh={zh} />, versusSurvivor.isError) },
    { key: 'versus-survivor-details', label: zh ? '对抗幸存者明细' : 'Versus survivor details', children: panel('versus-survivor-details', <PlayerVersus data={versusSurvivorDetails.data} loading={versusSurvivorDetails.isLoading} view="survivor-details" copy={copy} zh={zh} />, versusSurvivorDetails.isError) },
    { key: 'versus-infected', label: copy.versusInfected, children: panel('versus-infected', <PlayerVersus data={versusInfected.data} loading={versusInfected.isLoading} view="infected" copy={copy} zh={zh} />, versusInfected.isError) },
    { key: 'versus-infected-details', label: zh ? '对抗感染者明细' : 'Versus infected details', children: panel('versus-infected-details', <PlayerVersus data={versusInfectedDetails.data} loading={versusInfectedDetails.isLoading} view="infected-details" copy={copy} zh={zh} />, versusInfectedDetails.isError) },
    { key: 'relationships', label: zh ? '玩家关系' : 'Player relationships', children: panel('relationships', <PlayerRelationships key={`${range}:${server}:${relationshipMode}`} steamID={steamID} range={range} server={server} mode={relationshipMode} enabled={enabled && canView('relationships')} zh={zh} />) },
    { key: 'history', label: copy.history, children: panel('history', <PlayerHistory sessions={sessions} chapters={chapters} sessionPage={sessionPage} chapterPage={chapterPage} copy={copy} />, sessionPage.isError || chapterPage.isError) },
    { key: 'settings', label: zh ? '设置' : 'Settings', children: panel('settings', <PlayerVisibilitySettings key={steamID} value={profile.data?.visible_sections ?? []} onSaved={sections => queryClient.setQueryData(['player-profile', steamID], current => current && typeof current === 'object' ? { ...current, visible_sections: sections } : current)} zh={zh} />) },
  ]
  const tabItems = profile.data ? allTabItems.filter(item => item.key === 'settings' ? self : canView(item.key as PlayerProfileSection)) : []

  return <Layout className={styles.layout}><FloatingNav /><FloatingToolbar ariaLabel={zh ? '个人查询工具' : 'Player query tools'} items={toolbarItems} /><Layout.Content className={styles.content}>
    {!steamID && <div className={styles.empty}><Empty description={t('enterSteamID')} /></div>}
    {notFound && <Alert type="info" showIcon title={t('playerNotFound')} />}
    {profile.isError && !notFound && <Alert type="warning" showIcon title={t('statsUnavailable')} />}
    {profile.isLoading && <div className={styles.loading}><Spin /></div>}
    {profile.data && <>
      <section className={styles.identityPanel}>
        <div className={styles.identity}><div className={styles.identityPrimary}><div className={styles.identityNameLine}><Typography.Title level={2}>{profile.data.player_name || profile.data.steam_id}</Typography.Title>{achievements.data?.overview.badges.map(item => <AchievementBadge key={item.achievement_key} artworkKey={isAchievementArtworkKey(item.artwork_key) ? item.artwork_key : undefined} tier={item.tier} size={32} label={item.title} />)}</div><div className={styles.identityMeta}><Tag>{profile.data.steam_id}</Tag>{achievements.data && <button type="button" onClick={() => selectTab('achievements')}>{zh ? '成就' : 'Achievements'} {achievements.data.overview.unlocked}/{achievements.data.overview.total} · {zh ? '彩蛋' : 'Easter eggs'} {achievements.data.overview.easter_eggs}</button>}</div></div>{tabItems.length > 0 && <div className={styles.identityRange}><Segmented value={range} onChange={value => setRange(String(value))} options={[['all', t('allTime')], ['30d', t('days30')], ['90d', t('days90')], ['365d', t('days365')]].map(([value, label]) => ({ value, label }))} /></div>}</div>
        {summary.data && <MetricList items={[[copy.sessions, summary.data.session_count], [copy.connected, hours(summary.data.connected_seconds)], [copy.activeTime, hours(summary.data.active_play_seconds)], [copy.activeRatio, activeRatio], [copy.firstSeen, date(summary.data.first_seen_at)], [t('lastSeen'), date(summary.data.last_seen_at)]]} />}
      </section>
      {!self && tabItems.length === 0 && <Alert className={styles.privateProfile} type="info" showIcon title={zh ? '该玩家未公开个人中心内容' : 'This player has not shared any profile sections'} />}
      {tabItems.length > 0 && <Tabs className={styles.tabs} activeKey={currentTab} onChange={selectTab} items={tabItems} />}
    </>}
  </Layout.Content><PlayerPreviewModal open={previewOpen} steamID={savedSteamID} playerName={savedSteamID === steamID ? profile.data?.player_name : undefined} onClose={() => setPreviewOpen(false)} /><Modal title={zh ? '筛选统计范围' : 'Filter statistics'} open={statsFilterOpen} onCancel={() => setStatsFilterOpen(false)} onOk={applyStatsFilter} okText={zh ? '应用' : 'Apply'} cancelText={zh ? '取消' : 'Cancel'} destroyOnHidden>
    <div className={styles.statsFilterFields}>
      <label><span>{zh ? '服务器' : 'Server'}</span><Select value={draftServer} onChange={setDraftServer} options={[{ value: '', label: zh ? '全部服务器' : 'All servers' }, ...(activityServers.data?.servers ?? []).map(item => ({ value: item.server_key, label: item.server_key }))]} /></label>
      <label><span>{zh ? 'PvE 模式' : 'PvE mode'}</span><Select value={draftGameMode} onChange={setDraftGameMode} options={[{ value: '', label: zh ? '合作 + 写实' : 'Co-op + Realism' }, { value: 'coop', label: zh ? '合作' : 'Co-op' }, { value: 'realism', label: zh ? '写实' : 'Realism' }]} /></label>
    </div>
  </Modal><Modal title={zh ? '筛选玩家关系' : 'Filter player relationships'} open={relationshipFilterOpen} onCancel={() => setRelationshipFilterOpen(false)} onOk={() => setRelationshipFilterOpen(false)} okText={zh ? '完成' : 'Done'} cancelButtonProps={{ style: { display: 'none' } }}>
    <Segmented block value={relationshipMode} onChange={value => setRelationshipMode(String(value))} options={[{ value: 'all', label: zh ? 'PvE + 对抗' : 'PvE + Versus' }, { value: 'pve', label: 'PvE' }, { value: 'versus', label: zh ? '对抗' : 'Versus' }]} />
  </Modal><Modal title={zh ? '查询玩家' : 'Query player'} open={queryOpen} onCancel={() => setQueryOpen(false)} onOk={search} okButtonProps={{ disabled: !validSteamID(input.trim()) }} okText={t('query')}>
    <Input autoFocus value={input} maxLength={17} placeholder="SteamID64" onChange={event => setInput(event.target.value)} onPressEnter={search} />
  </Modal></Layout>
}
