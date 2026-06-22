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

// ChatRequest - те, що присилає користувач із фронтенду
type ChatRequest struct {
	Topic    string `json:"topic"`    // Наприклад: "У ресторані"
	Message  string `json:"message"`  // Наприклад: "I wants to order pizza"
	Language string `json:"language"` // Наприклад: "English"
	Level    string `json:"level"`    // Наприклад: "A2"
}

// AIResponse - те, що ми віддаємо фронтенду (і те, що просимо згенерувати Gemini)
type AIResponse struct {
	Reply    string `json:"reply"`
	Mistakes []struct {
		WrongText       string `json:"wrong_text"`
		CorrectText     string `json:"correct_text"`
		RuleExplanation string `json:"rule_explanation"`
	} `json:"mistakes"`
}

// --- 2. СТРУКТУРИ ДЛЯ GOOGLE GEMINI API ---
// Ці структури описують специфічний формат, який вимагає Google.

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
	ResponseMimeType string `json:"responseMimeType"` // Змушує ШІ віддавати тільки JSON
}

// Структури для читання відповіді від Google
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

	// Крок 1. Дістаємо userID з JWT токена через контекст (оскільки твоя middleware вже це робить)
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
	// Використовуємо fmt.Sprintf для підстановки змінних у рядок.
	// Вказуємо рівень A2, щоб ШІ підбирав правильну лексику і не перевантажував складними конструкціями.
	systemPrompt := fmt.Sprintf(`
Ти — дружній репетитор з "%s" мови. 
Зараз ми відпрацьовуємо тему: "%s".
Моє повідомлення: "%s"

Проаналізуй моє повідомлення. Використовуй лексику та граматику рівня "%s" для відповіді.
Поверни результат СУВОРО у форматі JSON з такими полями:
1. "reply": твоя відповідь ("%s" мова) для продовження діалогу.(тут не має бути жодних пояснень, тільки відповідь для користувача)
2. "mistakes": масив об'єктів з полями "wrong_text" (моя помилка), "correct_text" (як правильно) та "rule_explanation" (пояснення правила українською мовою). Якщо помилок немає, поверни порожній масив [].(Відповідай більш розгорнуто правило і по простому мені я ще не впевнений у своїх знаннях. Не використовуй складні конструкції, щоб не заплутати мене.)
`, req.Language, req.Topic, req.Message, req.Level, req.Language)

	// Крок 4. Пакуємо наш промпт у структуру, яку розуміє Google Gemini
	geminiReqData := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: systemPrompt}}},
		},
		GenerationConfig: GeminiConfig{
			ResponseMimeType: "application/json", // Магія: гарантує, що відповідь буде чистим JSON
		},
	}

	// Перетворюємо структуру Go у байти JSON для відправки
	requestBody, err := json.Marshal(geminiReqData)
	if err != nil {
		http.Error(w, `{"error": "Помилка формування запиту до ШІ"}`, http.StatusInternalServerError)
		return
	}

	// Крок 5. Відправляємо HTTP-запит до Google
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + apiKey

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Користувач %d: помилка з'єднання з ШІ: %v", userID, err)
		http.Error(w, `{"error": "Сервіс ШІ тимчасово недоступний"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Крок 6. Читаємо відповідь від Google
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, `{"error": "Помилка читання відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		// Якщо Google відповів помилкою (наприклад 400), ми виводимо це червоним у консоль
		log.Printf("🔴 Помилка від Google API (Статус %d):\n%s\n", resp.StatusCode, string(bodyBytes))
		http.Error(w, `{"error": "Помилка від Google API (дивись консоль бекенду)"}`, http.StatusInternalServerError)
		return
	}

	// Розпаковуємо відповідь Google у нашу структуру GeminiResponse
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

	// Крок 7. Дістаємо корисний JSON, який згенерував ШІ (він лежить усередині поля Text)
	aiGeneratedJSON := geminiResp.Candidates[0].Content.Parts[0].Text

	// Перевіряємо, чи згенерував ШІ правильну структуру AIResponse
	var finalResponse AIResponse
	if err := json.Unmarshal([]byte(aiGeneratedJSON), &finalResponse); err != nil {
		log.Printf("ШІ повернув невалідний формат даних: %s", aiGeneratedJSON)
		http.Error(w, `{"error": "Помилка форматування відповіді ШІ"}`, http.StatusInternalServerError)
		return
	}

	// Крок 8. Віддаємо ідеально відформатований результат у React!
	// Тепер на клієнті можна відобразити finalResponse.Reply у чаті
	// та перевірити if (finalResponse.Mistakes.length > 0)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(finalResponse)
}
