import { useEffect, type PropsWithChildren } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ConfigProvider, theme as antdTheme } from 'antd'
import enUS from 'antd/locale/en_US'
import zhCN from 'antd/locale/zh_CN'
import { useTranslation } from 'react-i18next'
import { api } from '../api'

export function DashboardTheme({ children }: PropsWithChildren) {
  const { i18n } = useTranslation()
	const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 5 * 60_000, retry: 1 })
	const mode = site.data?.theme === 'dark' ? 'dark' : 'light'

	useEffect(() => {
		document.documentElement.dataset.theme = mode
		document.documentElement.dataset.colorMode = mode
	}, [mode])

  return <ConfigProvider locale={i18n.resolvedLanguage === 'en' ? enUS : zhCN} form={{ requiredMark: false }} theme={{
	algorithm: mode === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
	token: {
    colorPrimary: '#c8753f',
    colorInfo: '#c8753f',
    colorSuccess: '#77865e',
    colorError: '#a85448',
		colorText: mode === 'dark' ? '#f2ebe4' : '#4e3c32',
		colorTextSecondary: mode === 'dark' ? '#b9aea5' : '#8d7b6f',
		colorBgBase: mode === 'dark' ? '#151719' : '#fffaf3',
		colorBgContainer: mode === 'dark' ? '#202326' : '#fffaf3',
		colorBgElevated: mode === 'dark' ? '#25282b' : '#fffaf3',
		colorBorder: mode === 'dark' ? '#4b4e50' : '#dccbbb',
    borderRadius: 14,
    fontFamily: 'Inter, Segoe UI, PingFang SC, Microsoft YaHei, sans-serif',
  } }}>
    {children}
  </ConfigProvider>
}
