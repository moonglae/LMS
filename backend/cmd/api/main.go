package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/internal/database"
	"backend/internal/handlers/admin"
	"backend/internal/handlers/analytics"
	"backend/internal/handlers/auth"
	"backend/internal/handlers/content"
	"backend/internal/handlers/practice"
	"backend/internal/handlers/user"

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
	if err := godotenv.Load(); err != nil {
		log.Println("Увага: файл .env не знайдено, використовуємо змінні оточення")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	db, err := database.InitDB(connStr)
	if err != nil {
		log.Fatalf("Критична помилка БД: %v", err)
	}
	defer db.Close()

	dictionary, err := practice.LoadDictionary("english.json")
	if err != nil {
		log.Fatalf("Помилка завантаження словника: %v", err)
	}
	adminH := admin.NewAdminHandler(db)

	authH := auth.NewAuthHandler(db)
	contentH := content.NewContentHandler(db)
	analyticsH := analytics.NewAnalyticsHandler(db)
	userH := user.NewUserHandler(db)
	practiceHandler := &practice.Handler{DB: db}

	mux := http.NewServeMux()

	// --- ВІДКРИТІ МАРШРУТИ ---
	mux.HandleFunc("/api/auth/register", methodHandler("POST", authH.Register))
	mux.HandleFunc("/api/auth/login", methodHandler("POST", authH.Login))

	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// --- ЗАХИЩЕНІ МАРШРУТИ ---
	mux.HandleFunc("/api/me", auth.Protect(db, methodHandler("GET", userH.GetMe)))

	mux.HandleFunc("/api/modules", auth.Protect(db, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			contentH.GetModules(w, r)
		case http.MethodPost:
			contentH.CreateModule(w, r)
		default:
			http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/modules/", auth.Protect(db, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			contentH.UpdateModule(w, r)
		default:
			http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/modules/flashcards", auth.Protect(db, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			contentH.GetFlashcards(w, r)
		case http.MethodPost:
			contentH.CreateFlashcard(w, r)
		default:
			http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/modules/students", auth.Protect(db, methodHandler("GET", contentH.GetModuleStudents)))
	mux.HandleFunc("/api/modules/enroll", auth.Protect(db, methodHandler("POST", contentH.EnrollStudent)))

	// --- ТРЕНУВАННЯ ---
	mux.HandleFunc("/api/content/quiz/generate", auth.Protect(db, methodHandler("GET", contentH.GenerateQuiz)))
	mux.HandleFunc("/api/content/quiz/submit", auth.Protect(db, methodHandler("POST", contentH.SubmitTestResult)))

	// --- АНАЛІТИКА ТА ПРОФІЛЬ ---
	mux.HandleFunc("/api/analytics/summary", auth.Protect(db, methodHandler("GET", analyticsH.GetSummary)))
	mux.HandleFunc("/api/analytics/mistakes", auth.Protect(db, methodHandler("GET", analyticsH.GetActiveMistakes)))
	mux.HandleFunc("/api/analytics/mistakes/resolve", auth.Protect(db, methodHandler("POST", analyticsH.ResolveMistake)))
	mux.HandleFunc("/api/analytics/mistakes-quiz", auth.Protect(db, methodHandler("GET", analyticsH.GetMistakesQuiz)))
	mux.HandleFunc("/api/analytics/progress", auth.Protect(db, methodHandler("GET", analyticsH.GetProgressData)))
	mux.HandleFunc("/api/profile/stats", auth.Protect(db, methodHandler("GET", analyticsH.GetProfileStats)))
	mux.HandleFunc("/api/profile/update", auth.Protect(db, methodHandler("POST", authH.UpdateProfile)))
	mux.HandleFunc("/api/profile/password", auth.Protect(db, methodHandler("POST", authH.UpdatePassword)))

	// --- ПРАКТИКА ТА АВТОКОМПЛІТ ---
	mux.HandleFunc("/api/practice/chat", auth.Protect(db, methodHandler("POST", practiceHandler.ChatWithAI)))
	mux.HandleFunc("/api/practice/generate-test", auth.Protect(db, methodHandler("POST", practiceHandler.GenerateAITest)))

	// Роути для словника з підтримкою DELETE, GET, POST
	mux.HandleFunc("/api/practice/vocab", auth.Protect(db, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			practiceHandler.GetMyVocabulary(w, r)
		case http.MethodPost:
			practiceHandler.SaveVocabulary(w, r)
		case http.MethodDelete:
			practiceHandler.DeleteVocabulary(w, r)
		default:
			http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	// Роути для збережених помилок з підтримкою DELETE, GET, POST
	mux.HandleFunc("/api/practice/mistakes", auth.Protect(db, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			practiceHandler.GetMyMistakes(w, r)
		case http.MethodPost:
			practiceHandler.SaveMistake(w, r)
		case http.MethodDelete:
			practiceHandler.DeleteMistake(w, r)
		default:
			http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/autocomplete", auth.Protect(db, methodHandler("GET", dictionary.AutocompleteHandler)))

	// --- ПАНЕЛЬ АДМІНІСТРАТОРА ---
	mux.HandleFunc("/api/admin/users", auth.Protect(db, auth.AdminOnly(db, methodHandler("GET", adminH.GetUsers))))
	mux.HandleFunc("/api/admin/alerts", auth.Protect(db, auth.AdminOnly(db, methodHandler("GET", adminH.GetSecurityAlerts))))
	mux.HandleFunc("/api/admin/alerts/resolve", auth.Protect(db, auth.AdminOnly(db, methodHandler("POST", adminH.ResolveAlert))))
	mux.HandleFunc("/api/admin/users/ban", auth.Protect(db, auth.AdminOnly(db, methodHandler("POST", adminH.BanUser))))
	mux.HandleFunc("/api/admin/users/restrict", auth.Protect(db, auth.AdminOnly(db, methodHandler("POST", adminH.RestrictFeature))))

	// --- МАРШРУТИ ДЛЯ ЦІЛЕЙ ---
	mux.HandleFunc("/api/profile/goals", auth.Protect(db, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			analyticsH.GetGoals(w, r)
		case http.MethodPost:
			analyticsH.AddGoal(w, r)
		case http.MethodPut:
			analyticsH.ToggleGoal(w, r)
		case http.MethodDelete:
			analyticsH.DeleteGoal(w, r)
		default:
			http.Error(w, "Метод заборонено", http.StatusMethodNotAllowed)
		}
	}))

	// 1. Управління самими папками
	mux.HandleFunc("POST /api/folders", auth.Protect(db, contentH.CreateFolder))
	mux.HandleFunc("GET /api/folders", auth.Protect(db, contentH.GetFolders))

	// 2. Дії з конкретною папкою (використовуємо {id} замість сліпого "/")
	mux.HandleFunc("DELETE /api/folders/{id}", auth.Protect(db, contentH.DeleteFolder))
	mux.HandleFunc("GET /api/folders/{id}", auth.Protect(db, contentH.GetFolderModules))

	// 3. Управління модулями всередині папок
	mux.HandleFunc("POST /api/folders/{folder_id}/modules", auth.Protect(db, contentH.AddModuleToFolder))
	mux.HandleFunc("DELETE /api/folders/{folder_id}/modules/{module_id}", auth.Protect(db, contentH.RemoveModuleFromFolder))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Сервер успішно запущено на порту %s...", port)
	if err := http.ListenAndServe(":"+port, enableCORS(mux)); err != nil {
		log.Fatalf("Помилка роботи сервера: %v", err)
	}
}
