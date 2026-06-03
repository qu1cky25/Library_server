package methods

import (
	"database/sql"
	"encoding/json"
	"errors"
	"library_server/internal/entities"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type API struct {
	DB     *sql.DB
	Logger *slog.Logger
}

func NewAPI(db *sql.DB, logger *slog.Logger) *API {
	return &API{DB: db, Logger: logger}
}

func (api *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /books", api.CreateBook)
	mux.HandleFunc("GET /books", api.GetBooks)
	mux.HandleFunc("GET /books/{id}", api.GetBookByID)
	mux.HandleFunc("PUT /books/{id}", api.UpdateBook)

	mux.HandleFunc("POST /users", api.CreateUser)
	mux.HandleFunc("GET /users/{id}/books", api.GetUserBooks)

	mux.HandleFunc("POST /issues", api.IssueBook)
	mux.HandleFunc("POST /returns", api.ReturnBook)
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *API) WriteError(w http.ResponseWriter, status int, msg string, err error) {
	api.Logger.Error(msg, slog.String("error", err.Error()))
	WriteJSON(w, status, map[string]string{"error": msg})
}

func (api *API) CreateBook(w http.ResponseWriter, r *http.Request) {
	var b entities.Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		api.WriteError(w, http.StatusBadRequest, "Неверный формат JSON", err)
		return
	}

	b.ID = uuid.New().String()
	b.Status = "Available"

	_, err := api.DB.Exec("INSERT INTO books (id, title, author, isbn, year, status) VALUES (?, ?, ?, ?, ?, ?)",
		b.ID, b.Title, b.Author, b.ISBN, b.Year, b.Status)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка сохранения книги в БД", err)
		return
	}

	api.Logger.Info("Книга успешно добавлена", slog.String("id", b.ID), slog.String("title", b.Title))
	WriteJSON(w, http.StatusCreated, b)
}

func (api *API) GetBooks(w http.ResponseWriter, r *http.Request) {
	rows, err := api.DB.Query("SELECT id, title, author, isbn, year, status FROM books")
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка получения списка книг", err)
		return
	}
	defer rows.Close()

	books := []entities.Book{}
	for rows.Next() {
		var b entities.Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Year, &b.Status); err != nil {
			api.WriteError(w, http.StatusInternalServerError, "Ошибка сканирования данных книги", err)
			return
		}
		books = append(books, b)
	}

	WriteJSON(w, http.StatusOK, books)
}

func (api *API) GetBookByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b entities.Book

	err := api.DB.QueryRow("SELECT id, title, author, isbn, year, status FROM books WHERE id = ?", id).
		Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Year, &b.Status)
	if errors.Is(err, sql.ErrNoRows) {
		api.WriteError(w, http.StatusNotFound, "Книга не найдена", err)
		return
	} else if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка запроса к БД", err)
		return
	}

	WriteJSON(w, http.StatusOK, b)
}

func (api *API) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b entities.Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		api.WriteError(w, http.StatusBadRequest, "Неверный формат JSON", err)
		return
	}

	_, err := api.DB.Exec("UPDATE books SET title = ?, author = ?, isbn = ?, year = ? WHERE id = ?",
		b.Title, b.Author, b.ISBN, b.Year, id)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка обновления книги", err)
		return
	}

	api.Logger.Info("Данные книги обновлены", slog.String("id", id))
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) CreateUser(w http.ResponseWriter, r *http.Request) {
	var u entities.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		api.WriteError(w, http.StatusBadRequest, "Неверный формат JSON", err)
		return
	}

	u.ID = uuid.New().String()
	u.RegistrationDate = time.Now()

	_, err := api.DB.Exec("INSERT INTO users (id, name, email, registration_date) VALUES (?, ?, ?, ?)",
		u.ID, u.Name, u.Email, u.RegistrationDate)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка регистрации пользователя", err)
		return
	}

	api.Logger.Info("Пользователь зарегистрирован", slog.String("id", u.ID), slog.String("name", u.Name))
	WriteJSON(w, http.StatusCreated, u)
}

func (api *API) GetUserBooks(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	rows, err := api.DB.Query(`
		SELECT b.id, b.title, b.author, b.isbn, b.year, b.status 
		FROM books b 
		JOIN issues i ON b.id = i.book_id 
		WHERE i.user_id = ? AND i.return_date IS NULL`, userID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка получения книг пользователя", err)
		return
	}
	defer rows.Close()

	books := []entities.Book{}
	for rows.Next() {
		var b entities.Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Year, &b.Status); err != nil {
			api.WriteError(w, http.StatusInternalServerError, "Ошибка чтения данных", err)
			return
		}
		books = append(books, b)
	}

	WriteJSON(w, http.StatusOK, books)
}

func (api *API) IssueBook(w http.ResponseWriter, r *http.Request) {
	var req entities.IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "Неверный формат JSON", err)
		return
	}

	var status string
	err := api.DB.QueryRow("SELECT status FROM books WHERE id = ?", req.BookID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		api.WriteError(w, http.StatusNotFound, "Книга не найдена", err)
		return
	}
	if status != "Available" {
		api.WriteError(w, http.StatusBadRequest, "Книга уже выдана", errors.New("book is not available"))
		return
	}

	tx, err := api.DB.Begin()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка транзакции", err)
		return
	}
	defer tx.Rollback()

	issueID := uuid.New().String()
	now := time.Now()
	due := now.AddDate(0, 0, 14) // на 14 дней

	_, err = tx.Exec("INSERT INTO issues (id, book_id, user_id, issue_date, due_date) VALUES (?, ?, ?, ?, ?)",
		issueID, req.BookID, req.UserID, now, due)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка создания записи выдачи", err)
		return
	}

	_, err = tx.Exec("UPDATE books SET status = 'Issued' WHERE id = ?", req.BookID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка обновления статуса книги", err)
		return
	}

	if err := tx.Commit(); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка коммита транзакции", err)
		return
	}

	api.Logger.Info("Книга успешно выдана читателю", slog.String("issue_id", issueID))
	w.WriteHeader(http.StatusCreated)
}

func (api *API) ReturnBook(w http.ResponseWriter, r *http.Request) {
	var req entities.IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "Неверный формат JSON", err)
		return
	}

	tx, err := api.DB.Begin()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка транзакции", err)
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec("UPDATE issues SET return_date = ? WHERE book_id = ? AND user_id = ? AND return_date IS NULL",
		time.Now(), req.BookID, req.UserID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка обновления записи возврата", err)
		return
	}

	RowsAffected, _ := res.RowsAffected()
	if RowsAffected == 0 {
		api.WriteError(w, http.StatusNotFound, "Активная запись о выдаче этой книги не найдена", errors.New("no active issue"))
		return
	}

	_, err = tx.Exec("UPDATE books SET status = 'Available' WHERE id = ?", req.BookID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка обновления статуса книги", err)
		return
	}

	if err := tx.Commit(); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "Ошибка коммита", err)
		return
	}

	api.Logger.Info("Книга успешно возвращена в библиотеку", slog.String("book_id", req.BookID))
	w.WriteHeader(http.StatusOK)
}