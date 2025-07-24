package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"gwent-backend/internal/auth"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type GameHub struct {
	games      map[int]*GameRoom
	register   chan *GameClient
	unregister chan *GameClient
	mu         sync.RWMutex
	db         *sql.DB
}

type GameRoom struct {
	ID       int
	Clients  map[int]*GameClient
	State    map[string]interface{}
	mu       sync.RWMutex
}

type GameClient struct {
	UserID   int
	GameID   int
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *GameHub
}

type GameMessage struct {
	Type    string                 `json:"type"`
	GameID  int                    `json:"game_id"`
	UserID  int                    `json:"user_id"`
	Payload map[string]interface{} `json:"payload"`
}

func NewGameHub(db *sql.DB) *GameHub {
	return &GameHub{
		games:      make(map[int]*GameRoom),
		register:   make(chan *GameClient),
		unregister: make(chan *GameClient),
		db:         db,
	}
}

func (h *GameHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			room, exists := h.games[client.GameID]
			if !exists {
				room = &GameRoom{
					ID:      client.GameID,
					Clients: make(map[int]*GameClient),
				}
				h.games[client.GameID] = room
				
				var stateJSON []byte
				err := h.db.QueryRow("SELECT state FROM games WHERE id = $1", client.GameID).Scan(&stateJSON)
				if err == nil {
					json.Unmarshal(stateJSON, &room.State)
				}
			}
			room.Clients[client.UserID] = client
			h.mu.Unlock()

			client.Send <- []byte(`{"type":"connected","payload":{"message":"Connected to game"}}`)
			
			if room.State != nil {
				stateMsg, _ := json.Marshal(GameMessage{
					Type:    "game_state",
					GameID:  client.GameID,
					Payload: room.State,
				})
				client.Send <- stateMsg
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if room, exists := h.games[client.GameID]; exists {
				if _, ok := room.Clients[client.UserID]; ok {
					delete(room.Clients, client.UserID)
					close(client.Send)
					
					if len(room.Clients) == 0 {
						delete(h.games, client.GameID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *GameHub) BroadcastToGame(gameID int, message []byte) {
	h.mu.RLock()
	room, exists := h.games[gameID]
	h.mu.RUnlock()
	
	if exists {
		room.mu.RLock()
		defer room.mu.RUnlock()
		
		for _, client := range room.Clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(room.Clients, client.UserID)
			}
		}
	}
}

func (h *GameHub) HandleGameMove(gameID int, userID int, move map[string]interface{}) error {
	h.mu.RLock()
	room, exists := h.games[gameID]
	h.mu.RUnlock()
	
	if !exists {
		return nil
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.State == nil {
		var stateJSON []byte
		err := h.db.QueryRow("SELECT state FROM games WHERE id = $1", gameID).Scan(&stateJSON)
		if err != nil {
			return err
		}
		json.Unmarshal(stateJSON, &room.State)
	}

	currentPlayer, ok := room.State["currentPlayer"].(float64)
	if !ok || int(currentPlayer) != userID {
		return nil
	}

	room.State["lastMove"] = move
	room.State["lastMoveBy"] = userID

	stateJSON, _ := json.Marshal(room.State)
	_, err := h.db.Exec("UPDATE games SET state = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", 
		stateJSON, gameID)
	if err != nil {
		return err
	}

	message, _ := json.Marshal(GameMessage{
		Type:    "game_update",
		GameID:  gameID,
		UserID:  userID,
		Payload: room.State,
	})
	
	h.BroadcastToGame(gameID, message)
	
	return nil
}

func (c *GameClient) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg GameMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "move":
			c.Hub.HandleGameMove(c.GameID, c.UserID, msg.Payload)
		case "game_state_request":
			c.Hub.mu.RLock()
			room := c.Hub.games[c.GameID]
			c.Hub.mu.RUnlock()
			
			if room != nil && room.State != nil {
				stateMsg, _ := json.Marshal(GameMessage{
					Type:    "game_state",
					GameID:  c.GameID,
					Payload: room.State,
				})
				c.Send <- stateMsg
			}
		}
	}
}

func (c *GameClient) WritePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func HandleWebSocket(hub *GameHub, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		gameID, err := strconv.Atoi(vars["gameId"])
		if err != nil {
			http.Error(w, "Invalid game ID", http.StatusBadRequest)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Token required", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(token, jwtSecret)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}

		client := &GameClient{
			UserID: claims.UserID,
			GameID: gameID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			Hub:    hub,
		}

		client.Hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}