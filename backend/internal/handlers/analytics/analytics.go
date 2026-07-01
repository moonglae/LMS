package analytics

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"math/rand"

	"backend/internal/handlers/auth"
)

type AnalyticsHandler struct {
	DB *sql.DB
}

func NewAnalyticsHandler(db *sql.DB) *AnalyticsHandler {
	return &AnalyticsHandler{DB: db}
}

// --- СТРУКТУРИ ДАНИХ ---

type QuestionResult struct {
	QuestionID int  `json:"question_id"`
	IsCorrect  bool `json:"is_correct"`
}

type QuizSubmitRequest struct {
	QuizID         int              `json:"quiz_id"`
	Score          int              `json:"score"`
	TotalQuestions int              `json:"total_questions"`
	Answers        []QuestionResult `json:"answers"`
}

type MistakeQuestionResponse struct {
	ID           int    `json:"id"`
	QuestionText string `json:"question_text"`
	ModuleTitle  string `json:"module_title"`
	ModuleID     int    `json:"module_id"`
	QuizID       int    `json:"quiz_id"`
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

// 1. Збереження результатів і автоматичне оновлення списку помилок
func (h *AnalyticsHandler) SubmitQuizAttempt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	var req QuizSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат"}`, http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, "Помилка сервера", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO quiz_attempts (user_id, quiz_id, score, total_questions, completed_at) 
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, req.QuizID, req.Score, req.TotalQuestions, time.Now(),
	)
	if err != nil {
		http.Error(w, "Помилка запису результату", http.StatusInternalServerError)
		return
	}

	for _, ans := range req.Answers {
		tx.Exec(`INSERT INTO user_question_attempts (user_id, question_id, is_correct, attempted_at) 
				 VALUES ($1, $2, $3, $4)`,
			userID, ans.QuestionID, ans.IsCorrect, time.Now())

		if !ans.IsCorrect {
			tx.Exec(`INSERT INTO user_active_mistakes (user_id, question_id) 
					 VALUES ($1, $2) ON CONFLICT (user_id, question_id) DO NOTHING`,
				userID, ans.QuestionID)
		} else {
			tx.Exec(`DELETE FROM user_active_mistakes WHERE user_id = $1 AND question_id = $2`, 
				userID, ans.QuestionID)
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Помилка фіксації транзакції", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Результат збережено"})
}

// 2. Отримання списку активних помилок користувача
func (h *AnalyticsHandler) GetActiveMistakes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := auth.GetUserID(r.Context())
	
	query := `
		SELECT q.id, q.question_text, m.id, m.title 
		FROM questions q 
		JOIN user_active_mistakes uam ON q.id = uam.question_id 
		JOIN quizzes qu ON q.quiz_id = qu.id
		JOIN modules m ON qu.module_id = m.id
		WHERE uam.user_id = $1
	`
	rows, _ := h.DB.Query(query, userID)
	defer rows.Close()

	type Mistake struct {
		ID          int    `json:"id"`
		Text        string `json:"question_text"`
		ModuleID    int    `json:"module_id"`
		ModuleTitle string `json:"module_title"`
	}
	
	var mistakes []Mistake
	for rows.Next() { 
		var m Mistake
		rows.Scan(&m.ID, &m.Text, &m.ModuleID, &m.ModuleTitle)
		mistakes = append(mistakes, m) 
	}
	if mistakes == nil { mistakes = []Mistake{} }
	json.NewEncoder(w).Encode(mistakes)
}

// 3. Генератор тесту з помилок
func (h *AnalyticsHandler) GetMistakesQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := auth.GetUserID(r.Context())
	moduleIDStr := r.URL.Query().Get("module_id")

	query := `
		SELECT q.id, q.quiz_id, q.question_text, qu.module_id 
		FROM user_active_mistakes uam
		JOIN questions q ON uam.question_id = q.id
		JOIN quizzes qu ON q.quiz_id = qu.id
		WHERE uam.user_id = $1
	`
	var args []interface{}
	args = append(args, userID)

	if moduleIDStr != "" && moduleIDStr != "null" {
		modID, _ := strconv.Atoi(moduleIDStr)
		query += ` AND qu.module_id = $2`
		args = append(args, modID)
	}

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, `{"error": "Помилка БД"}`, 500)
		return
	}
	defer rows.Close()

	var quiz []map[string]interface{}
	rand.Seed(time.Now().UnixNano())

	for rows.Next() {
		var qID, quizID, modID int
		var qText string
		rows.Scan(&qID, &quizID, &qText, &modID)

		var options []string
		var correct string

		err := h.DB.QueryRow(`SELECT answer FROM flashcards WHERE question = $1 AND module_id = $2 LIMIT 1`, qText, modID).Scan(&correct)
		if err == nil {
			options = append(options, correct)
			otherAns, _ := h.DB.Query(`SELECT answer FROM flashcards WHERE module_id = $1 AND answer != $2 ORDER BY RANDOM() LIMIT 3`, modID, correct)
			for otherAns.Next() {
				var oa string
				otherAns.Scan(&oa)
				options = append(options, oa)
			}
			otherAns.Close()
			
			rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
			quiz = append(quiz, map[string]interface{}{
				"id": qID, "quiz_id": quizID, "question_text": qText, "options": options, "correct": correct,
			})
		}
	}

	json.NewEncoder(w).Encode(quiz)
}

// 4. Ручне видалення помилки
func (h *AnalyticsHandler) ResolveMistake(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	var req struct { QuestionID int `json:"question_id"` }
	json.NewDecoder(r.Body).Decode(&req)
	h.DB.Exec("DELETE FROM user_active_mistakes WHERE user_id = $1 AND question_id = $2", userID, req.QuestionID)
	w.WriteHeader(http.StatusOK)
}

// 5. Отримання загальної статистики користувача (для дашборду/модуля)
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
			COUNT(qa.id), 
			COALESCE(FLOOR(AVG(qa.score::numeric / NULLIF(qa.total_questions, 0) * 100)), 0), 
			COALESCE(SUM(CASE WHEN qa.score = qa.total_questions THEN 1 ELSE 0 END), 0) 
		FROM quiz_attempts qa
		WHERE qa.user_id = $1
	`
	if moduleID != "" && moduleID != "all" {
		query += ` AND qa.quiz_id IN (SELECT id FROM quizzes WHERE module_id = $2)`
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

// 6. Дані для графіка прогресу
func (h *AnalyticsHandler) GetProgressData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизований", http.StatusUnauthorized)
		return
	}

	// 1. Формуємо запит одразу з userID за допомогою fmt.Sprintf
	// %d заміниться на число userID
	query := fmt.Sprintf(`
		SELECT 
			TO_CHAR(completed_at, 'DD.MM') as date, 
			FLOOR(AVG(score::numeric / total_questions * 100))::int as score
		FROM quiz_attempts 
		WHERE user_id = %d 
		GROUP BY DATE(completed_at), TO_CHAR(completed_at, 'DD.MM')
		ORDER BY DATE(completed_at) ASC
	`, userID)
	
	// 2. Викликаємо Query ТІЛЬКИ з текстом запиту (без userID другим параметром)
	// Це змусить Go використати простий протокол без кешування шаблонів
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
		err := rows.Scan(&p.Date, &p.Score)
		if err == nil {
			data = append(data, p)
		}
	}

	if data == nil {
		data = []ProgressItem{}
	}
	
	json.NewEncoder(w).Encode(data)
}

// 7. Отримання комплексної статистики для профілю (стріки, пройдені тести)
func (h *AnalyticsHandler) GetProfileStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	var stats ProfileStatsResponse

	err := h.DB.QueryRow(`SELECT COUNT(id) FROM user_question_attempts WHERE user_id = $1`, userID).Scan(&stats.TotalCardsLearned)
	if err != nil {
		stats.TotalCardsLearned = 0
	}

	err = h.DB.QueryRow(`SELECT COUNT(id) FROM quiz_attempts WHERE user_id = $1`, userID).Scan(&stats.TotalQuizzesTaken)
	if err != nil {
		stats.TotalQuizzesTaken = 0
	}

	err = h.DB.QueryRow(`
		SELECT m.id, m.title 
		FROM quiz_attempts qa
		JOIN quizzes q ON qa.quiz_id = q.id
		JOIN modules m ON q.module_id = m.id
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
	
	var req struct { Text string `json:"text"` }
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
