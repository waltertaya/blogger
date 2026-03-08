package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/waltertaya/blogging-app/internals/db"
	"github.com/waltertaya/blogging-app/internals/helpers"
	"github.com/waltertaya/blogging-app/internals/models"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	if user.Username == "" || user.Email == "" || user.Password == "" {
		http.Error(w, "username, email and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := helpers.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Error creating password hash", http.StatusInternalServerError)
		return
	}
	result, err := db.DB.NamedExec("INSERT INTO users (username, email, password) VALUES (:username, :email, :password)", map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"password": string(hashedPassword),
	})
	if err != nil {
		http.Error(w, "Error inserting user to db", http.StatusInternalServerError)
		return
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Error retrieving created user id", http.StatusInternalServerError)
		return
	}
	userID := uint(insertedID)

	code := helpers.GenerateCode()
	var verificationCode = models.VerificationCode{
		Code:   code,
		UserID: userID,
	}

	_, err = db.DB.NamedExec("INSERT INTO verification_codes (code, user_id) VALUES (:code, :user_id)", &verificationCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error inserting code to db: %v", err), http.StatusInternalServerError)
		return
	}

	err = helpers.SendVerifyMail(user.Username, user.Email, code)
	if err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	var token string
	token, err = helpers.GenerateJWT(strconv.FormatUint(uint64(userID), 10))
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, token)

	response := map[string]interface{}{
		"id":       userID,
		"username": user.Username,
		"email":    user.Email,
	}

	writeJSON(w, http.StatusCreated, response)
}

func VerifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code cannot be empty", http.StatusBadRequest)
		return
	}

	codeInt, err := strconv.Atoi(code)
	if err != nil {
		http.Error(w, "Verification code is invalid", http.StatusBadRequest)
		return
	}

	var verificationCode models.VerificationCode
	err = db.DB.Get(&verificationCode, "SELECT * FROM verification_codes WHERE code=$1", codeInt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Verification code is invalid", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error finding verification code", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec("UPDATE users SET email_verified_at=$1 WHERE id=$2", time.Now(), verificationCode.UserID)
	if err != nil {
		http.Error(w, "Error updating email verification in users", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Email verified successfully"))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var userCredential struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userCredential); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	if userCredential.Username == "" || userCredential.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	var userData models.User

	err := db.DB.Get(&userData, "SELECT * FROM users WHERE username=$1", userCredential.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := helpers.ComparePassword([]byte(userData.Password), userCredential.Password); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	var token string
	token, err = helpers.GenerateJWT(strconv.FormatUint(uint64(userData.ID), 10))
	if err != nil {
		http.Error(w, "could not generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, token)

	userData.Password = ""
	writeJSON(w, http.StatusOK, userData)
}

func RequestNewEmailVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	if body.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	var user models.User
	err := db.DB.Get(&user, "SELECT * FROM users WHERE email=$1", body.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}

	code := helpers.GenerateCode()
	verificationCode := models.VerificationCode{Code: code, UserID: user.ID}

	_, err = db.DB.NamedExec("INSERT INTO verification_codes (code, user_id) VALUES (:code, :user_id)", &verificationCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error inserting code to db: %v", err), http.StatusInternalServerError)
		return
	}

	err = helpers.SendVerifyMail(user.Username, user.Email, code)
	if err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "verification email sent"})
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDValue := r.Context().Value("userID")
	userIDStr, ok := userIDValue.(string)
	if !ok || userIDStr == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user id in token", http.StatusUnauthorized)
		return
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		http.Error(w, "current_password and new_password are required", http.StatusBadRequest)
		return
	}

	var user models.User
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=$1", userIDUint)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := helpers.ComparePassword([]byte(user.Password), body.CurrentPassword); err != nil {
		http.Error(w, "Current password is invalid", http.StatusUnauthorized)
		return
	}

	hashedPassword, err := helpers.HashPassword(body.NewPassword)
	if err != nil {
		http.Error(w, "Error creating password hash", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec("UPDATE users SET password=$1, updated_at=$2 WHERE id=$3", string(hashedPassword), time.Now(), userIDUint)
	if err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password reset successful"})
}

func Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDValue := r.Context().Value("userID")
	userIDStr, ok := userIDValue.(string)
	if !ok || userIDStr == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user id in token", http.StatusUnauthorized)
		return
	}

	var user models.User
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=$1", userIDUint)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error fetching user profile", http.StatusInternalServerError)
		return
	}

	user.Password = ""
	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, _ = w.Write(jsonData)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // true for https
		Path:     "/",
		MaxAge:   86400,
	}

	http.SetCookie(w, &cookie)
}
