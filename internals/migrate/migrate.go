package main

import (
	"log"

	"github.com/waltertaya/blogging-app/internals/db"
	"github.com/waltertaya/blogging-app/internals/initialisers"
)

var schemas = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(150) NOT NULL UNIQUE,
		email_verified_at TIMESTAMP NULL,
		password VARCHAR(256) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL
	)`,
	`CREATE TABLE IF NOT EXISTS posts (
		id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		title TEXT,
		description TEXT,
		tags VARCHAR(255),
		author BIGINT UNSIGNED,
		banner VARCHAR(150),
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
}

func main() {
	initialisers.LoadEnv()
	db.Connect()

	for _, schema := range schemas {
		db.DB.MustExec(schema)
	}

	log.Println("Tables migrated successfully")
}
