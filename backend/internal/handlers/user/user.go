package user

import (
	"database/sql"
	"encoding/json"
	"log"
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

// UserData відповідає структурі бази даних
type UserData struct {
	ID                 int             `json:"id"`
	Email              string          `json:"email"`
	FirstName          string          `json:"first_name"`
	LastName           string          `json:"last_name"`
	Role               string          `json:"role"`
	IsBanned           bool            `json:"is_banned"`
	BanReason          string          `json:"ban_reason"`
	RestrictedFeatures json.RawMessage `json:"restricted_features"` // JSON об'єкт
	CreatedAt          time.Time       `json:"created_at"`
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
	var restricted string // ПРОМІЖНА ЗМІННА: спочатку читаємо JSON як звичайний рядок

	err := h.DB.QueryRow(`
        SELECT 
            id, email, first_name, last_name, role, 
            is_banned, COALESCE(ban_reason, ''), 
            COALESCE(restricted_features::text, '{}'), created_at 
        FROM users 
        WHERE id = $1`, userID).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role,
		&u.IsBanned, &u.BanReason, &restricted, &u.CreatedAt, // <--- Скануємо в рядок
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Користувача не знайдено"}`, http.StatusNotFound)
		} else {
			// ТЕПЕР МИ БУДЕМО БАЧИТИ ПОМИЛКУ В КОНСОЛІ
			log.Printf("GetMe SQL Error для юзера %d: %v", userID, err)
			http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		}
		return
	}

	// Перетворюємо рядок у правильний JSON формат для фронтенду
	u.RestrictedFeatures = json.RawMessage(restricted)

	json.NewEncoder(w).Encode(u)
}
