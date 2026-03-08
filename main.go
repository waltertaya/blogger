package main

import (
	"log"
	"net/http"
	"os"

	"github.com/waltertaya/blogging-app/internals/db"
	"github.com/waltertaya/blogging-app/internals/handlers"
	"github.com/waltertaya/blogging-app/internals/initialisers"
	"github.com/waltertaya/blogging-app/internals/middlewares"
)

func init() {
	initialisers.LoadEnv()
	db.Connect()
}

func main() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", handlers.HealthHandler)

	handler := middlewares.CORSMiddleware(
		middlewares.LoggingMiddleware(mux),
	)

	logger.Println("Starting the server at http://localhost:8080")
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
