package practice

import (
	"encoding/json"
	"html"
	"net/http"
	"os"
	"strings"
)

// --- 1. СТРУКТУРИ ДАНИХ ---

// DictEntry описує одне слово та його переклад для відправки в React
type DictEntry struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
}

// MemoryDictionary зберігає весь словник в оперативній пам'яті (RAM)
// для миттєвого пошуку без звернення до диска
type MemoryDictionary struct {
	Entries []DictEntry
}

// --- 2. ФУНКЦІЯ ЗАВАНТАЖЕННЯ (Викликається один раз при старті сервера) ---

// LoadDictionary читає JSON файл словника та готує його до роботи
func LoadDictionary(filepath string) (*MemoryDictionary, error) {
	// 1. Читаємо файл з диска
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	// 2. Розпаковуємо JSON у формат ключ-значення
	var rawDict map[string]string
	if err := json.Unmarshal(bytes, &rawDict); err != nil {
		return nil, err
	}

	// 3. Фільтруємо та очищаємо дані
	var entries []DictEntry
	for word, translation := range rawDict {
		// Відкидаємо пусті ключі та технічні метадані словника (наприклад: "##name")
		if word == "" || strings.HasPrefix(word, "##") {
			continue
		}

		// Розкодовуємо HTML-сутності (перетворюємо &#x27; на апостроф ')
		cleanTranslation := html.UnescapeString(translation)

		// Додаємо слово до нашого масиву
		entries = append(entries, DictEntry{
			Word:        word,
			Translation: cleanTranslation,
		})
	}

	return &MemoryDictionary{Entries: entries}, nil
}

// --- 3. ХЕНДЛЕР ДЛЯ REACT (Викликається при кожному натисканні клавіші) ---

// AutocompleteHandler приймає запит (наприклад: ?q=app) і віддає підказки
func (md *MemoryDictionary) AutocompleteHandler(w http.ResponseWriter, r *http.Request) {
	// Вказуємо, що відповідь буде у форматі JSON
	w.Header().Set("Content-Type", "application/json")

	// 1. Отримуємо текст, який ввів користувач
	query := r.URL.Query().Get("q")
	query = strings.ToLower(strings.TrimSpace(query)) // Приводимо до нижнього регістру

	// 2. Захист від зайвих обчислень: якщо введено менше 2 букв, повертаємо пустий масив
	if len(query) < 2 {
		// Важливо віддавати [], а не null для React
		json.NewEncoder(w).Encode([]DictEntry{})
		return
	}

	// 3. Шукаємо співпадіння
	results := []DictEntry{} // Ініціалізуємо пустий масив
	limit := 10              // Максимальна кількість підказок у випадаючому списку

	for _, entry := range md.Entries {
		// Перевіряємо, чи починається слово з того, що ввів користувач
		if strings.HasPrefix(strings.ToLower(entry.Word), query) {
			results = append(results, entry)

			// Як тільки знайшли 10 слів - зупиняємо пошук для економії ресурсів
			if len(results) >= limit {
				break
			}
		}
	}

	// 4. Відправляємо результат назад на фронтенд
	json.NewEncoder(w).Encode(results)
}
