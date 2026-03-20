import { describe, expect, it } from 'vitest'
import { friendlyError, modeLabel, rank, resultLabel, toYouResult } from './utils'

describe('rank', () => {
  it('returns string of number', () => {
    expect(rank(1)).toBe('1')
    expect(rank(13)).toBe('13')
  })
})

describe('modeLabel', () => {
  it('returns プレイヤー for PLAYER', () => {
    expect(modeLabel('PLAYER')).toBe('プレイヤー')
  })
  it('returns ディーラー for DEALER', () => {
    expect(modeLabel('DEALER')).toBe('ディーラー')
  })
})

describe('toYouResult', () => {
  it('returns result as-is when not dealer', () => {
    expect(toYouResult('PLAYER_WIN', false)).toBe('PLAYER_WIN')
    expect(toYouResult('DEALER_WIN', false)).toBe('DEALER_WIN')
    expect(toYouResult('DRAW', false)).toBe('DRAW')
  })
  it('swaps when dealer', () => {
    expect(toYouResult('PLAYER_WIN', true)).toBe('DEALER_WIN')
    expect(toYouResult('DEALER_WIN', true)).toBe('PLAYER_WIN')
    expect(toYouResult('DRAW', true)).toBe('DRAW')
  })
})

describe('resultLabel', () => {
  it('returns empty for null', () => {
    expect(resultLabel(null)).toBe('')
  })
  it('returns labels for each result', () => {
    expect(resultLabel('PLAYER_WIN')).toBe('あなたの勝ち')
    expect(resultLabel('DEALER_WIN')).toBe('相手の勝ち')
    expect(resultLabel('DRAW')).toBe('引き分け')
  })
})

describe('friendlyError', () => {
  it('returns mapped messages for known codes', () => {
    expect(friendlyError('unauthorized', '')).toBe('ログインが必要です。')
    expect(friendlyError('forbidden', '')).toBe('操作できません。')
    expect(friendlyError('too_many_requests', '')).toBe('操作が早すぎます。少し待ってからもう一度お試しください。')
    expect(friendlyError('version_conflict', '')).toBe('状態が更新されました。もう一度お試しください。')
    expect(friendlyError('game_not_started', '')).toBe('ゲームが開始されていません。')
    expect(friendlyError('game_already_started', '')).toBe('ゲームは進行中です。')
    expect(friendlyError('game_not_finished', '')).toBe('まだゲームが終了していません。')
    expect(friendlyError('invalid_input', '')).toBe('入力内容が正しくありません。')
    expect(friendlyError('invalid_json', '')).toBe('入力内容が正しくありません。')
    expect(friendlyError('cheat_not_allowed', '')).toBe('このモードではチートは使えません。')
    expect(friendlyError('cheat_already_used', '')).toBe('チートはこのセットで1回までです。')
    expect(friendlyError('cheat_not_available', '')).toBe('チートを使えるカードが残っていません。')
  })
  it('returns message or default for unknown code', () => {
    expect(friendlyError('unknown', 'custom msg')).toBe('custom msg')
    expect(friendlyError('unknown', '')).toBe('エラーが発生しました。')
  })
})
