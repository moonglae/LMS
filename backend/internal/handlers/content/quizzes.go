package content

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand" // ДОДАЙ ЦЕ
	"net/http"
	"strconv"
	"strings"
	"time" // ДОДАЙ ЦЕ

	"backend/internal/handlers/auth"
)

func (h *ContentHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	var req CreateQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}
    
    // Переконайся, що імпорт "strings" є у файлі
	if req.ModuleID <= 0 || strings.TrimSpace(req.Title) == "" {
		http.Error(w, `{"error": "Вкажіть назву тесту та модуль"}`, http.StatusBadRequest)
		return
	}

	// 1. Перевірка доступу до модуля
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

	// 2. Перевірка Enrolled
	if ownerID != userID {
		var enrolledCount int
		err = h.DB.QueryRow(`SELECT COUNT(1) FROM enrollments WHERE user_id = $1 AND module_id = $2`, userID, req.ModuleID).Scan(&enrolledCount)
		if enrolledCount == 0 {
			http.Error(w, `{"error": "Ви не маєте прав створювати тести в цьому модулі"}`, http.StatusForbidden)
			return
		}
	}

	// 3. Вставляємо тест (БЕЗ created_by)
	var newQuizID int
	err = h.DB.QueryRow(
		`INSERT INTO quizzes (module_id, title) VALUES ($1, $2) RETURNING id`,
		req.ModuleID, req.Title,
	).Scan(&newQuizID)

	if err != nil {
		log.Printf("CreateQuiz insert error: %v", err)
		http.Error(w, `{"error": "Помилка збереження тесту"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Тест успішно створено",
		"id":      newQuizID,
	})
}


func (h *ContentHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	var req CreateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}
	req.QuestionText = strings.TrimSpace(req.QuestionText)
	if req.QuizID <= 0 || req.QuestionText == "" {
		http.Error(w, `{"error": "Вкажіть питання"}`, http.StatusBadRequest)
		return
	}
	if req.Type != "single" && req.Type != "multiple" && req.Type != "text" {
		http.Error(w, `{"error": "Некоректний тип питання"}`, http.StatusBadRequest)
		return
	}

	minAnswers := 2
	if req.Type == "text" {
		minAnswers = 1
	}
	if len(req.Answers) < minAnswers {
		http.Error(w, `{"error": "Вкажіть принаймні два варіанти відповіді"}`, http.StatusBadRequest)
		return
	}

	var ownerID, moduleID int
	err := h.DB.QueryRow(
		`SELECT m.created_by, q.module_id
         FROM quizzes q
         JOIN modules m ON q.module_id = m.id
         WHERE q.id = $1`,
		req.QuizID,
	).Scan(&ownerID, &moduleID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Тест не знайдено"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "Помилка перевірки прав"}`, http.StatusInternalServerError)
		return
	}
	if ownerID != userID {
		http.Error(w, `{"error": "Ви не власник курсу"}`, http.StatusForbidden)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var questionID int
	err = tx.QueryRow(
		`INSERT INTO questions (quiz_id, question_text, q_type)
         VALUES ($1, $2, $3) RETURNING id`,
		req.QuizID, req.QuestionText, req.Type,
	).Scan(&questionID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, `{"error": "Помилка створення питання"}`, http.StatusInternalServerError)
		return
	}

	correctAnswerFound := false
	for _, answer := range req.Answers {
		answer.AnswerText = strings.TrimSpace(answer.AnswerText)
		if answer.AnswerText == "" {
			_ = tx.Rollback()
			http.Error(w, `{"error": "Усі варіанти відповіді повинні мати текст"}`, http.StatusBadRequest)
			return
		}
		if answer.IsCorrect {
			correctAnswerFound = true
		}
		if _, err = tx.Exec(
			`INSERT INTO answers (question_id, answer_text, is_correct)
             VALUES ($1, $2, $3)`,
			questionID, answer.AnswerText, answer.IsCorrect,
		); err != nil {
			_ = tx.Rollback()
			http.Error(w, `{"error": "Помилка збереження варіантів відповіді"}`, http.StatusInternalServerError)
			return
		}
	}

	if req.Type != "text" && !correctAnswerFound {
		_ = tx.Rollback()
		http.Error(w, `{"error": "Позначте хоча б одну правильну відповідь"}`, http.StatusBadRequest)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, `{"error": "Помилка підтвердження транзакції"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Питання успішно створено",
		"question_id": questionID,
	})
}

func (h *ContentHandler) GetQuizzes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	moduleIDStr := r.URL.Query().Get("module_id")
	moduleID, err := strconv.Atoi(moduleIDStr)
	if err != nil || moduleID <= 0 {
		log.Printf("GetQuizzes invalid module_id=%q", moduleIDStr)
		http.Error(w, `{"error": "Некоректний ID модуля"}`, http.StatusBadRequest)
		return
	}

	log.Printf("GetQuizzes module_id=%d", moduleID)
	rows, err := h.DB.Query(`SELECT id, title FROM quizzes WHERE module_id = $1`, moduleID)
	if err != nil {
		log.Printf("GetQuizzes query error module_id=%d: %v", moduleID, err)
		http.Error(w, `{"error": "Помилка отримання тестів"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var quizzes []QuizResponse
	for rows.Next() {
		var q QuizResponse
		if err := rows.Scan(&q.ID, &q.Title); err != nil {
			log.Printf("GetQuizzes scan error module_id=%d: %v", moduleID, err)
			http.Error(w, `{"error": "Помилка обробки тестів"}`, http.StatusInternalServerError)
			return
		}
		quizzes = append(quizzes, q)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Помилка читання даних"}`, http.StatusInternalServerError)
		return
	}

	if quizzes == nil {
		quizzes = []QuizResponse{}
	}
	_ = json.NewEncoder(w).Encode(quizzes)
}

func (h *ContentHandler) GetQuizQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	quizIDStr := r.URL.Query().Get("quiz_id")
	quizID, err := strconv.Atoi(quizIDStr)
	if err != nil || quizID <= 0 {
		log.Printf("GetQuizQuestions invalid quiz_id=%q", quizIDStr)
		http.Error(w, `{"error": "Некоректний ID тесту"}`, http.StatusBadRequest)
		return
	}

	log.Printf("GetQuizQuestions quiz_id=%d", quizID)
	rows, err := h.DB.Query(
		`SELECT id, question_text, COALESCE(q_type, 'single')
         FROM questions WHERE quiz_id = $1`,
		quizID,
	)
	if err != nil {
		log.Printf("GetQuizQuestions query error quiz_id=%d: %v", quizID, err)
		http.Error(w, `{"error": "Помилка отримання питань"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var questions []QuestionResponse
	for rows.Next() {
		var q QuestionResponse
		if err := rows.Scan(&q.ID, &q.QuestionText, &q.Type); err != nil {
			http.Error(w, `{"error": "Помилка обробки питання"}`, http.StatusInternalServerError)
			return
		}

		optionRows, err := h.DB.Query(
			`SELECT id, answer_text, is_correct
             FROM answers WHERE question_id = $1 ORDER BY id`,
			q.ID,
		)
		if err != nil {
			http.Error(w, `{"error": "Помилка отримання варіантів відповіді"}`, http.StatusInternalServerError)
			return
		}

		var options []AnswerOption
		var correctIDs []int
		for optionRows.Next() {
			var opt AnswerOption
			var isCorrect bool
			if err := optionRows.Scan(&opt.ID, &opt.AnswerText, &isCorrect); err != nil {
				optionRows.Close()
				http.Error(w, `{"error": "Помилка обробки варіанта відповіді"}`, http.StatusInternalServerError)
				return
			}
			if isCorrect {
				correctIDs = append(correctIDs, opt.ID)
				if q.CorrectAnswerID == 0 {
					q.CorrectAnswerID = opt.ID
				}
			}
			options = append(options, opt)
		}
		optionRows.Close()
		if err := optionRows.Err(); err != nil {
			http.Error(w, `{"error": "Помилка читання варіантів відповіді"}`, http.StatusInternalServerError)
			return
		}

		q.Options = options
		q.CorrectAnswerIDs = correctIDs
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetQuizQuestions rows error quiz_id=%d: %v", quizID, err)
		http.Error(w, `{"error": "Помилка обробки питань"}`, http.StatusInternalServerError)
		return
	}

	if questions == nil {
		questions = []QuestionResponse{}
	}
	_ = json.NewEncoder(w).Encode(questions)
}
func (h *ContentHandler) GenerateQuiz(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    moduleID, _ := strconv.Atoi(r.URL.Query().Get("module_id"))
    if moduleID <= 0 {
        http.Error(w, `{"error": "Некоректний ID модуля"}`, http.StatusBadRequest)
        return
    }

    // 1. Забираємо всі картки модуля
    rows, err := h.DB.Query(`SELECT question, answer FROM flashcards WHERE module_id = $1`, moduleID)
    if err != nil {
        http.Error(w, `{"error": "Помилка бази даних"}`, http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    type Card struct { Q string; A string }
    var allCards []Card
    for rows.Next() {
        var c Card
        rows.Scan(&c.Q, &c.A)
        allCards = append(allCards, c)
    }

    if len(allCards) < 4 {
        http.Error(w, `{"error": "Для генерації тесту потрібно мінімум 4 картки в модулі"}`, http.StatusBadRequest)
        return
    }

    // --- НОВИЙ БЛОК: Запис у БД ---
    // Починаємо транзакцію, щоб створити реальний тест
    tx, err := h.DB.Begin()
    if err != nil {
        http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
        return
    }
    defer tx.Rollback() // Відкотить зміни, якщо щось піде не так

    // Створюємо запис про тест
    quizTitle := "Генерація " + time.Now().Format("02.01 15:04")
    var realQuizID int
    err = tx.QueryRow(`INSERT INTO quizzes (module_id, title) VALUES ($1, $2) RETURNING id`, moduleID, quizTitle).Scan(&realQuizID)
    if err != nil {
        log.Printf("Помилка створення тесту: %v", err)
        http.Error(w, `{"error": "Помилка бази даних"}`, http.StatusInternalServerError)
        return
    }

    rand.Seed(time.Now().UnixNano())

    // 2. Генеруємо питання та зберігаємо їх
    var quiz []map[string]interface{}
    for _, card := range allCards {
        var options []string
        options = append(options, card.A) // Додаємо правильну

        // Збираємо інші унікальні відповіді
        var others []string
        for _, c := range allCards {
            if c.A != card.A { 
                others = append(others, c.A)
            }
        }

        // Беремо 3 випадкові з інших
        rand.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
        if len(others) >= 3 {
            options = append(options, others[:3]...)
        } else {
            options = append(options, others...)
        }

        // Перемішуємо варіанти
        rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })

        // ВАЖЛИВО: Зберігаємо питання в БД, щоб отримати реальний ID
        var realQuestionID int
        err = tx.QueryRow(`INSERT INTO questions (quiz_id, question_text, q_type) VALUES ($1, $2, 'single') RETURNING id`, realQuizID, card.Q).Scan(&realQuestionID)
        if err != nil {
            log.Printf("Помилка створення питання: %v", err)
            http.Error(w, `{"error": "Помилка запису питання"}`, http.StatusInternalServerError)
            return
        }

        quiz = append(quiz, map[string]interface{}{
            "id":            realQuestionID, // ТЕПЕР ТУТ РЕАЛЬНИЙ ID З БД!
            "quiz_id":       realQuizID,     // ТЕПЕР ТУТ РЕАЛЬНИЙ ID ТЕСТУ!
            "question_text": card.Q,
            "options":       options,
            "correct":       card.A,
        })
    }

    // Підтверджуємо запис у базу
    if err := tx.Commit(); err != nil {
        http.Error(w, `{"error": "Помилка підтвердження транзакції"}`, http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(quiz)
}
