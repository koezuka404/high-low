package repository

import (
	"errors"
	"testing"
	"time"

	"backend/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGameTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Game{}); err != nil {
		t.Fatalf("failed to migrate game_sessions table: %v", err)
	}
	return db
}

func TestNewGameRepository(t *testing.T) {
	db := setupGameTestDB(t)
	r := NewGameRepository(db)
	if r == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestGameRepository_Create_And_GetByID_UserID(t *testing.T) {
	db := setupGameTestDB(t)
	r := NewGameRepository(db)

	now := time.Now()
	game := &model.Game{
		UserID:           1,
		Status:           model.GameStatusInProgress,
		Mode:             model.GameModePlayer,
		PlayerWins:       0,
		DealerWins:       0,
		ConsecutiveDraws: 0,
		Cheated:          false,
		CheatReserved:    false,
		CheatCard:        nil,
		Ver:              1,
		PlayerUsedCards:  model.IntSlice{1, 2},
		DealerUsedCards:  model.IntSlice{3, 4},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := r.Create(game); err != nil {
		t.Fatal(err)
	}
	if game.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	gotByID, err := r.GetGameByID(game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotByID == nil || gotByID.UserID != 1 {
		t.Fatalf("unexpected game: %+v", gotByID)
	}

	gotByUser, err := r.GetGameByUserID(1)
	if err != nil {
		t.Fatal(err)
	}
	if gotByUser == nil || gotByUser.ID != game.ID {
		t.Fatalf("unexpected game: %+v", gotByUser)
	}
}

func TestGameRepository_Get_NotFound_ReturnsNil(t *testing.T) {
	db := setupGameTestDB(t)
	r := NewGameRepository(db)

	g, err := r.GetGameByID(9999)
	if err != nil {
		t.Fatal(err)
	}
	if g != nil {
		t.Fatalf("expected nil, got %+v", g)
	}

	g, err = r.GetGameByUserID(9999)
	if err != nil {
		t.Fatal(err)
	}
	if g != nil {
		t.Fatalf("expected nil, got %+v", g)
	}
}

func TestGameRepository_Get_Error_WhenDBClosed(t *testing.T) {
	db := setupGameTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewGameRepository(db)
	if _, err := r.GetGameByID(1); err == nil {
		t.Fatal("expected error")
	}
	if _, err := r.GetGameByUserID(1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameRepository_UpdateWithVersion_Success(t *testing.T) {
	db := setupGameTestDB(t)
	r := NewGameRepository(db)

	game := &model.Game{
		UserID:           1,
		Status:           model.GameStatusInProgress,
		Mode:             model.GameModePlayer,
		PlayerWins:       0,
		DealerWins:       0,
		ConsecutiveDraws: 0,
		Cheated:          false,
		CheatReserved:    false,
		Ver:              1,
		PlayerUsedCards:  model.IntSlice{},
		DealerUsedCards:  model.IntSlice{},
		UpdatedAt:        time.Now(),
	}
	if err := r.Create(game); err != nil {
		t.Fatal(err)
	}

	game.PlayerWins = 1
	game.Ver = 2
	game.UpdatedAt = time.Now()
	if err := r.(*gameRepository).UpdateWithVersion(game, 1); err != nil {
		t.Fatal(err)
	}
}

func TestGameRepository_UpdateWithVersion_VersionConflict(t *testing.T) {
	db := setupGameTestDB(t)
	r := NewGameRepository(db).(*gameRepository)

	game := &model.Game{
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{},
		DealerUsedCards: model.IntSlice{},
		UpdatedAt:       time.Now(),
	}
	if err := r.Create(game); err != nil {
		t.Fatal(err)
	}

	game.Ver = 2
	game.UpdatedAt = time.Now()
	if err := r.UpdateWithVersion(game, 999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestGameRepository_UpdateWithVersion_Error_WhenDBClosed(t *testing.T) {
	db := setupGameTestDB(t)
	r := NewGameRepository(db).(*gameRepository)

	game := &model.Game{
		UserID:          1,
		Status:          model.GameStatusInProgress,
		Mode:            model.GameModePlayer,
		Ver:             1,
		PlayerUsedCards: model.IntSlice{},
		DealerUsedCards: model.IntSlice{},
		UpdatedAt:       time.Now(),
	}
	if err := r.Create(game); err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	game.Ver = 2
	game.UpdatedAt = time.Now()
	if err := r.UpdateWithVersion(game, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGameRepository_Create_Error_WhenDBClosed(t *testing.T) {
	db := setupGameTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewGameRepository(db)
	err = r.Create(&model.Game{UserID: 1, Status: model.GameStatusInProgress, Mode: model.GameModePlayer, Ver: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

