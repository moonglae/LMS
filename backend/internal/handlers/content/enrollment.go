package content

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"backend/internal/handlers/auth"
)

func (h *ContentHandler) EnrollStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	var moduleID int
	err := h.DB.QueryRow(`SELECT id FROM modules WHERE invite_code = $1`, req.InviteCode).Scan(&moduleID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Невірний код"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "Помилка пошуку модуля"}`, http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec(`INSERT INTO enrollments (user_id, module_id) VALUES ($1, $2)`, userID, moduleID)
	if err != nil {
		http.Error(w, `{"error": "Вже приєднані"}`, http.StatusConflict)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Успішно приєднано"})
}
