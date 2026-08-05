import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SiteFooter } from './SiteFooter'

describe('SiteFooter', () => {
  it('renders footer links with separators and safe external attributes', () => {
    render(<SiteFooter site={{
      language: 'zh-CN', browser_title: 'L4D2 Stats', theme: 'light', footer_enabled: true, background_image_url: '', steam_openid_enabled: false, a2s_refresh_seconds: 30, site_documents: [], configured: true,
      footer_links: [{ label: 'ICP 查询', url: 'https://beian.miit.gov.cn' }, { label: '隐私政策', url: 'https://example.com/privacy' }],
    }} />)
    expect(screen.getByRole('link', { name: 'ICP 查询' })).toHaveAttribute('rel', 'noopener noreferrer')
    expect(screen.getByRole('link', { name: '隐私政策' })).toHaveAttribute('target', '_blank')
    expect(screen.getByText('·')).toBeInTheDocument()
  })

  it('does not render an empty footer', () => {
    const { container } = render(<SiteFooter site={{ language: 'zh-CN', browser_title: 'L4D2 Stats', theme: 'light', footer_enabled: false, background_image_url: '', steam_openid_enabled: false, a2s_refresh_seconds: 30, site_documents: [], configured: true, footer_links: [] }} />)
    expect(container).toBeEmptyDOMElement()
  })
})
