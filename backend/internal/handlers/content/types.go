package content

// --- СТРУКТУРИ ДЛЯ ВІДПОВІДЕЙ (GET) ---
type ModuleResponse struct {
    ID           int    `json:"id"`
    Title        string `json:"title"`
    Description  string `json:"description"`
    Theory       string `json:"theory"`    // Обов'язково додайте це поле
    InviteCode   string `json:"invite_code"`
    StudentCount int    `json:"student_count"`
    CreatedBy    int    `json:"created_by"`
}

type FlashcardResponse struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type QuizResponse struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type QuestionResponse struct {
	ID               int            `json:"id"`
	QuestionText     string         `json:"question_text"`
	Type             string         `json:"type"`
	Options          []AnswerOption `json:"options"`
	CorrectAnswerID  int            `json:"correct_answer_id,omitempty"`
	CorrectAnswerIDs []int          `json:"correct_answer_ids,omitempty"`
}

type AnswerOption struct {
	ID         int    `json:"id"`
	AnswerText string `json:"answer_text"`
}

type ModuleStudentResponse struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// --- СТРУКТУРИ ДЛЯ ЗАПИТІВ (POST) ---
type CreateModuleRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CreateFlashcardRequest struct {
	ModuleID int    `json:"module_id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type CreateQuizRequest struct {
	ModuleID int    `json:"module_id"`
	Title    string `json:"title"`
}

type CreateAnswerOptionRequest struct {
	AnswerText string `json:"answer_text"`
	IsCorrect  bool   `json:"is_correct"`
}

type CreateQuestionRequest struct {
	QuizID       int                         `json:"quiz_id"`
	QuestionText string                      `json:"question_text"`
	Type         string                      `json:"type"`
	Answers      []CreateAnswerOptionRequest `json:"answers"`
}
