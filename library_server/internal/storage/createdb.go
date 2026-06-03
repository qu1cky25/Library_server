package storage

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(db_path string, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", db_path)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	logger.Info("Успешное подключение к SQLite", slog.String("db_path", db_path))

	if err := CreateTables(db); err != nil {
		return nil, fmt.Errorf("Ошибка создания таблиц: %w", err)
	}

	return db, nil
}

func CreateTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS books (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			author TEXT NOT NULL,
			isbn TEXT UNIQUE NOT NULL,
			year INTEGER,
			status TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			registration_date DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS issues (
			id TEXT PRIMARY KEY,
			book_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			issue_date DATETIME NOT NULL,
			due_date DATETIME NOT NULL,
			return_date DATETIME,
			FOREIGN KEY(book_id) REFERENCES books(id),
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("ошибка выполнения запроса таблицы: %w", err)
		}
	}
	return nil
}