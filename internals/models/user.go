package models

type User struct {
	ID              uint    `json:"id" db:"id"`
	Username        string  `json:"username" db:"username"`
	Email           string  `json:"email" db:"email"`
	EmailVerifiedAt *string `json:"email_verified_at,omitempty" db:"email_verified_at"`
	Password        string  `json:"-" db:"password"`
	ProfileImage    *string `json:"profile_image,omitempty" db:"profile_image"`
	CreatedAt       string  `json:"created_at" db:"created_at"`
	UpdatedAt       *string `json:"updated_at,omitempty" db:"updated_at"`
}

type VerificationCode struct {
	ID        uint    `json:"id" db:"id"`
	UserID    uint    `json:"user_id" db:"user_id"`
	Code      int     `json:"code" db:"code"`
	CreatedAt string  `json:"created_at" db:"created_at"`
	UpdatedAt *string `json:"updated_at,omitempty" db:"updated_at"`
}
