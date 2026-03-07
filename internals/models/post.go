package models

type Post struct {
	ID          uint    `json:"id" db:"id"`
	Title       string  `json:"title" db:"title"`
	Description string  `json:"description" db:"description"`
	Tags        string  `json:"tags" db:"tags"`
	Author      uint    `json:"author" db:"author"`
	Banner      string  `json:"banner" db:"banner"`
	Published   bool    `json:"published" db:"published"`
	CreatedAt   string  `json:"created_at" db:"created_at"`
	UpdatedAt   *string `json:"updated_at,omitempty" db:"updated_at"`
}
