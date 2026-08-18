import { afterEach, describe, expect, it, vi } from 'vitest'
import { copyTextToClipboard } from './clipboard'

afterEach(() => {
  Reflect.deleteProperty(navigator, 'clipboard')
  Reflect.deleteProperty(document, 'execCommand')
  document.querySelectorAll('textarea').forEach(element => element.remove())
})

describe('copyTextToClipboard', () => {
  it('uses the modern Clipboard API when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    await copyTextToClipboard('motd content')
    expect(writeText).toHaveBeenCalledWith('motd content')
  })

  it('falls back to execCommand when Clipboard API is rejected on HTTP', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('not allowed'))
    const execCommand = vi.fn(() => document.querySelector('textarea')?.value === 'motd content')
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    Object.defineProperty(document, 'execCommand', { configurable: true, value: execCommand })
    await copyTextToClipboard('motd content')
    expect(execCommand).toHaveBeenCalledWith('copy')
    expect(document.querySelector('textarea')).toBeNull()
  })
})
