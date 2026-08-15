import type { PlayerProfileSection } from '../../api'

export const playerProfileSections: PlayerProfileSection[] = [
  'overview', 'achievements', 'analysis', 'pve', 'pve-details', 'versus-survivor',
  'versus-survivor-details', 'versus-infected', 'versus-infected-details', 'relationships', 'history',
]

export const defaultPlayerProfileSections: PlayerProfileSection[] = ['overview', 'analysis', 'relationships']
