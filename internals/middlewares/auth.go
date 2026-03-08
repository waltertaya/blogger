package middlewares

import (
	"context"
	"net/http"

	"github.com/waltertaya/blogger/internals/helpers"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := helpers.VerifyJWT(cookie.Value)

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
