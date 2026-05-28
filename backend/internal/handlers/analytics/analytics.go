package analytics

import (
	"database/sql"
	"encoding/json"
	"log" // Додано лог, бо його не вистачало
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

type ModuleReportItem struct {
	StudentID        int    `json:"student_id"`
	StudentName      string `json:"student_name"`
	Attempts         int    `json:"attempts"`
	AverageScore     int    `json:"average_score"`
	RegistrationDate string `json:"registration_date"`
}

type StudentReportResponse struct {
	StudentID     int    `json:"student_id"`
	StudentName   string `json:"student_name"`
	ModuleID      int    `json:"module_id"`
	QuizzesTaken  int    `json:"quizzes_taken"`
	AverageScore  int    `json:"average_score"`
	PerfectScores int    `json:"perfect_scores"`
	MistakesCount int    `json:"mistakes_count"`
}

// 1. Збереження результатів і АВТОМАТИЧНЕ видалення помилок
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

    // Зберігаємо спробу з поточною датою та часом (completed_at)
    _, err = tx.Exec(
        `INSERT INTO quiz_attempts (user_id, quiz_id, score, total_questions, completed_at) 
         VALUES ($1, $2, $3, $4, $5)`,
        userID, req.QuizID, req.Score, req.TotalQuestions, time.Now(),
    )
    if err != nil {
        http.Error(w, "Помилка запису результату", http.StatusInternalServerError)
        return
    }

    // Оновлюємо статус відповідей та помилок
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

// 2. Отримання списку помилок з інформацією про модуль
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
		ID           int    `json:"id"`
		Text         string `json:"question_text"`
		ModuleID     int    `json:"module_id"`
		ModuleTitle  string `json:"module_title"`
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

// 3. Генератор тесту з помилок (з підтримкою конкретного модуля)
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

func (h *AnalyticsHandler) ResolveMistake(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	var req struct { QuestionID int `json:"question_id"` }
	json.NewDecoder(r.Body).Decode(&req)
	h.DB.Exec("DELETE FROM user_active_mistakes WHERE user_id = $1 AND question_id = $2", userID, req.QuestionID)
	w.WriteHeader(http.StatusOK)
}

func (h *AnalyticsHandler) GetMistakesTest(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	rows, _ := h.DB.Query(`SELECT q.id, q.question_text, COALESCE(m.title, ''), COALESCE(m.id, 0), COALESCE(qu.id, 0) FROM questions q JOIN user_question_attempts uqa ON q.id = uqa.question_id LEFT JOIN quizzes qu ON q.quiz_id = qu.id LEFT JOIN modules m ON qu.module_id = m.id WHERE uqa.user_id = $1 AND uqa.is_correct = false GROUP BY q.id, q.question_text, m.title, m.id, qu.id`, userID)
	defer rows.Close()
	var mistakes []MistakeQuestionResponse
	for rows.Next() { var m MistakeQuestionResponse; rows.Scan(&m.ID, &m.QuestionText, &m.ModuleTitle, &m.ModuleID, &m.QuizID); mistakes = append(mistakes, m) }
	json.NewEncoder(w).Encode(mistakes)
}

func (h *AnalyticsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	role, _ := auth.GetUserRole(r.Context())
	moduleID := r.URL.Query().Get("module_id")

	var query string
	args := []interface{}{userID}

	if role == "teacher" {
		// ВИКЛАДАЧ: Рахуємо спроби всіх студентів на всіх модулях цього викладача
		query = `
			SELECT 
				COUNT(qa.id), 
				COALESCE(FLOOR(AVG(qa.score::numeric / NULLIF(qa.total_questions, 0) * 100)), 0), 
				COALESCE(SUM(CASE WHEN qa.score = qa.total_questions THEN 1 ELSE 0 END), 0) 
			FROM quiz_attempts qa
			JOIN quizzes q ON qa.quiz_id = q.id
			JOIN modules m ON q.module_id = m.id
			JOIN users u ON qa.user_id = u.id
			WHERE m.created_by = $1 AND u.role = 'student'
		`
		if moduleID != "" && moduleID != "all" {
			query += ` AND m.id = $2`
			args = append(args, moduleID)
		}
	} else {
		// СТУДЕНТ: Рахуємо тільки його власні спроби
		query = `
			SELECT 
				COUNT(qa.id), 
				COALESCE(FLOOR(AVG(qa.score::numeric / NULLIF(qa.total_questions, 0) * 100)), 0), 
				COALESCE(SUM(CASE WHEN qa.score = qa.total_questions THEN 1 ELSE 0 END), 0) 
			FROM quiz_attempts qa
			JOIN users u ON qa.user_id = u.id
			WHERE qa.user_id = $1 AND u.role = 'student'
		`
		if moduleID != "" && moduleID != "all" {
			query += ` AND qa.quiz_id IN (SELECT id FROM quizzes WHERE module_id = $2)`
			args = append(args, moduleID)
		}
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

func (h *AnalyticsHandler) GetStudentReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	teacherID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	studentID, _ := strconv.Atoi(r.URL.Query().Get("student_id"))
	moduleID, _ := strconv.Atoi(r.URL.Query().Get("module_id"))

	if studentID == 0 || moduleID == 0 {
		http.Error(w, `{"error": "Некоректні дані"}`, http.StatusBadRequest)
		return
	}

	// 1. Перевірка: чи належить модуль вчителю
	var ownerID int
	err := h.DB.QueryRow(`SELECT created_by FROM modules WHERE id = $1`, moduleID).Scan(&ownerID)
	if err != nil {
		http.Error(w, `{"error": "Модуль не знайдено"}`, http.StatusNotFound)
		return
	}
	if ownerID != teacherID {
		http.Error(w, `{"error": "Заборонено"}`, http.StatusForbidden)
		return
	}

	// 2. Ізольований та точний SQL-запит (CTE)
	query := `
		WITH student_quizzes AS (
			SELECT 
				user_id,
				COUNT(id) as attempts,
				COALESCE(FLOOR(AVG(score::numeric / NULLIF(total_questions, 0) * 100)), 0) as avg_score,
				SUM(CASE WHEN score = total_questions THEN 1 ELSE 0 END) as perfect
			FROM quiz_attempts
			WHERE user_id = $1 AND quiz_id IN (SELECT id FROM quizzes WHERE module_id = $2)
			GROUP BY user_id
		),
		student_mistakes AS (
			SELECT 
				user_id,
				COUNT(DISTINCT question_id) as mistakes
			FROM user_question_attempts
			WHERE user_id = $1 AND is_correct = false AND question_id IN (
				SELECT q.id FROM questions q 
				JOIN quizzes qu ON q.quiz_id = qu.id 
				WHERE qu.module_id = $2
			)
			GROUP BY user_id
		)
		SELECT 
			u.id, 
			CONCAT(u.first_name, ' ', u.last_name),
			COALESCE(sq.attempts, 0),
			COALESCE(sq.avg_score, 0),
			COALESCE(sq.perfect, 0),
			COALESCE(sm.mistakes, 0)
		FROM users u
		LEFT JOIN student_quizzes sq ON u.id = sq.user_id
		LEFT JOIN student_mistakes sm ON u.id = sm.user_id
		WHERE u.id = $1
	`

	var report StudentReportResponse
	err = h.DB.QueryRow(query, studentID, moduleID).Scan(
		&report.StudentID, 
		&report.StudentName, 
		&report.QuizzesTaken, 
		&report.AverageScore, 
		&report.PerfectScores, 
		&report.MistakesCount,
	)

	if err != nil {
		log.Printf("Report Query Error: %v", err)
		http.Error(w, `{"error": "Помилка формування звіту"}`, http.StatusInternalServerError)
		return
	}

	report.ModuleID = moduleID
	_ = json.NewEncoder(w).Encode(report)
}

func (h *AnalyticsHandler) GetModuleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	teacherID, _ := auth.GetUserID(r.Context())
	moduleID, _ := strconv.Atoi(r.URL.Query().Get("module_id"))

	var ownerID int
	if err := h.DB.QueryRow(`SELECT created_by FROM modules WHERE id = $1`, moduleID).Scan(&ownerID); err != nil || ownerID != teacherID {
		http.Error(w, "Заборонено", 403); return
	}

	query := `
        SELECT u.id, 
               CONCAT(u.first_name, ' ', u.last_name),
               COUNT(DISTINCT qa.id)::int, 
               COALESCE(FLOOR(AVG(qa.score::numeric / NULLIF(qa.total_questions, 0) * 100)), 0)::int,
               TO_CHAR(u.created_at, 'DD.MM.YYYY')
        FROM users u
        JOIN enrollments e ON u.id = e.user_id
        LEFT JOIN quiz_attempts qa ON u.id = qa.user_id AND qa.quiz_id IN (SELECT id FROM quizzes WHERE module_id = $1)
        WHERE e.module_id = $1
        GROUP BY u.id, u.first_name, u.last_name, u.created_at
    `
	rows, err := h.DB.Query(query, moduleID)
	if err != nil { log.Printf("Report Error: %v", err); http.Error(w, "Помилка БД", 500); return }
	defer rows.Close()

	var report []ModuleReportItem
	for rows.Next() {
		var item ModuleReportItem
		err := rows.Scan(&item.StudentID, &item.StudentName, &item.Attempts, &item.AverageScore, &item.RegistrationDate)
		if err == nil { report = append(report, item) }
	}
	json.NewEncoder(w).Encode(report)
}
func (h *AnalyticsHandler) GetStudentMistakes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	studentID := r.URL.Query().Get("student_id")
	moduleID := r.URL.Query().Get("module_id")

	// Повертаємо текст питань, де студент помилився
	query := `
		SELECT DISTINCT q.question_text
		FROM questions q
		JOIN user_question_attempts uqa ON q.id = uqa.question_id
		JOIN quizzes qu ON q.quiz_id = qu.id
		WHERE uqa.user_id = $1 
		AND qu.module_id = $2 
		AND uqa.is_correct = false
	`
	rows, err := h.DB.Query(query, studentID, moduleID)
	if err != nil {
		http.Error(w, "Помилка БД", 500); return
	}
	defer rows.Close()

	var mistakes []string
	for rows.Next() {
		var text string
		rows.Scan(&text)
		mistakes = append(mistakes, text)
	}
	if mistakes == nil { mistakes = []string{} }
	json.NewEncoder(w).Encode(mistakes)
}
type ProgressItem struct {
    Date  string `json:"date"`
    Score int    `json:"score"`
}

func (h *AnalyticsHandler) GetProgressData(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, "Неавторизований", http.StatusUnauthorized)
        return
    }

    // Беремо середній бал за кожен день
    query := `
        SELECT 
            TO_CHAR(completed_at, 'DD.MM') as date, 
            FLOOR(AVG(score::numeric / total_questions * 100))::int as score
        FROM quiz_attempts 
        WHERE user_id = $1 
        GROUP BY DATE(completed_at), TO_CHAR(completed_at, 'DD.MM')
        ORDER BY DATE(completed_at) ASC
    `
    
    rows, err := h.DB.Query(query, userID)
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