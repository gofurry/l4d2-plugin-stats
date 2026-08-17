import {
  DashboardOutlined,
  DatabaseOutlined,
  AreaChartOutlined,
  GithubOutlined,
  GlobalOutlined,
  HomeOutlined,
  LogoutOutlined,
  NotificationOutlined,
  SafetyOutlined,
  SettingOutlined,
  TeamOutlined,
  TrophyOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'

type NavMode = 'public' | 'admin' | 'auth'

interface FloatingNavProps {
  mode?: NavMode
  loggingOut?: boolean
  onLogout?: () => void
  monitorEnabled?: boolean
}

interface NavItem {
  path: string
  label: string
  icon: ReactNode
  external?: boolean
}

export function FloatingNav({ mode = 'public', loggingOut = false, onLogout, monitorEnabled = false }: FloatingNavProps) {
  const { t } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const items: NavItem[] = mode === 'admin'
    ? [
        { path: '/', label: t('home'), icon: <HomeOutlined /> },
        { path: '/admin/site', label: t('navSiteSettings'), icon: <SettingOutlined /> },
        { path: '/admin/servers', label: t('navServerManagement'), icon: <UnorderedListOutlined /> },
        { path: '/admin/ingame', label: t('navIngame'), icon: <GlobalOutlined /> },
        { path: '/admin/announcements', label: t('navAnnouncementManagement'), icon: <NotificationOutlined /> },
        { path: '/admin/security', label: t('navSecurity'), icon: <SafetyOutlined /> },
        { path: '/admin/data', label: t('navDataMaintenance'), icon: <DatabaseOutlined /> },
        ...(monitorEnabled ? [{ path: '/api/v1/admin/monitor', label: t('runtimeMonitor'), icon: <DashboardOutlined />, external: true }] : []),
      ]
    : mode === 'auth'
      ? [{ path: '/', label: t('home'), icon: <HomeOutlined /> }]
      : [
          { path: '/', label: t('home'), icon: <HomeOutlined /> },
          { path: '/player', label: t('playerCenter'), icon: <TeamOutlined /> },
          { path: '/rankings', label: t('rankings'), icon: <TrophyOutlined /> },
          { path: '/analysis', label: t('analysis'), icon: <AreaChartOutlined /> },
          { path: '/announcements', label: t('announcements'), icon: <NotificationOutlined /> },
          { path: '/admin', label: t('admin'), icon: <SettingOutlined /> },
          { path: 'https://github.com/gofurry/l4d2-plugin-stats', label: 'GitHub', icon: <GithubOutlined />, external: true },
        ]

  const active = (path: string) => path === '/'
    ? location.pathname === '/'
    : path === '/admin'
      ? location.pathname.startsWith('/admin')
      : location.pathname === path

  return <nav aria-label={t('mainNavigation')} className="floating-nav">
    {items.map(item => <button
      aria-current={active(item.path) ? 'page' : undefined}
      aria-label={item.label}
      className={`floating-nav-button${active(item.path) ? ' active' : ''}`}
      key={item.path}
      onClick={() => item.external ? window.open(item.path, '_blank', 'noopener,noreferrer') : navigate(item.path)}
      title={item.label}
      type="button"
    >
      <span className="floating-nav-icon">{item.icon}</span>
      <span className="floating-nav-label">{item.label}</span>
    </button>)}
    {mode === 'admin' && onLogout && <button
      aria-label={t('logout')}
      className="floating-nav-button danger"
      disabled={loggingOut}
      onClick={onLogout}
      title={t('logout')}
      type="button"
    >
      <span className="floating-nav-icon"><LogoutOutlined /></span>
      <span className="floating-nav-label">{t('logout')}</span>
    </button>}
  </nav>
}
