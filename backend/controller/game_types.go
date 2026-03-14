package controller

import "backend/model"

type StartGameRequest struct {
	Ver *int64 `json:"ver"`
}

type StartGameResponse struct {
	SessionID  uint           `json:"session_id"`
	Mode       model.GameMode `json:"mode"`
	PlayerWins int            `json:"player_wins"`
	DealerWins int            `json:"dealer_wins"`
	Ver        int64          `json:"ver"`
}

type SelectCardRequest struct {
	SessionID uint  `json:"session_id"`
	Ver       int64 `json:"ver"`
}

type SelectCardResponse struct {
	PlayerCard int            `json:"player_card"`
	DealerCard int            `json:"dealer_card"`
	Result     model.RoundResult `json:"result"`
	PlayerWins int            `json:"player_wins"`
	DealerWins int            `json:"dealer_wins"`
	GameStatus model.GameStatus `json:"game_status"`
	Ver        int64          `json:"ver"`
}

type CheatRequest struct {
	Ver int64 `json:"ver"`
}

type CheatResponse struct {
	CheatReserved bool  `json:"cheat_reserved"`
	CheatCard     int   `json:"cheat_card"`
	Ver           int64 `json:"ver"`
}

type ChangeModeRequest struct {
	Mode model.GameMode `json:"mode"`
	Ver  int64          `json:"ver"`
}

type ChangeModeResponse struct {
	Mode model.GameMode `json:"mode"`
	Ver  int64          `json:"ver"`
}

type HistoryItem struct {
	Round            int            `json:"round"`
	PlayerCard       int            `json:"player_card"`
	DealerCard       int            `json:"dealer_card"`
	Result           model.RoundResult `json:"result"`
	ConsecutiveDraws int            `json:"consecutive_draws"`
}

type StatusResponse struct {
	SessionID  uint             `json:"session_id"`
	Status     model.GameStatus `json:"status"`
	Mode       model.GameMode   `json:"mode"`
	PlayerWins int              `json:"player_wins"`
	DealerWins int              `json:"dealer_wins"`
	Ver        int64            `json:"ver"`
	History    []HistoryItem    `json:"history"`
}
