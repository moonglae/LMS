package content

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/handlers/auth"
)

func (h *ContentHandler) GetModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	// ВИПРАВЛЕНО: Використання параметризованого запиту ($1, $2) замість fmt.Sprintf
	query := `
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
		WHERE m.created_by = $1 OR e.user_id = $2
		GROUP BY m.id, m.title, m.description, m.theory, m.invite_code, m.created_by
		ORDER BY m.id DESC
	`

	rows, err := h.DB.Query(query, userID, userID)
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

	if modules == nil {
		modules = []ModuleResponse{}
	}
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

	var ownerID int
	// ВИПРАВЛЕНО: Параметризований запит
	err := h.DB.QueryRow(`SELECT created_by FROM modules WHERE id = $1`, moduleID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, `{"error": "Доступ заборонено"}`, http.StatusForbidden)
		return
	}

	// ВИПРАВЛЕНО: Параметризований запит
	studentsQuery := `
		SELECT u.id, u.first_name, u.last_name, u.email
		FROM users u
		JOIN enrollments e ON u.id = e.user_id
		WHERE e.module_id = $1
		ORDER BY u.last_name
	`

	rows, err := h.DB.Query(studentsQuery, moduleID)
	if err != nil {
		http.Error(w, `{"error": "Помилка БД"}`, http.StatusInternalServerError)
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

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

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
		http.Error(w, `{"error": "Некоректні дані або запит занадто великий (макс 1MB)"}`, http.StatusBadRequest)
		return
	}

	if len(req.Title) == 0 || len(req.Title) > 150 {
		http.Error(w, `{"error": "Назва модуля має бути від 1 до 150 символів"}`, http.StatusBadRequest)
		return
	}
	if len(req.Description) > 500 {
		http.Error(w, `{"error": "Опис занадто довгий (макс. 500 символів)"}`, http.StatusBadRequest)
		return
	}

	if len(req.Cards) > 50 || len(req.Theory) > 10000 {
		auth.LogSecurityAlert(h.DB, userID, "data_flooding", "Спроба створити модуль з аномально великим об'ємом даних (>50 карток або >10000 симв.)")
		http.Error(w, `{"error": "Перевищено ліміт об'єму даних. Максимум 50 карток та 10000 символів теорії."}`, http.StatusRequestEntityTooLarge)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error": "Не вдалось створити модуль"}`, http.StatusInternalServerError)
		return
	}

	for _, card := range req.Cards {
		if strings.TrimSpace(card.Question) != "" && strings.TrimSpace(card.Answer) != "" {
			if len(card.Question) > 1000 || len(card.Answer) > 1000 {
				auth.LogSecurityAlert(h.DB, userID, "data_flooding", "Спроба зберегти завеликий текст у картці (>1000 симв.)")
				http.Error(w, `{"error": "Питання та відповідь не можуть перевищувати 1000 символів"}`, http.StatusBadRequest)
				return
			}

			_, err = tx.Exec("INSERT INTO flashcards (module_id, created_by, question, answer) VALUES ($1, $2, $3, $4)",
				moduleID, userID, card.Question, card.Answer)
			if err != nil {
				http.Error(w, `{"error": "Помилка створення карток"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	tx.Commit()
	json.NewEncoder(w).Encode(map[string]interface{}{"id": moduleID, "message": "Успішно"})
}

func (h *ContentHandler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	// ВИПРАВЛЕНО: Використовуємо вашу готову безпечну функцію
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	role, ok := r.Context().Value("role").(string)
	if !ok {
		role = "student"
	}

	parts := strings.Split(r.URL.Path, "/")
	moduleID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, `{"error": "Некоректний ID модуля"}`, http.StatusBadRequest)
		return
	}

	var ownerID int
	err = h.DB.QueryRow("SELECT created_by FROM modules WHERE id = $1", moduleID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Модуль не знайдено"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Помилка перевірки прав"}`, http.StatusInternalServerError)
		}
		return
	}

	if ownerID != userID && role != "admin" {
		description := fmt.Sprintf("Спроба редагування чужого модуля (ID: %d)", moduleID)
		auth.LogSecurityAlert(h.DB, userID, "unauthorized_content_access", description)
		http.Error(w, `{"error": "Доступ заборонено"}`, http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

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
		http.Error(w, `{"error": "Помилка даних або запит занадто великий"}`, http.StatusBadRequest)
		return
	}

	if len(req.Title) == 0 || len(req.Title) > 150 {
		http.Error(w, `{"error": "Назва модуля має бути від 1 до 150 символів"}`, http.StatusBadRequest)
		return
	}
	if len(req.Description) > 500 {
		http.Error(w, `{"error": "Опис занадто довгий (макс. 500 символів)"}`, http.StatusBadRequest)
		return
	}

	if len(req.Cards) > 50 || len(req.Theory) > 10000 {
		auth.LogSecurityAlert(h.DB, userID, "data_flooding", "Спроба оновити модуль аномально великим об'ємом даних")
		http.Error(w, `{"error": "Перевищено ліміт об'єму даних. Максимум 50 карток та 10000 символів теорії."}`, http.StatusRequestEntityTooLarge)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error": "Помилка транзакції"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE modules SET title = $1, description = $2, theory = $3 WHERE id = $4",
		req.Title, req.Description, req.Theory, moduleID)
	if err != nil {
		http.Error(w, `{"error": "Помилка оновлення модуля"}`, http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM flashcards WHERE module_id = $1", moduleID)
	if err != nil {
		http.Error(w, `{"error": "Помилка оновлення карток"}`, http.StatusInternalServerError)
		return
	}

	for _, card := range req.Cards {
		if strings.TrimSpace(card.Question) != "" && strings.TrimSpace(card.Answer) != "" {
			if len(card.Question) > 1000 || len(card.Answer) > 1000 {
				auth.LogSecurityAlert(h.DB, userID, "data_flooding", "Спроба зберегти завеликий текст у картці (>1000 симв.)")
				http.Error(w, `{"error": "Питання та відповідь не можуть перевищувати 1000 символів"}`, http.StatusBadRequest)
				return
			}

			_, err = tx.Exec("INSERT INTO flashcards (module_id, created_by, question, answer) VALUES ($1, $2, $3, $4)",
				moduleID, userID, card.Question, card.Answer)
			if err != nil {
				http.Error(w, `{"error": "Помилка збереження картки"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error": "Помилка фіксації даних"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Модуль успішно оновлено"}`))
}
