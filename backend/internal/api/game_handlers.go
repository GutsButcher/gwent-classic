package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"gwent-backend/internal/auth"
	"gwent-backend/internal/models"
	"github.com/gorilla/mux"
)

type GameHandler struct {
	db *sql.DB
}

func NewGameHandler(db *sql.DB) *GameHandler {
	return &GameHandler{
		db: db,
	}
}

func (h *GameHandler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}
	
	claims := r.Context().Value(UserContextKey).(*auth.Claims)
	
	var req struct {
		ChallengedNameID string `json:"challenged_name_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse name#id format
	var challengedName string
	var challengedID int
	
	lastHash := -1
	for i := len(req.ChallengedNameID) - 1; i >= 0; i-- {
		if req.ChallengedNameID[i] == '#' {
			lastHash = i
			break
		}
	}
	if lastHash == -1 {
		http.Error(w, "Invalid nameID format. Use name#id", http.StatusBadRequest)
		return
	}
	
	challengedName = req.ChallengedNameID[:lastHash]
	idStr := req.ChallengedNameID[lastHash+1:]
	
	var err error
	challengedID, err = strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID in nameID", http.StatusBadRequest)
		return
	}
	
	log.Printf("Parsed challenge request: name=%s, id=%d", challengedName, challengedID)

	if challengedID == claims.UserID {
		http.Error(w, "Cannot challenge yourself", http.StatusBadRequest)
		return
	}

	var userExists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE name = $1 AND id = $2)", 
		challengedName, challengedID).Scan(&userExists)
	log.Printf("Checking user existence: name=%s, id=%d, exists=%v, err=%v", challengedName, challengedID, userExists, err)
	if err != nil || !userExists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var challenge models.Challenge
	query := `INSERT INTO challenges (challenger_id, challenged_id, status) 
			  VALUES ($1, $2, 'pending') 
			  RETURNING id, challenger_id, challenged_id, status, created_at`
	err = h.db.QueryRow(query, claims.UserID, challengedID).Scan(
		&challenge.ID, &challenge.ChallengerID, &challenge.ChallengedID, 
		&challenge.Status, &challenge.CreatedAt,
	)

	if err != nil {
		http.Error(w, "Challenge already exists or database error", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challenge)
}

func (h *GameHandler) GetChallenges(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}
	
	claims := r.Context().Value(UserContextKey).(*auth.Claims)
	
	query := `SELECT c.id, c.challenger_id, c.challenged_id, c.status, c.game_id, c.created_at,
			  u1.name as challenger_name, u2.name as challenged_name
			  FROM challenges c
			  JOIN users u1 ON c.challenger_id = u1.id
			  JOIN users u2 ON c.challenged_id = u2.id
			  WHERE (c.challenger_id = $1 OR c.challenged_id = $1) AND c.status = 'pending'
			  ORDER BY c.created_at DESC`
	
	rows, err := h.db.Query(query, claims.UserID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var challenges []map[string]interface{}
	for rows.Next() {
		var c models.Challenge
		var challengerName, challengedName string
		err := rows.Scan(&c.ID, &c.ChallengerID, &c.ChallengedID, &c.Status, 
			&c.GameID, &c.CreatedAt, &challengerName, &challengedName)
		if err != nil {
			continue
		}
		
		challenge := map[string]interface{}{
			"id":             c.ID,
			"challenger":     map[string]interface{}{"id": c.ChallengerID, "name": challengerName},
			"challenged":     map[string]interface{}{"id": c.ChallengedID, "name": challengedName},
			"status":         c.Status,
			"is_challenger":  c.ChallengerID == claims.UserID,
			"created_at":     c.CreatedAt,
		}
		challenges = append(challenges, challenge)
	}

	if challenges == nil {
		challenges = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challenges)
}

func (h *GameHandler) RespondToChallenge(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}
	
	claims := r.Context().Value(UserContextKey).(*auth.Claims)
	vars := mux.Vars(r)
	challengeID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid challenge ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Accept bool `json:"accept"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var challenge models.Challenge
	err = tx.QueryRow("SELECT id, challenger_id, challenged_id FROM challenges WHERE id = $1 AND challenged_id = $2 AND status = 'pending'",
		challengeID, claims.UserID).Scan(&challenge.ID, &challenge.ChallengerID, &challenge.ChallengedID)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Challenge not found or unauthorized", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if req.Accept {
		var gameID int
		initialState := map[string]interface{}{
			"player1": challenge.ChallengerID,
			"player2": challenge.ChallengedID,
			"currentPlayer": challenge.ChallengerID,
			"round": 1,
			"player1Score": 0,
			"player2Score": 0,
			"board": map[string]interface{}{
				"player1": map[string][]interface{}{
					"close": []interface{}{},
					"ranged": []interface{}{},
					"siege": []interface{}{},
				},
				"player2": map[string][]interface{}{
					"close": []interface{}{},
					"ranged": []interface{}{},
					"siege": []interface{}{},
				},
			},
		}
		
		stateJSON, _ := json.Marshal(initialState)
		err = tx.QueryRow("INSERT INTO games (player1_id, player2_id, state, status) VALUES ($1, $2, $3, 'active') RETURNING id",
			challenge.ChallengerID, challenge.ChallengedID, stateJSON).Scan(&gameID)
		
		if err != nil {
			http.Error(w, "Failed to create game", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("UPDATE challenges SET status = 'accepted', game_id = $1 WHERE id = $2", gameID, challengeID)
		if err != nil {
			http.Error(w, "Failed to update challenge", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"challenge_id": challengeID,
			"game_id": gameID,
			"status": "accepted",
		})
	} else {
		_, err = tx.Exec("UPDATE challenges SET status = 'declined' WHERE id = $1", challengeID)
		if err != nil {
			http.Error(w, "Failed to update challenge", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"challenge_id": challengeID,
			"status": "declined",
		})
	}
}

func (h *GameHandler) GetActiveGames(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}
	
	claims := r.Context().Value(UserContextKey).(*auth.Claims)
	
	query := `SELECT g.id, g.player1_id, g.player2_id, g.status, g.created_at,
			  u1.name as player1_name, u2.name as player2_name
			  FROM games g
			  JOIN users u1 ON g.player1_id = u1.id
			  LEFT JOIN users u2 ON g.player2_id = u2.id
			  WHERE (g.player1_id = $1 OR g.player2_id = $1) AND g.status = 'active'
			  ORDER BY g.updated_at DESC`
	
	rows, err := h.db.Query(query, claims.UserID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []map[string]interface{}
	for rows.Next() {
		var game models.GameState
		var player1Name string
		var player2Name sql.NullString
		
		err := rows.Scan(&game.ID, &game.Player1ID, &game.Player2ID, 
			&game.Status, &game.CreatedAt, &player1Name, &player2Name)
		if err != nil {
			continue
		}
		
		gameInfo := map[string]interface{}{
			"id":        game.ID,
			"player1":   map[string]interface{}{"id": game.Player1ID, "name": player1Name},
			"status":    game.Status,
			"created_at": game.CreatedAt,
		}
		
		if player2Name.Valid && game.Player2ID != nil {
			gameInfo["player2"] = map[string]interface{}{"id": *game.Player2ID, "name": player2Name.String}
		} else {
			gameInfo["player2"] = nil
		}
		
		games = append(games, gameInfo)
	}

	if games == nil {
		games = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (h *GameHandler) CreateAIGame(w http.ResponseWriter, r *http.Request) {
	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}
	
	claims := r.Context().Value(UserContextKey).(*auth.Claims)
	
	initialState := map[string]interface{}{
		"player1": claims.UserID,
		"player2": nil,
		"currentPlayer": claims.UserID,
		"round": 1,
		"player1Score": 0,
		"player2Score": 0,
		"vsAI": true,
		"board": map[string]interface{}{
			"player1": map[string][]interface{}{
				"close": []interface{}{},
				"ranged": []interface{}{},
				"siege": []interface{}{},
			},
			"player2": map[string][]interface{}{
				"close": []interface{}{},
				"ranged": []interface{}{},
				"siege": []interface{}{},
			},
		},
	}
	
	stateJSON, _ := json.Marshal(initialState)
	
	var gameID int
	err := h.db.QueryRow("INSERT INTO games (player1_id, player2_id, state, status) VALUES ($1, NULL, $2, 'active') RETURNING id",
		claims.UserID, stateJSON).Scan(&gameID)
	
	if err != nil {
		http.Error(w, "Failed to create game", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"game_id": gameID,
		"vs_ai": true,
	})
}