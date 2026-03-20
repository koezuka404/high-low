import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import App from './App'
import { ApiException } from './api/http'
import * as auth from './api/auth'
import * as game from './api/game'

vi.mock('./api/auth')
vi.mock('./api/game')

const mockAuth = vi.mocked(auth)
const mockGame = vi.mocked(game)

const defaultStatus = {
  session_id: 1,
  status: 'IN_PROGRESS' as const,
  mode: 'PLAYER' as const,
  player_wins: 0,
  dealer_wins: 0,
  ver: 1,
  cheated: false,
  cheat_reserved: false,
  history: [] as { round: number; player_card: number; dealer_card: number; result: string; consecutive_draws: number; cheat_used: boolean }[],
}

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGame.gameStatus.mockResolvedValue({ ...defaultStatus })
    mockAuth.login.mockResolvedValue(undefined)
    mockAuth.signup.mockResolvedValue(undefined)
    mockAuth.logout.mockResolvedValue(undefined)
    mockGame.gameStart.mockResolvedValue({ session_id: 1, mode: 'PLAYER', player_wins: 0, dealer_wins: 0, ver: 2 })
    mockGame.gameSelect.mockResolvedValue({
      player_card: 7,
      dealer_card: 10,
      result: 'DEALER_WIN',
      player_wins: 0,
      dealer_wins: 1,
      game_status: 'IN_PROGRESS',
      ver: 2,
    })
    mockGame.gameCheat.mockResolvedValue({ cheat_reserved: true, cheat_card: 13, ver: 2 })
    mockGame.gameResetSet.mockResolvedValue({ status: 'IN_PROGRESS', mode: 'PLAYER', ver: 2 })
    mockGame.gameChangeMode.mockResolvedValue({ mode: 'DEALER', ver: 2 })
  })

  it('renders login form when unauthenticated', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await waitFor(() => {
      expect(screen.getByPlaceholderText('user@example.com')).toBeInTheDocument()
    })
    expect(screen.getByPlaceholderText('8文字以上')).toBeInTheDocument()
    expect(screen.getAllByText('ログイン').length).toBeGreaterThanOrEqual(1)
  })

  it('switches to login tab from signup', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    const user = userEvent.setup()
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await user.click(screen.getByRole('button', { name: '新規登録' }))
    const loginButtons = screen.getAllByRole('button', { name: 'ログイン' })
    await user.click(loginButtons[0])
    expect(loginButtons[0]).toHaveClass('active')
  })

  it('switches to signup tab', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    const user = userEvent.setup()
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '新規登録' })).toBeInTheDocument()
    })
    const signupTab = screen.getAllByRole('button', { name: '新規登録' })[0]
    await user.click(signupTab)
    expect(signupTab).toHaveClass('active')
  })

  it('shows authenticated game UI when gameStatus returns data', async () => {
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('セット開始')).toBeInTheDocument()
    })
    expect(screen.getByText('勝負')).toBeInTheDocument()
    expect(screen.getByText('リセット')).toBeInTheDocument()
  })

  it('calls login on submit when in login mode', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    const user = userEvent.setup()
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await waitFor(() => {
      expect(screen.getByPlaceholderText('user@example.com')).toBeInTheDocument()
    })
    await user.type(screen.getByPlaceholderText('user@example.com'), 'a@b.com')
    await user.type(screen.getByPlaceholderText('8文字以上'), 'password123')
    await user.click(screen.getAllByRole('button', { name: 'ログイン' })[1])
    await waitFor(() => {
      expect(mockAuth.login).toHaveBeenCalledWith({ email: 'a@b.com', password: 'password123' })
    })
  })

  it('calls signup then login when in signup mode', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    const user = userEvent.setup()
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '新規登録' })).toBeInTheDocument()
    })
    await user.click(screen.getAllByRole('button', { name: '新規登録' })[0])
    await user.type(screen.getByPlaceholderText('user@example.com'), 'a@b.com')
    await user.type(screen.getByPlaceholderText('8文字以上'), 'password123')
    await user.click(screen.getAllByRole('button', { name: '新規登録' })[1])
    await waitFor(() => {
      expect(mockAuth.signup).toHaveBeenCalledWith({ email: 'a@b.com', password: 'password123' })
      expect(mockAuth.login).toHaveBeenCalled()
    })
  })

  it('calls gameStart on セット開始 click', async () => {
    mockGame.gameStatus.mockResolvedValue({ ...defaultStatus, status: 'FINISHED' })
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('セット開始')).toBeInTheDocument()
    })
    await user.click(screen.getByText('セット開始'))
    await waitFor(() => {
      expect(mockGame.gameStart).toHaveBeenCalled()
    })
  })

  it('calls gameSelect on 勝負 click', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('勝負')).toBeInTheDocument()
    })
    await user.click(screen.getByText('勝負'))
    await waitFor(() => {
      expect(mockGame.gameSelect).toHaveBeenCalledWith({ sessionId: 1, ver: 1 })
    })
  })

  it('calls gameResetSet on リセット click', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('リセット')).toBeInTheDocument()
    })
    await user.click(screen.getByText('リセット'))
    await waitFor(() => {
      expect(mockGame.gameResetSet).toHaveBeenCalledWith({ ver: 1 })
    })
  })

  it('calls logout on ログアウト click', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('ログアウト')).toBeInTheDocument()
    })
    await user.click(screen.getByText('ログアウト'))
    await waitFor(() => {
      expect(mockAuth.logout).toHaveBeenCalled()
    })
  })

  it('shows internal error on non-ApiException', async () => {
    mockGame.gameStatus.mockResolvedValue({ ...defaultStatus, status: 'FINISHED' })
    mockGame.gameStart.mockRejectedValue(new Error('network'))
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('セット開始')).toBeInTheDocument()
    })
    await user.click(screen.getByText('セット開始'))
    await waitFor(() => {
      expect(screen.getByText('エラーが発生しました。')).toBeInTheDocument()
    })
  })

  it('shows error on ApiException', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    mockAuth.login.mockRejectedValue(new ApiException({ code: 'unauthorized', message: 'x', status: 401 }))
    const user = userEvent.setup()
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await waitFor(() => {
      expect(screen.getByPlaceholderText('user@example.com')).toBeInTheDocument()
    })
    await user.type(screen.getByPlaceholderText('user@example.com'), 'a@b.com')
    await user.type(screen.getByPlaceholderText('8文字以上'), 'password123')
    await user.click(screen.getAllByRole('button', { name: 'ログイン' })[1])
    await waitFor(() => {
      expect(screen.getByText('ログインが必要です。')).toBeInTheDocument()
    })
  })

  it('clears notice on 閉じる click', async () => {
    mockGame.gameStatus.mockRejectedValue(new Error('unauthorized'))
    mockAuth.login.mockRejectedValue(new ApiException({ code: 'unauthorized', message: 'x', status: 401 }))
    const user = userEvent.setup()
    render(<App />)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50))
    })
    await user.type(screen.getByPlaceholderText('user@example.com'), 'a@b.com')
    await user.type(screen.getByPlaceholderText('8文字以上'), 'password123')
    await user.click(screen.getAllByRole('button', { name: 'ログイン' })[1])
    await waitFor(() => {
      expect(screen.getByText('閉じる')).toBeInTheDocument()
    })
    await user.click(screen.getByText('閉じる'))
    await waitFor(() => {
      expect(screen.queryByText('閉じる')).not.toBeInTheDocument()
    })
  })

  it('shows 表示を更新 and calls refreshStatus', async () => {
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('表示を更新')).toBeInTheDocument()
    })
    await user.click(screen.getByText('表示を更新'))
    await waitFor(() => {
      expect(mockGame.gameStatus).toHaveBeenCalledTimes(2)
    })
  })

  it('shows イカサマ button when DEALER mode and not cheated', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      mode: 'DEALER',
      cheated: false,
      cheat_reserved: false,
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('イカサマ')).toBeInTheDocument()
    })
  })

  it('shows イカサマ使用済み when DEALER and cheated', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      mode: 'DEALER',
      cheated: true,
      cheat_reserved: false,
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('イカサマ使用済み')).toBeInTheDocument()
    })
  })

  it('shows モード切替 when FINISHED', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      status: 'FINISHED',
      mode: 'PLAYER',
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText(/モード切替/)).toBeInTheDocument()
    })
  })

  it('calls gameChangeMode on モード切替 click', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      status: 'FINISHED',
      mode: 'PLAYER',
    })
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText(/モード切替/)).toBeInTheDocument()
    })
    await user.click(screen.getByText(/モード切替/))
    await waitFor(() => {
      expect(mockGame.gameChangeMode).toHaveBeenCalledWith({ mode: 'DEALER', ver: 1 })
    })
  })

  it('shows history when history exists', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      history: [
        {
          round: 1,
          player_card: 7,
          dealer_card: 10,
          result: 'DEALER_WIN',
          consecutive_draws: 0,
          cheat_used: false,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('履歴表示')).toBeInTheDocument()
    })
    expect(screen.getByText((_, el) => el?.classList?.contains('historyRound') && el?.textContent === 'ラウンド 1')).toBeInTheDocument()
    expect(screen.getByText('7')).toBeInTheDocument()
    expect(screen.getByText('10')).toBeInTheDocument()
  })

  it('shows まだ履歴がありません when no history', async () => {
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('まだ履歴がありません')).toBeInTheDocument()
    })
  })

  it('retries on version_conflict for start', async () => {
    mockGame.gameStatus
      .mockResolvedValueOnce({ ...defaultStatus, status: 'FINISHED' })
      .mockResolvedValueOnce({ ...defaultStatus, status: 'FINISHED', ver: 2 })
      .mockResolvedValueOnce({ ...defaultStatus, status: 'FINISHED', ver: 3 })
    mockGame.gameStart
      .mockRejectedValueOnce(new ApiException({ code: 'version_conflict', message: 'x', status: 409 }))
      .mockResolvedValueOnce({ session_id: 1, mode: 'PLAYER', player_wins: 0, dealer_wins: 0, ver: 3 })
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('セット開始')).toBeInTheDocument()
    })
    await user.click(screen.getByText('セット開始'))
    await waitFor(() => {
      expect(mockGame.gameStart).toHaveBeenCalledTimes(2)
    })
  })

  it('calls gameCheat before gameSelect when cheat pending', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      mode: 'DEALER',
      cheated: false,
      cheat_reserved: false,
    })
    const user = userEvent.setup()
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('イカサマ')).toBeInTheDocument()
    })
    await user.click(screen.getByText('イカサマ'))
    await user.click(screen.getByText('勝負'))
    await waitFor(() => {
      expect(mockGame.gameCheat).toHaveBeenCalled()
      expect(mockGame.gameSelect).toHaveBeenCalledWith({ sessionId: 1, ver: 2 })
    })
  })

  it('shows Win/Lose when FINISHED with history', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      status: 'FINISHED',
      player_wins: 2,
      dealer_wins: 0,
      history: [
        {
          round: 1,
          player_card: 10,
          dealer_card: 5,
          result: 'PLAYER_WIN',
          consecutive_draws: 0,
          cheat_used: false,
        },
        {
          round: 2,
          player_card: 13,
          dealer_card: 7,
          result: 'PLAYER_WIN',
          consecutive_draws: 0,
          cheat_used: false,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('セット開始')).toBeInTheDocument()
    })
    expect(screen.getByText('Win')).toBeInTheDocument()
  })

  it('shows history with DRAW and consecutive_draws', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      history: [
        {
          round: 1,
          player_card: 5,
          dealer_card: 5,
          result: 'DRAW',
          consecutive_draws: 2,
          cheat_used: false,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('引き分け連続:')).toBeInTheDocument()
    })
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(document.querySelector('.historyItem:not(.win):not(.lose)')).toBeInTheDocument()
  })

  it('shows history with DRAW when dealer mode', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      mode: 'DEALER',
      history: [
        {
          round: 1,
          player_card: 5,
          dealer_card: 5,
          result: 'DRAW',
          consecutive_draws: 0,
          cheat_used: false,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('引き分け')).toBeInTheDocument()
    })
  })

  it('shows チート使用 in history when cheat_used', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      history: [
        {
          round: 1,
          player_card: 10,
          dealer_card: 5,
          result: 'PLAYER_WIN',
          consecutive_draws: 0,
          cheat_used: true,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('チート使用')).toBeInTheDocument()
    })
  })

  it('shows history from dealer perspective when mode is DEALER', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      mode: 'DEALER',
      status: 'FINISHED',
      history: [
        {
          round: 1,
          player_card: 10,
          dealer_card: 5,
          result: 'PLAYER_WIN',
          consecutive_draws: 0,
          cheat_used: false,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('履歴表示')).toBeInTheDocument()
    })
    expect(screen.getByText('相手の勝ち')).toBeInTheDocument()
  })

  it('shows あなたの勝ち in history when dealer mode and DEALER_WIN', async () => {
    mockGame.gameStatus.mockResolvedValue({
      ...defaultStatus,
      mode: 'DEALER',
      status: 'FINISHED',
      history: [
        {
          round: 1,
          player_card: 5,
          dealer_card: 10,
          result: 'DEALER_WIN',
          consecutive_draws: 0,
          cheat_used: false,
        },
      ],
    })
    render(<App />)
    await waitFor(() => {
      expect(screen.getByText('履歴表示')).toBeInTheDocument()
    })
    expect(screen.getByText('あなたの勝ち')).toBeInTheDocument()
  })
})
