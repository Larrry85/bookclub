package post

import (
	"database/sql"
	"lions/database"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/sessions"
)

// Define your Post struct with appropriate field tags
type Post struct {
	ID        int
	Title     string
	Content   string
	Username  string
	Category  string
	Likes     int
	Dislikes  int
	CategoryID int // Added field
	UserID    int // Added field
}

// Define your User struct for session management
type User struct {
	Username string
}

type Reply struct {
	Content string
}

type PostViewData struct {
	Post          Post
	Replies       []Reply
	Authenticated bool
}

var (
	// Replace with your own secret key
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

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

    // Retrieve the post's category name
    var categoryName string
    err = database.DB.QueryRow(`SELECT CategoryName FROM Category WHERE CategoryID = ?`, post.CategoryID).Scan(&categoryName)
    if err != nil {
        http.Error(w, "Could not retrieve category", http.StatusInternalServerError)
        return
    }
    post.Category = categoryName

    // Retrieve the post's author
    var username string
    err = database.DB.QueryRow(`SELECT Username FROM User WHERE UserID = ?`, post.UserID).Scan(&username)
    if err != nil {
        http.Error(w, "Could not retrieve username", http.StatusInternalServerError)
        return
    }
    post.Username = username

    // Retrieve the number of likes and dislikes
    err = database.DB.QueryRow(`
        SELECT 
            (SELECT COUNT(*) FROM LikesDislikes WHERE PostID = ? AND IsLike = 1) AS Likes,
            (SELECT COUNT(*) FROM LikesDislikes WHERE PostID = ? AND IsLike = 0) AS Dislikes
        `, postID, postID).Scan(&post.Likes, &post.Dislikes)
    if err != nil {
        http.Error(w, "Could not retrieve likes/dislikes", http.StatusInternalServerError)
        return
    }

    // Retrieve comments
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

    // Retrieve session data
    session, _ := store.Get(r, "session")
    authenticated := session.Values["username"] != nil
    username, _ = session.Values["username"].(string)

    // Prepare data for template
    data := struct {
        Post          Post
        Replies       []Reply
        Authenticated bool
        Username      string
    }{
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
		// Get form values
		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")

		// Get session
		session, _ := store.Get(r, "session")
		username, ok := session.Values["username"].(string)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Retrieve user ID
		var userID int
		err := database.DB.QueryRow(`SELECT UserID FROM User WHERE Username = ?`, username).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("No user found with username: %s", username)
				http.Error(w, "Could not find user", http.StatusInternalServerError)
			} else {
				log.Printf("Database error retrieving user ID: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Creating post for user ID: %d", userID)

		// Retrieve or create category ID
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

		// Insert post
		_, err = database.DB.Exec(`INSERT INTO Post (Title, Content, UserID, CategoryID) VALUES (?, ?, ?, ?)`,
			title, content, userID, categoryID)
		if err != nil {
			http.Error(w, "Could not create post", http.StatusInternalServerError)
			return
		}

		log.Printf("Post created successfully for user ID: %d", userID)
		http.Redirect(w, r, "/post", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func ListPosts(w http.ResponseWriter, r *http.Request) {
    // Retrieve posts from database
    rows, err := database.DB.Query(`
        SELECT p.PostID, p.Title, p.Content, u.Username, c.CategoryName,
               (SELECT COUNT(*) FROM LikesDislikes l WHERE l.PostID = p.PostID AND l.IsLike = 1) AS Likes,
               (SELECT COUNT(*) FROM LikesDislikes l WHERE l.PostID = p.PostID AND l.IsLike = 0) AS Dislikes
        FROM Post p
        JOIN User u ON p.UserID = u.UserID
        LEFT JOIN Category c ON p.CategoryID = c.CategoryID
        ORDER BY p.PostID DESC
    `)
    if err != nil {
        http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var posts []Post
    for rows.Next() {
        var post Post
        if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.Category, &post.Likes, &post.Dislikes); err != nil {
            http.Error(w, "Could not scan post", http.StatusInternalServerError)
            return
        }
        posts = append(posts, post)
    }

    if err = rows.Err(); err != nil {
        http.Error(w, "Error occurred while processing posts", http.StatusInternalServerError)
        return
    }

    session, err := store.Get(r, "session")
    if err != nil {
        http.Error(w, "Session error", http.StatusInternalServerError)
        return
    }
    authenticated := session.Values["username"] != nil
    username, _ := session.Values["username"].(string)

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
        return
    }
    if err := tmpl.Execute(w, data); err != nil {
        http.Error(w, "Template execution error", http.StatusInternalServerError)
        return
    }
}

// AddReply handles adding a reply to a post
func AddReply(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Get form values
		postID := r.FormValue("post_id")
		content := r.FormValue("content")

		// Validate input
		if postID == "" || content == "" {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// Insert reply into database
		_, err := database.DB.Exec(`INSERT INTO Comment (PostID, Content) VALUES (?, ?)`, postID, content)
		if err != nil {
			http.Error(w, "Could not add reply", http.StatusInternalServerError)
			return
		}

		// Redirect to the post view page
		http.Redirect(w, r, "/post/view?id="+postID, http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

