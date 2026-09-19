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
)

// --- 1. СТРУКТУРИ ДЛЯ НАШОГО ФРОНТЕНДУ (REACT) ---
// (Твої структури без змін)
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

// --- 2. СТРУКТУРИ ДЛЯ GOOGLE GEMINI API ---
// (Твої структури без змін)
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

// --- 3. ГОЛОВНА ФУНКЦІЯ ОБРОБКИ ---

func (h *Handler) ChatWithAI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Крок 1. Дістаємо userID
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	// Крок 2. Читаємо повідомлення від фронтенду
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат запиту"}`, http.StatusBadRequest)
		return
	}

	// Крок 3. Формуємо промпт (інструкцію) для ШІ
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

	// Крок 4. Пакуємо наш промпт
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

	// Крок 5. Механізм перемикання API-ключів (Fallback)
	apiKeys := []string{
		os.Getenv("API_KEY_1"),
		os.Getenv("API_KEY_2"),
		os.Getenv("API_KEY_3"),
	}

	var bodyBytes []byte
	var isSuccess bool

	for i, apiKey := range apiKeys {
		// Пропускаємо порожні ключі (якщо ти забув додати їх у .env)
		if apiKey == "" {
			continue
		}

		url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=" + apiKey

		// Важливо: для кожної спроби створюємо новий буфер, бо після попереднього читання він порожній
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			log.Printf("Користувач %d: помилка з'єднання з ШІ (Ключ %d): %v", userID, i+1, err)
			continue // Йдемо до наступного ключа, якщо відпав інтернет
		}

		// Читаємо відповідь одразу, щоб можна було закрити Body всередині циклу
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close() // ЗАКРИВАЄМО ВРУЧНУ, без defer!

		if err != nil {
			log.Printf("Помилка читання відповіді (Ключ %d): %v", i+1, err)
			continue
		}

		// Якщо зловили ліміт запитів (429 Too Many Requests)
		if resp.StatusCode == http.StatusTooManyRequests {
			log.Printf("⚠️ Ключ %d зловив ліміт (429). Перемикаємось на наступний...", i+1)
			continue
		}

		// Інші помилки від Google (400, 500)
		if resp.StatusCode != http.StatusOK {
			log.Printf("🔴 Помилка від Google API (Ключ %d, Статус %d):\n%s", i+1, resp.StatusCode, string(respBody))
			continue
		}

		// Якщо ми тут, запит успішний!
		bodyBytes = respBody
		isSuccess = true
		break // Виходимо з циклу, інші ключі не чіпаємо
	}

	// Якщо всі ключі вичерпані або не спрацювали
	if !isSuccess {
		log.Printf("Користувач %d: Всі API ключі вичерпані або не працюють", userID)
		http.Error(w, `{"error": "Всі сервіси ШІ тимчасово перевантажені. Спробуйте через кілька хвилин."}`, http.StatusTooManyRequests)
		return
	}

	// Крок 6. Розпаковуємо відповідь Google у нашу структуру GeminiResponse
	var geminiResp GeminiResponse
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		log.Printf("Помилка парсингу Gemini JSON: %v", err)
		http.Error(w, `{"error": "Помилка обробки відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	// Перевіряємо, чи Google взагалі щось повернув
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		http.Error(w, `{"error": "ШІ повернув порожню відповідь"}`, http.StatusInternalServerError)
		return
	}

	// Крок 7. Дістаємо корисний JSON, який згенерував ШІ
	aiGeneratedJSON := geminiResp.Candidates[0].Content.Parts[0].Text

	// Перевіряємо, чи згенерував ШІ правильну структуру AIResponse
	var finalResponse AIResponse
	if err := json.Unmarshal([]byte(aiGeneratedJSON), &finalResponse); err != nil {
		log.Printf("ШІ повернув невалідний формат даних: %s", aiGeneratedJSON)
		http.Error(w, `{"error": "Помилка форматування відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	// Крок 8. Віддаємо ідеально відформатований результат у React!
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(finalResponse)
}
