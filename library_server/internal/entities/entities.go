package entities

import "time"

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	ISBN   string `json:"isbn"`
	Year   int    `json:"year"`
	Status string `json:"status"`
}

type User struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	RegistrationDate time.Time `json:"registration_date"`
}

type Issue struct {
	ID         string     `json:"id"`
	BookID     string     `json:"book_id"`
	UserID     string     `json:"user_id"`
	IssueDate  time.Time  `json:"issue_date"`
	DueDate    time.Time  `json:"due_date"`
	ReturnDate *time.Time `json:"return_date,omitempty"`
}

type IssueRequest struct {
	BookID string `json:"book_id"`
	UserID string `json:"user_id"`
}
