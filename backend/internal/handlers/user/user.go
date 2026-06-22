package user

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"backend/internal/handlers/auth"
)

type UserHandler struct {
	DB *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{DB: db}
}

// UserData відповідає структурі вашої бази даних
type UserData struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}

// GetMe повертає профіль поточного користувача
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	var u UserData
	err := h.DB.QueryRow(`
		SELECT id, email, first_name, last_name, created_at 
		FROM users 
		WHERE id = $1`, userID).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Користувача не знайдено"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(u)
}