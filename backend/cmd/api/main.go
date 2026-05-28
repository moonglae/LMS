package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/internal/database"
	"backend/internal/handlers/analytics"
	"backend/internal/handlers/auth"
	"backend/internal/handlers/content"
	"backend/internal/handlers/user" // Підключено новий пакет для роботи з профілем
	"github.com/joho/godotenv"
)

// Helper для перевірки методів
func methodHandler(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, fmt.Sprintf(`{"error": "Метод %s заборонено"}`, r.Method), http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

// CORS Middleware
func enableCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Динамічно дозволяємо будь-який Origin (зручно для Vercel та Localhost)
        origin := r.Header.Get("Origin")
        if origin != "" {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        } else {
            w.Header().Set("Access-Control-Allow-Origin", "*")
        }
        
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func main() {
	// Завантаження .env
	if err := godotenv.Load(); err != nil {
		log.Println("Увага: файл .env не знайдено, використовуємо змінні оточення")
	}

	// Ініціалізація БД
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	db, err := database.InitDB(connStr)
	if err != nil {
		log.Fatalf("Критична помилка БД: %v", err)
	}
	defer db.Close()

	// Ініціалізація хендлерів
	authH := auth.NewAuthHandler(db)
	contentH := content.NewContentHandler(db)
	analyticsH := analytics.NewAnalyticsHandler(db)
	userH := user.NewUserHandler(db) // Ініціалізація хендлера користувача

	mux := http.NewServeMux()

	// --- ВІДКРИТІ МАРШРУТИ ---
	mux.HandleFunc("/api/auth/register", methodHandler("POST", authH.Register))
	mux.HandleFunc("/api/auth/login", methodHandler("POST", authH.Login))

	// --- ЗАХИЩЕНІ МАРШРУТИ ---

	// User info - тепер використовує userH.GetMe
	mux.HandleFunc("/api/me", auth.Protect(methodHandler("GET", userH.GetMe)))

	// Модулі (CRUD)
	mux.HandleFunc("/api/modules", auth.Protect(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet: contentH.GetModules(w, r)
		case http.MethodPost: contentH.CreateModule(w, r)
		default: http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))
	// Редагування модуля (PUT)
	mux.HandleFunc("/api/modules/", auth.Protect(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut { contentH.UpdateModule(w, r) } else { http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed) }
	}))

	// Картки
	mux.HandleFunc("/api/modules/flashcards", auth.Protect(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet: contentH.GetFlashcards(w, r)
		case http.MethodPost: contentH.CreateFlashcard(w, r)
		case http.MethodDelete: contentH.DeleteFlashcards(w, r)
		default: http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	// Студенти та доступ
	mux.HandleFunc("/api/modules/students", auth.Protect(methodHandler("GET", contentH.GetModuleStudents)))
	mux.HandleFunc("/api/modules/enroll", auth.Protect(methodHandler("POST", contentH.EnrollStudent)))

	// Тести та питання
	mux.HandleFunc("/api/quizzes", auth.Protect(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" { contentH.GetQuizzes(w, r) } else { contentH.CreateQuiz(w, r) }
	}))
	mux.HandleFunc("/api/quizzes/questions", auth.Protect(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" { contentH.GetQuizQuestions(w, r) } else { contentH.CreateQuestion(w, r) }
	}))
	mux.HandleFunc("/api/generate-quiz", auth.Protect(methodHandler("GET", contentH.GenerateQuiz)))

	// Аналітика
	mux.HandleFunc("/api/analytics/quiz/submit", auth.Protect(methodHandler("POST", analyticsH.SubmitQuizAttempt)))
	mux.HandleFunc("/api/analytics/quiz/mistakes", auth.Protect(methodHandler("GET", analyticsH.GetMistakesTest)))
	mux.HandleFunc("/api/analytics/student-report", auth.Protect(methodHandler("GET", analyticsH.GetStudentReport)))
	mux.HandleFunc("/api/analytics/module-report", auth.Protect(methodHandler("GET", analyticsH.GetModuleReport)))
	mux.HandleFunc("/api/analytics/summary", auth.Protect(methodHandler("GET", analyticsH.GetSummary)))
	mux.HandleFunc("/api/analytics/mistakes", auth.Protect(methodHandler("GET", analyticsH.GetActiveMistakes)))
	mux.HandleFunc("/api/analytics/mistakes/resolve", auth.Protect(methodHandler("POST", analyticsH.ResolveMistake)))
	mux.HandleFunc("/api/analytics/mistakes-quiz", auth.Protect(methodHandler("GET", analyticsH.GetMistakesQuiz)))
	mux.HandleFunc("/api/analytics/student-mistakes", auth.Protect(methodHandler("GET", analyticsH.GetStudentMistakes)))
	mux.HandleFunc("/api/analytics/progress", auth.Protect(methodHandler("GET", analyticsH.GetProgressData)))

	// Запуск
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	log.Printf("Сервер успішно запущено на порту %s...", port)
	if err := http.ListenAndServe(":"+port, enableCORS(mux)); err != nil {
		log.Fatalf("Помилка роботи сервера: %v", err)
	}
}