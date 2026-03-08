package main

import (
	"log"

	"github.com/waltertaya/blogger/internals/db"
	"github.com/waltertaya/blogger/internals/initialisers"
)

var schemas = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(150) NOT NULL UNIQUE,
		email_verified_at TIMESTAMP NULL,
		password VARCHAR(256) NOT NULL,
		profile_image VARCHAR(300) NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL
	)`,
	`CREATE TABLE IF NOT EXISTS blogs (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		title TEXT,
		description TEXT,
		tags VARCHAR(255),
		author BIGINT UNSIGNED,
		banner VARCHAR(150),
		likes INT DEFAULT 0,
		published BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL,
		FOREIGN KEY (author) REFERENCES users(id)
	)`,
	`CREATE TABLE IF NOT EXISTS verification_codes (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT UNSIGNED,
		code INT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL,
	)`,
	`CREATE TABLE IF NOT EXISTS blog_comments (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		blog_id BIGINT UNSIGNED,
		user_id BIGINT UNSIGNED,
		comment TEXT,
		likes INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL,
		FOREIGN KEY (blog_id) REFERENCES blogs(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	)`,
}

func main() {
	initialisers.LoadEnv()
	db.Connect()

	for _, schema := range schemas {
		db.DB.MustExec(schema)
	}

	log.Println("Tables migrated successfully")
}
