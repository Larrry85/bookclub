// post/post.go
package post

import (
	"database/sql"
	"lions/database"
	"lions/session"
	"log"
	"net/http"
	"text/template"

	"github.com/google/uuid"
)

// Define your Post struct with appropriate field tags
type Post struct {
	ID         string
	Title      string
	Content    string
	Username   string
	Category   string
	Likes      int
	Dislikes   int
	CategoryID int
	UserID     int
	RepliesCount  int
    Views         int
    LastReplyDate string
    LastReplyUser string
    IsPopular     bool
}

type PageData struct {
    Authenticated bool
    Username      string
    Posts         []Post
    Post          Post
    Replies       []Reply
}

type Reply struct {
	Content string
}

type PostViewData struct {
	Post          Post
	Replies       []Reply
	Authenticated bool
	Username      string
}

func ViewPost(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("id")

	var post Post
	err := database.DB.QueryRow(`
        SELECT PostID, Title, Content, CategoryID, UserID
        FROM Post
        WHERE PostID = ?`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.CategoryID, &post.UserID)
	if err != nil {
		http.Error(w, "Could not retrieve post", http.StatusInternalServerError)
		return
	}

	var categoryName string
	err = database.DB.QueryRow(`SELECT CategoryName FROM Category WHERE CategoryID = ?`, post.CategoryID).Scan(&categoryName)
	if err != nil {
		http.Error(w, "Could not retrieve category", http.StatusInternalServerError)
		return
	}
	post.Category = categoryName

	var username string
	err = database.DB.QueryRow(`SELECT Username FROM User WHERE UserID = ?`, post.UserID).Scan(&username)
	if err != nil {
		http.Error(w, "Could not retrieve username", http.StatusInternalServerError)
		return
	}
	post.Username = username

	err = database.DB.QueryRow(`
        SELECT 
            (SELECT COUNT(*) FROM LikesDislikes WHERE PostID = ? AND IsLike = 1) AS Likes,
            (SELECT COUNT(*) FROM LikesDislikes WHERE PostID = ? AND IsLike = 0) AS Dislikes
        `, postID, postID).Scan(&post.Likes, &post.Dislikes)
	if err != nil {
		http.Error(w, "Could not retrieve likes/dislikes", http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query(`SELECT Content FROM Comment WHERE PostID = ?`, postID)
	if err != nil {
		http.Error(w, "Could not retrieve comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var replies []Reply
	for rows.Next() {
		var reply Reply
		if err := rows.Scan(&reply.Content); err != nil {
			http.Error(w, "Could not scan comment", http.StatusInternalServerError)
			return
		}
		replies = append(replies, reply)
	}

	// Retrieve session data from context
	authenticated, _ := r.Context().Value("authenticated").(bool)
	username, _ = r.Context().Value("username").(string)

	data := PostViewData{
		Post:          post,
		Replies:       replies,
		Authenticated: authenticated,
		Username:      username,
	}
	tmpl, err := template.ParseFiles("static/html/view_post.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")

		postID := uuid.New().String()

		sessionID, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		sessionData, authenticated := session.GetSession(sessionID.Value)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var userID int
		err = database.DB.QueryRow(`SELECT UserID FROM User WHERE Username = ?`, sessionData.Username).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("No user found with username: %s", sessionData.Username)
				http.Error(w, "Could not find user", http.StatusInternalServerError)
			} else {
				log.Printf("Database error retrieving user ID: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Creating post for user ID: %d", userID)

		var categoryID int
		err = database.DB.QueryRow(`SELECT CategoryID FROM Category WHERE CategoryName = ?`, category).Scan(&categoryID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Could not find category", http.StatusInternalServerError)
			return
		}

		if err == sql.ErrNoRows {
			result, err := database.DB.Exec(`INSERT INTO Category (CategoryName) VALUES (?)`, category)
			if err != nil {
				http.Error(w, "Could not create category", http.StatusInternalServerError)
				return
			}
			categoryID64, err := result.LastInsertId()
			if err != nil {
				http.Error(w, "Could not retrieve category ID", http.StatusInternalServerError)
				return
			}
			categoryID = int(categoryID64)
		}

		log.Printf("Using category ID: %d for category: %s", categoryID, category)

		_, err = database.DB.Exec(`INSERT INTO Post (PostID, Title, Content, UserID, CategoryID) VALUES (?, ?, ?, ?, ?)`,
			postID, title, content, userID, categoryID)
		if err != nil {
			http.Error(w, "Could not create post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/post", http.StatusSeeOther)
	} else {
		tmpl, err := template.ParseFiles("static/html/create_post.html")
		if err != nil {
			http.Error(w, "Template parsing error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, "Template execution error", http.StatusInternalServerError)
			return
		}
	}
}

func ListPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
        SELECT PostID, Title, Content, CategoryID, UserID
        FROM Post`)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		log.Printf("Error retrieving posts: %v", err)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		var userID int

		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.CategoryID, &userID); err != nil {
			http.Error(w, "Could not scan post", http.StatusInternalServerError)
			log.Printf("Error scanning post: %v", err)
			return
		}

		post.UserID = userID

		var categoryName string
		err = database.DB.QueryRow(`SELECT CategoryName FROM Category WHERE CategoryID = ?`, post.CategoryID).Scan(&categoryName)
		if err != nil {
			http.Error(w, "Could not retrieve category", http.StatusInternalServerError)
			log.Printf("Error retrieving category: %v", err)
			return
		}
		post.Category = categoryName

		var username string
		err = database.DB.QueryRow(`SELECT Username FROM User WHERE UserID = ?`, post.UserID).Scan(&username)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("No username found for UserID: %d", post.UserID)
				username = "Unknown"
			} else {
				http.Error(w, "Could not retrieve username", http.StatusInternalServerError)
				log.Printf("Error retrieving username: %v", err)
				return
			}
		}
		post.Username = username

		posts = append(posts, post)
	}

	// Check for session cookie
	sessionCookie, err := r.Cookie("session_id")
	var authenticated bool
	var username string

	if err == nil {
		sessionData, exists := session.GetSession(sessionCookie.Value)
		if exists {
			authenticated = sessionData.Authenticated
			username = sessionData.Username
		}
	}

	data := struct {
		Posts         []Post
		Authenticated bool
		Username      string
	}{
		Posts:         posts,
		Authenticated: authenticated,
		Username:      username,
	}

	tmpl, err := template.ParseFiles("static/html/post.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		log.Printf("Template parsing error: %v", err)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		return
	}
}

// AddReply handles adding a reply to a post
func AddReply(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		postID := r.FormValue("post_id")
		content := r.FormValue("content")

		if postID == "" || content == "" {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		_, err := database.DB.Exec(`INSERT INTO Comment (PostID, Content) VALUES (?, ?)`, postID, content)
		if err != nil {
			http.Error(w, "Could not add reply", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/post/view?id="+postID, http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
