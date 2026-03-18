package usecase

import (
	"backend/domain"
	"backend/model"
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type IGameUsecase interface {
	Start(userID uint, ver *int64) (*model.Game, error)
	Select(userID uint, sessionID uint, ver int64) (*model.Game, *model.Round, error)
	Cheat(userID uint, ver int64) (*model.Game, error)
	ResetSet(userID uint, ver int64) (*model.Game, error)
	ChangeMode(userID uint, mode model.GameMode, ver int64) (*model.Game, error)
	Status(userID uint) (*model.Game, error)
}

type gameUsecase struct {
	gr IGameRepository
	rr IGameRoundLogRepository
}

func NewGameUsecase(gr IGameRepository, rr IGameRoundLogRepository) IGameUsecase {
	return &gameUsecase{gr: gr, rr: rr}
}

func (gu *gameUsecase) Start(userID uint, ver *int64) (*model.Game, error) {
	game, err := gu.gr.GetGameByUserID(userID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		newGame := &model.Game{
			UserID:           userID,
			Status:           model.GameStatusInProgress,
			Mode:             model.GameModePlayer,
			PlayerWins:       0,
			DealerWins:       0,
			ConsecutiveDraws: 0,
			Cheated:          false,
			CheatReserved:    false,
			CheatCard:        nil,
			Ver:              1,
			PlayerUsedCards:  model.IntSlice{},
			DealerUsedCards:  model.IntSlice{},
			Rounds:           []model.Round{},
		}
		if err := gu.gr.Create(newGame); err != nil {
			return nil, err
		}
		return newGame, nil
	}
	if game.Status == model.GameStatusInProgress {
		return nil, errGameAlreadyStarted
	}
	if game.Status != model.GameStatusFinished {
		return nil, errGameNotStarted
	}
	expectedVer := int64(0)
	if ver != nil {
		expectedVer = *ver
	}
	if game.Ver != expectedVer {
		return nil, errVersionConflict
	}
	if err := gu.rr.DeleteByGameID(game.ID); err != nil {
		return nil, err
	}
	game.Status = model.GameStatusInProgress
	game.PlayerWins = 0
	game.DealerWins = 0
	game.ConsecutiveDraws = 0
	game.Cheated = false
	game.CheatReserved = false
	game.CheatCard = nil
	game.PlayerUsedCards = model.IntSlice{}
	game.DealerUsedCards = model.IntSlice{}
	game.Rounds = []model.Round{}
	game.Ver = game.Ver + 1
	game.UpdatedAt = time.Now()
	if err := gu.gr.UpdateWithVersion(game, expectedVer); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errVersionConflict
		}
		return nil, err
	}
	return game, nil
}

func (gu *gameUsecase) Status(userID uint) (*model.Game, error) {
	game, err := gu.gr.GetGameByUserID(userID)
	if err != nil {
		return nil, err
	}
	if game != nil {
		rounds, err := gu.rr.GetRoundLogsByGameID(game.ID)
		if err != nil {
			return nil, err
		}
		game.Rounds = rounds
	}
	return game, nil
}

func (gu *gameUsecase) Select(userID uint, sessionID uint, ver int64) (*model.Game, *model.Round, error) {
	if sessionID == 0 {
		return nil, nil, errInvalidInput
	}
	game, err := gu.gr.GetGameByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	if game == nil {
		return nil, nil, errSessionNotFound
	}
	if game.ID != sessionID {
		return nil, nil, errForbidden
	}
	if game.Status != model.GameStatusInProgress {
		return nil, nil, errGameNotStarted
	}
	if game.Ver != ver {
		return nil, nil, errVersionConflict
	}

	playerRemaining := domain.RemainingCards([]int(game.PlayerUsedCards))
	dealerRemaining := domain.RemainingCards([]int(game.DealerUsedCards))
	if len(playerRemaining) == 0 || len(dealerRemaining) == 0 {
		return nil, nil, errNoSelectableCard
	}

	playerCard := pickRandom(playerRemaining)
	var dealerCard int
	if game.CheatReserved && game.CheatCard != nil {
		dealerCard = *game.CheatCard
	} else {
		dealerCard = pickRandom(dealerRemaining)
	}

	game.PlayerUsedCards = append(game.PlayerUsedCards, playerCard)
	game.DealerUsedCards = append(game.DealerUsedCards, dealerCard)

	result := domain.JudgeRound(playerCard, dealerCard)
	roundCheatUsed := game.CheatReserved && game.CheatCard != nil

	switch result {
	case model.RoundResultPlayerWin:
		game.PlayerWins++
		game.ConsecutiveDraws = 0
	case model.RoundResultDealerWin:
		game.DealerWins++
		game.ConsecutiveDraws = 0
	case model.RoundResultDraw:
		game.ConsecutiveDraws++
	}

	if game.ConsecutiveDraws >= 5 {
		game.PlayerUsedCards = model.IntSlice{}
		game.DealerUsedCards = model.IntSlice{}
		game.ConsecutiveDraws = 0
	}

	if roundCheatUsed {
		game.CheatReserved = false
		game.CheatCard = nil
	}

	count, err := gu.rr.GetRoundLogCountByGameID(game.ID)
	if err != nil {
		return nil, nil, err
	}
	roundNum := int(count) + 1
	round := model.Round{
		Number:           roundNum,
		PlayerCard:       playerCard,
		DealerCard:       dealerCard,
		Result:           result,
		ConsecutiveDraws: game.ConsecutiveDraws,
		CheatUsed:        roundCheatUsed,
		PlayedAt:         time.Now(),
	}
	game.Rounds = append(game.Rounds, round)

	if game.PlayerWins >= 2 || game.DealerWins >= 2 {
		game.Status = model.GameStatusFinished
	}

	game.Ver = game.Ver + 1
	game.UpdatedAt = time.Now()
	if err := gu.gr.UpdateWithVersion(game, ver); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errVersionConflict
		}
		return nil, nil, err
	}
	log := &model.GameRoundLog{
		GameID:           game.ID,
		Number:           round.Number,
		PlayerCard:       round.PlayerCard,
		DealerCard:       round.DealerCard,
		Result:           round.Result,
		ConsecutiveDraws: round.ConsecutiveDraws,
		CheatUsed:        round.CheatUsed,
		PlayedAt:         round.PlayedAt,
	}
	if err := gu.rr.Create(log); err != nil {
		return nil, nil, err
	}
	return game, &round, nil
}

func (gu *gameUsecase) Cheat(userID uint, ver int64) (*model.Game, error) {
	game, err := gu.gr.GetGameByUserID(userID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errSessionNotFound
	}
	if game.Status != model.GameStatusInProgress {
		return nil, errGameNotStarted
	}
	if game.Mode != model.GameModeDealer {
		return nil, errCheatNotAllowed
	}
	if game.Cheated {
		return nil, errCheatAlreadyUsed
	}
	if game.Ver != ver {
		return nil, errVersionConflict
	}

	dealerRem := domain.RemainingCards([]int(game.DealerUsedCards))
	if len(dealerRem) == 0 {
		return nil, errCheatNotAvailable
	}
	cheatCard := domain.MaxInt(dealerRem)
	game.CheatCard = &cheatCard
	game.Cheated = true
	game.CheatReserved = true
	game.Ver = game.Ver + 1
	game.UpdatedAt = time.Now()
	if err := gu.gr.UpdateWithVersion(game, ver); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errVersionConflict
		}
		return nil, err
	}
	return game, nil
}

func (gu *gameUsecase) ResetSet(userID uint, ver int64) (*model.Game, error) {
	game, err := gu.gr.GetGameByUserID(userID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errSessionNotFound
	}
	if game.Ver != ver {
		return nil, errVersionConflict
	}
	if err := gu.rr.DeleteByGameID(game.ID); err != nil {
		return nil, err
	}

	// Reset current set progress. Keep the current mode.
	game.Status = model.GameStatusInProgress
	game.PlayerWins = 0
	game.DealerWins = 0
	game.ConsecutiveDraws = 0
	game.Cheated = false
	game.CheatReserved = false
	game.CheatCard = nil
	game.PlayerUsedCards = model.IntSlice{}
	game.DealerUsedCards = model.IntSlice{}
	game.Rounds = []model.Round{}
	game.Ver = game.Ver + 1
	game.UpdatedAt = time.Now()
	if err := gu.gr.UpdateWithVersion(game, ver); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errVersionConflict
		}
		return nil, err
	}
	return game, nil
}

func (gu *gameUsecase) ChangeMode(userID uint, mode model.GameMode, ver int64) (*model.Game, error) {
	game, err := gu.gr.GetGameByUserID(userID)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errGameNotStarted
	}
	if game.Status != model.GameStatusFinished {
		return nil, errGameNotFinished
	}
	if mode != model.GameModePlayer && mode != model.GameModeDealer {
		return nil, errInvalidMode
	}
	if game.Ver != ver {
		return nil, errVersionConflict
	}
	game.Mode = mode
	game.Ver = game.Ver + 1
	game.UpdatedAt = time.Now()
	if err := gu.gr.UpdateWithVersion(game, ver); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errVersionConflict
		}
		return nil, err
	}
	return game, nil
}

func pickRandom(cards []int) int {
	return cards[rand.Intn(len(cards))]
}
