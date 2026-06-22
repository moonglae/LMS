package auth

import (
	"context"
	"fmt"
	"log" // Додано логування
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
)

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

func Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Protect Middleware: Запит до %s", r.URL.Path)

		// 1. Шукаємо заголовок
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("Protect Error: Заголовок Authorization відсутній!")
			http.Error(w, `{"error": "Відсутній токен авторизації"}`, http.StatusUnauthorized)
			return
		}

		// 2. Перевіряємо формат
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("Protect Error: Некоректний формат токена: %s", authHeader)
			http.Error(w, `{"error": "Некоректний формат токена"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// 3. Перевірка токена
		secretKey := os.Getenv("JWT_SECRET")
		if secretKey == "" {
			log.Println("CRITICAL ERROR: JWT_SECRET не встановлено в .env!")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if err != nil {
			log.Printf("Protect Error: Помилка розбору JWT: %v", err)
			http.Error(w, `{"error": "Недійсний токен"}`, http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			log.Println("Protect Error: Токен не є валідним")
			http.Error(w, `{"error": "Недійсний токен"}`, http.StatusUnauthorized)
			return
		}

		// 4. Дістаємо Claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Println("Protect Error: Не вдалося привести claims до MapClaims")
			http.Error(w, `{"error": "Помилка читання токена"}`, http.StatusUnauthorized)
			return
		}

		userID, ok := getIntFromClaim(claims["sub"])
		if !ok {
			log.Printf("Protect Error: Не вдалося отримати userID (sub) з токена. Claims: %v", claims)
			http.Error(w, `{"error": "Помилка читання ID користувача"}`, http.StatusUnauthorized)
			return
		}

		log.Printf("Protect Success: Користувач %d авторизований", userID)

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

