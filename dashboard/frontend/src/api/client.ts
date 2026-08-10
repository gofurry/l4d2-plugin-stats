interface Envelope<T> { data: T; request_id: string }
interface ErrorEnvelope { error?: { code?: string; message?: string } }

export class APIError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message) }
}

let csrfToken = ''

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body) headers.set('Content-Type', 'application/json')
  const response = await fetch(path, { ...init, headers, credentials: 'same-origin' })
  let payload: Envelope<T> & ErrorEnvelope
  try { payload = await response.json() as Envelope<T> & ErrorEnvelope } catch { throw new APIError(response.status, 'invalid_response', `HTTP ${response.status}`) }
  if (!response.ok) throw new APIError(response.status, payload.error?.code ?? 'request_failed', payload.error?.message ?? `HTTP ${response.status}`)
  return payload.data
}

async function csrf(): Promise<string> {
  if (csrfToken) return csrfToken
  const result = await request<{ token: string }>('/api/v1/admin/auth/csrf')
  csrfToken = result.token
  return csrfToken
}

export async function adminWrite<T>(path: string, method: string, body?: unknown): Promise<T> {
  const execute = async () => request<T>(path, { method, headers: { 'X-Csrf-Token': await csrf() }, body: body === undefined ? undefined : JSON.stringify(body) })
  try { return await execute() } catch (error) { if (error instanceof APIError && error.code === 'csrf_invalid') { csrfToken = ''; return execute() } throw error }
}

export function queryString(values: Record<string, string | number | undefined>) {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(values)) if (value !== undefined && value !== '') query.set(key, String(value))
  return query.toString()
}

export function resetCSRF() { csrfToken = '' }
