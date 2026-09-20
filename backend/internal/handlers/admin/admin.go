package admin

import (
	"backend/internal/handlers/auth"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type AdminHandler struct {
	DB *sql.DB
}

type SecurityAlert struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	ActivityType string    `json:"activity_type"`
	Description  string    `json:"description"`
	Resolved     bool      `json:"resolved"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

// BanUser: Повне блокування користувача
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	// 1. Оголошуємо структуру, яка приймає reason!
	var req struct {
		Ban    bool   `json:"ban"`
		Reason string `json:"reason"` // <--- ЦЕ ДУЖЕ ВАЖЛИВО
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний запит"}`, http.StatusBadRequest)
		return
	}

	userID := r.URL.Query().Get("id")

	// 2. Оновлюємо і статус, і причину в базі
	query := `UPDATE users SET is_banned = $1, ban_reason = $2 WHERE id = $3`
	_, err := h.DB.Exec(query, req.Ban, req.Reason, userID)

	if err != nil {
		http.Error(w, `{"error": "Помилка бази даних"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Статус користувача оновлено"}`))
}

// RestrictFeature: Блокування окремих функцій (наприклад, AI-чату)
func (h *AdminHandler) RestrictFeature(w http.ResponseWriter, r *http.Request) {
	targetUserID := r.URL.Query().Get("id")

	// ЗАХИСТ: Перевірка на самоблокування функцій
	adminID, _ := auth.GetUserID(r.Context())
	targetIDInt, _ := strconv.Atoi(targetUserID)
	if targetIDInt == adminID {
		http.Error(w, `{"error": "Ви не можете обмежувати функції самі собі!"}`, http.StatusForbidden)
		return
	}

	var req struct {
		Feature string `json:"feature"`
		Lock    bool   `json:"lock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат"}`, http.StatusBadRequest)
		return
	}

	lockVal := "false"
	if req.Lock {
		lockVal = "true"
	}

	query := `
        UPDATE users 
        SET restricted_features = jsonb_set(COALESCE(restricted_features, '{}'::jsonb), array[$1], $2::jsonb, true)
        WHERE id = $3`

	_, err := h.DB.Exec(query, req.Feature, lockVal, targetUserID)
	if err != nil {
		http.Error(w, `{"error": "Помилка обмеження функції"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	alertID := r.URL.Query().Get("id")

	_, err := h.DB.Exec("UPDATE security_alerts SET resolved = true WHERE id = $1", alertID)
	if err != nil {
		http.Error(w, `{"error": "Помилка оновлення алерта"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ВИДАЛЕНО ВІДОБРАЖЕННЯ АДМІНІСТРАТОРІВ ЧЕРЕЗ WHERE
	query := `
        SELECT id, email, first_name, last_name, 
               COALESCE(role, 'student'), 
               COALESCE(is_banned, false),
               COALESCE(restricted_features::text, '{}')
        FROM users 
        WHERE role IS NULL OR role != 'admin'
        ORDER BY id DESC
    `
	rows, err := h.DB.Query(query)
	if err != nil {
		http.Error(w, `{"error": "Помилка отримання користувачів"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserItem struct {
		ID                 int    `json:"id"`
		Email              string `json:"email"`
		FirstName          string `json:"first_name"`
		LastName           string `json:"last_name"`
		Role               string `json:"role"`
		IsBanned           bool   `json:"is_banned"`
		RestrictedFeatures string `json:"restricted_features"`
	}

	users := make([]UserItem, 0)
	for rows.Next() {
		var u UserItem
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.IsBanned, &u.RestrictedFeatures); err != nil {
			continue
		}
		users = append(users, u)
	}

	json.NewEncoder(w).Encode(users)
}

func (h *AdminHandler) GetSecurityAlerts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, user_id, activity_type, description, resolved, created_at FROM security_alerts WHERE resolved = false ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		http.Error(w, `{"error": "Помилка БД"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var alerts []SecurityAlert
	for rows.Next() {
		var a SecurityAlert
		rows.Scan(&a.ID, &a.UserID, &a.ActivityType, &a.Description, &a.Resolved, &a.CreatedAt)
		alerts = append(alerts, a)
	}

	if len(alerts) == 0 {
		alerts = make([]SecurityAlert, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}
