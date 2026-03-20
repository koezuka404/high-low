import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  gameChangeMode,
  gameCheat,
  gameResetSet,
  gameSelect,
  gameStart,
  gameStatus,
} from './game'

describe('game', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const mockOk = (data: unknown) =>
    new Response(JSON.stringify({ success: true, data }), { status: 200 })

  it('gameStatus calls GET /api/game/status', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      mockOk({
        session_id: 1,
        status: 'IN_PROGRESS',
        mode: 'PLAYER',
        player_wins: 0,
        dealer_wins: 0,
        ver: 1,
        cheated: false,
        cheat_reserved: false,
        history: [],
      }),
    )
    const r = await gameStatus()
    expect(r.session_id).toBe(1)
    expect(mockFetch).toHaveBeenCalledWith('/api/game/status', {
      method: 'GET',
      credentials: 'include',
    })
  })

  it('gameStart calls POST /api/game/start with ver when provided', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      mockOk({ session_id: 1, mode: 'PLAYER', player_wins: 0, dealer_wins: 0, ver: 2 }),
    )
    await gameStart(1)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/game/start',
      expect.objectContaining({ body: '{"ver":1}' }),
    )
  })

  it('gameStart calls POST with empty object when ver undefined', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      mockOk({ session_id: 1, mode: 'PLAYER', player_wins: 0, dealer_wins: 0, ver: 1 }),
    )
    await gameStart()
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/game/start',
      expect.objectContaining({ body: '{}' }),
    )
  })

  it('gameSelect calls POST with session_id and ver', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      mockOk({
        player_card: 7,
        dealer_card: 10,
        result: 'DEALER_WIN',
        player_wins: 0,
        dealer_wins: 1,
        game_status: 'IN_PROGRESS',
        ver: 2,
      }),
    )
    await gameSelect({ sessionId: 1, ver: 1 })
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/game/select',
      expect.objectContaining({ body: '{"session_id":1,"ver":1}' }),
    )
  })

  it('gameCheat calls POST with ver', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(mockOk({ cheat_reserved: true, cheat_card: 13, ver: 2 }))
    await gameCheat({ ver: 1 })
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/game/cheat',
      expect.objectContaining({ body: '{"ver":1}' }),
    )
  })

  it('gameResetSet calls POST /api/game/reset', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(
      mockOk({ status: 'IN_PROGRESS', mode: 'PLAYER', ver: 2 }),
    )
    await gameResetSet({ ver: 1 })
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/game/reset',
      expect.objectContaining({ body: '{"ver":1}' }),
    )
  })

  it('gameChangeMode calls POST with mode and ver', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValue(mockOk({ mode: 'DEALER', ver: 2 }))
    await gameChangeMode({ mode: 'DEALER', ver: 1 })
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/game/mode',
      expect.objectContaining({ body: '{"mode":"DEALER","ver":1}' }),
    )
  })
})
