package domain

import "testing"

func TestJudgeRound(t *testing.T) {
	if got := JudgeRound(10, 3); got != "PLAYER_WIN" {
		t.Fatalf("expected PLAYER_WIN, got %s", got)
	}
	if got := JudgeRound(2, 13); got != "DEALER_WIN" {
		t.Fatalf("expected DEALER_WIN, got %s", got)
	}
	if got := JudgeRound(7, 7); got != "DRAW" {
		t.Fatalf("expected DRAW, got %s", got)
	}
}

func TestRemainingCards(t *testing.T) {
	got := RemainingCards([]int{2, 5, 9, 13})
	want := []int{1, 3, 4, 6, 7, 8, 10, 11, 12}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%d want=%d got=%v", i, got[i], want[i], got)
		}
	}
}

func TestMaxInt(t *testing.T) {
	if got := MaxInt([]int{3, 7, 11}); got != 11 {
		t.Fatalf("expected 11, got %d", got)
	}
}

