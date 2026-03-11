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

// Round は 1 ラウンドのログ（history の 1 件）。API/ドメイン用。DB には GameRoundLog で保存する。
type Round struct {
	Number           int          `json:"number"`
	PlayerCard       int          `json:"player_card"`
	DealerCard       int          `json:"dealer_card"`
	Result           RoundResult  `json:"result"`
	ConsecutiveDraws int          `json:"consecutive_draws"`
	CheatUsed        bool         `json:"cheat_used"`
	PlayedAt         time.Time    `json:"played_at"`
}

// GameRoundLog は game_round_logs テーブルの 1 行（ラウンドログ）。
type GameRoundLog struct {
	ID               uint       `gorm:"primaryKey"`
	GameID           uint       `gorm:"not null;index"`
	Number           int        `gorm:"not null"`
	PlayerCard       int        `gorm:"not null"`
	DealerCard       int        `gorm:"not null"`
	Result           RoundResult `gorm:"not null;type:text"`
	ConsecutiveDraws int        `gorm:"not null"`
	CheatUsed        bool       `gorm:"not null"`
	PlayedAt         time.Time  `gorm:"not null"`
}

func (GameRoundLog) TableName() string {
	return "game_round_logs"
}

// IntSlice for DB storage of []int as JSON.
type IntSlice []int

func (s IntSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
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
	Rounds            []Round   `gorm:"-"` // history は game_round_logs で管理。読み込み時に付与。
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Game) TableName() string {
	return "game_sessions"
}
