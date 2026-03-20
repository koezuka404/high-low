package usecase

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"backend/model"

	"gorm.io/gorm"
)

type mockGameRepository struct {
	gameByUser map[uint]*model.Game
	nextID     uint
	getErr     error
	updateErr  error
	createErr  error
}

func newMockGameRepository() *mockGameRepository {
	return &mockGameRepository{gameByUser: map[uint]*model.Game{}, nextID: 1}
}

func (m *mockGameRepository) Create(game *model.Game) error {
	if m.createErr != nil {
		return m.createErr
	}
	if game.ID == 0 {
		game.ID = m.nextID
		m.nextID++
	}
	cp := *game
	m.gameByUser[game.UserID] = &cp
	return nil
}

func (m *mockGameRepository) GetGameByID(id uint) (*model.Game, error) {
	for _, g := range m.gameByUser {
		if g != nil && g.ID == id {
			cp := *g
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockGameRepository) GetGameByUserID(userID uint) (*model.Game, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	g := m.gameByUser[userID]
	if g == nil {
		return nil, nil
	}
	cp := *g
	return &cp, nil
}

func (m *mockGameRepository) UpdateWithVersion(game *model.Game, expectedVer int64) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	cur := m.gameByUser[game.UserID]
	if cur == nil {
		return gorm.ErrRecordNotFound
	}
	if cur.Ver != expectedVer {
		return gorm.ErrRecordNotFound
	}
	cp := *game
	m.gameByUser[game.UserID] = &cp
	return nil
}

type mockRoundLogRepository struct {
	logsByGame map[uint][]model.GameRoundLog
	createErr  error
	countErr   error
	deleteErr  error
	getErr     error
}

func newMockRoundLogRepository() *mockRoundLogRepository {
	return &mockRoundLogRepository{logsByGame: map[uint][]model.GameRoundLog{}}
}

func (m *mockRoundLogRepository) Create(log *model.GameRoundLog) error {
	if m.createErr != nil {
		return m.createErr
	}
	if log == nil {
		return errors.New("nil log")
	}
	cp := *log
	m.logsByGame[log.GameID] = append(m.logsByGame[log.GameID], cp)
	return nil
}

func (m *mockRoundLogRepository) GetRoundLogsByGameID(gameID uint) ([]model.Round, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	logs := m.logsByGame[gameID]
	out := make([]model.Round, 0, len(logs))
	for _, l := range logs {
		out = append(out, model.Round{
			Number:           l.Number,
			PlayerCard:       l.PlayerCard,
			DealerCard:       l.DealerCard,
			Result:           l.Result,
			ConsecutiveDraws: l.ConsecutiveDraws,
			CheatUsed:        l.CheatUsed,
			PlayedAt:         l.PlayedAt,
		})
	}
	return out, nil
}

func (m *mockRoundLogRepository) GetRoundLogCountByGameID(gameID uint) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return int64(len(m.logsByGame[gameID])), nil
}

func (m *mockRoundLogRepository) DeleteByGameID(gameID uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.logsByGame, gameID)
	return nil
}

func TestGameUsecase_Start_NewGame(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game, err := gu.Start(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if game == nil || game.ID == 0 {
		t.Fatalf("expected game created, got %+v", game)
	}
	if game.Status != model.GameStatusInProgress {
		t.Fatalf("status=%s", game.Status)
	}
	if game.Mode != model.GameModePlayer {
		t.Fatalf("mode=%s", game.Mode)
	}
	if game.Ver != 1 {
		t.Fatalf("ver=%d", game.Ver)
	}
}

func TestGameUsecase_Start_NewGame_CreateError(t *testing.T) {
	gr := newMockGameRepository()
	gr.createErr = errors.New("create error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, err := gu.Start(1, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Start_GameRepoError(t *testing.T) {
	gr := newMockGameRepository()
	gr.getErr = errors.New("get error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, err := gu.Start(1, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Start_InProgressRejected(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	if _, err := gu.Start(1, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := gu.Start(1, nil); err == nil {
		t.Fatal("expected error")
	} else if err != errGameAlreadyStarted {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Start_InvalidState(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusNotStarted, Mode: model.GameModePlayer, Ver: 1}
	if _, err := gu.Start(1, nil); err == nil {
		t.Fatal("expected error")
	} else if err != errGameNotStarted {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Start_Restart_VersionConflict_VerNilExpected0(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	gr.gameByUser[1] = &model.Game{
		ID:     10,
		UserID: 1,
		Status: model.GameStatusFinished,
		Mode:   model.GameModePlayer,
		Ver:    5,
	}
	if _, err := gu.Start(1, nil); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Start_Restart_VersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	finished := &model.Game{
		ID:     10,
		UserID: 1,
		Status: model.GameStatusFinished,
		Mode:   model.GameModePlayer,
		Ver:    5,
	}
	gr.gameByUser[1] = finished

	ver := int64(4)
	if _, err := gu.Start(1, &ver); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Start_Restart_Success(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	finished := &model.Game{
		ID:               10,
		UserID:           1,
		Status:           model.GameStatusFinished,
		Mode:             model.GameModeDealer,
		PlayerWins:       2,
		DealerWins:       0,
		ConsecutiveDraws: 3,
		Cheated:          true,
		CheatReserved:    true,
		Ver:              5,
		PlayerUsedCards:  model.IntSlice{1, 2},
		DealerUsedCards:  model.IntSlice{3, 4},
	}
	gr.gameByUser[1] = finished
	_ = rr.Create(&model.GameRoundLog{GameID: 10, Number: 1, PlayerCard: 1, DealerCard: 2, Result: model.RoundResultDealerWin, ConsecutiveDraws: 0, CheatUsed: false, PlayedAt: timeNowForTest()})

	ver := int64(5)
	game, err := gu.Start(1, &ver)
	if err != nil {
		t.Fatal(err)
	}
	if game.Status != model.GameStatusInProgress || game.Ver != 6 {
		t.Fatalf("unexpected status/ver: %s %d", game.Status, game.Ver)
	}
	if game.PlayerWins != 0 || game.DealerWins != 0 || game.ConsecutiveDraws != 0 {
		t.Fatalf("unexpected wins/draws: %d %d %d", game.PlayerWins, game.DealerWins, game.ConsecutiveDraws)
	}
	if game.Cheated || game.CheatReserved || game.CheatCard != nil {
		t.Fatalf("unexpected cheat state: %+v", game)
	}
	if len(game.PlayerUsedCards) != 0 || len(game.DealerUsedCards) != 0 {
		t.Fatalf("expected used cards reset")
	}
}

func TestGameUsecase_Start_Restart_DeleteLogsError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	rr.deleteErr = errors.New("delete error")
	gu := NewGameUsecase(gr, rr)

	finished := &model.Game{
		ID:     10,
		UserID: 1,
		Status: model.GameStatusFinished,
		Mode:   model.GameModePlayer,
		Ver:    5,
	}
	gr.gameByUser[1] = finished

	ver := int64(5)
	if _, err := gu.Start(1, &ver); err == nil {
		t.Fatal("expected error")
	} else if err.Error() != "delete error" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Start_Restart_UpdateVersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	finished := &model.Game{
		ID:     10,
		UserID: 1,
		Status: model.GameStatusFinished,
		Mode:   model.GameModePlayer,
		Ver:    5,
	}
	gr.gameByUser[1] = finished

	gr.updateErr = gorm.ErrRecordNotFound
	ver := int64(5)
	if _, err := gu.Start(1, &ver); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Start_Restart_UpdateOtherError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	finished := &model.Game{
		ID:     10,
		UserID: 1,
		Status: model.GameStatusFinished,
		Mode:   model.GameModePlayer,
		Ver:    5,
	}
	gr.gameByUser[1] = finished

	gr.updateErr = errors.New("update error")
	ver := int64(5)
	if _, err := gu.Start(1, &ver); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Status_ReturnsRounds(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:     10,
		UserID: 1,
		Status: model.GameStatusInProgress,
		Mode:   model.GameModePlayer,
		Ver:    1,
	}
	gr.gameByUser[1] = game
	_ = rr.Create(&model.GameRoundLog{GameID: 10, Number: 1, PlayerCard: 7, DealerCard: 10, Result: model.RoundResultDealerWin, ConsecutiveDraws: 0, CheatUsed: false, PlayedAt: timeNowForTest()})

	got, err := gu.Status(1)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %+v", got)
	}
}

func TestGameUsecase_Status_NoGameReturnsNil(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game, err := gu.Status(1)
	if err != nil {
		t.Fatal(err)
	}
	if game != nil {
		t.Fatalf("expected nil game, got %+v", game)
	}
}

func TestGameUsecase_Status_RoundRepoError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	rr.getErr = errors.New("get rounds error")
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, Ver: 1}
	gr.gameByUser[1] = game
	if _, err := gu.Status(1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Status_GameRepoError(t *testing.T) {
	gr := newMockGameRepository()
	gr.getErr = errors.New("get game error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, err := gu.Status(1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Select_Draw5ResetsUsedCards(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:               10,
		UserID:           1,
		Status:           model.GameStatusInProgress,
		Mode:             model.GameModePlayer,
		PlayerWins:       0,
		DealerWins:       0,
		ConsecutiveDraws: 4,
		Cheated:          false,
		CheatReserved:    false,
		CheatCard:        nil,
		Ver:              7,
		PlayerUsedCards:  model.IntSlice{1, 2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13},
		DealerUsedCards:  model.IntSlice{1, 2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13},
	}
	gr.gameByUser[1] = game

	updated, round, err := gu.Select(1, 10, 7)
	if err != nil {
		t.Fatal(err)
	}
	if round == nil {
		t.Fatal("expected round")
	}
	if round.Result != model.RoundResultDraw {
		t.Fatalf("expected DRAW, got %s", round.Result)
	}
	if updated.ConsecutiveDraws != 0 {
		t.Fatalf("expected consecutive_draws reset to 0, got %d", updated.ConsecutiveDraws)
	}
	if len(updated.PlayerUsedCards) != 0 || len(updated.DealerUsedCards) != 0 {
		t.Fatalf("expected used cards reset, got player=%v dealer=%v", updated.PlayerUsedCards, updated.DealerUsedCards)
	}
}

func TestGameUsecase_Select_PlayerWin_FinishesAt2Wins(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		PlayerWins:      1,
		DealerWins:      0,
		ConsecutiveDraws: 2,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		DealerUsedCards: model.IntSlice{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
	}
	gr.gameByUser[1] = game

	updated, round, err := gu.Select(1, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if round.Result != model.RoundResultPlayerWin {
		t.Fatalf("expected PLAYER_WIN got %s", round.Result)
	}
	if updated.PlayerWins != 2 || updated.Status != model.GameStatusFinished {
		t.Fatalf("expected finish at 2 wins, got wins=%d status=%s", updated.PlayerWins, updated.Status)
	}
	if updated.ConsecutiveDraws != 0 {
		t.Fatalf("expected consecutive_draws reset, got %d", updated.ConsecutiveDraws)
	}
}

func TestGameUsecase_Select_DealerWin(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		PlayerWins:      0,
		DealerWins:      0,
		ConsecutiveDraws: 2,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
		DealerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	gr.gameByUser[1] = game

	updated, round, err := gu.Select(1, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if round.Result != model.RoundResultDealerWin {
		t.Fatalf("expected DEALER_WIN got %s", round.Result)
	}
	if updated.DealerWins != 1 || updated.ConsecutiveDraws != 0 {
		t.Fatalf("unexpected dealer_wins/draws: %d %d", updated.DealerWins, updated.ConsecutiveDraws)
	}
}

func TestGameUsecase_Select_GameRepoError(t *testing.T) {
	gr := newMockGameRepository()
	gr.getErr = errors.New("get game error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Select_UpdateOtherError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		DealerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	gr.gameByUser[1] = game
	gr.updateErr = errors.New("update error")
	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Select_InvalidInput(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	if _, _, err := gu.Select(1, 0, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errInvalidInput {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_SessionNotFound(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errSessionNotFound {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_Forbidden(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{ID: 99, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, Ver: 1}
	gr.gameByUser[1] = game

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errForbidden {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_NotInProgress(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModePlayer, Ver: 1}
	gr.gameByUser[1] = game

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errGameNotStarted {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_VersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, Ver: 2}
	gr.gameByUser[1] = game

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_NoSelectableCard(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
		DealerUsedCards: model.IntSlice{},
	}
	gr.gameByUser[1] = game

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errNoSelectableCard {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_CountError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	rr.countErr = errors.New("count error")
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		DealerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	gr.gameByUser[1] = game

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Select_UpdateVersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		DealerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	gr.gameByUser[1] = game
	gr.updateErr = gorm.ErrRecordNotFound

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Select_LogCreateError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	rr.createErr = errors.New("create log error")
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:              10,
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		DealerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	gr.gameByUser[1] = game

	if _, _, err := gu.Select(1, 10, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Cheat_ReservesMaxDealerRemaining(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:            10,
		UserID:        1,
		Status:        model.GameStatusInProgress,
		Mode:          model.GameModeDealer,
		Cheated:       false,
		CheatReserved: false,
		CheatCard:     nil,
		Ver:           3,
		DealerUsedCards: model.IntSlice{
			1, 2, 4, 5, 6, 8, 9, 10, 12, 13,
		},
	}
	gr.gameByUser[1] = game

	updated, err := gu.Cheat(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.CheatReserved || !updated.Cheated {
		t.Fatalf("expected cheat reserved+cheated true, got reserved=%v cheated=%v", updated.CheatReserved, updated.Cheated)
	}
	if updated.CheatCard == nil || *updated.CheatCard != 11 {
		t.Fatalf("expected cheat_card=11, got %v", updated.CheatCard)
	}
}

func TestGameUsecase_Cheat_SessionNotFound(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errSessionNotFound {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_NotInProgress(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModeDealer, Ver: 1}
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errGameNotStarted {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_NotAllowedMode(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, Ver: 1}
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errCheatNotAllowed {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_AlreadyUsed(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Cheated: true, Ver: 1}
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errCheatAlreadyUsed {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_VersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Cheated: false, Ver: 2}
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_NotAvailable(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{
		ID:             10,
		UserID:         1,
		Status:         model.GameStatusInProgress,
		Mode:           model.GameModeDealer,
		Cheated:        false,
		Ver:            1,
		DealerUsedCards: model.IntSlice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
	}
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errCheatNotAvailable {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_UpdateVersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Cheated: false, Ver: 1}
	gr.updateErr = gorm.ErrRecordNotFound
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_Cheat_GameRepoError(t *testing.T) {
	gr := newMockGameRepository()
	gr.getErr = errors.New("get game error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Cheat_UpdateOtherError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Cheated: false, Ver: 1}
	gr.updateErr = errors.New("update error")
	if _, err := gu.Cheat(1, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_ResetSet_SessionNotFound(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	if _, err := gu.ResetSet(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errSessionNotFound {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_ResetSet_GameRepoError(t *testing.T) {
	gr := newMockGameRepository()
	gr.getErr = errors.New("get game error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	if _, err := gu.ResetSet(1, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_ResetSet_VersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Ver: 2}

	if _, err := gu.ResetSet(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_ResetSet_DeleteLogsError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	rr.deleteErr = errors.New("delete error")
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Ver: 1}

	if _, err := gu.ResetSet(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err.Error() != "delete error" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_ResetSet_UpdateVersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Ver: 1}
	gr.updateErr = gorm.ErrRecordNotFound

	if _, err := gu.ResetSet(1, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_ResetSet_UpdateOtherError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, Ver: 1}
	gr.updateErr = errors.New("update error")

	if _, err := gu.ResetSet(1, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_ResetSet_Success(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	cheatCard := 13
	gr.gameByUser[1] = &model.Game{
		ID:               10,
		UserID:           1,
		Status:           model.GameStatusFinished,
		Mode:             model.GameModeDealer,
		PlayerWins:       2,
		DealerWins:       1,
		ConsecutiveDraws: 3,
		Cheated:          true,
		CheatReserved:    true,
		CheatCard:        &cheatCard,
		Ver:              1,
		PlayerUsedCards:  model.IntSlice{1, 2, 3},
		DealerUsedCards:  model.IntSlice{4, 5, 6},
	}
	_ = rr.Create(&model.GameRoundLog{
		GameID:           10,
		Number:           1,
		PlayerCard:       7,
		DealerCard:       10,
		Result:           model.RoundResultDealerWin,
		ConsecutiveDraws: 0,
		CheatUsed:        true,
		PlayedAt:         timeNowForTest(),
	})

	got, err := gu.ResetSet(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.GameStatusInProgress {
		t.Fatalf("unexpected status: %s", got.Status)
	}
	if got.Mode != model.GameModeDealer {
		t.Fatalf("expected mode preserved, got %s", got.Mode)
	}
	if got.PlayerWins != 0 || got.DealerWins != 0 || got.ConsecutiveDraws != 0 {
		t.Fatalf("unexpected wins/draws: %d %d %d", got.PlayerWins, got.DealerWins, got.ConsecutiveDraws)
	}
	if got.Cheated || got.CheatReserved || got.CheatCard != nil {
		t.Fatalf("unexpected cheat state: %+v", got)
	}
	if len(got.PlayerUsedCards) != 0 || len(got.DealerUsedCards) != 0 || len(got.Rounds) != 0 {
		t.Fatalf("expected cards/rounds reset")
	}
	if got.Ver != 2 {
		t.Fatalf("expected ver=2, got %d", got.Ver)
	}
	if c, err := rr.GetRoundLogCountByGameID(10); err != nil || c != 0 {
		t.Fatalf("expected logs deleted, got count=%d err=%v", c, err)
	}
}

func TestGameUsecase_ChangeMode_Errors(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	if _, err := gu.ChangeMode(1, model.GameModeDealer, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errGameNotStarted {
		t.Fatalf("unexpected err: %v", err)
	}

	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, Ver: 1}
	if _, err := gu.ChangeMode(1, model.GameModeDealer, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errGameNotFinished {
		t.Fatalf("unexpected err: %v", err)
	}

	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModePlayer, Ver: 1}
	if _, err := gu.ChangeMode(1, model.GameMode("BAD"), 1); err == nil {
		t.Fatal("expected error")
	} else if err != errInvalidMode {
		t.Fatalf("unexpected err: %v", err)
	}

	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModePlayer, Ver: 2}
	if _, err := gu.ChangeMode(1, model.GameModeDealer, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_ChangeMode_UpdateVersionConflict(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModePlayer, Ver: 1}
	gr.updateErr = gorm.ErrRecordNotFound
	if _, err := gu.ChangeMode(1, model.GameModeDealer, 1); err == nil {
		t.Fatal("expected error")
	} else if err != errVersionConflict {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestGameUsecase_ChangeMode_Success(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModePlayer, Ver: 1}
	game, err := gu.ChangeMode(1, model.GameModeDealer, 1)
	if err != nil {
		t.Fatal(err)
	}
	if game.Mode != model.GameModeDealer || game.Ver != 2 {
		t.Fatalf("unexpected mode/ver: %s %d", game.Mode, game.Ver)
	}
}

func TestGameUsecase_ChangeMode_GameRepoError(t *testing.T) {
	gr := newMockGameRepository()
	gr.getErr = errors.New("get game error")
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	if _, err := gu.ChangeMode(1, model.GameModeDealer, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_ChangeMode_UpdateOtherError(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)
	gr.gameByUser[1] = &model.Game{ID: 10, UserID: 1, Status: model.GameStatusFinished, Mode: model.GameModePlayer, Ver: 1}
	gr.updateErr = errors.New("update error")
	if _, err := gu.ChangeMode(1, model.GameModeDealer, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameUsecase_Select_ConsumesCheatReservation(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	cheatCard := 13
	game := &model.Game{
		ID:            10,
		UserID:        1,
		Status:        model.GameStatusInProgress,
		Mode:          model.GameModeDealer,
		PlayerWins:    0,
		DealerWins:    0,
		Ver:           5,
		CheatReserved: true,
		Cheated:       true,
		CheatCard:     &cheatCard,
		PlayerUsedCards: model.IntSlice{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
		DealerUsedCards: model.IntSlice{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	}
	gr.gameByUser[1] = game

	rand.Seed(1)
	updated, round, err := gu.Select(1, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	if round.DealerCard != 13 {
		t.Fatalf("expected dealer_card=13, got %d", round.DealerCard)
	}
	if updated.CheatReserved || updated.CheatCard != nil {
		t.Fatalf("expected cheat reservation consumed, got reserved=%v card=%v", updated.CheatReserved, updated.CheatCard)
	}
}

func TestGameUsecase_Select_CheatReservedButNilCardDoesNotConsume(t *testing.T) {
	gr := newMockGameRepository()
	rr := newMockRoundLogRepository()
	gu := NewGameUsecase(gr, rr)

	game := &model.Game{
		ID:            10,
		UserID:        1,
		Status:        model.GameStatusInProgress,
		Mode:          model.GameModeDealer,
		Ver:           1,
		CheatReserved: true,
		CheatCard:     nil,
		PlayerUsedCards: model.IntSlice{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
		DealerUsedCards: model.IntSlice{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	}
	gr.gameByUser[1] = game

	rand.Seed(1)
	updated, round, err := gu.Select(1, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if round == nil {
		t.Fatal("expected round")
	}
	if !updated.CheatReserved {
		t.Fatal("expected cheat_reserved to remain true")
	}
}

func timeNowForTest() time.Time {
	return time.Unix(1712345678, 0)
}

