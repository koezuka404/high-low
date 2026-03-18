export type ApiSuccess<T> = { success: true; data: T }
export type ApiError = { success: false; error: { code: string; message: string } }
export type ApiEnvelope<T> = ApiSuccess<T> | ApiError

export type GameMode = 'PLAYER' | 'DEALER'
export type GameStatus = 'NOT_STARTED' | 'IN_PROGRESS' | 'FINISHED'
export type RoundResult = 'PLAYER_WIN' | 'DEALER_WIN' | 'DRAW'

export type StatusResponse = {
  session_id: number
  status: GameStatus
  mode: GameMode
  player_wins: number
  dealer_wins: number
  ver: number
  cheated: boolean
  cheat_reserved: boolean
  history: Array<{
    round: number
    player_card: number
    dealer_card: number
    result: RoundResult
    consecutive_draws: number
    cheat_used: boolean
  }>
}

export type StartResponse = {
  session_id: number
  mode: GameMode
  player_wins: number
  dealer_wins: number
  ver: number
}

export type SelectResponse = {
  player_card: number
  dealer_card: number
  result: RoundResult
  player_wins: number
  dealer_wins: number
  game_status: GameStatus
  ver: number
}

export type CheatResponse = {
  cheat_reserved: boolean
  cheat_card: number
  ver: number
}

export type ResetSetResponse = {
  status: GameStatus
  mode: GameMode
  ver: number
}

export type ChangeModeResponse = {
  mode: GameMode
  ver: number
}

