import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiException, apiGet, apiPost } from './http'

describe('ApiException', () => {
  it('creates error with code, message, status', () => {
    const e = new ApiException({ code: 'x', message: 'msg', status: 400 })
    expect(e).toBeInstanceOf(Error)
    expect(e.code).toBe('x')
    expect(e.message).toBe('msg')
    expect(e.status).toBe(400)
    expect(e.retryAfter).toBeUndefined()
  })
  it('includes retryAfter when provided', () => {
    const e = new ApiException({ code: 'x', message: 'm', status: 429, retryAfter: 5 })
    expect(e.retryAfter).toBe(5)
  })
})

describe('apiPost', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
    document.cookie = ''
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends POST with JSON body and credentials', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: { x: 1 } }), { status: 200 }),
    )
    const result = await apiPost<{ x: number }>('/path', { a: 1 })
    expect(result).toEqual({ x: 1 })
    expect(mockFetch).toHaveBeenCalledWith('/path', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: '{"a":1}',
    })
  })
  it('sends empty object when body undefined', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: null }), { status: 200 }),
    )
    await apiPost('/path')
    expect(mockFetch).toHaveBeenCalledWith(
      '/path',
      expect.objectContaining({ body: '{}' }),
    )
  })
  it('adds X-CSRF-Token when csrf_token cookie present', async () => {
    document.cookie = 'csrf_token=abc123'
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: null }), { status: 200 }),
    )
    await apiPost('/path')
    expect(mockFetch).toHaveBeenCalledWith(
      '/path',
      expect.objectContaining({
        headers: expect.objectContaining({ 'X-CSRF-Token': 'abc123' }),
      }),
    )
  })
  it('throws ApiException on error envelope', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(
        JSON.stringify({ success: false, error: { code: 'bad', message: 'err' } }),
        { status: 400, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    try {
      await apiPost('/path')
      expect.fail('should throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException)
      expect((e as ApiException).code).toBe('bad')
      expect((e as ApiException).message).toBe('err')
      expect((e as ApiException).status).toBe(400)
    }
  })
  it('throws on invalid JSON response', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(new Response('not json', { status: 200 }))
    await expect(apiPost('/path')).rejects.toThrow(ApiException)
  })
  it('throws not_found on 404 non-envelope response', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ message: 'Not Found' }), { status: 404 }),
    )
    try {
      await apiPost('/path')
      expect.fail('should throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException)
      expect((e as ApiException).code).toBe('not_found')
      expect((e as ApiException).message).toBe('Not Found')
    }
  })
  it('throws http_error on non-404 non-envelope error response', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ message: 'Server error' }), { status: 500 }),
    )
    try {
      await apiPost('/path')
      expect.fail('should throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException)
      expect((e as ApiException).code).toBe('http_error')
      expect((e as ApiException).message).toBe('Server error')
    }
  })
  it('throws invalid_json on 200 non-envelope response', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ message: 'ok' }), { status: 200 }),
    )
    try {
      await apiPost('/path')
      expect.fail('should throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException)
      expect((e as ApiException).code).toBe('invalid_json')
      expect((e as ApiException).message).toContain('unexpected response shape')
    }
  })
  it('uses statusText when non-envelope error has no string message', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({}), {
        status: 500,
        statusText: 'Internal Server Error',
      }),
    )
    try {
      await apiPost('/path')
      expect.fail('should throw')
    } catch (e) {
      expect((e as ApiException).message).toBe('Internal Server Error')
    }
  })
  it('throws ApiException with default code/message when error envelope has missing fields', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: false, error: {} }), { status: 400 }),
    )
    try {
      await apiPost('/path')
      expect.fail('should throw')
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException)
      expect((e as ApiException).code).toBe('internal_error')
      expect((e as ApiException).message).toBe('unknown error')
    }
  })
  it('passes Retry-After header to exception', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(
        JSON.stringify({ success: false, error: { code: 'x', message: 'm' } }),
        { status: 429, headers: { 'Retry-After': '10' } },
      ),
    )
    try {
      await apiPost('/path')
    } catch (e) {
      expect((e as ApiException).retryAfter).toBe(10)
    }
  })
})

describe('apiGet', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends GET and returns data', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: { a: 1 } }), { status: 200 }),
    )
    const result = await apiGet<{ a: number }>('/path')
    expect(result).toEqual({ a: 1 })
    expect(mockFetch).toHaveBeenCalledWith('/path', { method: 'GET', credentials: 'include' })
  })
})
