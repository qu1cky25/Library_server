package main

import (
	"log/slog"
	"net/http"
	"os"

	"library_server/internal/methods"
	"library_server/internal/storage"

	"github.com/joho/godotenv"
)

func LoggingMiddleWare(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Получен HTTP-запрос",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
		)
		next.ServeHTTP(w, r)
	})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		logger.Warn("Файл .env не найден", slog.String("error", err.Error()))
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	db_path := os.Getenv("DB_PATH")
	if db_path == "" {
		db_path = "./library.db"
	}

	db, err := storage.InitDB(db_path, logger)
	if err != nil {
		logger.Error("Ошибка инициализации БД", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()
	api := methods.NewAPI(db, logger)
	api.RegisterRoutes(mux)

	handler_with_logging := LoggingMiddleWare(logger, mux)
	logger.Info("Сервер успешно запущен", slog.String("port", port))

	if err := http.ListenAndServe(":"+port, handler_with_logging); err != nil {
		logger.Error("Ошибка сервера", slog.String("error", err.Error()))
		os.Exit(1)
	}
}