import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { IngameMapNamesEditor } from './IngameMapNamesEditor'

describe('IngameMapNamesEditor', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response(JSON.stringify({ data: [{ map_name: 'custom_map', display_name: '三方图第一章', updated_at: 1 }], request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))))
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
  })
})
