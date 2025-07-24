package models

import (
	"time"
)

type GameState struct {
	ID          int                    `json:"id" db:"id"`
	Player1ID   int                    `json:"player1_id" db:"player1_id"`
	Player2ID   *int                   `json:"player2_id" db:"player2_id"`
	State       map[string]interface{} `json:"state" db:"state"`
	Status      string                 `json:"status" db:"status"`
	WinnerID    *int                   `json:"winner_id" db:"winner_id"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

type GameMove struct {
	GameID    int                    `json:"game_id"`
	PlayerID  int                    `json:"player_id"`
	Move      map[string]interface{} `json:"move"`
	Timestamp time.Time              `json:"timestamp"`
}

type Challenge struct {
	ID           int       `json:"id" db:"id"`
	ChallengerID int       `json:"challenger_id" db:"challenger_id"`
	ChallengedID int       `json:"challenged_id" db:"challenged_id"`
	Status       string    `json:"status" db:"status"`
	GameID       *int      `json:"game_id" db:"game_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}