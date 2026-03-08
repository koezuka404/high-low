package db

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB() (*gorm.DB, error) {
	_ = godotenv.Load()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, errors.New("DB_DSN is empty")
	}

	fmt.Println("NewDB called, DB_DSN len =", len(dsn))

	var dialector gorm.Dialector
	if dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
		dialector = sqlite.Open(dsn)
	} else {
		dialector = postgres.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

