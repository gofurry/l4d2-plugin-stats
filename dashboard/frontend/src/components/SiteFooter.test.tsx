import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SiteFooter } from './SiteFooter'

describe('SiteFooter', () => {
  it('renders plain multiline text and safe external links', () => {
    render(<SiteFooter site={{
      title: 'Stats', footer_text: '备案信息\n运营说明',
      footer_links: [{ label: 'ICP 查询', url: 'https://beian.miit.gov.cn', open_new_tab: true }],
    }} />)
    expect(screen.getByText(/备案信息/)).toHaveTextContent('备案信息 运营说明')
    expect(screen.getByRole('link', { name: 'ICP 查询' })).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('does not render an empty footer', () => {
    const { container } = render(<SiteFooter site={{ title: 'Stats', footer_text: '', footer_links: [] }} />)
    expect(container).toBeEmptyDOMElement()
  })
})
