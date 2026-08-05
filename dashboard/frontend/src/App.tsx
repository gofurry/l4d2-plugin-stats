import { lazy, Suspense, useEffect, useRef } from 'react'
import { Spin } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { api } from './api'
import i18n from './i18n'
import { HomePage } from './pages/HomePage'
import { PageScrollOrb } from './components/PageScrollOrb'

const PlayerPage=lazy(()=>import('./pages/PlayerPage').then(module=>({default:module.PlayerPage})))
const RankingsPage=lazy(()=>import('./pages/RankingsPage').then(module=>({default:module.RankingsPage})))
const AnnouncementsPage=lazy(()=>import('./pages/AnnouncementsPage').then(module=>({default:module.AnnouncementsPage})))
const AdminSetupPage=lazy(()=>import('./pages/AdminSetupPage').then(module=>({default:module.AdminSetupPage})))
const AdminLoginPage=lazy(()=>import('./pages/AdminLoginPage').then(module=>({default:module.AdminLoginPage})))
const AdminLayout=lazy(()=>import('./pages/AdminPages').then(module=>({default:module.AdminLayout})))
const AdminSitePage=lazy(()=>import('./pages/AdminPages').then(module=>({default:module.AdminSitePage})))
const AdminServersPage=lazy(()=>import('./pages/AdminPages').then(module=>({default:module.AdminServersPage})))
const AdminSecurityPage=lazy(()=>import('./pages/AdminPages').then(module=>({default:module.AdminSecurityPage})))
const AdminAnnouncementsPage=lazy(()=>import('./pages/AdminAnnouncementsPage').then(module=>({default:module.AdminAnnouncementsPage})))
const AdminDataPage=lazy(()=>import('./pages/AdminDataPage').then(module=>({default:module.AdminDataPage})))

function SiteAppearanceSync() {
  const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 5 * 60_000, retry: 1 })
  useEffect(() => {
    const language = site.data?.language ?? 'zh-CN'
    void i18n.changeLanguage(language)
    document.documentElement.lang = language
	document.title = site.data?.browser_title?.trim() || 'L4D2 Stats'
	const backgroundURL = site.data?.background_image_url?.trim() ?? ''
	if (backgroundURL) {
		document.body.style.setProperty('--site-background-image', `url(${JSON.stringify(backgroundURL)})`)
		document.body.classList.add('has-site-background')
	} else {
		document.body.style.removeProperty('--site-background-image')
		document.body.classList.remove('has-site-background')
	}
  }, [site.data?.language, site.data?.browser_title, site.data?.background_image_url])
  return null
}

function AppRoutes() {
  const location = useLocation()
  const scrollerRef = useRef<HTMLDivElement>(null)
  useEffect(() => { scrollerRef.current?.scrollTo({ top: 0 }) }, [location.pathname])
  return <><SiteAppearanceSync/><div className="app-scroll" ref={scrollerRef}><Suspense fallback={<div style={{display:'grid',placeItems:'center',minHeight:'60vh'}}><Spin size="large"/></div>}><Routes>
    <Route path="/" element={<HomePage />} />
    <Route path="/player" element={<PlayerPage />} />
    <Route path="/rankings" element={<RankingsPage />} />
    <Route path="/announcements" element={<AnnouncementsPage />} />
    <Route path="/admin/setup" element={<AdminSetupPage />} />
    <Route path="/admin/login" element={<AdminLoginPage />} />
    <Route path="/admin" element={<AdminLayout />}>
      <Route index element={<Navigate to="site" replace />} />
      <Route path="site" element={<AdminSitePage />} />
      <Route path="servers" element={<AdminServersPage />} />
      <Route path="announcements" element={<AdminAnnouncementsPage />} />
      <Route path="security" element={<AdminSecurityPage />} />
      <Route path="data" element={<AdminDataPage />} />
    </Route>
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes></Suspense></div><PageScrollOrb scrollerRef={scrollerRef} refreshKey={location.pathname}/></>
}

export default function App() {
  return <BrowserRouter><AppRoutes/></BrowserRouter>
}
