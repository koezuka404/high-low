package model

import "time"

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
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (GameRoundLog) TableName() string {
	return "game_round_logs"
}
