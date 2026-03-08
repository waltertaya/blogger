package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/waltertaya/blogging-app/internals/db"
	"github.com/waltertaya/blogging-app/internals/models"
)

const (
	bannersDir = "internals/resources/banners"
)

func CreateBlogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	tags := strings.TrimSpace(r.FormValue("tags"))
	published, _ := strconv.ParseBool(r.FormValue("published"))

	if title == "" || description == "" {
		http.Error(w, "title and description are required", http.StatusBadRequest)
		return
	}

	bannerName := ""
	bannerFile, bannerHeader, fileErr := r.FormFile("banner")
	if fileErr == nil {
		defer bannerFile.Close()

		bannerName, err = saveImageFile(bannerFile, bannerHeader, bannersDir, "banner")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if !errors.Is(fileErr, http.ErrMissingFile) {
		http.Error(w, "Invalid banner file", http.StatusBadRequest)
		return
	}

	blog := models.Blog{
		Title:       title,
		Description: description,
		Tags:        tags,
		Author:      uint(userID),
		Banner:      bannerName,
		Published:   published,
	}

	result, err := db.DB.NamedExec(`INSERT INTO blogs (title, description, tags, author, banner, published)
		VALUES (:title, :description, :tags, :author, :banner, :published)`, &blog)
	if err != nil {
		http.Error(w, "Error creating blog", http.StatusInternalServerError)
		return
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Error retrieving created blog id", http.StatusInternalServerError)
		return
	}

	err = db.DB.Get(&blog, "SELECT * FROM blogs WHERE id=?", insertedID)
	if err != nil {
		http.Error(w, "Error fetching created blog", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, blog)
}

func PublishBlogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	blogID, err := parseUintQueryParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid blog id", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("UPDATE blogs SET published=?, updated_at=? WHERE id=? AND author=?", true, time.Now(), blogID, userID)
	if err != nil {
		http.Error(w, "Error publishing blog", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Blog not found or unauthorized", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "blog published"})
}

func GetBlogByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blogID, err := parseUintQueryParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid blog id", http.StatusBadRequest)
		return
	}

	var blog models.Blog
	err = db.DB.Get(&blog, "SELECT * FROM blogs WHERE id=?", blogID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Blog not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error fetching blog", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, blog)
}

func GetAuthorBlogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authorID, err := parseUintQueryParam(r, "author_id")
	if err != nil {
		http.Error(w, "Invalid author id", http.StatusBadRequest)
		return
	}

	blogs := []models.Blog{}
	err = db.DB.Select(&blogs, "SELECT * FROM blogs WHERE author=? ORDER BY created_at DESC", authorID)
	if err != nil {
		http.Error(w, "Error fetching author blogs", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, blogs)
}

func GetAllBlogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blogs := []models.Blog{}
	err := db.DB.Select(&blogs, "SELECT * FROM blogs WHERE published=? ORDER BY created_at DESC", true)
	if err != nil {
		http.Error(w, "Error fetching blogs", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, blogs)
}

func GetTrendingBlogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blogs := []models.Blog{}
	err := db.DB.Select(&blogs, "SELECT * FROM blogs WHERE published=? ORDER BY likes DESC, created_at DESC LIMIT 10", true)
	if err != nil {
		http.Error(w, "Error fetching trending blogs", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, blogs)
}

func GetBlogsByTagHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if tag == "" {
		http.Error(w, "tag is required", http.StatusBadRequest)
		return
	}

	blogs := []models.Blog{}
	err := db.DB.Select(&blogs, "SELECT * FROM blogs WHERE published=? AND tags LIKE ? ORDER BY created_at DESC", true, "%"+tag+"%")
	if err != nil {
		http.Error(w, "Error fetching blogs by tag", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, blogs)
}

func GetRecentBlogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blogs := []models.Blog{}
	err := db.DB.Select(&blogs, "SELECT * FROM blogs WHERE published=? ORDER BY created_at DESC LIMIT 10", true)
	if err != nil {
		http.Error(w, "Error fetching recent blogs", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, blogs)
}

func LikeBlogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blogID, err := parseUintQueryParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid blog id", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("UPDATE blogs SET likes=likes+1, updated_at=? WHERE id=?", time.Now(), blogID)
	if err != nil {
		http.Error(w, "Error liking blog", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Blog not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "blog liked"})
}

func CommentOnBlogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	blogID, err := parseUintQueryParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid blog id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	payload.Comment = strings.TrimSpace(payload.Comment)
	if payload.Comment == "" {
		http.Error(w, "comment is required", http.StatusBadRequest)
		return
	}

	comment := models.BlogComments{BlogID: uint(blogID), UserID: uint(userID), Comment: payload.Comment}
	_, err = db.DB.NamedExec("INSERT INTO blog_comments (blog_id, user_id, comment) VALUES (:blog_id, :user_id, :comment)", &comment)
	if err != nil {
		http.Error(w, "Error adding comment", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "comment added"})
}

func GetAnotherUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := parseUintQueryParam(r, "user_id")
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		return
	}

	var user models.User
	err = db.DB.Get(&user, "SELECT * FROM users WHERE id=?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error fetching user profile", http.StatusInternalServerError)
		return
	}

	blogs := []models.Blog{}
	if err := db.DB.Select(&blogs, "SELECT * FROM blogs WHERE author=? AND published=? ORDER BY created_at DESC", userID, true); err != nil {
		http.Error(w, "Error fetching user blogs", http.StatusInternalServerError)
		return
	}

	user.Password = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"user":  user,
		"blogs": blogs,
	})
}

func UpdateBlogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	blogID, err := parseUintQueryParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid blog id", http.StatusBadRequest)
		return
	}

	var currentBlog models.Blog
	err = db.DB.Get(&currentBlog, "SELECT * FROM blogs WHERE id=? AND author=?", blogID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Blog not found or unauthorized", http.StatusNotFound)
			return
		}
		http.Error(w, "Error fetching blog", http.StatusInternalServerError)
		return
	}

	updates := map[string]any{}
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Invalid multipart form", http.StatusBadRequest)
			return
		}

		if title := strings.TrimSpace(r.FormValue("title")); title != "" {
			updates["title"] = title
		}
		if description := strings.TrimSpace(r.FormValue("description")); description != "" {
			updates["description"] = description
		}
		if tags := strings.TrimSpace(r.FormValue("tags")); tags != "" {
			updates["tags"] = tags
		}
		if published := strings.TrimSpace(r.FormValue("published")); published != "" {
			publishedValue, err := strconv.ParseBool(published)
			if err != nil {
				http.Error(w, "Invalid published value", http.StatusBadRequest)
				return
			}
			updates["published"] = publishedValue
		}

		bannerFile, bannerHeader, fileErr := r.FormFile("banner")
		if fileErr == nil {
			defer bannerFile.Close()

			bannerName, err := saveImageFile(bannerFile, bannerHeader, bannersDir, "banner")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			updates["banner"] = bannerName
		} else if !errors.Is(fileErr, http.ErrMissingFile) {
			http.Error(w, "Invalid banner file", http.StatusBadRequest)
			return
		}
	} else {
		var payload struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
			Tags        *string `json:"tags"`
			Published   *bool   `json:"published"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid json", http.StatusBadRequest)
			return
		}

		if payload.Title != nil {
			title := strings.TrimSpace(*payload.Title)
			if title == "" {
				http.Error(w, "title cannot be empty", http.StatusBadRequest)
				return
			}
			updates["title"] = title
		}
		if payload.Description != nil {
			description := strings.TrimSpace(*payload.Description)
			if description == "" {
				http.Error(w, "description cannot be empty", http.StatusBadRequest)
				return
			}
			updates["description"] = description
		}
		if payload.Tags != nil {
			updates["tags"] = strings.TrimSpace(*payload.Tags)
		}
		if payload.Published != nil {
			updates["published"] = *payload.Published
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
	args = append(args, time.Now(), blogID, userID)

	query := "UPDATE blogs SET " + strings.Join(queryParts, ", ") + " WHERE id=? AND author=?"
	_, err = db.DB.Exec(query, args...)
	if err != nil {
		http.Error(w, "Error updating blog", http.StatusInternalServerError)
		return
	}

	if newBanner, ok := updates["banner"].(string); ok && currentBlog.Banner != "" && currentBlog.Banner != newBanner {
		_ = os.Remove(bannersDir + "/" + currentBlog.Banner)
	}

	var updatedBlog models.Blog
	err = db.DB.Get(&updatedBlog, "SELECT * FROM blogs WHERE id=?", blogID)
	if err != nil {
		http.Error(w, "Error fetching updated blog", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, updatedBlog)
}

func DeleteBlogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := currentUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	blogID, err := parseUintQueryParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid blog id", http.StatusBadRequest)
		return
	}

	var blog models.Blog
	err = db.DB.Get(&blog, "SELECT * FROM blogs WHERE id=? AND author=?", blogID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Blog not found or unauthorized", http.StatusNotFound)
			return
		}
		http.Error(w, "Error fetching blog", http.StatusInternalServerError)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM blog_comments WHERE blog_id=?", blogID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting blog comments", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec("DELETE FROM blogs WHERE id=? AND author=?", blogID, userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Error deleting blog", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		_ = tx.Rollback()
		http.Error(w, "Blog not found or unauthorized", http.StatusNotFound)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Error committing transaction", http.StatusInternalServerError)
		return
	}

	if blog.Banner != "" {
		_ = os.Remove(bannersDir + "/" + blog.Banner)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "blog deleted"})
}

func parseUintQueryParam(r *http.Request, key string) (uint64, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0, errors.New("missing query parameter")
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}
