import type { ApiEnvelope, ApiError } from './types'

function getCookie(name: string): string | null {
  const parts = document.cookie.split(';').map((p) => p.trim())
  for (const p of parts) {
    if (p.startsWith(`${name}=`)) return decodeURIComponent(p.slice(name.length + 1))
  }
  return null
}

export class ApiException extends Error {
  readonly code: string
  readonly status: number
  readonly retryAfter?: number

  constructor(params: { code: string; message: string; status: number; retryAfter?: number }) {
    super(params.message)
    this.code = params.code
    this.status = params.status
    this.retryAfter = params.retryAfter
  }
}

async function parseEnvelope<T>(res: Response): Promise<T> {
  const retryAfterHeader = res.headers.get('Retry-After')
  const retryAfter = retryAfterHeader ? Number(retryAfterHeader) : undefined

  let json: unknown
  try {
    json = (await res.json()) as unknown
  } catch {
    throw new ApiException({
      code: 'invalid_json',
      message: `invalid json response (status ${res.status})`,
      status: res.status,
      retryAfter,
    })
  }

  const env = json as ApiEnvelope<T>
  if (!env || typeof env !== 'object' || !('success' in env)) {
    if (!res.ok) {
      const obj = json as { message?: unknown } | null
      const msg = typeof obj?.message === 'string' ? obj.message : res.statusText || `HTTP ${res.status}`
      throw new ApiException({
        code: res.status === 404 ? 'not_found' : 'http_error',
        message: msg,
        status: res.status,
        retryAfter,
      })
    }
    throw new ApiException({
      code: 'invalid_json',
      message: `unexpected response shape (status ${res.status})`,
      status: res.status,
      retryAfter,
    })
  }

  if (env.success) return env.data

  const err = (env as ApiError).error
  throw new ApiException({
    code: err?.code ?? 'internal_error',
    message: err?.message ?? 'unknown error',
    status: res.status,
    retryAfter,
  })
}

export async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const csrf = getCookie('csrf_token')
  const headers: HeadersInit = { 'Content-Type': 'application/json' }
  if (csrf) headers['X-CSRF-Token'] = csrf

  const res = await fetch(path, {
    method: 'POST',
    credentials: 'include',
    headers,
    body: body === undefined ? '{}' : JSON.stringify(body),
  })
  return await parseEnvelope<T>(res)
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path, { method: 'GET', credentials: 'include' })
  return await parseEnvelope<T>(res)
}

