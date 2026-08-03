import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

const resources = {
  'zh-CN': { translation: {
    subtitle: '求生之路 2 服务器数据中心', language: 'English',
    serverStatus: '主服务器', online: '在线', offline: '离线', stale: '最后已知状态', notConfigured: '尚未配置主服务器',
    map: '当前地图', players: '玩家', bots: 'Bot', latency: '延迟', join: '加入服务器', lastSuccess: '最后成功查询',
    overview: '全服概览', totalPlayers: '累计玩家', active7d: '7 日活跃', playTime: '有效游玩时长', completedRuns: '完成对局',
    pve: 'PvE 战役', versus: '对抗模式', commonKills: '普通感染者击杀', specialKills: '特殊感染者击杀', bossKills: 'Tank / Witch 击杀', rescues: '团队救援',
    matches: '完成比赛', halves: '完成半场', humanSIKills: '真人特感 / Tank 击杀', controls: '控制真人幸存者',
    pveRuns: 'PvE', versusRuns: '对抗', hours: '小时', noData: '统计数据库中暂时没有可展示的数据。', statsUnavailable: '统计数据暂时不可用，服务器状态仍会独立更新。',
    serverUnavailable: '服务器状态暂时不可用。', siteUnavailable: '站点配置暂时不可用，正在使用默认标题。', retry: '重试',
  } },
  en: { translation: {
    subtitle: 'Left 4 Dead 2 server data center', language: '中文',
    serverStatus: 'Primary server', online: 'Online', offline: 'Offline', stale: 'Last known status', notConfigured: 'No primary server configured',
    map: 'Current map', players: 'Players', bots: 'Bots', latency: 'Latency', join: 'Join server', lastSuccess: 'Last successful query',
    overview: 'Network overview', totalPlayers: 'Total players', active7d: 'Active in 7 days', playTime: 'Active play time', completedRuns: 'Completed runs',
    pve: 'PvE campaigns', versus: 'Versus', commonKills: 'Common infected kills', specialKills: 'Special infected kills', bossKills: 'Tank / Witch kills', rescues: 'Team rescues',
    matches: 'Completed matches', halves: 'Completed halves', humanSIKills: 'Human SI / Tank kills', controls: 'Human survivor controls',
    pveRuns: 'PvE', versusRuns: 'Versus', hours: 'hours', noData: 'There is no dashboard data to display yet.', statsUnavailable: 'Statistics are temporarily unavailable. Server status continues to update independently.',
    serverUnavailable: 'Server status is temporarily unavailable.', siteUnavailable: 'Site settings are unavailable; the default title is being used.', retry: 'Retry',
  } },
}

const stored = localStorage.getItem('l4d2-stats.locale')
const detected = navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en'

void i18n.use(initReactI18next).init({ resources, lng: stored ?? detected, fallbackLng: 'zh-CN', interpolation: { escapeValue: false } })

export function toggleLanguage(): void {
  const next = i18n.language === 'zh-CN' ? 'en' : 'zh-CN'
  localStorage.setItem('l4d2-stats.locale', next)
  void i18n.changeLanguage(next)
  document.documentElement.lang = next
}

export default i18n
