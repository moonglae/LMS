package practice

import (
	"backend/internal/handlers/auth"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// --- АНТИСПАМ СИСТЕМА ДЛЯ ШІ ---
var aiRequests sync.Map

type aiRateLimitData struct {
	Count     int
	LastReset time.Time
}

// checkAILimitAndLog перевіряє, чи не перевищив користувач ліміт у 10 запитів/хв
func checkAILimitAndLog(db *sql.DB, userID int) bool {
	now := time.Now()
	val, _ := aiRequests.LoadOrStore(userID, &aiRateLimitData{Count: 0, LastReset: now})
	data := val.(*aiRateLimitData)

	// Скидаємо лічильник щохвилини
	if now.Sub(data.LastReset) > time.Minute {
		data.Count = 0
		data.LastReset = now
	}

	data.Count++

	if data.Count > 5 { // ЛІМІТ: 5 запитів на хвилину
		if data.Count == 6 {
			// Логуємо тільки на 6-й раз, щоб не спамити базу логами
			auth.LogSecurityAlert(db, userID, "ai_spam_attempt", "Аномально висока активність ШІ (>5 запитів/хв). Рекомендується заблокувати ШІ для цього користувача.")
		}
		return false // Ліміт перевищено
	}
	return true // Все ок
}

// ---------------------------------

type GenerateTestRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	QuestionCount int    `json:"question_count"`
}

type AITestResponse struct {
	Rules     []string         `json:"rules"`
	Questions []AITestQuestion `json:"questions"`
}

type AITestQuestion struct {
	Type          string   `json:"type"`
	Question      string   `json:"question"`
	Options       []string `json:"options,omitempty"`
	CorrectAnswer string   `json:"correct_answer"`
	Explanation   string   `json:"explanation"`
}

func (h *Handler) GenerateAITest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	// 1. ПЕРЕВІРКА ОБМЕЖЕНЬ АДМІНІСТРАТОРА
	var restrictedFeatures string
	err := h.DB.QueryRow("SELECT COALESCE(restricted_features::text, '{}') FROM users WHERE id = $1", userID).Scan(&restrictedFeatures)
	if err == nil {
		if strings.Contains(restrictedFeatures, `"ai_chat": true`) || strings.Contains(restrictedFeatures, `"ai_chat":true`) {
			auth.LogSecurityAlert(h.DB, userID, "blocked_feature_access", "Спроба згенерувати тест заблокованим користувачем")
			http.Error(w, `{"error": "Генерація за допомогою AI заблокована для вашого акаунту"}`, http.StatusForbidden)
			return
		}
	}

	// 2. АНТИСПАМ ПЕРЕВІРКА ДЛЯ ШІ
	if !checkAILimitAndLog(h.DB, userID) {
		http.Error(w, `{"error": "Занадто багато запитів до ШІ. Будь ласка, зачекайте хвилину."}`, http.StatusTooManyRequests)
		return
	}

	// 3. ЗАХИСТ ВІД JSON-БОМБ: Обмежуємо розмір тіла запиту до 1 МБ (1048576 байт)
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req GenerateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Невірний формат запиту або перевищено ліміт об'єму (макс. 1MB)"}`, http.StatusBadRequest)
		return
	}

	// Базові перевірки значень
	if req.QuestionCount < 1 {
		req.QuestionCount = 5
	} else if req.QuestionCount > 20 {
		req.QuestionCount = 20
	}

	title := strings.TrimSpace(req.Title)
	if len([]rune(title)) > 150 {
		title = string([]rune(title)[:150])
	}
	if title == "" {
		http.Error(w, `{"error": "Назва теми (Title) є обов'язковою"}`, http.StatusBadRequest)
		return
	}

	description := strings.TrimSpace(req.Description)
	if len([]rune(description)) > 500 {
		description = string([]rune(description)[:500])
	}
	// Якщо опису немає, передаємо порожній рядок, ШІ орієнтуватиметься лише на Title
	if description == "" {
		description = "Generate general questions covering all aspects of the theme."
	}


	systemPrompt := fmt.Sprintf(`Generate an English grammar test.

THEME: "%s"
SPECIFIC DETAILS / FOCUS: "%s"

REQUIREMENTS:
1. Output exactly %d questions. Types: "choice" and "fill".
2. CONTEXT MARKER: If the correct tense or form depends on specific context, add a brief hint in brackets at the end of the sentence. Example: "I ___ (to do) my homework. [Context: action happened yesterday]".
3. "fill" QUESTIONS: Use "___" for blanks. ALWAYS provide the base word in parentheses next to the blank. CRITICAL: If the correct answer requires negative words ('not') or adverbs ('never', 'just', 'already', 'ever'), you MUST include them inside the parentheses. Example: "I ___ (never / to see) that movie." -> correct answer: "have never seen".
4. "rules" FIELD: Provide 1-2 technical UI/UX input instructions ONLY (e.g., "Вводьте з маленької літери"). NO grammar theory.
5. LANGUAGES: "rules" and "explanation" MUST be strictly in Ukrainian. Test sentences in English.
6. FORMAT: Return strictly valid JSON. NO markdown formatting or code blocks.

JSON SCHEMA:
{
  "rules": ["technical instruction"],
  "questions": [
    {
      "type": "choice" | "fill",
      "question": "Sentence text. Blank format: ___ (base word). [Context: optional hint]",
      "options": ["opt1", "opt2", "opt3"], // ONLY if type="choice"
      "correct_answer": "exact answer or '-'",
      "explanation": "Why this answer is correct (in Ukrainian)"
    }
  ]
}`, title, description, req.QuestionCount)

	geminiReqData := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: systemPrompt}}},
		},
		GenerationConfig: GeminiConfig{
			ResponseMimeType: "application/json",
		},
	}

	requestBody, err := json.Marshal(geminiReqData)
	if err != nil {
		http.Error(w, `{"error": "Помилка формування запиту до ШІ"}`, http.StatusInternalServerError)
		return
	}

	apiKeys := []string{os.Getenv("API_KEY_1"), os.Getenv("API_KEY_2"), os.Getenv("API_KEY_3")}
	var bodyBytes []byte
	var isSuccess bool

	for i, apiKey := range apiKeys {
		if apiKey == "" {
			continue
		}
		url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + apiKey
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			log.Printf("Генерація тесту: помилка з'єднання з ШІ (Ключ %d): %v", i+1, err)
			continue
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode == http.StatusTooManyRequests {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Помилка API (Статус %d): %s", resp.StatusCode, string(respBody))
			continue
		}
		bodyBytes = respBody
		isSuccess = true
		break
	}

	if !isSuccess {
		http.Error(w, `{"error": "Всі сервіси ШІ тимчасово перевантажені."}`, http.StatusTooManyRequests)
		return
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 {
		http.Error(w, `{"error": "Помилка обробки відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	aiGeneratedJSON := geminiResp.Candidates[0].Content.Parts[0].Text

	var finalResponse AITestResponse
	if err := json.Unmarshal([]byte(aiGeneratedJSON), &finalResponse); err != nil {
		log.Printf("ШІ повернув невалідний формат даних: %s", aiGeneratedJSON)
		http.Error(w, `{"error": "Помилка форматування відповіді від ШІ"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(finalResponse)
}
