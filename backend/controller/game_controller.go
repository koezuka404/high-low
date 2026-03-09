package controller

import (
	"backend/controller/dto"
	"backend/model"
	"backend/usecase"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type IGameController interface {
	StartGame(c echo.Context) error
	SelectCard(c echo.Context) error
	GetGameState(c echo.Context) error
}

type gameController struct {
	gu usecase.IGameUsecase
}

func NewGameController(gu usecase.IGameUsecase) IGameController {
	return &gameController{gu: gu}
}

func (gc *gameController) StartGame(c echo.Context) error {
	req := dto.StartGameRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	userID := c.Get("user_id").(uint)

	game, err := gc.gu.StartGame(userID, req.Mode)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	res := dto.StartGameResponse{
		ID:           game.ID,
		Status:       string(game.Status),
		Mode:         string(game.Mode),
		PlayerScore:  game.PlayerScore,
		DealerScore:  game.DealerScore,
		DrawCount:    game.DrawCount,
		CurrentRound: game.CurrentRound,
		CheatUsed:    game.CheatUsed,
	}

	return c.JSON(http.StatusCreated, res)
}

func (gc *gameController) SelectCard(c echo.Context) error {
	req := dto.SelectCardRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	userID := c.Get("user_id").(uint)

	game, round, err := gc.gu.SelectCard(userID, req.UseCheat)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	res := dto.SelectCardResponse{
		GameStateResponse: toGameStateResponse(game),
		LastRound:         toRoundLogResponse(*round),
	}

	return c.JSON(http.StatusOK, res)
}

func (gc *gameController) GetGameState(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	game, err := gc.gu.GetGameState(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toGameStateResponse(game))
}

func toGameStateResponse(game *model.Game) dto.GameStateResponse {
	rounds := make([]dto.RoundLogResponse, 0, len(game.Rounds))
	for _, r := range game.Rounds {
		rounds = append(rounds, toRoundLogResponse(r))
	}

	return dto.GameStateResponse{
		ID:              game.ID,
		Status:          string(game.Status),
		Mode:            string(game.Mode),
		PlayerScore:     game.PlayerScore,
		DealerScore:     game.DealerScore,
		DrawCount:       game.DrawCount,
		CurrentRound:    game.CurrentRound,
		CheatUsed:       game.CheatUsed,
		PlayerUsedCards: game.PlayerUsedCards,
		DealerUsedCards: game.DealerUsedCards,
		Rounds:          rounds,
	}
}

func toRoundLogResponse(round model.Round) dto.RoundLogResponse {
	return dto.RoundLogResponse{
		Number:     round.Number,
		PlayerCard: round.PlayerCard,
		DealerCard: round.DealerCard,
		Result:     string(round.Result),
		CheatUsed:  round.CheatUsed,
		PlayedAt:   round.PlayedAt.Format(time.RFC3339),
	}
}
