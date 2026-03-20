import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { login, logout, signup } from './auth'

describe('auth', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('signup calls POST /signup', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: null }), { status: 201 }),
    )
    await signup({ email: 'a@b.com', password: 'pass1234' })
    expect(mockFetch).toHaveBeenCalledWith(
      '/signup',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ email: 'a@b.com', password: 'pass1234' }),
      }),
    )
  })

  it('login calls POST /login', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: null }), { status: 200 }),
    )
    await login({ email: 'a@b.com', password: 'pass1234' })
    expect(mockFetch).toHaveBeenCalledWith(
      '/login',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ email: 'a@b.com', password: 'pass1234' }),
      }),
    )
  })

  it('logout calls POST /logout', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ success: true, data: null }), { status: 200 }),
    )
    await logout()
    expect(mockFetch).toHaveBeenCalledWith(
      '/logout',
      expect.objectContaining({
        method: 'POST',
        body: '{}',
      }),
    )
  })
})
