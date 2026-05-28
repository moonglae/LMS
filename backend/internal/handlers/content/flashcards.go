package content

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/handlers/auth"
)

func (h *ContentHandler) GetFlashcards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	moduleIDStr := r.URL.Query().Get("module_id")
	moduleID, err := strconv.Atoi(moduleIDStr)
	if err != nil || moduleID <= 0 {
		http.Error(w, `{"error": "Некоректний ID модуля"}`, http.StatusBadRequest)
		return
	}

	var ownerID int
	err = h.DB.QueryRow(`SELECT created_by FROM modules WHERE id = $1`, moduleID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Модуль не знайдено"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "Помилка перевірки прав"}`, http.StatusInternalServerError)
		return
	}

	// Доступ: або власник, або студент, що зарахований на курс
	if ownerID != userID {
		var enrolled int
		err = h.DB.QueryRow(
			`SELECT 1 FROM enrollments WHERE user_id = $1 AND module_id = $2`,
			userID, moduleID,
		).Scan(&enrolled)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, `{"error": "У вас немає доступу до цього курсу"}`, http.StatusForbidden)
				return
			}
			http.Error(w, `{"error": "Помилка перевірки доступу"}`, http.StatusInternalServerError)
			return
		}
	}

	rows, err := h.DB.Query(`SELECT id, question, answer FROM flashcards WHERE module_id = $1`, moduleID)
	if err != nil {
		http.Error(w, `{"error": "Помилка отримання карток"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var flashcards []FlashcardResponse
	for rows.Next() {
		var f FlashcardResponse
		if err := rows.Scan(&f.ID, &f.Question, &f.Answer); err != nil {
			http.Error(w, `{"error": "Помилка обробки карток"}`, http.StatusInternalServerError)
			return
		}
		flashcards = append(flashcards, f)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Помилка читання даних"}`, http.StatusInternalServerError)
		return
	}

	if flashcards == nil {
		flashcards = []FlashcardResponse{}
	}

	_ = json.NewEncoder(w).Encode(flashcards)
}

func (h *ContentHandler) CreateFlashcard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	var req CreateFlashcardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	var ownerID int
	err := h.DB.QueryRow(`SELECT created_by FROM modules WHERE id = $1`, req.ModuleID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Модуль не знайдено"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "Помилка перевірки прав"}`, http.StatusInternalServerError)
		return
	}

	if ownerID != userID {
		http.Error(w, `{"error": "Ви не власник курсу"}`, http.StatusForbidden)
		return
	}

	_, err = h.DB.Exec(
		`INSERT INTO flashcards (module_id, created_by, question, answer)
         VALUES ($1, $2, $3, $4)`,
		req.ModuleID, userID, req.Question, req.Answer,
	)
	if err != nil {
		http.Error(w, `{"error": "Помилка збереження"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Картку успішно додано"})
}
