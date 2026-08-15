import { describe, expect, it } from 'vitest'
import { preferredPlayerSteamID } from './playerFormat'

describe('preferredPlayerSteamID', () => {
  const self = '76561198000000001'
  const other = '76561198000000002'

  it('keeps an explicitly linked player ahead of the signed-in identity', () => {
    expect(preferredPlayerSteamID(other, self, self)).toBe(other)
  })

  it('uses the signed-in identity for a plain player-center visit', () => {
    expect(preferredPlayerSteamID('', self, other)).toBe(self)
  })

  it('falls back to the saved query only while signed out', () => {
    expect(preferredPlayerSteamID('', '', other)).toBe(other)
  })
})
