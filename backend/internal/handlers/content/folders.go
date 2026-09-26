package content

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/handlers/auth"
)

type FolderResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ModuleCount int    `json:"module_count"`
	CreatedAt   string `json:"created_at"`
}

// CreateFolder створює нову папку користувача
func (h *ContentHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576) // Обмеження 1MB

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректні дані"}`, http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 || len(req.Name) > 100 {
		http.Error(w, `{"error": "Назва папки має бути від 1 до 100 символів"}`, http.StatusBadRequest)
		return
	}

	var folderID int
	err := h.DB.QueryRow(
		`INSERT INTO folders (user_id, name) VALUES ($1, $2) RETURNING id`,
		userID, req.Name,
	).Scan(&folderID)

	if err != nil {
		http.Error(w, `{"error": "Помилка створення папки"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      folderID,
		"message": "Папку успішно створено",
	})
}

// GetFolders повертає список папок користувача з кількістю модулів у них
func (h *ContentHandler) GetFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	query := `
		SELECT f.id, f.name, f.created_at, COUNT(fm.module_id) AS module_count
		FROM folders f
		LEFT JOIN folder_modules fm ON f.id = fm.folder_id
		WHERE f.user_id = $1
		GROUP BY f.id, f.name, f.created_at
		ORDER BY f.created_at DESC
	`

	rows, err := h.DB.Query(query, userID)
	if err != nil {
		http.Error(w, `{"error": "Помилка завантаження папок"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	folders := []FolderResponse{}
	for rows.Next() {
		var f FolderResponse
		if err := rows.Scan(&f.ID, &f.Name, &f.CreatedAt, &f.ModuleCount); err != nil {
			continue
		}
		folders = append(folders, f)
	}

	json.NewEncoder(w).Encode(folders)
}

// DeleteFolder видаляє папку (зв'язки з модулями видаляться автоматично завдяки ON DELETE CASCADE)
func (h *ContentHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	folderID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, `{"error": "Некоректний ID папки"}`, http.StatusBadRequest)
		return
	}

	// Перевірка власника та видалення одним запитом
	res, err := h.DB.Exec(`DELETE FROM folders WHERE id = $1 AND user_id = $2`, folderID, userID)
	if err != nil {
		http.Error(w, `{"error": "Помилка видалення"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error": "Папку не знайдено або доступ заборонено"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "Папку успішно видалено"}`))
}

// AddModuleToFolder додає існуючий модуль до папки
func (h *ContentHandler) AddModuleToFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	// Очікуємо URL виду: /api/folders/{folder_id}/modules
	parts := strings.Split(r.URL.Path, "/")
	folderID, err := strconv.Atoi(parts[len(parts)-2])
	if err != nil {
		http.Error(w, `{"error": "Некоректний ID папки"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		ModuleID int `json:"module_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректні дані"}`, http.StatusBadRequest)
		return
	}

	// Перевіряємо, чи належить папка цьому користувачу
	var ownerID int
	err = h.DB.QueryRow(`SELECT user_id FROM folders WHERE id = $1`, folderID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, `{"error": "Папку не знайдено або доступ заборонено"}`, http.StatusForbidden)
		return
	}

	// Додаємо зв'язок (ON CONFLICT DO NOTHING запобігає дублям, якщо модуль вже в папці)
	_, err = h.DB.Exec(
		`INSERT INTO folder_modules (folder_id, module_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		folderID, req.ModuleID,
	)

	if err != nil {
		http.Error(w, `{"error": "Помилка додавання модуля до папки"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "Модуль додано до папки"}`))
}

// RemoveModuleFromFolder прибирає модуль з папки
func (h *ContentHandler) RemoveModuleFromFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	// Очікуємо URL виду: /api/folders/{folder_id}/modules/{module_id}
	parts := strings.Split(r.URL.Path, "/")
	moduleID, errMod := strconv.Atoi(parts[len(parts)-1])
	folderID, errFol := strconv.Atoi(parts[len(parts)-3])

	if errMod != nil || errFol != nil {
		http.Error(w, `{"error": "Некоректні ID"}`, http.StatusBadRequest)
		return
	}

	// Видаляємо зв'язок лише якщо папка належить користувачу (вкладений SELECT для безпеки)
	query := `
		DELETE FROM folder_modules 
		WHERE folder_id = $1 AND module_id = $2 
		AND EXISTS (SELECT 1 FROM folders WHERE id = $1 AND user_id = $3)
	`

	res, err := h.DB.Exec(query, folderID, moduleID, userID)
	if err != nil {
		http.Error(w, `{"error": "Помилка видалення модуля з папки"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, `{"error": "Модуль не знайдено в папці або доступ заборонено"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "Модуль вилучено з папки"}`))
}

// GetFolderModules повертає масив модулів, що лежать у конкретній папці
func (h *ContentHandler) GetFolderModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований"}`, http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	folderID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, `{"error": "Некоректний ID папки"}`, http.StatusBadRequest)
		return
	}

	// Перевіряємо права на папку
	var ownerID int
	err = h.DB.QueryRow(`SELECT user_id FROM folders WHERE id = $1`, folderID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, `{"error": "Папку не знайдено або доступ заборонено"}`, http.StatusForbidden)
		return
	}

	// Отримуємо модулі (використовуємо ту саму структуру ModuleResponse, що й у modules.go)
	query := `
		SELECT DISTINCT
			m.id, 
			m.title, 
			m.description, 
			m.theory, 
			COALESCE(m.invite_code, '') AS invite_code,
			(SELECT COUNT(DISTINCT e.user_id) FROM enrollments e WHERE e.module_id = m.id) AS student_count,
			m.created_by
		FROM modules m
		JOIN folder_modules fm ON m.id = fm.module_id
		WHERE fm.folder_id = $1
		ORDER BY m.id DESC
	`

	rows, err := h.DB.Query(query, folderID)
	if err != nil {
		http.Error(w, `{"error": "Помилка отримання модулів папки"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	modules := []ModuleResponse{} // ModuleResponse визначено у вашому файлі modules.go
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
