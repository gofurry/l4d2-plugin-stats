import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { IngameGroupSettingsEditor } from './IngameGroupSettingsEditor'

const group = {
  server_key: 'main', title: '主服务器',
  instances: [{ server_id: 'server-1', name: '官图 #1', address: '127.0.0.1:27015', sort_order: 0, online: true, stale: false }],
}

const response = {
  settings: {
    server_key: 'main', title_mode: 'inherit', title: '', description_mode: 'inherit', description: '', short_description: '',
    banner_mode: 'inherit', banner_url: '', background_mode: 'inherit', background_url: '', website_mode: 'inherit', website_url: '',
    highlight_mode: 'inherit', highlight_metrics: ['active_play_seconds', 'special_kills', 'rescues'], updated_at: 0,
  },
  documents: [
    { server_key: 'main', key: 'introduction', mode: 'inherit', content_markdown: '', updated_at: 0 },
    { server_key: 'main', key: 'commands', mode: 'inherit', content_markdown: '', updated_at: 0 },
    { server_key: 'main', key: 'resources', mode: 'inherit', content_markdown: '', updated_at: 0 },
  ],
  quick_links: [{ server_key: 'main', label: '原链接', url: 'https://example.com', sort_order: 0, enabled: true }],
  metric_catalog: [
    { key: 'active_play_seconds', label: '实际游戏时间', mode: 'activity', ranking_metric: 'active_time', format: 'duration' },
    { key: 'special_kills', label: '特殊感染者击杀', mode: 'pve', ranking_metric: 'special_kills', format: 'integer' },
    { key: 'rescues', label: '队友救援', mode: 'pve', ranking_metric: 'rescues', format: 'integer' },
  ],
  server_key: 'main', title: '主服务器', instances: group.instances, public_origin: 'https://stats.example.com',
}

describe('IngameGroupSettingsEditor', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('zh-CN')
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response(JSON.stringify({ data: response, request_id: 'test' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('keeps the quick-link label input mounted and focused while typing', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
    render(<QueryClientProvider client={client}><IngameGroupSettingsEditor group={group} /></QueryClientProvider>)
    const input = await screen.findByRole('textbox', { name: '链接 1 名称' })
    input.focus()
    fireEvent.change(input, { target: { value: '地' } })
    expect(screen.getByRole('textbox', { name: '链接 1 名称' })).toBe(input)
    expect(input).toHaveFocus()
    fireEvent.change(input, { target: { value: '地图合集' } })
    expect(input).toHaveValue('地图合集')
    expect(input).toHaveFocus()
  })
})
