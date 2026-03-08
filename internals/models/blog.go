package models

type Blog struct {
	ID          uint    `json:"id" db:"id"`
	Title       string  `json:"title" db:"title"`
	Description string  `json:"description" db:"description"`
	Tags        string  `json:"tags" db:"tags"`
	Author      uint    `json:"author" db:"author"`
	Banner      string  `json:"banner" db:"banner"`
	Likes       int     `json:"likes" db:"likes"`
	Published   bool    `json:"published" db:"published"`
	CreatedAt   string  `json:"created_at" db:"created_at"`
	UpdatedAt   *string `json:"updated_at,omitempty" db:"updated_at"`
}

type BlogComments struct {
	ID        uint    `json:"id" db:"id"`
	BlogID    uint    `json:"blog_id" db:"blog_id"`
	UserID    uint    `json:"user_id" db:"user_id"`
	Comment   string  `json:"comment" db:"comment"`
	Likes     int     `json:"likes" db:"likes"`
	CreatedAt string  `json:"created_at" db:"created_at"`
	UpdatedAt *string `json:"updated_at,omitempty" db:"updated_at"`
}
