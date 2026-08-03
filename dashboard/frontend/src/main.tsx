import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfigProvider } from 'antd'
import 'antd/dist/reset.css'
import './i18n'
import './styles/tailwind.css'
import './styles/global.scss'
import App from './App'

const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false } } })

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider theme={{ token: {
      colorPrimary: '#c8753f', colorInfo: '#c8753f', colorSuccess: '#77865e', colorError: '#a85448',
      colorText: '#4e3c32', colorTextSecondary: '#8d7b6f', colorBgBase: '#fffaf3', colorBorder: '#dccbbb',
      borderRadius: 14, fontFamily: 'Inter, Segoe UI, PingFang SC, Microsoft YaHei, sans-serif',
    } }}>
      <QueryClientProvider client={queryClient}><App /></QueryClientProvider>
    </ConfigProvider>
  </React.StrictMode>,
)
