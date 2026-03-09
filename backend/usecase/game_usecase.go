package usecase

import (
	"backend/model"
	"backend/repository"
	"errors"
	"math/rand"
	"time"
)

type IGameUsecase interface {
	StartGame(userID uint, mode string) (*model.Game, error)
	SelectCard(userID uint, useCheat bool) (*model.Game, *model.Round, error)
	GetGameState(userID uint) (*model.Game, error)
}

type gameUsecase struct {
	gr repository.IGameRepository
}

func NewGameUsecase(gr repository.IGameRepository) IGameUsecase {
	return &gameUsecase{gr: gr}
}

func (gu *gameUsecase) StartGame(userID uint, mode string) (*model.Game, error) {
	if mode != string(model.GameModePlayer) && mode != string(model.GameModeDealer) {
		return nil, errors.New("invalid mode")
	}

	game := &model.Game{
		UserID:          userID,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameMode(mode),
		PlayerScore:     0,
		DealerScore:     0,
		DrawCount:       0,
		CheatUsed:       false,
		CurrentRound:    0,
		PlayerUsedCards: []int{},
		DealerUsedCards: []int{},
		Rounds:          []model.Round{},
	}

	if err := gu.gr.Create(game); err != nil {
		return nil, err
	}

	return game, nil
}

func (gu *gameUsecase) GetGameState(userID uint) (*model.Game, error) {
	game, err := gu.gr.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	return game, nil
}

func (gu *gameUsecase) SelectCard(userID uint, useCheat bool) (*model.Game, *model.Round, error) {
	game, err := gu.gr.FindByUserID(userID)
	if err != nil {
		return nil, nil, err
	}

	if game.Status != model.GameStatusInProgress {
		return nil, nil, errors.New("game is not in progress")
	}

	playerCard := gu.drawRandomCard(game.PlayerUsedCards)
	dealerCard := gu.drawRandomCard(game.DealerUsedCards)

	roundCheatUsed := false

	if game.Mode == model.GameModeDealer && useCheat {
		if game.CheatUsed {
			return nil, nil, errors.New("cheat already used")
		}
		dealerCard = gu.drawCheatCard(game.DealerUsedCards)
		game.CheatUsed = true
		roundCheatUsed = true
	}

	if playerCard == 0 || dealerCard == 0 {
		return nil, nil, errors.New("no selectable card")
	}

	game.PlayerUsedCards = append(game.PlayerUsedCards, playerCard)
	game.DealerUsedCards = append(game.DealerUsedCards, dealerCard)
	game.CurrentRound++

	result := judgeRound(playerCard, dealerCard)

	switch result {
	case model.RoundResultPlayerWin:
		game.PlayerScore++
		game.DrawCount = 0
	case model.RoundResultDealerWin:
		game.DealerScore++
		game.DrawCount = 0
	case model.RoundResultDraw:
		game.DrawCount++
	}

	if game.DrawCount >= 5 {
		game.PlayerUsedCards = []int{}
		game.DealerUsedCards = []int{}
		game.DrawCount = 0
	}

	if game.PlayerScore >= 2 || game.DealerScore >= 2 {
		game.Status = model.GameStatusFinished
	}

	round := model.Round{
		Number:     game.CurrentRound,
		PlayerCard: playerCard,
		DealerCard: dealerCard,
		Result:     result,
		CheatUsed:  roundCheatUsed,
		PlayedAt:   time.Now(),
	}

	game.Rounds = append(game.Rounds, round)

	if err := gu.gr.Save(game); err != nil {
		return nil, nil, err
	}

	return game, &round, nil
}

func judgeRound(playerCard, dealerCard int) model.RoundResult {
	if playerCard > dealerCard {
		return model.RoundResultPlayerWin
	}
	if dealerCard > playerCard {
		return model.RoundResultDealerWin
	}
	return model.RoundResultDraw
}

func (gu *gameUsecase) drawRandomCard(used []int) int {
	candidates := availableCards(used, 1, 13)
	if len(candidates) == 0 {
		return 0
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return candidates[r.Intn(len(candidates))]
}

func (gu *gameUsecase) drawCheatCard(used []int) int {
	candidates := availableCards(used, 11, 13)
	if len(candidates) == 0 {
		return 0
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return candidates[r.Intn(len(candidates))]
}

func availableCards(used []int, min, max int) []int {
	usedMap := map[int]bool{}
	for _, v := range used {
		usedMap[v] = true
	}

	result := []int{}
	for i := min; i <= max; i++ {
		if !usedMap[i] {
			result = append(result, i)
		}
	}
	return result
}
