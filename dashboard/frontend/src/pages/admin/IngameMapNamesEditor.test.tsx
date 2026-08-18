import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { IngameMapNamesEditor } from './IngameMapNamesEditor'

describe('IngameMapNamesEditor', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      const values = init?.method === 'PUT'
        ? [{ map_name: 'custom_campaign_m1', display_name: '三方图第一章', updated_at: 2 }]
        : [{ map_name: 'custom_map', display_name: '三方图第一章', updated_at: 1 }]
      return Promise.resolve(new Response(JSON.stringify({ data: values, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('keeps map-name inputs mounted while editing their values', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<QueryClientProvider client={client}><IngameMapNamesEditor zh /></QueryClientProvider>)
    const mapInput = await screen.findByRole('textbox', { name: '地图代码 1' })
    mapInput.focus()
    fireEvent.change(mapInput, { target: { value: 'custom_campaign_m1' } })
    expect(screen.getByRole('textbox', { name: '地图代码 1' })).toBe(mapInput)
    expect(mapInput).toHaveValue('custom_campaign_m1')
    expect(mapInput).toHaveFocus()
    expect(screen.queryByRole('button', { name: '保存地图名称' })).not.toBeInTheDocument()
    await waitFor(() => expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'PUT')).toBe(true), { timeout: 2500 })
  })
})
