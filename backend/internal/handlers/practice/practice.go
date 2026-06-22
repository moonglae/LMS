package practice

import (
	"backend/internal/handlers/auth"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// --- 1. ГОЛОВНА СТРУКТУРА ХЕНДЛЕРА ---

// Handler зберігає пул з'єднань з базою даних.
// Це дозволяє всім функціям у цьому файлі робити запити до БД через h.DB
type Handler struct {
	DB *sql.DB
}

// --- 2. СТРУКТУРИ ДЛЯ ВХІДНИХ ДАНИХ (ВІД REACT) ---

// SaveMistakeRequest - те, що приходить, коли ти тиснеш "Зберегти помилку"
type SaveMistakeRequest struct {
	WrongText       string `json:"wrong_text"`
	CorrectText     string `json:"correct_text"`
	RuleExplanation string `json:"rule_explanation"`
}

// SaveVocabRequest - те, що приходить, коли ти додаєш нове слово
type SaveVocabRequest struct {
	Word            string `json:"word"`
	Translation     string `json:"translation"`
	ContextSentence string `json:"context_sentence"`
}

// --- 3. СТРУКТУРИ ДЛЯ ВІДПОВІДЕЙ (ДЛЯ ВКЛАДКИ "ВИВЧЕННЯ") ---

// Ці структури майже ідентичні тим, що ми писали для БД,
// але вони використовуються для того, щоб відправити масив даних назад у React
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

// SaveMistake приймає помилку від ШІ та зберігає її у твій зошит
func (h *Handler) SaveMistake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Отримуємо ID користувача з токена (middleware)
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	// 2. Читаємо JSON від клієнта
	var req SaveMistakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	// 3. Робимо запит до БД
	// Використовуємо Exec, бо нам не треба нічого читати з бази, тільки записати
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

	// 4. Успіх!
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Помилку збережено!"})
}

// SaveVocabulary зберігає нове слово у твій словник
func (h *Handler) SaveVocabulary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

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

// GetMyMistakes віддає React-у всі збережені помилки користувача для відображення карток
func (h *Handler) GetMyMistakes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Ідентифікуємо користувача
	userIDValue := r.Context().Value("userID")
	if userIDValue == nil {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}
	userID := userIDValue.(int)

	// 2. Робимо вибірку з БД (сортуємо від найновіших до найстаріших)
	query := `
		SELECT id, wrong_text, correct_text, rule_explanation, created_at 
		FROM saved_mistakes 
		WHERE user_id = $1 
		ORDER BY created_at DESC
	`

	// Використовуємо Query, бо очікуємо багато рядків у відповідь
	rows, err := h.DB.Query(query, userID)
	if err != nil {
		log.Printf("Помилка отримання помилок: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close() // Обов'язково закриваємо з'єднання після читання!

	// 3. Збираємо результати в масив (slice)
	var mistakes []MistakeResponse
	for rows.Next() {
		var m MistakeResponse
		// Scan "розкладає" дані з рядка БД по змінних нашої структури
		if err := rows.Scan(&m.ID, &m.WrongText, &m.CorrectText, &m.RuleExplanation, &m.CreatedAt); err != nil {
			log.Printf("Помилка парсингу рядка БД: %v", err)
			continue // Якщо один рядок битий, просто йдемо до наступного
		}
		mistakes = append(mistakes, m)
	}

	// 4. Відправляємо масив JSON-ом на клієнт
	// Якщо помилок немає, відправиться пустий масив [], що абсолютно нормально для React
	json.NewEncoder(w).Encode(mistakes)
}
