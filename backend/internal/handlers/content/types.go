package content

// --- СТРУКТУРИ ДЛЯ ВІДПОВІДЕЙ (GET) ---
type ModuleResponse struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Theory       string `json:"theory"`
	InviteCode   string `json:"invite_code"`
	StudentCount int    `json:"student_count"`
	CreatedBy    int    `json:"created_by"`
}

type FlashcardResponse struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type ModuleStudentResponse struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// Структура для питання, згенерованого "на льоту"
type GeneratedTestQuestion struct {
	FlashcardID int      `json:"flashcard_id"`
	Question    string   `json:"question"`
	Options     []string `json:"options"` // Правильна відповідь + 3 дистрактори (перемішані)
	Answer      string   `json:"answer"`  // Правильна відповідь (для перевірки на фронтенді, або можна приховати і перевіряти на бекенді)
}

// --- СТРУКТУРИ ДЛЯ ЗАПИТІВ (POST) ---
type CreateModuleRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Theory      string `json:"theory"`
}

type CreateFlashcardRequest struct {
	ModuleID int    `json:"module_id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Структура для збереження результатів пройденого тесту
type SubmitTestResultRequest struct {
	ModuleID            int   `json:"module_id"`
	Score               int   `json:"score"`
	TotalQuestions      int   `json:"total_questions"`
	MistakeFlashcardIDs []int `json:"mistake_flashcard_ids"`
	CorrectFlashcardIDs []int `json:"correct_flashcard_ids"`
}
