package controller

type StartGameRequest struct {
	Mode string `json:"mode"`
}

type StartGameResponse struct {
	ID           uint   `json:"id"`
	Status       string `json:"status"`
	Mode         string `json:"mode"`
	PlayerScore  int    `json:"playerScore"`
	DealerScore  int    `json:"dealerScore"`
	DrawCount    int    `json:"drawCount"`
	CurrentRound int    `json:"currentRound"`
	CheatUsed    bool   `json:"cheatUsed"`
}

type SelectCardRequest struct {
	UseCheat bool `json:"useCheat"`
}

type RoundLogResponse struct {
	Number     int    `json:"number"`
	PlayerCard int    `json:"playerCard"`
	DealerCard int    `json:"dealerCard"`
	Result     string `json:"result"`
	CheatUsed  bool   `json:"cheatUsed"`
	PlayedAt   string `json:"playedAt"`
}

type GameStateResponse struct {
	ID              uint               `json:"id"`
	Status          string             `json:"status"`
	Mode            string             `json:"mode"`
	PlayerScore     int                `json:"playerScore"`
	DealerScore     int                `json:"dealerScore"`
	DrawCount       int                `json:"drawCount"`
	CurrentRound    int                `json:"currentRound"`
	CheatUsed       bool               `json:"cheatUsed"`
	PlayerUsedCards []int              `json:"playerUsedCards"`
	DealerUsedCards []int              `json:"dealerUsedCards"`
	Rounds          []RoundLogResponse `json:"rounds"`
}

type SelectCardResponse struct {
	GameStateResponse
	LastRound RoundLogResponse `json:"lastRound"`
}
