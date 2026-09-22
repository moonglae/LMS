package practice

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend/internal/handlers/auth"
)

// --- 1. ГОЛОВНА СТРУКТУРА ХЕНДЛЕРА ---

type Handler struct {
	DB *sql.DB
}

// --- 2. СТРУКТУРИ ДЛЯ ВХІДНИХ ДАНИХ (ВІД REACT) ---

type SaveMistakeRequest struct {
	WrongText       string `json:"wrong_text"`
	CorrectText     string `json:"correct_text"`
	RuleExplanation string `json:"rule_explanation"`
}

type SaveVocabRequest struct {
	Word            string `json:"word"`
	Translation     string `json:"translation"`
	ContextSentence string `json:"context_sentence"`
}

// --- 3. СТРУКТУРИ ДЛЯ ВІДПОВІДЕЙ ---

type MistakeResponse struct {
	ID              int       `json:"id"`
	WrongText       string    `json:"wrong_text"`
	CorrectText     string    `json:"correct_text"`
	RuleExplanation string    `json:"rule_explanation"`
	CreatedAt       time.Time `json:"created_at"`
}

type VocabResponse struct {
	ID              int       `json:"id"`
	Word            string    `json:"word"`
	Translation     string    `json:"translation"`
	ContextSentence string    `json:"context_sentence"`
	CreatedAt       time.Time `json:"created_at"`
}

// --- 4. ФУНКЦІЇ ЗБЕРЕЖЕННЯ (POST-ЗАПИТИ) ---

func (h *Handler) SaveMistake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*10) // ДОДАНО: Ліміт 10KB

	var req SaveMistakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	query := `
        INSERT INTO saved_mistakes (user_id, wrong_text, correct_text, rule_explanation)
        VALUES ($1, $2, $3, $4)
    `
	_, err := h.DB.Exec(query, userID, req.WrongText, req.CorrectText, req.RuleExplanation)
	if err != nil {
		log.Printf("Помилка збереження помилки для юзера %d: %v", userID, err)
		http.Error(w, `{"error": "Не вдалося зберегти помилку"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Помилку збережено!"})
}

func (h *Handler) SaveVocabulary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*10) // ДОДАНО: Ліміт 10KB

	var req SaveVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	query := `
        INSERT INTO saved_vocabulary (user_id, word, translation, context_sentence)
        VALUES ($1, $2, $3, $4)
    `
	_, err := h.DB.Exec(query, userID, req.Word, req.Translation, req.ContextSentence)
	if err != nil {
		log.Printf("Помилка збереження слова для юзера %d: %v", userID, err)
		http.Error(w, `{"error": "Не вдалося зберегти слово"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Слово додано до словника!"})
}

// --- 5. ФУНКЦІЇ ОТРИМАННЯ ДАНИХ (GET-ЗАПИТИ) ---

func (h *Handler) GetMyMistakes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ВИПРАВЛЕНО: Використовуємо auth.GetUserID
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	query := `
        SELECT id, wrong_text, correct_text, rule_explanation, created_at 
        FROM saved_mistakes 
        WHERE user_id = $1 
        ORDER BY created_at DESC
    `

	rows, err := h.DB.Query(query, userID)
	if err != nil {
		log.Printf("Помилка отримання помилок: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var mistakes []MistakeResponse
	for rows.Next() {
		var m MistakeResponse
		if err := rows.Scan(&m.ID, &m.WrongText, &m.CorrectText, &m.RuleExplanation, &m.CreatedAt); err != nil {
			log.Printf("Помилка парсингу рядка БД: %v", err)
			continue
		}
		mistakes = append(mistakes, m)
	}

	if mistakes == nil {
		mistakes = []MistakeResponse{} // Повертаємо пустий масив, якщо немає даних
	}
	json.NewEncoder(w).Encode(mistakes)
}

// ДОДАНО: Отримання особистого словника користувача
func (h *Handler) GetMyVocabulary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	query := `
        SELECT id, word, translation, context_sentence, created_at 
        FROM saved_vocabulary 
        WHERE user_id = $1 
        ORDER BY created_at DESC
    `

	rows, err := h.DB.Query(query, userID)
	if err != nil {
		log.Printf("Помилка отримання словника: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var vocab []VocabResponse
	for rows.Next() {
		var v VocabResponse
		if err := rows.Scan(&v.ID, &v.Word, &v.Translation, &v.ContextSentence, &v.CreatedAt); err != nil {
			log.Printf("Помилка парсингу рядка БД (словник): %v", err)
			continue
		}
		vocab = append(vocab, v)
	}

	if vocab == nil {
		vocab = []VocabResponse{} // Повертаємо пустий масив, якщо немає даних
	}

	json.NewEncoder(w).Encode(vocab)
}
func (h *Handler) DeleteVocabulary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	vocabID := r.URL.Query().Get("id")
	if vocabID == "" {
		http.Error(w, `{"error": "Відсутній ID"}`, http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec("DELETE FROM saved_vocabulary WHERE id = $1 AND user_id = $2", vocabID, userID)
	if err != nil {
		log.Printf("Помилка видалення слова: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Видалення збереженої помилки
func (h *Handler) DeleteMistake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	mistakeID := r.URL.Query().Get("id")
	if mistakeID == "" {
		http.Error(w, `{"error": "Відсутній ID"}`, http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec("DELETE FROM saved_mistakes WHERE id = $1 AND user_id = $2", mistakeID, userID)
	if err != nil {
		log.Printf("Помилка видалення помилки: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
