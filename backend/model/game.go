package model

import "time"

type GameStatus string
type GameMode string
type RoundResult string

const (
	GameStatusNotStarted GameStatus = "NOT_STARTED"
	GameStatusInProgress GameStatus = "IN_PROGRESS"
	GameStatusFinished   GameStatus = "FINISHED"
)

const (
	GameModePlayer GameMode = "PLAYER"
	GameModeDealer GameMode = "DEALER"
)

const (
	RoundResultPending   RoundResult = "PENDING"
	RoundResultPlayerWin RoundResult = "PLAYER_WIN"
	RoundResultDealerWin RoundResult = "DEALER_WIN"
	RoundResultDraw      RoundResult = "DRAW"
)

type Round struct {
	Number     int
	PlayerCard int
	DealerCard int
	Result     RoundResult
	CheatUsed  bool
	PlayedAt   time.Time
}

type Game struct {
	ID              uint
	UserID          uint
	Status          GameStatus
	Mode            GameMode
	PlayerScore     int
	DealerScore     int
	DrawCount       int
	CheatUsed       bool
	CurrentRound    int
	PlayerUsedCards []int
	DealerUsedCards []int
	Rounds          []Round
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
