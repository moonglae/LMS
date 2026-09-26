package content

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"backend/internal/handlers/auth"
)

// GenerateQuiz генерує тест "на льоту" з карток модуля БЕЗ збереження в БД
func (h *ContentHandler) GenerateQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleID, _ := strconv.Atoi(r.URL.Query().Get("module_id"))
	if moduleID <= 0 {
		http.Error(w, `{"error": "Некоректний ID модуля"}`, http.StatusBadRequest)
		return
	}

	// 1. Забираємо всі картки модуля (ID, Питання, Відповідь)
	rows, err := h.DB.Query(`SELECT id, question, answer FROM flashcards WHERE module_id = $1`, moduleID)
	if err != nil {
		log.Printf("GenerateQuiz DB error: %v", err)
		http.Error(w, `{"error": "Помилка бази даних"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Card struct {
		ID int
		Q  string
		A  string
	}
	var allCards []Card
	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.Q, &c.A); err != nil {
			continue
		}
		allCards = append(allCards, c)
	}

	if len(allCards) < 4 {
		http.Error(w, `{"error": "Для генерації тесту потрібно мінімум 4 картки в модулі"}`, http.StatusBadRequest)
		return
	}

	rand.Seed(time.Now().UnixNano())

	// 2. Генеруємо питання БЕЗ ЗАПИСУ в базу (on the fly)
	var generatedTest []GeneratedTestQuestion
	for _, card := range allCards {
		var options []string
		options = append(options, card.A) // Правильна відповідь

		// Збираємо дистрактори (неправильні варіанти)
		var others []string
		for _, c := range allCards {
			if c.A != card.A {
				others = append(others, c.A)
			}
		}

		// Беремо 3 випадкові дистрактори
		rand.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
		if len(others) >= 3 {
			options = append(options, others[:3]...)
		} else {
			options = append(options, others...) // Якщо слів мало, беремо всі, що є
		}

		// Перемішуємо всі варіанти, щоб правильна відповідь не була завжди першою
		rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })

		// Додаємо згенероване питання до масиву
		generatedTest = append(generatedTest, GeneratedTestQuestion{
			FlashcardID: card.ID, // Прив'язка до конкретної картки (для сторінки помилок)
			Question:    card.Q,
			Options:     options,
			Answer:      card.A, // Віддаємо клієнту для перевірки на фронтенді
		})
	}

	// Віддаємо готовий тест фронтенду (у базі він не існує!)
	json.NewEncoder(w).Encode(generatedTest)
}

// SubmitTestResult зберігає фінальний бал та записує помилки (і ВИДАЛЯЄ виправлені)
func (h *ContentHandler) SubmitTestResult(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	var req SubmitTestResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	if req.ModuleID <= 0 || req.TotalQuestions <= 0 {
		http.Error(w, `{"error": "Некоректні дані тесту"}`, http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// 1. Зберігаємо загальний результат (для графіка)
	_, err = tx.Exec(
		`INSERT INTO quiz_attempts (user_id, module_id, score, total_questions) VALUES ($1, $2, $3, $4)`,
		userID, req.ModuleID, req.Score, req.TotalQuestions,
	)
	if err != nil {
		log.Printf("SubmitTestResult score error: %v", err)
		http.Error(w, `{"error": "Помилка збереження результату"}`, http.StatusInternalServerError)
		return
	}

	// 2. Зберігаємо помилки (Upsert: якщо помилка вже є - збільшуємо лічильник, якщо ні - додаємо)
	for _, flashcardID := range req.MistakeFlashcardIDs {
		_, err = tx.Exec(`
			INSERT INTO user_active_mistakes (user_id, flashcard_id, error_count) 
			VALUES ($1, $2, 1)
			ON CONFLICT (user_id, flashcard_id) 
			DO UPDATE SET error_count = user_active_mistakes.error_count + 1
		`, userID, flashcardID)
		if err != nil {
			log.Printf("SubmitTestResult mistake error for flashcard %d: %v", flashcardID, err)
			http.Error(w, `{"error": "Помилка збереження помилок"}`, http.StatusInternalServerError)
			return
		}
	}

	// 3. НОВЕ: ВИДАЛЯЄМО правильні відповіді з таблиці помилок
	for _, flashcardID := range req.CorrectFlashcardIDs {
		_, err = tx.Exec(`
			DELETE FROM user_active_mistakes 
			WHERE user_id = $1 AND flashcard_id = $2
		`, userID, flashcardID)
		if err != nil {
			log.Printf("SubmitTestResult clear mistake error for flashcard %d: %v", flashcardID, err)
			http.Error(w, `{"error": "Помилка очищення виправлених помилок"}`, http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error": "Помилка транзакції"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Результати збережено"})
}
