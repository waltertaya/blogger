package main

import (
	"log"
	"net/http"
	"os"

	"github.com/waltertaya/blogger/internals/db"
	"github.com/waltertaya/blogger/internals/handlers"
	"github.com/waltertaya/blogger/internals/initialisers"
	"github.com/waltertaya/blogger/internals/middlewares"
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
	mux.Handle("/api/v1/me/update", middlewares.AuthMiddleware(http.HandlerFunc(handlers.UpdateProfileHandler)))
	mux.Handle("/api/v1/me/deactivate", middlewares.AuthMiddleware(http.HandlerFunc(handlers.DeactivateAccountHandler)))

	mux.Handle("/api/v1/blogs/create", middlewares.AuthMiddleware(http.HandlerFunc(handlers.CreateBlogHandler)))
	mux.Handle("/api/v1/blogs/publish", middlewares.AuthMiddleware(http.HandlerFunc(handlers.PublishBlogHandler)))
	mux.Handle("/api/v1/blogs/update", middlewares.AuthMiddleware(http.HandlerFunc(handlers.UpdateBlogHandler)))
	mux.Handle("/api/v1/blogs/delete", middlewares.AuthMiddleware(http.HandlerFunc(handlers.DeleteBlogHandler)))
	mux.Handle("/api/v1/blogs/comment", middlewares.AuthMiddleware(http.HandlerFunc(handlers.CommentOnBlogHandler)))
	mux.HandleFunc("/api/v1/blogs/one", handlers.GetBlogByIDHandler)
	mux.HandleFunc("/api/v1/blogs/author", handlers.GetAuthorBlogsHandler)
	mux.HandleFunc("/api/v1/blogs", handlers.GetAllBlogsHandler)
	mux.HandleFunc("/api/v1/blogs/trending", handlers.GetTrendingBlogsHandler)
	mux.HandleFunc("/api/v1/blogs/tag", handlers.GetBlogsByTagHandler)
	mux.HandleFunc("/api/v1/blogs/recent", handlers.GetRecentBlogsHandler)
	mux.HandleFunc("/api/v1/blogs/like", handlers.LikeBlogHandler)
	mux.HandleFunc("/api/v1/users/profile", handlers.GetAnotherUserProfileHandler)

	templateResourcesJS := http.StripPrefix("/resources/js/", http.FileServer(http.Dir("templates/resources/js")))
	mux.Handle("/resources/js/", templateResourcesJS)

	templateResourcesCSS := http.StripPrefix("/resources/css/", http.FileServer(http.Dir("templates/resources/css")))
	mux.Handle("/resources/css/", templateResourcesCSS)

	resources := http.StripPrefix("/resources/", http.FileServer(http.Dir("internals/resources")))
	mux.Handle("/resources/", resources)

	handler := middlewares.CORSMiddleware(
		middlewares.LoggingMiddleware(mux),
	)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.ServeFile(w, r, "templates/landing.html")
		case "/auth":
			http.ServeFile(w, r, "templates/auth.html")
		case "/home":
			http.ServeFile(w, r, "templates/home.html")
		default:
			http.NotFound(w, r)
		}
	})

	logger.Println("Starting the server at http://localhost:8080")
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
