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
	mux.HandleFunc("/api/v1/register", handlers.RegisterHandler)
	mux.HandleFunc("/api/v1/verify", handlers.VerifyEmailHandler)
	mux.HandleFunc("/api/v1/login", handlers.LoginHandler)
	mux.HandleFunc("/api/v1/logout", handlers.LogoutHandler)
	mux.Handle("/api/v1/request-verification", middlewares.AuthMiddleware(http.HandlerFunc(handlers.RequestNewEmailVerification)))
	mux.Handle("/api/v1/reset-password", middlewares.AuthMiddleware(http.HandlerFunc(handlers.ResetPassword)))
	mux.Handle("/api/v1/me", middlewares.AuthMiddleware(http.HandlerFunc(handlers.Profile)))

	handler := middlewares.CORSMiddleware(
		middlewares.LoggingMiddleware(mux),
	)

	logger.Println("Starting the server at http://localhost:8080")
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
