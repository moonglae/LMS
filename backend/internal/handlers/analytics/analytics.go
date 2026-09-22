package analytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"backend/internal/handlers/auth"
)

type AnalyticsHandler struct {
	DB *sql.DB
}

func NewAnalyticsHandler(db *sql.DB) *AnalyticsHandler {
	return &AnalyticsHandler{DB: db}
}

// --- СТРУКТУРИ ДАНИХ ---

type MistakeResponse struct {
	ID           int    `json:"id"`
	QuestionText string `json:"question_text"`
	ModuleID     int    `json:"module_id"`
	ModuleTitle  string `json:"module_title"`
}

type ProgressItem struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

type LastModule struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type ProfileStatsResponse struct {
	TotalCardsLearned int        `json:"total_cards_learned"`
	TotalQuizzesTaken int        `json:"total_quizzes_taken"`
	CurrentStreak     int        `json:"current_streak"`
	LastModule        LastModule `json:"last_module"`
}

type Goal struct {
	ID          int    `json:"id"`
	Text        string `json:"text"`
	IsCompleted bool   `json:"is_completed"`
}

// --- ФУНКЦІЇ ---

// 1. Отримання списку активних помилок користувача
func (h *AnalyticsHandler) GetActiveMistakes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	// Оновлений SQL-запит (джоінимо flashcards, а не questions)
	query := `
		SELECT 
			uam.id, 
			f.question AS question_text, 
			m.id AS module_id, 
			m.title AS module_title
		FROM user_active_mistakes uam
		JOIN flashcards f ON uam.flashcard_id = f.id
		JOIN modules m ON f.module_id = m.id
		WHERE uam.user_id = $1
		ORDER BY m.id, uam.created_at DESC
	`

	rows, err := h.DB.Query(query, userID)
	if err != nil {
		log.Printf("Помилка отримання помилок для юзера %d: %v", userID, err)
		http.Error(w, `{"error": "Помилка сервера при отриманні помилок"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var mistakes []MistakeResponse
	for rows.Next() {
		var m MistakeResponse
		if err := rows.Scan(&m.ID, &m.QuestionText, &m.ModuleID, &m.ModuleTitle); err != nil {
			log.Printf("Помилка сканування помилки: %v", err)
			continue
		}
		mistakes = append(mistakes, m)
	}

	if mistakes == nil {
		mistakes = []MistakeResponse{} // Віддаємо пустий масив, якщо помилок немає
	}

	json.NewEncoder(w).Encode(mistakes)
}

// 2. Ручне видалення помилки (якщо потрібно)
func (h *AnalyticsHandler) ResolveMistake(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		FlashcardID int `json:"flashcard_id"` // Змінено з question_id
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат"}`, http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec("DELETE FROM user_active_mistakes WHERE user_id = $1 AND flashcard_id = $2", userID, req.FlashcardID)
	if err != nil {
		http.Error(w, `{"error": "Помилка видалення"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// 3. Отримання загальної статистики користувача (для дашборду/модуля)
func (h *AnalyticsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	moduleID := r.URL.Query().Get("module_id")

	var query string
	args := []interface{}{userID}

	query = `
        SELECT 
            COUNT(id) as total_attempts, 
            COALESCE(FLOOR(AVG(score::numeric / NULLIF(total_questions, 0) * 100)), 0) as average_score, 
            COALESCE(SUM(CASE WHEN score = total_questions THEN 1 ELSE 0 END), 0) as perfect_scores
        FROM quiz_attempts
        WHERE user_id = $1
    `
	if moduleID != "" && moduleID != "all" {
		query += ` AND module_id = $2` // Змінено з quiz_id IN ...
		args = append(args, moduleID)
	}

	var totalAttempts, perfectScores int
	var averageScore float64

	err := h.DB.QueryRow(query, args...).Scan(&totalAttempts, &averageScore, &perfectScores)
	if err != nil {
		log.Printf("GetSummary Error: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_attempts": totalAttempts,
		"perfect_scores": perfectScores,
		"average_score":  int(averageScore),
	})
}

// 4. Дані для графіка прогресу
func (h *AnalyticsHandler) GetProgressData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизований", http.StatusUnauthorized)
		return
	}

	query := fmt.Sprintf(`
        SELECT 
            TO_CHAR(completed_at, 'DD.MM') as date, 
            FLOOR(AVG(score::numeric / total_questions * 100))::int as score
        FROM quiz_attempts 
        WHERE user_id = %d 
        GROUP BY DATE(completed_at), TO_CHAR(completed_at, 'DD.MM')
        ORDER BY DATE(completed_at) ASC
    `, userID)

	rows, err := h.DB.Query(query)
	if err != nil {
		log.Printf("Progress Data Error: %v", err)
		http.Error(w, "Помилка отримання даних графіка", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var data []ProgressItem
	for rows.Next() {
		var p ProgressItem
		if err := rows.Scan(&p.Date, &p.Score); err == nil {
			data = append(data, p)
		}
	}

	if data == nil {
		data = []ProgressItem{}
	}

	json.NewEncoder(w).Encode(data)
}

// 5. Отримання комплексної статистики для профілю (стріки, пройдені тести)
func (h *AnalyticsHandler) GetProfileStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	var stats ProfileStatsResponse

	// Рахуємо загальну кількість вивчених слів (унікальних карток, які юзер проходив у тестах)
	// Оскільки ми видалили user_question_attempts, ми просто підраховуємо загальну кількість тестів
	stats.TotalCardsLearned = 0 // Це поле можна або видалити з фронтенду, або рахувати інакше, якщо потрібно

	err := h.DB.QueryRow(`SELECT COUNT(id) FROM quiz_attempts WHERE user_id = $1`, userID).Scan(&stats.TotalQuizzesTaken)
	if err != nil {
		stats.TotalQuizzesTaken = 0
	}

	err = h.DB.QueryRow(`
        SELECT m.id, m.title 
        FROM quiz_attempts qa
        JOIN modules m ON qa.module_id = m.id
        WHERE qa.user_id = $1
        ORDER BY qa.completed_at DESC
        LIMIT 1
    `, userID).Scan(&stats.LastModule.ID, &stats.LastModule.Title)

	if err != nil {
		stats.LastModule = LastModule{ID: 0, Title: ""}
	}

	rows, err := h.DB.Query(`
        SELECT DISTINCT DATE(completed_at) 
        FROM quiz_attempts 
        WHERE user_id = $1 
        ORDER BY DATE(completed_at) DESC
    `, userID)

	if err == nil {
		defer rows.Close()
		streak := 0
		expectedDate := time.Now().Truncate(24 * time.Hour)

		for rows.Next() {
			var dateStr string
			if err := rows.Scan(&dateStr); err != nil {
				continue
			}

			parsedDate, err := time.Parse(time.RFC3339, dateStr)
			if err != nil {
				parsedDate, _ = time.Parse("2006-01-02", dateStr[:10])
			}

			activityDate := parsedDate.Truncate(24 * time.Hour)

			if streak == 0 && (activityDate.Equal(expectedDate) || activityDate.Equal(expectedDate.Add(-24*time.Hour))) {
				streak++
				expectedDate = activityDate.Add(-24 * time.Hour)
			} else if activityDate.Equal(expectedDate) {
				streak++
				expectedDate = expectedDate.Add(-24 * time.Hour)
			} else {
				break
			}
		}
		stats.CurrentStreak = streak
	}

	json.NewEncoder(w).Encode(stats)
}

func (h *AnalyticsHandler) GetGoals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := auth.GetUserID(r.Context())

	rows, err := h.DB.Query(`SELECT id, text, is_completed FROM user_goals WHERE user_id = $1 ORDER BY created_at ASC`, userID)
	if err != nil {
		http.Error(w, `{"error": "Помилка БД"}`, 500)
		return
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var g Goal
		rows.Scan(&g.ID, &g.Text, &g.IsCompleted)
		goals = append(goals, g)
	}
	if goals == nil {
		goals = []Goal{}
	}
	json.NewEncoder(w).Encode(goals)
}

func (h *AnalyticsHandler) AddGoal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := auth.GetUserID(r.Context())

	var req struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var newGoal Goal
	newGoal.Text = req.Text
	newGoal.IsCompleted = false

	err := h.DB.QueryRow(
		`INSERT INTO user_goals (user_id, text) VALUES ($1, $2) RETURNING id`,
		userID, req.Text,
	).Scan(&newGoal.ID)

	if err != nil {
		http.Error(w, `{"error": "Помилка збереження"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(newGoal)
}

func (h *AnalyticsHandler) ToggleGoal(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	var req struct {
		ID          int  `json:"id"`
		IsCompleted bool `json:"is_completed"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.DB.Exec(`UPDATE user_goals SET is_completed = $1 WHERE id = $2 AND user_id = $3`, req.IsCompleted, req.ID, userID)
	w.WriteHeader(http.StatusOK)
}

func (h *AnalyticsHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	id := r.URL.Query().Get("id")
	h.DB.Exec(`DELETE FROM user_goals WHERE id = $1 AND user_id = $2`, id, userID)
	w.WriteHeader(http.StatusOK)
}

func (h *AnalyticsHandler) GetMistakesQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	moduleIDStr := r.URL.Query().Get("module_id")
	if moduleIDStr == "" {
		http.Error(w, `{"error": "Відсутній module_id"}`, http.StatusBadRequest)
		return
	}

	// 1. Отримуємо всі помилки юзера для цього модуля
	query := `
		SELECT f.id, f.question, f.answer 
		FROM user_active_mistakes uam
		JOIN flashcards f ON uam.flashcard_id = f.id
		WHERE uam.user_id = $1 AND f.module_id = $2
	`
	rows, err := h.DB.Query(query, userID, moduleIDStr)
	if err != nil {
		log.Printf("Помилка БД в GetMistakesQuiz: %v", err)
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type MistakeCard struct {
		FlashcardID int
		Question    string
		Answer      string
	}
	var mistakes []MistakeCard
	for rows.Next() {
		var mc MistakeCard
		if err := rows.Scan(&mc.FlashcardID, &mc.Question, &mc.Answer); err == nil {
			mistakes = append(mistakes, mc)
		}
	}

	if len(mistakes) == 0 {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	// 2. Отримуємо всі відповіді з цього ж модуля (для генерації хибних варіантів A, B, C)
	allAnsRows, err := h.DB.Query(`SELECT answer FROM flashcards WHERE module_id = $1`, moduleIDStr)
	var allAnswers []string
	if err == nil {
		defer allAnsRows.Close()
		for allAnsRows.Next() {
			var ans string
			if err := allAnsRows.Scan(&ans); err == nil {
				allAnswers = append(allAnswers, ans)
			}
		}
	}

	// 3. Формуємо результат у форматі, який очікує наш оновлений Quiz.tsx
	type QuizQuestion struct {
		FlashcardID int      `json:"flashcard_id"`
		Question    string   `json:"question"`
		Options     []string `json:"options"`
		Answer      string   `json:"answer"`
	}

	var quiz []QuizQuestion
	for _, mc := range mistakes {
		var wrongOptions []string

		// Відфільтровуємо правильну відповідь
		for _, a := range allAnswers {
			if a != mc.Answer {
				wrongOptions = append(wrongOptions, a)
			}
		}

		// Перемішуємо хибні варіанти
		rand.Shuffle(len(wrongOptions), func(i, j int) {
			wrongOptions[i], wrongOptions[j] = wrongOptions[j], wrongOptions[i]
		})

		// Беремо максимум 3 хибних
		if len(wrongOptions) > 3 {
			wrongOptions = wrongOptions[:3]
		}

		// Додаємо правильну відповідь і перемішуємо ще раз (щоб вона не була завжди останньою)
		options := append(wrongOptions, mc.Answer)
		rand.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})

		quiz = append(quiz, QuizQuestion{
			FlashcardID: mc.FlashcardID,
			Question:    mc.Question,
			Options:     options,
			Answer:      mc.Answer,
		})
	}

	json.NewEncoder(w).Encode(quiz)
}
