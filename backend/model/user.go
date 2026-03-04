package model

import "time"

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	IsActive  bool      `json:"isActive" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RequestBodyUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResponseUser struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

func (User) TableName() string {
	return "users"
}
