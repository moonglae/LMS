package content

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/handlers/auth"
)

// GetModules повертає список курсів для користувача
func (h *ContentHandler) GetModules(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
        return
    }

    // Повертаємо модулі, які користувач створив або в яких зареєстрований
    rows, err := h.DB.Query(`
        SELECT DISTINCT
            m.id, 
            m.title, 
            m.description, 
            m.theory, 
            COALESCE(m.invite_code, '') AS invite_code,
            COUNT(DISTINCT e.user_id) AS student_count,
            m.created_by
        FROM modules m
        LEFT JOIN enrollments e ON m.id = e.module_id
        WHERE m.created_by = $1 OR e.user_id = $1
        GROUP BY m.id, m.title, m.description, m.theory, m.invite_code, m.created_by
        ORDER BY m.id DESC
    `, userID)

    if err != nil {
        log.Printf("GetModules Error: %v", err)
        http.Error(w, `{"error": "Помилка отримання курсів"}`, http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    modules := []ModuleResponse{}
    for rows.Next() {
        var m ModuleResponse
        if err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.Theory, &m.InviteCode, &m.StudentCount, &m.CreatedBy); err != nil {
            continue
        }
        modules = append(modules, m)
    }

    if modules == nil { modules = []ModuleResponse{} }
    json.NewEncoder(w).Encode(modules)
}

func (h *ContentHandler) GetModuleStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	moduleIDStr := r.URL.Query().Get("module_id")
	moduleID, _ := strconv.Atoi(moduleIDStr)

	// Перевірка власника
	var ownerID int
	err := h.DB.QueryRow(`SELECT created_by FROM modules WHERE id = $1`, moduleID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, `{"error": "Доступ заборонено"}`, http.StatusForbidden)
		return
	}

	rows, err := h.DB.Query(`
		SELECT u.id, u.first_name, u.last_name, u.email
		FROM users u
		JOIN enrollments e ON u.id = e.user_id
		WHERE e.module_id = $1
		ORDER BY u.last_name
	`, moduleID)
	if err != nil {
		http.Error(w, "Помилка БД", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	students := []ModuleStudentResponse{}
	for rows.Next() {
		var s ModuleStudentResponse
		rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email)
		students = append(students, s)
	}
	json.NewEncoder(w).Encode(students)
}

func (h *ContentHandler) CreateModule(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
        return
    }

    var req struct {
        Title       string `json:"title"`
        Description string `json:"description"`
        Theory      string `json:"theory"`
        Cards       []struct {
            Question string `json:"question"`
            Answer   string `json:"answer"`
        } `json:"cards"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error": "Некоректні дані"}`, http.StatusBadRequest)
        return
    }

    // Починаємо транзакцію для безпечного збереження
    tx, err := h.DB.Begin()
    if err != nil {
        http.Error(w, "Помилка сервера", http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    inviteCode := fmt.Sprintf("CRS-%d", time.Now().Unix()%1000000)
    var moduleID int
    err = tx.QueryRow(
        `INSERT INTO modules (title, description, theory, created_by, invite_code) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
        req.Title, req.Description, req.Theory, userID, inviteCode,
    ).Scan(&moduleID)

    if err != nil {
        http.Error(w, "Не вдалось створити модуль", http.StatusInternalServerError)
        return
    }

    // Зберігаємо картки
    for _, card := range req.Cards {
        if strings.TrimSpace(card.Question) != "" && strings.TrimSpace(card.Answer) != "" {
            _, err = tx.Exec("INSERT INTO flashcards (module_id, created_by, question, answer) VALUES ($1, $2, $3, $4)",
                moduleID, userID, card.Question, card.Answer)
            if err != nil {
                http.Error(w, "Помилка створення карток", http.StatusInternalServerError)
                return
            }
        }
    }

    tx.Commit()
    json.NewEncoder(w).Encode(map[string]interface{}{"id": moduleID, "message": "Успішно"})
}
func (h *ContentHandler) UpdateModule(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(r.URL.Path, "/")
    id, _ := strconv.Atoi(parts[len(parts)-1])

    var req struct {
        Title       string `json:"title"`
        Description string `json:"description"`
        Theory      string `json:"theory"`
        Cards       []struct {
            Question string `json:"question"`
            Answer   string `json:"answer"`
        } `json:"cards"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error": "Помилка даних"}`, http.StatusBadRequest)
        return
    }

    tx, err := h.DB.Begin() // Додано обробку помилки Begin
    if err != nil {
        http.Error(w, "Помилка транзакції", http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    _, err = tx.Exec("UPDATE modules SET title = $1, description = $2, theory = $3 WHERE id = $4", 
        req.Title, req.Description, req.Theory, id)
    if err != nil {
        http.Error(w, "Помилка оновлення модуля", http.StatusInternalServerError)
        return
    }

    tx.Exec("DELETE FROM flashcards WHERE module_id = $1", id)
    for _, card := range req.Cards {
        if card.Question != "" || card.Answer != "" {
            tx.Exec("INSERT INTO flashcards (module_id, question, answer) VALUES ($1, $2, $3)", id, card.Question, card.Answer)
        }
    }

    if err := tx.Commit(); err != nil {
        http.Error(w, "Помилка фіксації", http.StatusInternalServerError)
        return
    }

    // --- ПРАВИЛЬНИЙ ПОРЯДОК ---
    w.Header().Set("Content-Type", "application/json") // 1. Спочатку заголовок
    w.WriteHeader(http.StatusOK)                       // 2. Потім статус
    json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Оновлено"}) // 3. Потім JSON
}