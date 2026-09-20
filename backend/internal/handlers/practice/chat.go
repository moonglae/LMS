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

	// ВИПРАВЛЕНО: перевіряємо ключ "ai_chat" замість "chat"
	var restrictedFeatures string
	err := h.DB.QueryRow("SELECT COALESCE(restricted_features::text, '{}') FROM users WHERE id = $1", userID).Scan(&restrictedFeatures)
	if err == nil {
		if strings.Contains(restrictedFeatures, `"ai_chat": true`) || strings.Contains(restrictedFeatures, `"ai_chat":true`) {
			auth.LogSecurityAlert(h.DB, userID, "blocked_feature_access", "Спроба використати заблокований AI-чат")
			http.Error(w, `{"error": "Функція AI-чату заблокована для вашого акаунту"}`, http.StatusForbidden)
			return
		}
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат запиту"}`, http.StatusBadRequest)
		return
	}

	systemPrompt := fmt.Sprintf(`
Ти — дружній репетитор з "%s" мови. 
Зараз ми відпрацьовуємо тему: "%s".
Моє повідомлення: "%s"
(Запамятовуй контекст, бо ми можемо вести довгу розмову)
(Я люблю вчити мови на розмовному рівні, тому відповідай мені так, ніби ми спілкуємось у реальному житті)

Проаналізуй моє повідомлення. Використовуй лексику та граматику рівня "%s" для відповіді.
Поверни результат СУВОРО у форматі JSON з такими полями:
1. "reply": твоя відповідь ("%s" мова) для продовження діалогу.(тут не має бути жодних пояснень, тільки відповідь для користувача)
2. "mistakes": масив об'єктів з полями "wrong_text" (моя помилка), "correct_text" (як правильно) та "rule_explanation" (пояснення правила українською мовою). Якщо помилок немає, поверни порожній масив [].(Відповідай більш розгорнуто правило і по простому мені я ще не впевнений у своїх знаннях. Не використовуй складні конструкції, щоб не заплутати мене.)
`, req.Language, req.Topic, req.Message, req.Level, req.Language)

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
