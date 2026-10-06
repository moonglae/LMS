package practice

import (
	"backend/internal/handlers/auth"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type ChatRequest struct {
	Topic    string `json:"topic"`
	Message  string `json:"message"`
	Language string `json:"language"`
	Level    string `json:"level"`
}

type AIResponse struct {
	Reply    string `json:"reply"`
	Mistakes []struct {
		WrongText       string `json:"wrong_text"`
		CorrectText     string `json:"correct_text"`
		RuleExplanation string `json:"rule_explanation"`
	} `json:"mistakes"`
}

type GeminiRequest struct {
	Contents         []GeminiContent `json:"contents"`
	GenerationConfig GeminiConfig    `json:"generationConfig"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiConfig struct {
	ResponseMimeType string `json:"responseMimeType"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (h *Handler) ChatWithAI(w http.ResponseWriter, r *http.Request) {
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
			auth.LogSecurityAlert(h.DB, userID, "blocked_feature_access", "Спроба використати заблокований AI-чат")
			http.Error(w, `{"error": "Функція AI-чату заблокована для вашого акаунту"}`, http.StatusForbidden)
			return
		}
	}

	// 2. АНТИСПАМ ПЕРЕВІРКА ДЛЯ ШІ (перевикористовуємо функцію з ai_generator.go)
	if !checkAILimitAndLog(h.DB, userID) {
		http.Error(w, `{"error": "Занадто багато запитів до ШІ. Будь ласка, зачекайте хвилину."}`, http.StatusTooManyRequests)
		return
	}

	// 3. ЗАХИСТ ВІД JSON-БОМБ: Обмежуємо весь запит до 1 МБ
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат запиту або перевищено ліміт об'єму (макс. 1MB)"}`, http.StatusBadRequest)
		return
	}

	// 4. ЗАХИСТ ВІД ДОВГОГО ТЕКСТУ: Обмежуємо повідомлення в чаті
	req.Message = strings.TrimSpace(req.Message)
	if len(req.Message) > 2000 {
		auth.LogSecurityAlert(h.DB, userID, "payload_too_large", "Спроба відправити занадто довге повідомлення в ШІ-чат (>2000 символів)")
		http.Error(w, `{"error": "Повідомлення занадто довге (максимум 2000 символів)"}`, http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, `{"error": "Повідомлення не може бути порожнім"}`, http.StatusBadRequest)
		return
	}

	systemPrompt := fmt.Sprintf(`You are a friendly tutor of the "%s" language. 
We are currently practicing the topic: "%s".
My message: "%s"

Context rules:
- Remember the context of our chat, as we might have a long conversation.
- I prefer learning languages at a conversational level, so reply to me naturally, as if we were chatting in real life.

Analyze my message. Use vocabulary and grammar at the "%s" proficiency level for your reply.

LANGUAGE REQUIREMENTS (CRITICAL):
1. The conversational response ("reply" field) MUST be in the "%s" language.
2. The explanation of mistakes ("rule_explanation" field) MUST be strictly in Ukrainian (українською мовою).

OUTPUT FORMAT:
Return the result STRICTLY as a valid JSON object. Do not use markdown code blocks or add any conversational text outside the JSON.

Expected JSON structure:
{
  "reply": "Your response to continue the dialogue. There must be NO explanations here, ONLY the conversational reply to the user.",
  "mistakes": [
    {
      "wrong_text": "the exact text of my mistake",
      "correct_text": "how to write it correctly",
      "rule_explanation": "Пояснення правила СУВОРО УКРАЇНСЬКОЮ МОВОЮ. Explain it in detail but in very simple terms, as I am not confident in my knowledge yet. Do not use complex grammatical jargon."
    }
  ]
}
Note: If there are no mistakes in my message, return an empty array [] for "mistakes".`, 
		req.Language, req.Topic, req.Message, req.Level, req.Language)

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

	apiKeys := []string{
		os.Getenv("API_KEY_1"),
		os.Getenv("API_KEY_2"),
		os.Getenv("API_KEY_3"),
	}

	var bodyBytes []byte
	var isSuccess bool

	for i, apiKey := range apiKeys {
		if apiKey == "" {
			continue
		}

		url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=" + apiKey

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			log.Printf("Користувач %d: помилка з'єднання з ШІ (Ключ %d): %v", userID, i+1, err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			log.Printf("Помилка читання відповіді (Ключ %d): %v", i+1, err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			log.Printf("⚠️ Ключ %d зловив ліміт (429). Перемикаємось на наступний...", i+1)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("🔴 Помилка від Google API (Ключ %d, Статус %d):\n%s", i+1, resp.StatusCode, string(respBody))
			continue
		}

		bodyBytes = respBody
		isSuccess = true
		break
	}

	if !isSuccess {
		log.Printf("Користувач %d: Всі API ключі вичерпані або не працюють", userID)
		http.Error(w, `{"error": "Всі сервіси ШІ тимчасово перевантажені. Спробуйте через кілька хвилин."}`, http.StatusTooManyRequests)
		return
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		log.Printf("Помилка парсингу Gemini JSON: %v", err)
		http.Error(w, `{"error": "Помилка обробки відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		http.Error(w, `{"error": "ШІ повернув порожню відповідь"}`, http.StatusInternalServerError)
		return
	}

	aiGeneratedJSON := geminiResp.Candidates[0].Content.Parts[0].Text

	var finalResponse AIResponse
	if err := json.Unmarshal([]byte(aiGeneratedJSON), &finalResponse); err != nil {
		log.Printf("ШІ повернув невалідний формат даних: %s", aiGeneratedJSON)
		http.Error(w, `{"error": "Помилка форматування відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(finalResponse)
}
