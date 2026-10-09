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
	Topic         string `json:"topic"`
	Theory        string `json:"theory"`
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

	topic := strings.TrimSpace(req.Topic)
	if len(topic) > 150 {
		topic = topic[:150]
	}
	if topic == "" {
		http.Error(w, `{"error": "Тема є обов'язковою для створення тесту"}`, http.StatusBadRequest)
		return
	}

	theory := strings.TrimSpace(req.Theory)
	if len(theory) > 6000 {
		// Якщо користувач намагається пропхати текст понад 4000 символів, відхиляємо і логуємо
		auth.LogSecurityAlert(h.DB, userID, "payload_too_large", "Спроба відправити занадто великий текст теорії для ШІ (>6000 символів)")
		http.Error(w, `{"error": "Текст теорії занадто довгий (макс. 6000 символів)"}`, http.StatusBadRequest)
		return
	}

	var theoryBlock string
	if theory != "" {
		theoryBlock = fmt.Sprintf("Для створення питань ОБОВ'ЯЗКОВО спирайся на цей теоретичний матеріал:\n%s\n", theory)
	} else {
		theoryBlock = "Теоретичний матеріал не надано. Використовуй свої знання загальних правил англійської граматики для цієї теми."
	}

	systemPrompt := fmt.Sprintf(`You are a leading methodologist and expert in creating interactive English learning materials.
Your task is to generate a grammar test on the topic: "%s".

Theory/Context:
%s

TECHNICAL REQUIREMENTS:
1. Number of questions: exactly %d.
2. Question types: "choice" (choose one correct option) and "fill" (fill in the missing word).
   - For "fill" questions: Use "___" to indicate the blank.
   - CRITICAL FOR "fill": Because this is a grammar test, the user must NOT guess vocabulary or synonyms. You MUST ALWAYS provide the base (dictionary) form of the required word in parentheses immediately after the blank. Example: "She ___ (to read) a book now." or "This is the ___ (good) day of my life."

REQUIREMENTS FOR THE "rules" FIELD (CRITICAL):
This field is EXCLUSIVELY for technical text input instructions (UI/UX hints for the user).
IT IS STRICTLY FORBIDDEN to write grammar rules, theory, or explanations of the topic in this array.
Generate 1-2 short formatting rules for answers. To ensure correct validation of user inputs, the rules must be highly technical and specific (e.g., "Вводьте відповідь з маленької літери", "Не ставте крапку в кінці").

LANGUAGE REQUIREMENT:
All generated text inside the JSON for "rules" and "explanation" MUST be in Ukrainian. The English sentences for the tasks themselves must remain in English.

OUTPUT FORMAT (JSON SCHEMA):
Return the result STRICTLY as a valid JSON object. No conversational text, no Markdown wrappers (do NOT use markdown code blocks or similar formatting).
Object structure:
{
  "rules": ["technical rule 1", "technical rule 2"],
  "questions": [
    {
      "type": "choice" or "fill",
      "question": "question text (for 'fill' it MUST include the base word in parentheses, e.g., 'I ___ (to go)')",
      "options": ["option1", "option2", "option3"], // Include this key ONLY if type="choice"
      "correct_answer": "correct answer (if no word is needed, strictly use '-')",
      "explanation": "explanation of the correct answer in Ukrainian"
    }
  ]
}`, topic, theoryBlock, req.QuestionCount)

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
