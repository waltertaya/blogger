package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/waltertaya/blogging-app/internals/db"
	"github.com/waltertaya/blogging-app/internals/helpers"
	"github.com/waltertaya/blogging-app/internals/models"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Inavalid json", http.StatusBadRequest)
		return
	}

	hashedPassword, err := helpers.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Error creating password hash", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	_, err = db.DB.NamedExec("INSERT INTO users (username, email, password) VALUES (:username, :email, :password)", &user)
	if err != nil {
		http.Error(w, "Error inserting user to db", http.StatusInternalServerError)
		return
	}

	var token string
	token, err = helpers.GenerateJWT(string(user.ID))
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // true for https
		Path:     "/",
		MaxAge:   86400,
	}

	http.SetCookie(w, &cookie)

	user.Password = ""
	jsonData, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}
