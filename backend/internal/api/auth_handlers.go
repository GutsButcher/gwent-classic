package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"fmt"
	"gwent-backend/internal/auth"
	"gwent-backend/internal/models"
)

type AuthHandler struct {
	db        *sql.DB
	jwtSecret string
}

func NewAuthHandler(db *sql.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Name == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	var user models.User
	query := `INSERT INTO users (email, name, password) VALUES ($1, $2, $3) 
			  RETURNING id, email, name, created_at, updated_at`
	err = h.db.QueryRow(query, req.Email, req.Name, hashedPassword).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.Name, h.jwtSecret)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	response := models.AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user models.User
	query := `SELECT id, email, name, password, created_at, updated_at FROM users WHERE email = $1`
	err := h.db.QueryRow(query, req.Email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if !auth.CheckPassword(req.Password, user.Password) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, user.Name, h.jwtSecret)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	user.Password = ""
	response := models.AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) GetUserByNameID(w http.ResponseWriter, r *http.Request) {
	nameID := r.URL.Query().Get("nameID")
	if nameID == "" {
		http.Error(w, "nameID parameter required", http.StatusBadRequest)
		return
	}

	var name string
	var id int
	_, err := fmt.Sscanf(nameID, "%s#%d", &name, &id)
	if err != nil {
		http.Error(w, "Invalid nameID format. Use name#id", http.StatusBadRequest)
		return
	}

	var user models.User
	query := `SELECT id, email, name, created_at, updated_at FROM users WHERE name = $1 AND id = $2`
	err = h.db.QueryRow(query, name, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}