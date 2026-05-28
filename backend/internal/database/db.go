package database

import (
	"database/sql"
	"fmt"
	"log"

	// Анонімний імпорт драйвера PostgreSQL
	_ "github.com/lib/pq"
)

// InitDB ініціалізує пул з'єднань з базою даних та перевіряє доступність сервера
func InitDB(connectionString string) (*sql.DB, error) {
	// Створюємо конфігурацію підключення
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("помилка ініціалізації драйвера БД: %w", err)
	}

	// Налаштування пулу з'єднань
	db.SetMaxOpenConns(25) // Максимальна кількість одночасних відкритих з'єднань
	db.SetMaxIdleConns(5)  // Максимальна кількість "сплячих" з'єднань

	// Перевіряємо фізичний зв'язок із сервером
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("не вдалося підключитися до бази даних (Ping failed): %w", err)
	}

	log.Println("Успішне підключення до PostgreSQL!")

	return db, nil
}