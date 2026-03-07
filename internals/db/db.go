package db

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

func Connect() {
	logger := log.New(os.Stdout, "DB: ", log.LstdFlags)
	var err error
	var (
		DB_USER = os.Getenv("DB_USER")
		DB_PASS = os.Getenv("DB_PASS")
		DB_HOST = os.Getenv("DB_HOST")
		DB_PORT = os.Getenv("DB_PORT")
		DB_NAME = os.Getenv("DB_NAME")
	)
	DB, err = sqlx.Connect("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", DB_USER, DB_PASS, DB_HOST, DB_PORT, DB_NAME))
	if err == nil {
		logger.Println("MySQL database connected")
	} else {
		logger.Fatal("Connection fail: ", err)
	}
}
