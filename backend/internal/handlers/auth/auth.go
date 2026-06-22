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

// Структури для прийому JSON від фронтенду
type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
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
}

// Register створює нового користувача
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Читаємо JSON з тіла запиту
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	// 2. Хешуємо пароль (ніколи не зберігаємо паролі у відкритому вигляді!)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error": "Помилка шифрування пароля"}`, http.StatusInternalServerError)
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		http.Error(w, `{"error": "Будь ласка, заповніть всі обов'язкові поля"}`, http.StatusBadRequest)
		return
	}

	// 3. Зберігаємо в PostgreSQL
	query := `INSERT INTO users (email, password_hash, first_name, last_name) 
	          VALUES ($1, $2, $3, $4) RETURNING id`
	
	var newUserID int
	err = h.DB.QueryRow(query, req.Email, string(hashedPassword), req.FirstName, req.LastName).Scan(&newUserID)
	if err != nil {
		// Якщо такий email вже є в базі, Postgres видасть помилку унікальності
		http.Error(w, `{"error": "Користувач з таким email вже існує або сталася помилка БД"}`, http.StatusConflict)
		return
	}

	// 4. Повертаємо успішну відповідь
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Реєстрація успішна",
		"user_id": newUserID,
	})
}

// Login перевіряє дані та видає JWT-токен
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Некоректний формат даних"}`, http.StatusBadRequest)
		return
	}

	// 1. Шукаємо користувача за email
	var storedHash string
	var userID int
	var firstName string
	var lastName string
	var email string

	query := `SELECT id, password_hash, first_name, last_name, email FROM users WHERE email = $1`
	err := h.DB.QueryRow(query, req.Email).Scan(&userID, &storedHash, &firstName, &lastName, &email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Невірний email або пароль"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, `{"error": "Помилка сервера"}`, http.StatusInternalServerError)
		return
	}

	// 2. Порівнюємо пароль із хешем із бази
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		http.Error(w, `{"error": "Невірний email або пароль"}`, http.StatusUnauthorized)
		return
	}

	// 3. Генеруємо JWT-токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
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
		"user": UserResponse{
			ID:        userID,
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
		},
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
	query := `SELECT id, email, first_name, last_name FROM users WHERE id = $1`
	err := h.DB.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName)
	if err != nil {
		http.Error(w, `{"error": "Помилка отримання даних користувача"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}