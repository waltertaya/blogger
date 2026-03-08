package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/waltertaya/blogger/internals/db"
	"github.com/waltertaya/blogger/internals/helpers"
	"github.com/waltertaya/blogger/internals/models"
)

const (
	profilesDir  = "internals/resources/profiles"
	maxImageSize = 5 << 20
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
	err = db.DB.Get(&verificationCode, "SELECT * FROM verification_codes WHERE code=?", codeInt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Verification code is invalid", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error finding verification code", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec("UPDATE users SET email_verified_at=? WHERE id=?", time.Now(), verificationCode.UserID)
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

	err := db.DB.Get(&userData, "SELECT * FROM users WHERE username=?", userCredential.Username)
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
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=?", userIDUint)
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
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=?", userIDUint)
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
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=?", userIDUint)
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

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clearAuthCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func DeactivateAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}

	banners := []string{}
	if err := db.DB.Select(&banners, "SELECT banner FROM blogs WHERE author=? AND banner <> ''", userID); err != nil {
		http.Error(w, "Error fetching blog banners", http.StatusInternalServerError)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM blog_comments WHERE user_id=?", userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting user comments", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM blog_comments WHERE blog_id IN (SELECT id FROM blogs WHERE author=?)", userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting comments on user blogs", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM blogs WHERE author=?", userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting user blogs", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM verification_codes WHERE user_id=?", userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting verification codes", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM users WHERE id=?", userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Error committing transaction", http.StatusInternalServerError)
		return
	}

	if user.ProfileImage != nil && *user.ProfileImage != "" {
		_ = os.Remove(filepath.Join(profilesDir, *user.ProfileImage))
	}

	for _, banner := range banners {
		if banner == "" {
			continue
		}
		_ = os.Remove(filepath.Join(bannersDir, banner))
	}

	clearAuthCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "account deactivated successfully"})
}

func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var currentUser models.User
	err = db.DB.Get(&currentUser, "SELECT * FROM users WHERE id=?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}

	updates := map[string]any{}
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Invalid multipart form", http.StatusBadRequest)
			return
		}

		if username := strings.TrimSpace(r.FormValue("username")); username != "" {
			updates["username"] = username
		}
		if email := strings.TrimSpace(r.FormValue("email")); email != "" {
			updates["email"] = email
		}

		profileFile, profileHeader, fileErr := r.FormFile("profile_image")
		if fileErr == nil {
			defer profileFile.Close()

			fileName, err := saveImageFile(profileFile, profileHeader, profilesDir, "profile")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			updates["profile_image"] = fileName
		} else if !errors.Is(fileErr, http.ErrMissingFile) {
			http.Error(w, "Invalid profile image", http.StatusBadRequest)
			return
		}
	} else {
		var payload struct {
			Username *string `json:"username"`
			Email    *string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid json", http.StatusBadRequest)
			return
		}

		if payload.Username != nil {
			username := strings.TrimSpace(*payload.Username)
			if username == "" {
				http.Error(w, "username cannot be empty", http.StatusBadRequest)
				return
			}
			updates["username"] = username
		}
		if payload.Email != nil {
			email := strings.TrimSpace(*payload.Email)
			if email == "" {
				http.Error(w, "email cannot be empty", http.StatusBadRequest)
				return
			}
			updates["email"] = email
		}
	}

	if len(updates) == 0 {
		http.Error(w, "No updates provided", http.StatusBadRequest)
		return
	}

	queryParts := []string{}
	args := []any{}
	for key, value := range updates {
		queryParts = append(queryParts, key+"=?")
		args = append(args, value)
	}
	queryParts = append(queryParts, "updated_at=?")
	args = append(args, time.Now(), userID)

	query := "UPDATE users SET " + strings.Join(queryParts, ", ") + " WHERE id=?"
	_, err = db.DB.Exec(query, args...)
	if err != nil {
		http.Error(w, "Error updating profile", http.StatusInternalServerError)
		return
	}

	if newImage, ok := updates["profile_image"].(string); ok && currentUser.ProfileImage != nil && *currentUser.ProfileImage != "" && *currentUser.ProfileImage != newImage {
		_ = os.Remove(filepath.Join(profilesDir, *currentUser.ProfileImage))
	}

	var updatedUser models.User
	err = db.DB.Get(&updatedUser, "SELECT * FROM users WHERE id=?", userID)
	if err != nil {
		http.Error(w, "Error fetching updated profile", http.StatusInternalServerError)
		return
	}

	updatedUser.Password = ""
	writeJSON(w, http.StatusOK, updatedUser)
}

func currentUserIDFromContext(r *http.Request) (uint64, error) {
	userIDValue := r.Context().Value("userID")
	userIDStr, ok := userIDValue.(string)
	if !ok || userIDStr == "" {
		return 0, errors.New("unauthorized")
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func saveImageFile(file multipart.File, fileHeader *multipart.FileHeader, targetDir, prefix string) (string, error) {
	if fileHeader.Size > maxImageSize {
		return "", errors.New("image size cannot exceed 5MB")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}

	if !allowedExt[ext] {
		return "", errors.New("unsupported image format")
	}

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return "", errors.New("failed to prepare upload directory")
	}

	fileName := fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), ext)
	fullPath := filepath.Join(targetDir, fileName)

	destination, err := os.Create(fullPath)
	if err != nil {
		return "", errors.New("failed to save image")
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		return "", errors.New("failed to write image")
	}

	return fileName, nil
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

func clearAuthCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		HttpOnly: true,
		Secure:   false, // true for https
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}

	http.SetCookie(w, &cookie)
}
