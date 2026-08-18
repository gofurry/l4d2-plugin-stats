import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AdminAuditPage } from './AdminAuditPage'

function response(data: unknown): Response {
  return new Response(JSON.stringify({ data, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('AdminAuditPage', () => {
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('uses POST for sensitive searches and never returns the GeoIP key', async () => {
    const fetcher = vi.fn((input: RequestInfo | URL, _init?: RequestInit) => {
		void _init
      const path = String(input)
      if (path.endsWith('/geoip/settings')) return Promise.resolve(response({ enabled: true, provider: 'baidu', api_key_configured: true, api_key_masked: '****1234', ipv4_status: 'working', ipv6_status: 'unknown', cache_count: 2, pending_count: 0 }))
      if (path.endsWith('/connections/search')) return Promise.resolve(response({ items: [] }))
      return Promise.resolve(response({ token: 'csrf' }))
    })
    vi.stubGlobal('fetch', fetcher)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
    render(<QueryClientProvider client={client}><AdminAuditPage /></QueryClientProvider>)
    expect(await screen.findByText('****1234')).toBeInTheDocument()
    expect(screen.queryByText('secret-key')).not.toBeInTheDocument()
    await waitFor(() => expect(fetcher.mock.calls.some(call => String(call[0]).endsWith('/connections/search') && call[1]?.method === 'POST')).toBe(true))
  })

  it('shows chat completeness status without exposing it publicly', async () => {
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const path = String(input)
      if (path.endsWith('/chat/settings')) return Promise.resolve(response({ enabled: true, retention_days: 30, last_cleanup_at: 0, updated_at: 0 }))
      if (path.endsWith('/chat/status')) return Promise.resolve(response({ database: { driver: 'sqlite', bytes: 1 }, message_count: 12, oldest_message_at: 1, newest_message_at: 2, retention_days: 30, last_cleanup_at: 0, ingestion_lag: 3, dropped_count: 1, known_gap_count: 2, last_ingest_at: 2 }))
      if (path.endsWith('/chat/search')) return Promise.resolve(response({ items: [] }))
      if (path.endsWith('/geoip/settings')) return Promise.resolve(response({ enabled: false, provider: 'baidu', api_key_configured: false, ipv4_status: 'unknown', ipv6_status: 'unknown', cache_count: 0, pending_count: 0 }))
      if (path.endsWith('/connections/search')) return Promise.resolve(response({ items: [] }))
      return Promise.resolve(response({ token: 'csrf' }))
    }))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
    render(<QueryClientProvider client={client}><AdminAuditPage /></QueryClientProvider>)
    fireEvent.click(screen.getByRole('tab', { name: '聊天记录' }))
    expect(await screen.findByText('聊天审计状态')).toBeInTheDocument()
    expect(await screen.findByText('12')).toBeInTheDocument()
  })
})
