package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	IsBanned  bool   `json:"is_banned"`
	BanReason string `json:"ban_reason"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		http.Error(w, `{"error": "Будь ласка, заповніть всі обов'язкові поля"}`, http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error": "Помилка шифрування пароля"}`, http.StatusInternalServerError)
		return
	}

	query := `INSERT INTO users (email, password_hash, first_name, last_name) 
			  VALUES ($1, $2, $3, $4) RETURNING id`

	var newUserID int
	err = h.DB.QueryRow(query, req.Email, string(hashedPassword), req.FirstName, req.LastName).Scan(&newUserID)
	if err != nil {
		http.Error(w, `{"error": "Користувач з таким email вже існує або сталася помилка БД"}`, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Реєстрація успішна",
		"user_id": newUserID,
	})
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат"}`, http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec("UPDATE users SET first_name = $1, last_name = $2 WHERE id = $3", req.FirstName, req.LastName, userID)
	if err != nil {
		http.Error(w, `{"error": "Помилка оновлення профілю"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Дані успішно оновлено"}`))
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Неавторизований доступ"}`, http.StatusUnauthorized)
		return
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат"}`, http.StatusBadRequest)
		return
	}

	var currentHash string
	err := h.DB.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword))
	if err != nil {
		LogSecurityAlert(h.DB, userID, "failed_password_change", "Невдала спроба зміни пароля (невірний старий пароль)")
		http.Error(w, `{"error": "Невірний поточний пароль"}`, http.StatusForbidden)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error": "Помилка шифрування"}`, http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", newHash, userID)
	if err != nil {
		http.Error(w, `{"error": "Помилка збереження пароля"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Пароль успішно змінено"}`))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	var storedHash string
	var user UserResponse

	// Витягуємо також is_banned та ban_reason
	query := `SELECT id, password_hash, first_name, last_name, email, role, is_banned, COALESCE(ban_reason, '') 
	          FROM users WHERE email = $1`
	err := h.DB.QueryRow(query, req.Email).Scan(
		&user.ID, &storedHash, &user.FirstName, &user.LastName, &user.Email, &user.Role, &user.IsBanned, &user.BanReason,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Невірний email або пароль"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		LogSecurityAlert(h.DB, user.ID, "failed_login", "Невдала спроба входу (невірний пароль)")
		http.Error(w, `{"error": "Невірний email або пароль"}`, http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})

	secretKey := os.Getenv("JWT_SECRET")
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		http.Error(w, `{"error": "Помилка генерації токена"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": tokenString,
		"user":  user,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error": "Не вдалося визначити користувача"}`, http.StatusUnauthorized)
		return
	}

	var user UserResponse
	// Додано перевірку is_banned та ban_reason
	query := `SELECT id, email, first_name, last_name, role, is_banned, COALESCE(ban_reason, '') 
	          FROM users WHERE id = $1`
	err := h.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role, &user.IsBanned, &user.BanReason,
	)
	if err != nil {
		http.Error(w, `{"error": "Помилка отримання даних користувача"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}
