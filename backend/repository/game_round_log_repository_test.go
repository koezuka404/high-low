package repository

import (
	"testing"
	"time"

	"backend/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRoundLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.GameRoundLog{}); err != nil {
		t.Fatalf("failed to migrate game_round_logs table: %v", err)
	}
	return db
}

func TestNewGameRoundLogRepository(t *testing.T) {
	db := setupRoundLogTestDB(t)
	r := NewGameRoundLogRepository(db)
	if r == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestGameRoundLogRepository_CRUD(t *testing.T) {
	db := setupRoundLogTestDB(t)
	r := NewGameRoundLogRepository(db)

	now := time.Now()
	log2 := &model.GameRoundLog{GameID: 10, Number: 2, PlayerCard: 5, DealerCard: 7, Result: model.RoundResultDealerWin, ConsecutiveDraws: 0, CheatUsed: false, PlayedAt: now}
	log1 := &model.GameRoundLog{GameID: 10, Number: 1, PlayerCard: 7, DealerCard: 7, Result: model.RoundResultDraw, ConsecutiveDraws: 1, CheatUsed: true, PlayedAt: now}
	if err := r.Create(log2); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(log1); err != nil {
		t.Fatal(err)
	}

	count, err := r.GetRoundLogCountByGameID(10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}

	rounds, err := r.GetRoundLogsByGameID(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rounds) != 2 {
		t.Fatalf("expected 2 rounds, got %d", len(rounds))
	}
	if rounds[0].Number != 1 || rounds[1].Number != 2 {
		t.Fatalf("expected ordered by number, got %+v", rounds)
	}
	if rounds[0].CheatUsed != true || rounds[0].ConsecutiveDraws != 1 {
		t.Fatalf("unexpected mapped fields: %+v", rounds[0])
	}

	if err := r.DeleteByGameID(10); err != nil {
		t.Fatal(err)
	}
	count, err = r.GetRoundLogCountByGameID(10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestGameRoundLogRepository_Errors_WhenDBClosed(t *testing.T) {
	db := setupRoundLogTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewGameRoundLogRepository(db)
	if err := r.Create(&model.GameRoundLog{GameID: 1, Number: 1, PlayerCard: 1, DealerCard: 1, Result: model.RoundResultDraw, ConsecutiveDraws: 1, CheatUsed: false, PlayedAt: time.Now()}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := r.GetRoundLogsByGameID(1); err == nil {
		t.Fatal("expected error")
	}
	if _, err := r.GetRoundLogCountByGameID(1); err == nil {
		t.Fatal("expected error")
	}
	if err := r.DeleteByGameID(1); err == nil {
		t.Fatal("expected error")
	}
}

