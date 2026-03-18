package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/middleware"
	"backend/model"
	"backend/usecase"

	"github.com/labstack/echo/v4"
)

type mockGameUsecase struct {
	startFn      func(userID uint, ver *int64) (*model.Game, error)
	selectFn     func(userID uint, sessionID uint, ver int64) (*model.Game, *model.Round, error)
	cheatFn      func(userID uint, ver int64) (*model.Game, error)
	resetSetFn   func(userID uint, ver int64) (*model.Game, error)
	changeModeFn func(userID uint, mode model.GameMode, ver int64) (*model.Game, error)
	statusFn     func(userID uint) (*model.Game, error)
}

func (m *mockGameUsecase) Start(userID uint, ver *int64) (*model.Game, error) {
	if m.startFn == nil {
		return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, PlayerWins: 0, DealerWins: 0, Ver: 1}, nil
	}
	return m.startFn(userID, ver)
}
func (m *mockGameUsecase) Select(userID uint, sessionID uint, ver int64) (*model.Game, *model.Round, error) {
	if m.selectFn == nil {
		g := &model.Game{ID: sessionID, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, PlayerWins: 0, DealerWins: 0, Ver: ver + 1}
		r := &model.Round{Number: 1, PlayerCard: 7, DealerCard: 10, Result: model.RoundResultDealerWin, ConsecutiveDraws: 0, CheatUsed: false, PlayedAt: time.Now()}
		return g, r, nil
	}
	return m.selectFn(userID, sessionID, ver)
}
func (m *mockGameUsecase) Cheat(userID uint, ver int64) (*model.Game, error) {
	if m.cheatFn == nil {
		card := 13
		return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, CheatReserved: true, CheatCard: &card, Ver: ver + 1}, nil
	}
	return m.cheatFn(userID, ver)
}
func (m *mockGameUsecase) ResetSet(userID uint, ver int64) (*model.Game, error) {
	if m.resetSetFn == nil {
		return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, PlayerWins: 0, DealerWins: 0, Ver: ver + 1}, nil
	}
	return m.resetSetFn(userID, ver)
}
func (m *mockGameUsecase) ChangeMode(userID uint, mode model.GameMode, ver int64) (*model.Game, error) {
	if m.changeModeFn == nil {
		return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusFinished, Mode: mode, Ver: ver + 1}, nil
	}
	return m.changeModeFn(userID, mode, ver)
}
func (m *mockGameUsecase) Status(userID uint) (*model.Game, error) {
	if m.statusFn == nil {
		return nil, nil
	}
	return m.statusFn(userID)
}

func newJSONRequest(method, path string, body any) *http.Request {
	if body == nil {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		return req
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return req
}

func TestGetUserID_Errors(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/game/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if _, err := getUserID(c); err == nil {
		t.Fatal("expected error")
	}
	c.Set(middleware.CtxUserIDKey, "bad")
	if _, err := getUserID(c); err == nil {
		t.Fatal("expected error")
	}
}

func TestIsEOF(t *testing.T) {
	if !isEOF(io.EOF) {
		t.Fatal("expected true")
	}
	if isEOF(errors.New("x")) {
		t.Fatal("expected false")
	}
}

func TestHandleGameError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/game/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handleGameError(c, nil); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	if err := handleGameError(c, &usecase.AppError{Code: "version_conflict", Message: "version conflict"}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	if err := handleGameError(c, errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHttpStatusFromCode_Default(t *testing.T) {
	if got := httpStatusFromCode("unknown"); got != http.StatusInternalServerError {
		t.Fatalf("unexpected: %d", got)
	}
}

func TestHttpStatusFromCode_AllCases(t *testing.T) {
	cases := map[string]int{
		"invalid_json":         http.StatusBadRequest,
		"invalid_input":        http.StatusBadRequest,
		"invalid_game_state":   http.StatusBadRequest,
		"invalid_mode":         http.StatusBadRequest,
		"game_not_started":     http.StatusBadRequest,
		"game_not_finished":    http.StatusBadRequest,
		"game_already_started": http.StatusBadRequest,
		"cheat_not_available":  http.StatusBadRequest,
		"cheat_already_used":   http.StatusBadRequest,
		"cheat_not_allowed":    http.StatusBadRequest,
		"unauthorized":         http.StatusUnauthorized,
		"forbidden":            http.StatusForbidden,
		"session_not_found":    http.StatusNotFound,
		"version_conflict":     http.StatusConflict,
		"too_many_requests":    http.StatusTooManyRequests,
	}
	for code, want := range cases {
		if got := httpStatusFromCode(code); got != want {
			t.Fatalf("code=%s got=%d want=%d", code, got, want)
		}
	}
}

func TestGameController_Start_Unauthorized(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodPost, "/api/game/start", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	gc := NewGameController(&mockGameUsecase{})
	if err := gc.Start(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGameController_Start_BindError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/game/start", bytes.NewReader([]byte(`{"ver":`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))

	gc := NewGameController(&mockGameUsecase{})
	if err := gc.Start(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGameController_Start_EOF_Allows(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodPost, "/api/game/start", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))

	mock := &mockGameUsecase{
		startFn: func(userID uint, ver *int64) (*model.Game, error) {
			if ver != nil {
				t.Fatal("expected nil ver")
			}
			return &model.Game{ID: 10, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, PlayerWins: 0, DealerWins: 0, Ver: 1}, nil
		},
	}
	gc := NewGameController(mock)
	if err := gc.Start(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGameController_Start_UsecaseError(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodPost, "/api/game/start", map[string]any{"ver": 1})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))

	mock := &mockGameUsecase{
		startFn: func(userID uint, ver *int64) (*model.Game, error) {
			return nil, &usecase.AppError{Code: "version_conflict", Message: "version conflict"}
		},
	}
	gc := NewGameController(mock)
	if err := gc.Start(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestGameController_Select_Validation(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/api/game/select", bytes.NewReader([]byte(`{"session_id":`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	gc := NewGameController(&mockGameUsecase{})
	if err := gc.Select(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = newJSONRequest(http.MethodPost, "/api/game/select", map[string]any{"ver": 1})
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Select(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = newJSONRequest(http.MethodPost, "/api/game/select", map[string]any{"session_id": 1})
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Select(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGameController_Select_Unauthorized(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodPost, "/api/game/select", map[string]any{"session_id": 1, "ver": 1})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	gc := NewGameController(&mockGameUsecase{})
	if err := gc.Select(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGameController_Select_Success_And_UsecaseError(t *testing.T) {
	e := echo.New()
	gc := NewGameController(&mockGameUsecase{
		selectFn: func(userID uint, sessionID uint, ver int64) (*model.Game, *model.Round, error) {
			g := &model.Game{ID: sessionID, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, PlayerWins: 1, DealerWins: 0, Ver: ver + 1}
			r := &model.Round{Number: 1, PlayerCard: 7, DealerCard: 10, Result: model.RoundResultDealerWin, ConsecutiveDraws: 0, CheatUsed: false, PlayedAt: time.Now()}
			return g, r, nil
		},
	})

	rec := httptest.NewRecorder()
	req := newJSONRequest(http.MethodPost, "/api/game/select", map[string]any{"session_id": 1, "ver": 1})
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Select(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	gc = NewGameController(&mockGameUsecase{
		selectFn: func(userID uint, sessionID uint, ver int64) (*model.Game, *model.Round, error) {
			return nil, nil, &usecase.AppError{Code: "forbidden", Message: "forbidden"}
		},
	})
	rec = httptest.NewRecorder()
	req = newJSONRequest(http.MethodPost, "/api/game/select", map[string]any{"session_id": 1, "ver": 1})
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Select(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestGameController_Cheat_And_ChangeMode_Validation(t *testing.T) {
	e := echo.New()
	gc := NewGameController(&mockGameUsecase{})

	req := httptest.NewRequest(http.MethodPost, "/api/game/cheat", bytes.NewReader([]byte(`{"ver":`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Cheat(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	req = newJSONRequest(http.MethodPost, "/api/game/cheat", map[string]any{})
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Cheat(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/game/mode", bytes.NewReader([]byte(`{"mode":`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.ChangeMode(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	req = newJSONRequest(http.MethodPost, "/api/game/mode", map[string]any{"mode": "PLAYER"})
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.ChangeMode(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGameController_Cheat_Unauthorized(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodPost, "/api/game/cheat", map[string]any{"ver": 1})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	gc := NewGameController(&mockGameUsecase{})
	if err := gc.Cheat(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGameController_Cheat_Success_NilAndNonNilCheatCard(t *testing.T) {
	e := echo.New()

	gc := NewGameController(&mockGameUsecase{
		cheatFn: func(userID uint, ver int64) (*model.Game, error) {
			return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, CheatReserved: true, CheatCard: nil, Ver: ver + 1}, nil
		},
	})
	rec := httptest.NewRecorder()
	req := newJSONRequest(http.MethodPost, "/api/game/cheat", map[string]any{"ver": 1})
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Cheat(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	card := 13
	gc = NewGameController(&mockGameUsecase{
		cheatFn: func(userID uint, ver int64) (*model.Game, error) {
			return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusInProgress, Mode: model.GameModeDealer, CheatReserved: true, CheatCard: &card, Ver: ver + 1}, nil
		},
	})
	rec = httptest.NewRecorder()
	req = newJSONRequest(http.MethodPost, "/api/game/cheat", map[string]any{"ver": 1})
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Cheat(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGameController_Cheat_UsecaseError(t *testing.T) {
	e := echo.New()
	gc := NewGameController(&mockGameUsecase{
		cheatFn: func(userID uint, ver int64) (*model.Game, error) {
			return nil, &usecase.AppError{Code: "cheat_not_allowed", Message: "cheat not allowed"}
		},
	})
	rec := httptest.NewRecorder()
	req := newJSONRequest(http.MethodPost, "/api/game/cheat", map[string]any{"ver": 1})
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.Cheat(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGameController_ChangeMode_Success_And_UsecaseError(t *testing.T) {
	e := echo.New()
	gc := NewGameController(&mockGameUsecase{
		changeModeFn: func(userID uint, mode model.GameMode, ver int64) (*model.Game, error) {
			return &model.Game{ID: 1, UserID: userID, Status: model.GameStatusFinished, Mode: mode, Ver: ver + 1}, nil
		},
	})
	rec := httptest.NewRecorder()
	req := newJSONRequest(http.MethodPost, "/api/game/mode", map[string]any{"mode": "DEALER", "ver": 1})
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.ChangeMode(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	gc = NewGameController(&mockGameUsecase{
		changeModeFn: func(userID uint, mode model.GameMode, ver int64) (*model.Game, error) {
			return nil, &usecase.AppError{Code: "invalid_mode", Message: "invalid mode"}
		},
	})
	rec = httptest.NewRecorder()
	req = newJSONRequest(http.MethodPost, "/api/game/mode", map[string]any{"mode": "BAD", "ver": 1})
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	if err := gc.ChangeMode(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGameController_ChangeMode_Unauthorized(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodPost, "/api/game/mode", map[string]any{"mode": "PLAYER", "ver": 1})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	gc := NewGameController(&mockGameUsecase{})
	if err := gc.ChangeMode(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGameController_Status_NilAndHistory(t *testing.T) {
	e := echo.New()

	rec := httptest.NewRecorder()
	req := newJSONRequest(http.MethodGet, "/api/game/status", nil)
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	gc := NewGameController(&mockGameUsecase{statusFn: func(userID uint) (*model.Game, error) { return nil, nil }})
	if err := gc.Status(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = newJSONRequest(http.MethodGet, "/api/game/status", nil)
	c = e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	now := time.Now()
	game := &model.Game{
		ID:         10,
		UserID:     1,
		Status:     model.GameStatusInProgress,
		Mode:       model.GameModePlayer,
		PlayerWins: 1,
		DealerWins: 0,
		Ver:        3,
		Rounds: []model.Round{
			{Number: 1, PlayerCard: 7, DealerCard: 7, Result: model.RoundResultDraw, ConsecutiveDraws: 1, CheatUsed: false, PlayedAt: now},
		},
	}
	gc = NewGameController(&mockGameUsecase{statusFn: func(userID uint) (*model.Game, error) { return game, nil }})
	if err := gc.Status(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGameController_Status_Unauthorized(t *testing.T) {
	e := echo.New()
	req := newJSONRequest(http.MethodGet, "/api/game/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	gc := NewGameController(&mockGameUsecase{})
	if err := gc.Status(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGameController_Status_UsecaseError(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	req := newJSONRequest(http.MethodGet, "/api/game/status", nil)
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxUserIDKey, uint(1))
	gc := NewGameController(&mockGameUsecase{statusFn: func(userID uint) (*model.Game, error) {
		return nil, errors.New("boom")
	}})
	if err := gc.Status(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

