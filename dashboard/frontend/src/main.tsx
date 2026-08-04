import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import 'antd/dist/reset.css'
import './i18n'
import './styles/tailwind.css'
import './styles/global.scss'
import App from './App'
import { DashboardTheme } from './components/DashboardTheme'

const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false } } })

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}><DashboardTheme><App /></DashboardTheme></QueryClientProvider>
  </React.StrictMode>,
)
