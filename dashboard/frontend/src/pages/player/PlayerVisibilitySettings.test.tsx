import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api'
import { PlayerVisibilitySettings } from './PlayerVisibilitySettings'
import { defaultPlayerProfileSections } from './playerVisibility'

describe('PlayerVisibilitySettings', () => {
  afterEach(() => vi.restoreAllMocks())

  it('starts with the three default public tabs and saves explicit choices', async () => {
    const save = vi.spyOn(api, 'savePlayerProfileVisibility').mockResolvedValue({ visible_sections: [...defaultPlayerProfileSections, 'pve'] })
    const onSaved = vi.fn()
    render(<PlayerVisibilitySettings value={defaultPlayerProfileSections} onSaved={onSaved} zh />)

    expect(screen.getByRole('switch', { name: '概览' })).toBeChecked()
    expect(screen.getByRole('switch', { name: '分析' })).toBeChecked()
    expect(screen.getByRole('switch', { name: '玩家关系' })).toBeChecked()
    expect(screen.getByRole('switch', { name: 'PvE' })).not.toBeChecked()

    fireEvent.click(screen.getByRole('switch', { name: 'PvE' }))
    fireEvent.click(screen.getByRole('button', { name: '保存设置' }))
    await waitFor(() => expect(save).toHaveBeenCalledWith([...defaultPlayerProfileSections, 'pve']))
    expect(onSaved).toHaveBeenCalledWith([...defaultPlayerProfileSections, 'pve'])
  })

  it('explains what visitors see when every tab is private', () => {
    render(<PlayerVisibilitySettings value={[]} onSaved={() => undefined} zh />)
    expect(screen.getByText('关闭全部栏目后，访客只能看到你的玩家名称和 SteamID。')).toBeInTheDocument()
  })
})
