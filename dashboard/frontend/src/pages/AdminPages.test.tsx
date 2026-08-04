import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import i18n from '../i18n'
import { AdminLayout } from './AdminPages'

function response(data: unknown): Response {
  return new Response(JSON.stringify({ data, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('AdminLayout', () => {
  beforeEach(async () => { await i18n.changeLanguage('zh-CN') })

  it('provides home navigation and a working global logout action', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.endsWith('/auth/me')) return Promise.resolve(response({ username: 'admin' }))
      if (path.endsWith('/auth/csrf')) return Promise.resolve(response({ token: 'csrf-token' }))
      return Promise.resolve(response({}))
    }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<MemoryRouter initialEntries={['/admin/site']}><QueryClientProvider client={client}><Routes>
      <Route path="/admin" element={<AdminLayout />}><Route path="site" element={<div>管理工作区</div>} /></Route>
      <Route path="/admin/login" element={<div>登录页</div>} />
    </Routes></QueryClientProvider></MemoryRouter>)

    expect(await screen.findByRole('button', { name: '主页' })).toBeInTheDocument()
    expect(screen.getByText('管理工作区')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '退出登录' }))
    await waitFor(() => expect(screen.getByText('登录页')).toBeInTheDocument())
  })
})
