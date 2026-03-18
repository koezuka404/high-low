package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

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
	RoundResultPlayerWin RoundResult = "PLAYER_WIN"
	RoundResultDealerWin RoundResult = "DEALER_WIN"
	RoundResultDraw      RoundResult = "DRAW"
)

type Round struct {
	Number           int          `json:"number"`
	PlayerCard       int          `json:"player_card"`
	DealerCard       int          `json:"dealer_card"`
	Result           RoundResult  `json:"result"`
	ConsecutiveDraws int          `json:"consecutive_draws"`
	CheatUsed        bool         `json:"cheat_used"`
	PlayedAt         time.Time    `json:"played_at"`
}

type IntSlice []int

func (s IntSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, _ := json.Marshal(s)
	return string(b), nil
}

func (s *IntSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("invalid type for IntSlice")
	}
	return json.Unmarshal(b, s)
}

type Game struct {
	ID               uint       `gorm:"primaryKey"`
	UserID           uint       `gorm:"not null;uniqueIndex"`
	Status           GameStatus `gorm:"not null"`
	Mode             GameMode   `gorm:"not null"`
	PlayerWins       int        `gorm:"not null"`
	DealerWins       int        `gorm:"not null"`
	ConsecutiveDraws int        `gorm:"not null"`
	Cheated          bool       `gorm:"not null"`
	CheatReserved    bool       `gorm:"not null"`
	CheatCard        *int
	Ver              int64     `gorm:"not null"`
	PlayerUsedCards   IntSlice  `gorm:"type:text"`
	DealerUsedCards   IntSlice  `gorm:"type:text"`
	Rounds            []Round   `gorm:"-"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Game) TableName() string {
	return "game_sessions"
}
