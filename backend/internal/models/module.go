package models

import "time"

// User представляє користувача системи
type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Завдяки "-" хеш пароля НІКОЛИ не потрапить у JSON-відповідь на фронтенд
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// Module представляє навчальний курс
type Module struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// Theory представляє теоретичний матеріал у форматі Markdown
type Theory struct {
	ID        int       `json:"id"`
	ModuleID  int       `json:"module_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Flashcard представляє картку для самоперевірки (підготовлено для TTS)
type Flashcard struct {
	ID        int       `json:"id"`
	ModuleID  int       `json:"module_id"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	CreatedAt time.Time `json:"created_at"`
}

// Quiz представляє тест
type Quiz struct {
	ID        int       `json:"id"`
	ModuleID  int       `json:"module_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Question представляє питання в тесті
type Question struct {
	ID           int       `json:"id"`
	QuizID       int       `json:"quiz_id"`
	QuestionText string    `json:"question_text"`
	Type         string    `json:"q_type"` // 'single', 'multiple', 'text'
	CreatedAt    time.Time `json:"created_at"`
}

// Answer представляє варіант відповіді
type Answer struct {
	ID         int    `json:"id"`
	QuestionID int    `json:"question_id"`
	AnswerText string `json:"answer_text"`
	IsCorrect  bool   `json:"is_correct"`
}

// QuizAttempt фіксує загальний результат проходження тесту
type QuizAttempt struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	QuizID         int       `json:"quiz_id"`
	Score          int       `json:"score"`
	TotalQuestions int       `json:"total_questions"`
	CompletedAt    time.Time `json:"completed_at"`
}

// UserQuestionAttempt фіксує відповіді на конкретні питання (для роботи над помилками)
type UserQuestionAttempt struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	QuestionID  int       `json:"question_id"`
	IsCorrect   bool      `json:"is_correct"`
	AttemptedAt time.Time `json:"attempted_at"`
}

type SavedMistake struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	WrongText       string    `json:"wrong_text"`
	CorrectText     string    `json:"correct_text"`
	RuleExplanation string    `json:"rule_explanation"`
	CreatedAt       time.Time `json:"created_at"`
}
type SavedVocabulary struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	Word            string    `json:"word"`
	Translation     string    `json:"translation"`
	ContextSentence string    `json:"context_sentence"`
	CreatedAt       time.Time `json:"created_at"`
}
