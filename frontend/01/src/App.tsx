import './App.css'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { ApiException as ApiExceptionClass } from './api/http'
import { login, logout, signup } from './api/auth'
import { gameChangeMode, gameCheat, gameResetSet, gameSelect, gameStart, gameStatus } from './api/game'
import type { GameMode, StatusResponse, SelectResponse } from './api/types'
import { friendlyError, modeLabel, rank, resultLabel, toYouResult } from './utils'

function ResultBadge({ result }: { result: SelectResponse['result'] | null }) {
  if (!result) return null
  const cls = result === 'DRAW' ? 'badge draw' : result === 'PLAYER_WIN' ? 'badge win' : 'badge lose'
  return <div className={cls}>{resultLabel(result)}</div>
}

function HistoryBadge({ result }: { result: SelectResponse['result'] }) {
  const cls = result === 'DRAW' ? 'badge draw' : result === 'PLAYER_WIN' ? 'badge win' : 'badge lose'
  return <div className={cls}>{resultLabel(result)}</div>
}

function Card({ value, dim, overlay }: { value: number | null; dim?: boolean; overlay?: 'win' | 'lose' | null }) {
  const shown = value ?? 0
  return (
    <div className={`cardUi ${dim ? 'dim' : ''}`}>
      <div className="cardPip">{value ? rank(shown) : '—'}</div>
      {overlay ? <div className={`cardOverlay ${overlay}`}>{overlay === 'win' ? 'Win' : 'Lose'}</div> : null}
    </div>
  )
}

function App() {
  const [authMode, setAuthMode] = useState<'login' | 'signup'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const [status, setStatus] = useState<StatusResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<{ code: string; message: string; status?: number } | null>(null)
  const [lastRound, setLastRound] = useState<SelectResponse | null>(null)
  const [cheatPending, setCheatPending] = useState(false)
  const [displayMode, setDisplayMode] = useState<GameMode | null>(null)

  const isAuthed = status !== null

  const clearNotice = useCallback(() => {
    setMessage(null)
    setError(null)
  }, [])

  const refreshStatus = useCallback(async () => {
    const s = await gameStatus()
    setStatus(s)
    return s
  }, [])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const s = await gameStatus()
        if (!cancelled) setStatus(s)
      } catch (e) {
        if (!cancelled) setStatus(null)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!status) {
      setDisplayMode(null)
      return
    }
    if (status.status === 'IN_PROGRESS') {
      setDisplayMode(status.mode)
      return
    }
    setDisplayMode((prev) => prev ?? status.mode)
  }, [status])

  useEffect(() => {
    if (!status || status.status !== 'IN_PROGRESS' || status.mode !== 'DEALER') {
      setCheatPending(false)
    }
  }, [status])

  const handleApiError = useCallback((e: unknown) => {
    if (e instanceof ApiExceptionClass) {
      setError({ code: e.code, message: friendlyError(e.code, e.message) })
      return
    }
    setError({ code: 'internal_error', message: 'エラーが発生しました。' })
  }, [])

  const run = useCallback(
    async <T,>(fn: () => Promise<T>) => {
      clearNotice()
      setLoading(true)
      try {
        const v = await fn()
        return v
      } catch (e) {
        handleApiError(e)
        throw e
      } finally {
        setLoading(false)
      }
    },
    [clearNotice, handleApiError],
  )

  const runWithVersionRetry = useCallback(
    async <T,>(fn: (s: StatusResponse) => Promise<T>) => {
      return await run(async () => {
        const s0 = status ?? (await refreshStatus())
        try {
          const out = await fn(s0)
          await refreshStatus()
          return out
        } catch (e) {
          if (e instanceof ApiExceptionClass && e.code === 'version_conflict') {
            const latest = await refreshStatus()
            const out2 = await fn(latest)
            await refreshStatus()
            return out2
          }
          throw e
        }
      })
    },
    [refreshStatus, run, status],
  )

  const canStart = useMemo(() => {
    if (!status) return false
    return status.status !== 'IN_PROGRESS'
  }, [status])

  const canSelect = useMemo(() => {
    if (!status) return false
    return status.status === 'IN_PROGRESS'
  }, [status])

  const canCheat = useMemo(() => {
    if (!status) return false
    return status.status === 'IN_PROGRESS' && status.mode === 'DEALER'
  }, [status])

  const canChangeMode = useMemo(() => {
    if (!status) return false
    return status.status === 'FINISHED'
  }, [status])

  const canResetSet = useMemo(() => {
    if (!status) return false
    return status.ver > 0
  }, [status])

  const effectiveMode: GameMode = displayMode ?? status?.mode ?? 'PLAYER'
  const youAreDealer = effectiveMode === 'DEALER'

  const yourCard = youAreDealer ? lastRound?.dealer_card ?? null : lastRound?.player_card ?? null
  const oppCard = youAreDealer ? lastRound?.player_card ?? null : lastRound?.dealer_card ?? null
  const yourResult = lastRound ? toYouResult(lastRound.result, youAreDealer) : null

  const finishedHistory = status?.status === 'FINISHED' && status.history.length ? status.history[status.history.length - 1] : null
  const finishedYourResult = finishedHistory ? toYouResult(finishedHistory.result, youAreDealer) : null
  const finishedWinLose: 'win' | 'lose' | null =
    finishedYourResult === 'PLAYER_WIN' ? 'win' : finishedYourResult === 'DEALER_WIN' ? 'lose' : null

  const yourWins = status ? (youAreDealer ? status.dealer_wins : status.player_wins) : 0
  const oppWins = status ? (youAreDealer ? status.player_wins : status.dealer_wins) : 0

  const onSubmitAuth = useCallback(async () => {
    await run(async () => {
      setStatus(null)
      setLastRound(null)
      setCheatPending(false)
      setDisplayMode(null)
      setMessage(null)
      setError(null)

      if (authMode === 'signup') {
        await signup({ email, password })
      }

      await login({ email, password })
      setMessage(authMode === 'signup' ? 'アカウントを作成してログインしました。' : 'ログインしました。')
      await refreshStatus()
    })
  }, [authMode, email, password, refreshStatus, run])

  const onLogout = useCallback(async () => {
    await run(async () => {
      await logout()
      setStatus(null)
      setLastRound(null)
      setCheatPending(false)
      setDisplayMode(null)
      setMessage('ログアウトしました。')
    })
  }, [run])

  const onResetSet = useCallback(async () => {
    await run(async () => {
      let s = status ?? (await refreshStatus())
      for (let i = 0; i < 3; i++) {
        try {
          const r = await gameResetSet({ ver: s.ver })
          setStatus((prev) =>
            prev
              ? {
                  ...prev,
                  status: r.status,
                  mode: r.mode,
                  ver: r.ver,
                  player_wins: 0,
                  dealer_wins: 0,
                  cheated: false,
                  cheat_reserved: false,
                  history: [],
                }
              : prev,
          )
      setLastRound(null)
      setCheatPending(false)
          await refreshStatus()
          return
        } catch (e) {
          if (e instanceof ApiExceptionClass && e.code === 'version_conflict') {
            s = await refreshStatus()
            continue
          }
          throw e
        }
      }
      throw new ApiExceptionClass({ code: 'version_conflict', message: 'retry exhausted', status: 409 })
    })
  }, [refreshStatus, run, status])

  return (
    <div className="tableBg">
      <div className={`page ${!isAuthed ? 'page-auth' : ''}`}>
      <header className="header">
        <div>
          <div className="title">High &amp; Low</div>
        </div>
        <div className="headerRight">
          {isAuthed ? (
            <>
              <button className="btn" disabled={loading} onClick={() => run(refreshStatus)}>
                表示を更新
              </button>
              <button className="btn danger" disabled={loading} onClick={onLogout}>
                ログアウト
              </button>
            </>
          ) : null}
        </div>
      </header>

      {(message || error) && (
        <div className={`notice ${error ? 'error' : 'ok'}`}>
          {error ? (
            <div>
              <b>エラー</b>
              <div className="small">{error.message}</div>
            </div>
          ) : (
            <div>{message}</div>
          )}
          <button className="btn ghost" onClick={clearNotice}>
            閉じる
          </button>
        </div>
      )}

      <main className="grid">
        {!isAuthed ? (
          <section className="panel auth-panel">
            <div className="tabs">
              <button
                className={`tab ${authMode === 'login' ? 'active' : ''}`}
                onClick={() => {
                  setAuthMode('login')
                  setEmail('')
                  setPassword('')
                }}
                disabled={loading}
              >
                ログイン
              </button>
              <button
                className={`tab ${authMode === 'signup' ? 'active' : ''}`}
                onClick={() => {
                  setAuthMode('signup')
                  setEmail('')
                  setPassword('')
                }}
                disabled={loading}
              >
                新規登録
              </button>
            </div>

            <label className="field">
              <div className="label">メールアドレス</div>
              <input
                className="input"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="user@example.com"
                autoComplete="email"
              />
            </label>
            <label className="field">
              <div className="label">パスワード</div>
              <input
                className="input"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="8文字以上"
                autoComplete={authMode === 'login' ? 'current-password' : 'new-password'}
              />
            </label>
            <button className="btn primary" disabled={loading} onClick={onSubmitAuth}>
              {authMode === 'login' ? 'ログイン' : '新規登録'}
            </button>
          </section>
        ) : (
          <>
            <section className="panel stage">
              <div className="stageTop">
                <div>
                  <div className="stageTitle">HIGH &amp; LOW</div>
                  <div className="stageSub">モード: {modeLabel(effectiveMode)}</div>
                </div>
                <div className="chips">
                  <div className="chip player">
                    あなた <span className="mono">{yourWins}</span>
                  </div>
                  <div className="chip dealer">
                    相手 <span className="mono">{oppWins}</span>
                  </div>
                </div>
              </div>

              <div className="arena">
                <div className="seat">
                  <div className="seatLabel">あなた</div>
                  <Card value={yourCard} dim={status?.status !== 'IN_PROGRESS'} overlay={null} />
                </div>
                <div className="center">
                  {status?.status === 'FINISHED' ? null : <ResultBadge result={yourResult} />}
                  <div className="metaRow">
                    <div className="meta">
                      ラウンド <span className="mono">{status?.history?.length ?? 0}</span>
                    </div>
                    {status?.status === 'FINISHED' && finishedWinLose ? (
                      <div className={`finishWinLose ${finishedWinLose}`}>
                        {finishedWinLose === 'win' ? 'Win' : 'Lose'}
                      </div>
                    ) : null}
                  </div>
                </div>
                <div className="seat">
                  <div className="seatLabel">相手</div>
                  <Card value={oppCard} dim={status?.status !== 'IN_PROGRESS'} />
                </div>
              </div>

              <div className="controls">
                <button
                  className="btn primary"
                  disabled={loading || !canStart}
                  onClick={() =>
                    runWithVersionRetry(async (s) => {
                      await gameStart(s.ver || undefined)
                      setLastRound(null)
                    })
                  }
                >
                  セット開始
                </button>
                <button
                  className="btn"
                  disabled={loading || !canSelect}
                  onClick={() =>
                    runWithVersionRetry(async (s) => {
                      let current = s
                      if (
                        cheatPending &&
                        current.mode === 'DEALER' &&
                        !current.cheated &&
                        !current.cheat_reserved
                      ) {
                        const cheatRes = await gameCheat({ ver: current.ver })
                        current = {
                          ...current,
                          ver: cheatRes.ver,
                          cheated: true,
                          cheat_reserved: true,
                        }
                      }
                      const r = await gameSelect({ sessionId: current.session_id, ver: current.ver })
                      setLastRound(r)
                      setCheatPending(false)
                    })
                  }
                >
                  勝負
                </button>
                <button className="btn" disabled={loading || !canResetSet} onClick={onResetSet}>
                  リセット
                </button>
                {status?.mode === 'DEALER' ? (
                  status.cheated && !status.cheat_reserved ? (
                    <div className="usedTag">イカサマ使用済み</div>
                  ) : (
                    <button
                      className={`btn warn ${cheatPending || status.cheat_reserved ? 'cheat-active' : ''}`}
                      disabled={loading || !canCheat || status.cheated}
                      onClick={() => setCheatPending((prev) => !prev)}
                    >
                      イカサマ
                    </button>
                  )
                ) : null}
                {status?.status === 'FINISHED' ? (
                  <button
                    className="btn"
                    disabled={loading || !canChangeMode}
                    onClick={() =>
                      runWithVersionRetry(async (s) => {
                        const next: GameMode = s.mode === 'PLAYER' ? 'DEALER' : 'PLAYER'
                        const r = await gameChangeMode({ mode: next, ver: s.ver })
                        void r
                      })
                    }
                  >
                    モード切替: {status ? `${modeLabel(status.mode)} → ${modeLabel(status.mode === 'PLAYER' ? 'DEALER' : 'PLAYER')}` : '—'}
                  </button>
                ) : null}
              </div>

            </section>

            <section className="panel">
              <div className="panelTitle">履歴表示</div>
              {status?.history?.length ? (
                <div className="historyList">
                  {status.history
                    .slice()
                    .reverse()
                    .map((h) => {
                      const yourResult = youAreDealer
                        ? h.result === 'PLAYER_WIN'
                          ? 'DEALER_WIN'
                          : h.result === 'DEALER_WIN'
                            ? 'PLAYER_WIN'
                            : 'DRAW'
                        : h.result
                      const itemCls =
                        yourResult === 'DRAW'
                          ? 'historyItem'
                          : `historyItem ${yourResult === 'PLAYER_WIN' ? 'win' : 'lose'}`
                      return (
                      <div className={itemCls} key={h.round}>
                        <div className="historyLeft">
                          <div className="historyRound">
                            ラウンド <span className="mono">{h.round}</span>
                          </div>
                          <div className="historyCards">
                            <div className="miniLabel">あなた</div>
                            <div className="miniCard mono">{rank(youAreDealer ? h.dealer_card : h.player_card)}</div>
                            <div className="vs">VS</div>
                            <div className="miniCard mono">{rank(youAreDealer ? h.player_card : h.dealer_card)}</div>
                            <div className="miniLabel">相手</div>
                          </div>
                          {h.result === 'DRAW' ? (
                            <div className="historySub">
                              引き分け連続: <span className="mono">{h.consecutive_draws}</span>
                            </div>
                          ) : null}
                          {h.cheat_used ? <div className="historySub">チート使用</div> : null}
                        </div>
                        <div className="historyRight">
                          <HistoryBadge
                            result={
                              youAreDealer
                                ? h.result === 'PLAYER_WIN'
                                  ? 'DEALER_WIN'
                                  : h.result === 'DEALER_WIN'
                                    ? 'PLAYER_WIN'
                                    : 'DRAW'
                                : h.result
                            }
                          />
                        </div>
                      </div>
                    )})}
                </div>
              ) : (
                <div className="muted">まだ履歴がありません</div>
              )}
            </section>
          </>
        )}
      </main>
      </div>
    </div>
  )
}

export default App
