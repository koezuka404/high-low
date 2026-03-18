package model

import (
	"database/sql/driver"
	"testing"
	"time"
)

func TestGame_TableName(t *testing.T) {
	var g Game
	if g.TableName() != "game_sessions" {
		t.Fatalf("expected game_sessions, got %s", g.TableName())
	}
}

func TestGameRoundLog_TableName(t *testing.T) {
	var l GameRoundLog
	if l.TableName() != "game_round_logs" {
		t.Fatalf("expected game_round_logs, got %s", l.TableName())
	}
}

func TestIntSlice_Value(t *testing.T) {
	var s IntSlice
	v, err := s.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v.(string) != "[]" {
		t.Fatalf("expected [], got %#v", v)
	}

	s = IntSlice{1, 2, 13}
	v, err = s.Value()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(string); !ok {
		t.Fatalf("expected string, got %T", v)
	}
}

func TestIntSlice_Scan(t *testing.T) {
	var s IntSlice

	if err := (&s).Scan(nil); err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}

	if err := (&s).Scan([]byte(`[1,2,3]`)); err != nil {
		t.Fatal(err)
	}
	if len(s) != 3 || s[0] != 1 || s[2] != 3 {
		t.Fatalf("unexpected slice: %v", s)
	}

	if err := (&s).Scan(`[4,5]`); err != nil {
		t.Fatal(err)
	}
	if len(s) != 2 || s[0] != 4 || s[1] != 5 {
		t.Fatalf("unexpected slice: %v", s)
	}

	if err := (&s).Scan(driver.Value(123)); err == nil {
		t.Fatal("expected error")
	}
}

func TestGame_StructFields(t *testing.T) {
	now := time.Now()
	card := 13

	g := Game{
		ID:               1,
		UserID:           2,
		Status:           GameStatusInProgress,
		Mode:             GameModeDealer,
		PlayerWins:       1,
		DealerWins:       0,
		ConsecutiveDraws: 4,
		Cheated:          true,
		CheatReserved:    true,
		CheatCard:        &card,
		Ver:              7,
		PlayerUsedCards:  IntSlice{1, 2},
		DealerUsedCards:  IntSlice{3, 4},
		Rounds: []Round{
			{Number: 1, PlayerCard: 7, DealerCard: 10, Result: RoundResultDealerWin, ConsecutiveDraws: 0, CheatUsed: false, PlayedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if g.ID != 1 || g.UserID != 2 || g.Ver != 7 {
		t.Fatalf("unexpected ids/ver: %+v", g)
	}
	if g.Status != GameStatusInProgress || g.Mode != GameModeDealer {
		t.Fatalf("unexpected status/mode: %+v", g)
	}
	if g.CheatCard == nil || *g.CheatCard != 13 {
		t.Fatalf("unexpected cheat_card: %+v", g.CheatCard)
	}
	if len(g.Rounds) != 1 || g.Rounds[0].Number != 1 {
		t.Fatalf("unexpected rounds: %+v", g.Rounds)
	}
}

