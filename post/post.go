//post.go
package post

import (
	"database/sql"
	"net/http"
	"text/template"
	"github.com/gorilla/sessions"
	"lions/database"

)

type Post struct {
	ID       int
	Title    string
	Content  string
	Username string
	Category string
}

var (
	// Replace with your own secret key
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")

		session, _ := store.Get(r, "session")
		username, _ := session.Values["username"].(string)

		var userID int
		err := database.DB.QueryRow(`SELECT UserID FROM User WHERE Username = ?`, username).Scan(&userID)
		if err != nil {
			http.Error(w, "Could not find user", http.StatusInternalServerError)
			return
		}

		var categoryID int
		err = database.DB.QueryRow(`SELECT CategoryID FROM Category WHERE CategoryName = ?`, category).Scan(&categoryID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Could not find category", http.StatusInternalServerError)
			return
		}

		if err == sql.ErrNoRows {
			_, err = database.DB.Exec(`INSERT INTO Category (CategoryName) VALUES (?)`, category)
			if err != nil {
				http.Error(w, "Could not create category", http.StatusInternalServerError)
				return
			}
			err = database.DB.QueryRow(`SELECT last_insert_rowid()`).Scan(&categoryID)
			if err != nil {
				http.Error(w, "Could not retrieve category ID", http.StatusInternalServerError)
				return
			}
		}

		_, err = database.DB.Exec(`INSERT INTO Post (Title, Content, UserID, CategoryID) VALUES (?, ?, ?, ?)`, title, content, userID, categoryID)
		if err != nil {
			http.Error(w, "Could not create post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/posts", http.StatusSeeOther)
	}
}

func ListPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT Post.PostID, Post.Title, Post.Content, User.Username, Category.CategoryName
		FROM Post
		JOIN User ON Post.UserID = User.UserID
		LEFT JOIN Category ON Post.CategoryID = Category.CategoryID
	`)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.Category); err != nil {
			http.Error(w, "Could not scan post", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	tmpl, err := template.ParseFiles("static/html/posts.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, posts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

