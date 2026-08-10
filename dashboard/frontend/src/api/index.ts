import { adminAPI } from './admin'
import { playersAPI } from './players'
import { rankingsAPI } from './rankings'
import { serversAPI } from './servers'
import { siteAPI } from './site'

export { APIError, resetCSRF } from './client'
export * from './admin'
export * from './players'
export * from './rankings'
export * from './servers'
export * from './site'

export const api = {
  ...siteAPI,
  ...serversAPI,
  ...playersAPI,
  ...rankingsAPI,
  ...adminAPI,
}
