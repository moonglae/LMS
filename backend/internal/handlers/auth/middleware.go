package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

var (
	userRequests sync.Map
)

type rateLimitData struct {
	Count     int
	LastReset time.Time
}

const (
	UserIDKey contextKey = "userID"
)

// Утиліта для фонового запису алертів у базу даних
func LogSecurityAlert(db *sql.DB, userID int, activityType string, description string) {
	if db == nil {
		return
	}
	// Запускаємо в горутині, щоб не гальмувати HTTP-відповідь користувачу
	go func() {
		query := `INSERT INTO security_alerts (user_id, activity_type, description) VALUES ($1, $2, $3)`
		_, err := db.Exec(query, userID, activityType, description)
		if err != nil {
			log.Printf("Помилка запису алерта: %v", err)
		}
	}()
}

func AntiSpamMiddleware(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok || userID == 0 {
			next(w, r)
			return
		}

		now := time.Now()
		val, _ := userRequests.LoadOrStore(userID, &rateLimitData{Count: 0, LastReset: now})
		data := val.(*rateLimitData)

		// Скидаємо лічильник кожну хвилину
		if now.Sub(data.LastReset) > time.Minute {
			data.Count = 0
			data.LastReset = now
		}

		data.Count++

		if data.Count > 50 { // Якщо більше 50 запитів за хвилину
			if data.Count == 51 { // Записуємо алерт ТІЛЬКИ один раз за хвилину
				LogSecurityAlert(db, userID, "rate_limit_exceeded", "Більше 50 запитів за хвилину. Можливий DDoS або бот.")
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Забагато запитів. Зачекайте хвилину."}`))
			return
		}

		next(w, r)
	}
}

func getIntFromClaim(value interface{}) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case string:
		var result int
		_, err := fmt.Sscanf(v, "%d", &result)
		return result, err == nil
	default:
		return 0, false
	}
}

// 🔴 ОНОВЛЕНО: Тепер приймає *sql.DB для перевірки на бан
func Protect(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Protect Middleware: Запит до %s", r.URL.Path)

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Відсутній токен авторизації"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Некоректний формат токена"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		secretKey := os.Getenv("JWT_SECRET")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"error": "Недійсний токен"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error": "Помилка читання токена"}`, http.StatusUnauthorized)
			return
		}

		userID, ok := getIntFromClaim(claims["sub"])
		if !ok {
			http.Error(w, `{"error": "Помилка читання ID користувача"}`, http.StatusUnauthorized)
			return
		}

		// 🔴 НОВЕ: ЖОРСТКА ПЕРЕВІРКА НА БАН ПРИ КОЖНОМУ ЗАПИТІ
		if db != nil {
			var isBanned bool
			err := db.QueryRow("SELECT is_banned FROM users WHERE id = $1", userID).Scan(&isBanned)
			if err == nil && isBanned {
				// Якщо забанений — повертаємо 403, щоб apiFetch миттєво показав екран блокування
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "banned"}`))
				return
			}
		}

		// 1. Створюємо контекст з ID користувача
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// 2. ВИТЯГУЄМО РОЛЬ ІЗ ТОКЕНА і теж кладемо в контекст
		if roleClaim, ok := claims["role"].(string); ok {
			ctx = context.WithValue(ctx, "role", roleClaim)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AdminOnly тепер приймає *sql.DB для можливості логування
func AdminOnly(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("role").(string)

		if !ok || role != "admin" {
			// АВТОМАТИЧНИЙ АЛЕРТ: Звичайний користувач намагається зайти в адмінку
			userID, _ := GetUserID(r.Context())
			if userID != 0 {
				LogSecurityAlert(db, userID, "unauthorized_admin_access", "Спроба доступу до API адміністратора звичайним користувачем")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Доступ заборонено. Потрібні права адміністратора"}`))
			return
		}

		next(w, r)
	}
}

func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}
