import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Layout } from 'antd'
import { Navigate, Outlet, useNavigate } from 'react-router-dom'
import { api, APIError, resetCSRF } from '../../api'
import { FloatingNav } from '../../components/FloatingNav'
import styles from '../Portal.module.scss'

export function AdminLayout() {
  const nav = useNavigate()
  const client = useQueryClient()
  const me = useQuery({ queryKey: ['admin-me'], queryFn: api.adminMe, retry: false })
  const logout = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      resetCSRF()
      client.clear()
      nav('/admin/login')
    },
  })
  if (me.isLoading) return <Layout className={styles.layout} />
  if (me.error instanceof APIError && me.error.status === 401) return <Navigate to="/admin/login" replace />
  return <Layout className={styles.adminShell}>
    <FloatingNav mode="admin" loggingOut={logout.isPending} monitorEnabled={me.data?.monitor_enabled} onLogout={() => logout.mutate()} />
    <Layout.Content className={styles.adminContent}><Outlet /></Layout.Content>
  </Layout>
}
