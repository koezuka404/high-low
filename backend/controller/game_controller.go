package controller

import (
	"errors"
	"io"
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/usecase"

	"github.com/labstack/echo/v4"
)

type IGameController interface {
	Start(c echo.Context) error
	Select(c echo.Context) error
	Cheat(c echo.Context) error
	ChangeMode(c echo.Context) error
	Status(c echo.Context) error
}

type gameController struct {
	gu usecase.IGameUsecase
}

func NewGameController(gu usecase.IGameUsecase) IGameController {
	return &gameController{gu: gu}
}

func (gc *gameController) Start(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "unauthorized", err.Error())
	}
	var req StartGameRequest
	if err := c.Bind(&req); err != nil && !isEOF(err) {
		return respondError(c, http.StatusBadRequest, "invalid_json", err.Error())
	}
	game, err := gc.gu.Start(userID, req.Ver)
	if err != nil {
		return handleGameError(c, err)
	}
	return respondSuccess(c, http.StatusOK, StartGameResponse{
		SessionID:  game.ID,
		Mode:       game.Mode,
		PlayerWins: game.PlayerWins,
		DealerWins: game.DealerWins,
		Ver:        game.Ver,
	})
}

func (gc *gameController) Select(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "unauthorized", err.Error())
	}
	var req SelectCardRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid_json", err.Error())
	}
	if req.SessionID == 0 {
		return respondError(c, http.StatusBadRequest, "invalid_input", "session_id is required")
	}
	game, round, err := gc.gu.Select(userID, req.SessionID, req.Ver)
	if err != nil {
		return handleGameError(c, err)
	}
	return respondSuccess(c, http.StatusOK, SelectCardResponse{
		PlayerCard: round.PlayerCard,
		DealerCard: round.DealerCard,
		Result:     round.Result,
		PlayerWins: game.PlayerWins,
		DealerWins: game.DealerWins,
		GameStatus: game.Status,
		Ver:        game.Ver,
	})
}

func (gc *gameController) Cheat(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "unauthorized", err.Error())
	}
	var req CheatRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid_json", err.Error())
	}
	game, err := gc.gu.Cheat(userID, req.Ver)
	if err != nil {
		return handleGameError(c, err)
	}
	cheatCard := 0
	if game.CheatCard != nil {
		cheatCard = *game.CheatCard
	}
	return respondSuccess(c, http.StatusOK, CheatResponse{
		CheatReserved: game.CheatReserved,
		CheatCard:     cheatCard,
		Ver:           game.Ver,
	})
}

func (gc *gameController) ChangeMode(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "unauthorized", err.Error())
	}
	var req ChangeModeRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid_json", err.Error())
	}
	game, err := gc.gu.ChangeMode(userID, req.Mode, req.Ver)
	if err != nil {
		return handleGameError(c, err)
	}
	return respondSuccess(c, http.StatusOK, ChangeModeResponse{
		Mode: game.Mode,
		Ver:  game.Ver,
	})
}

func (gc *gameController) Status(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "unauthorized", err.Error())
	}
	game, err := gc.gu.Status(userID)
	if err != nil {
		return handleGameError(c, err)
	}
	if game == nil {
		return respondSuccess(c, http.StatusOK, StatusResponse{
			SessionID:  0,
			Status:     model.GameStatusNotStarted,
			Mode:       model.GameModePlayer,
			PlayerWins: 0,
			DealerWins: 0,
			Ver:        0,
			History:    []HistoryItem{},
		})
	}
	history := make([]HistoryItem, 0, len(game.Rounds))
	for _, r := range game.Rounds {
		history = append(history, HistoryItem{
			Round:            r.Number,
			PlayerCard:       r.PlayerCard,
			DealerCard:       r.DealerCard,
			Result:           r.Result,
			ConsecutiveDraws: r.ConsecutiveDraws,
		})
	}
	return respondSuccess(c, http.StatusOK, StatusResponse{
		SessionID:  game.ID,
		Status:     game.Status,
		Mode:       game.Mode,
		PlayerWins: game.PlayerWins,
		DealerWins: game.DealerWins,
		Ver:        game.Ver,
		History:    history,
	})
}

func getUserID(c echo.Context) (uint, error) {
	v := c.Get(middleware.CtxUserIDKey)
	if v == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "user_id missing")
	}
	id, ok := v.(uint)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "user_id invalid")
	}
	return id, nil
}

func isEOF(err error) bool {
	return err == io.EOF
}

// handleGameError は Usecase の AppError を HTTP レスポンスに変換する。
// invalid_mode（7.5）含め、httpStatusFromCode で code ごとのステータス（400/404/409 等）を決定する。
func handleGameError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}
	var appErr *usecase.AppError
	if errors.As(err, &appErr) && appErr != nil {
		status := httpStatusFromCode(appErr.Code)
		return respondError(c, status, appErr.Code, appErr.Message)
	}
	return respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
}

// httpStatusFromCode は仕様 7.1 エラーコード一覧＋7.3 session_not_found、7.5 invalid_mode に従い HTTP ステータスを返す。
// 7.1: invalid_json, invalid_input, invalid_game_state, game_not_started, game_not_finished,
// game_already_started, cheat_not_available, cheat_already_used, cheat_not_allowed, unauthorized,
// forbidden, version_conflict, too_many_requests / 7.3: session_not_found(404) / 7.5: invalid_mode(400)
func httpStatusFromCode(code string) int {
	switch code {
	case "invalid_json", "invalid_input", "invalid_game_state", "invalid_mode",
		"game_not_started", "game_not_finished", "game_already_started",
		"cheat_not_available", "cheat_already_used", "cheat_not_allowed":
		return http.StatusBadRequest
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "session_not_found": // 7.3 エラー表: セッション不存在
		return http.StatusNotFound
	case "version_conflict":
		return http.StatusConflict
	case "too_many_requests":
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
