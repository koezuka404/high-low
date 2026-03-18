import { apiGet, apiPost } from './http'
import type { ChangeModeResponse, CheatResponse, SelectResponse, StartResponse, StatusResponse, GameMode, ResetSetResponse } from './types'

export async function gameStatus(): Promise<StatusResponse> {
  return await apiGet<StatusResponse>('/api/game/status')
}

export async function gameStart(ver?: number): Promise<StartResponse> {
  return await apiPost<StartResponse>('/api/game/start', ver ? { ver } : {})
}

export async function gameSelect(params: { sessionId: number; ver: number }): Promise<SelectResponse> {
  return await apiPost<SelectResponse>('/api/game/select', { session_id: params.sessionId, ver: params.ver })
}

export async function gameCheat(params: { ver: number }): Promise<CheatResponse> {
  return await apiPost<CheatResponse>('/api/game/cheat', { ver: params.ver })
}

export async function gameResetSet(params: { ver: number }): Promise<ResetSetResponse> {
  return await apiPost<ResetSetResponse>('/api/game/reset', { ver: params.ver })
}

export async function gameChangeMode(params: { mode: GameMode; ver: number }): Promise<ChangeModeResponse> {
  return await apiPost<ChangeModeResponse>('/api/game/mode', { mode: params.mode, ver: params.ver })
}

