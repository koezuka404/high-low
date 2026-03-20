import type { GameMode } from './api/types'

export type RoundResult = 'PLAYER_WIN' | 'DEALER_WIN' | 'DRAW'

export function rank(n: number): string {
  return String(n)
}

export function modeLabel(m: GameMode): string {
  return m === 'PLAYER' ? 'プレイヤー' : 'ディーラー'
}

export function toYouResult(result: RoundResult, youAreDealer: boolean): RoundResult {
  if (!youAreDealer) return result
  if (result === 'PLAYER_WIN') return 'DEALER_WIN'
  if (result === 'DEALER_WIN') return 'PLAYER_WIN'
  return 'DRAW'
}

export function resultLabel(r: RoundResult | null): string {
  if (!r) return ''
  if (r === 'PLAYER_WIN') return 'あなたの勝ち'
  if (r === 'DEALER_WIN') return '相手の勝ち'
  return '引き分け'
}

export function friendlyError(code: string, message: string): string {
  switch (code) {
    case 'unauthorized':
      return 'ログインが必要です。'
    case 'forbidden':
      return '操作できません。'
    case 'too_many_requests':
      return '操作が早すぎます。少し待ってからもう一度お試しください。'
    case 'version_conflict':
      return '状態が更新されました。もう一度お試しください。'
    case 'game_not_started':
      return 'ゲームが開始されていません。'
    case 'game_already_started':
      return 'ゲームは進行中です。'
    case 'game_not_finished':
      return 'まだゲームが終了していません。'
    case 'invalid_input':
    case 'invalid_json':
      return '入力内容が正しくありません。'
    case 'cheat_not_allowed':
      return 'このモードではチートは使えません。'
    case 'cheat_already_used':
      return 'チートはこのセットで1回までです。'
    case 'cheat_not_available':
      return 'チートを使えるカードが残っていません。'
    default:
      return message || 'エラーが発生しました。'
  }
}
