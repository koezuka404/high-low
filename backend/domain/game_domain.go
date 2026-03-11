package domain

import "backend/model"

func JudgeRound(playerCard, dealerCard int) model.RoundResult {
	if playerCard > dealerCard {
		return model.RoundResultPlayerWin
	}
	if dealerCard > playerCard {
		return model.RoundResultDealerWin
	}
	return model.RoundResultDraw
}

func RemainingCards(used []int) []int {
	usedMap := make(map[int]bool)
	for _, v := range used {
		usedMap[v] = true
	}
	var out []int
	for i := 1; i <= 13; i++ {
		if !usedMap[i] {
			out = append(out, i)
		}
	}
	return out
}

func MaxInt(arr []int) int {
	m := arr[0]
	for i := 1; i < len(arr); i++ {
		if arr[i] > m {
			m = arr[i]
		}
	}
	return m
}
