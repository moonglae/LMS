package models

import (
	"encoding/json"
	"time"
)

// User представляє користувача системи
type User struct {
	ID                 int             `json:"id"`
	Email              string          `json:"email"`
	PasswordHash       string          `json:"-"` // Завдяки "-" хеш пароля НІКОЛИ не потрапить у JSON-відповідь на фронтенд
	FirstName          string          `json:"first_name"`
	LastName           string          `json:"last_name"`
	Role               string          `json:"role"`      // Роль (admin/student)
	IsBanned           bool            `json:"is_banned"` // Статус блокування (потрібен для адмінки)
	CreatedAt          time.Time       `json:"created_at"`
	RestrictedFeatures json.RawMessage `json:"restricted_features"` // Зберігає JSON: {"chat": true}
}

type SecurityAlert struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	ActivityType string    `json:"activity_type"`
	Description  string    `json:"description"`
	Resolved     bool      `json:"resolved"`
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

// Flashcard представляє картку для самоперевірки
type Flashcard struct {
	ID        int       `json:"id"`
	ModuleID  int       `json:"module_id"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	CreatedAt time.Time `json:"created_at"`
}

// QuizAttempt фіксує загальний результат проходження тесту (для графіка)
type QuizAttempt struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	ModuleID       int       `json:"module_id"` // ТЕПЕР ТУТ ModuleID!
	Score          int       `json:"score"`
	TotalQuestions int       `json:"total_questions"`
	CompletedAt    time.Time `json:"completed_at"`
}

// UserActiveMistake фіксує помилки користувача на рівні карток
type UserActiveMistake struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	FlashcardID int       `json:"flashcard_id"`
	ErrorCount  int       `json:"error_count"`
	CreatedAt   time.Time `json:"created_at"`
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
