package post

import (
	"database/sql"
	"lions/database"
	"lions/session"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/google/uuid"
)

// Define your Post struct with appropriate field tags
type Post struct {
	ID            string
	Title         string
	Content       string
	Username      string
	Category      string
	Likes         int
	Dislikes      int
	CategoryID    int
	UserID        int
	RepliesCount  int
	Views         int
	LastReplyDate string
	LastReplyUser string
	IsPopular     bool
}

type Pagination struct {
	CurrentPage int
	TotalPages  int
}

type PageData struct {
	Authenticated bool
	Username      string
	Posts         []Post
	Post          Post
	Replies       []Reply
	Pagination    Pagination
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

// Define the template functions
func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func getSessionData(r *http.Request) (authenticated bool, username string, userID int, err error) {
	sessionID, err := r.Cookie("session_id")
	if err != nil {
		return false, "", 0, nil
	}

	sessionData, authenticated := session.GetSession(sessionID.Value)
	if !authenticated {
		return false, "", 0, nil
	}

	username = sessionData.Username
	userID = sessionData.UserID
	return authenticated, username, userID, nil
}

// ViewPost handles displaying a single post and its replies
func ViewPost(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("id")
	if postID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

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

	// Use helper function to get session data
	authenticated, username, _, err := getSessionData(r)
	if err != nil {
		http.Error(w, "Could not retrieve session data", http.StatusInternalServerError)
		return
	}

	data := PostViewData{
		Post:          post,
		Replies:       replies,
		Authenticated: authenticated,
		Username:      username,
	}
	tmpl, err := template.New("view_post.html").Funcs(template.FuncMap{
		"add": add,
		"sub": sub,
	}).ParseFiles("static/html/view_post.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "view_post.html", data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

// CreatePost handles the creation of a new post
func CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")

		if title == "" || content == "" || category == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		postID := uuid.New().String()

		authenticated, username, _, err := getSessionData(r)
		if err != nil {
			http.Error(w, "Could not retrieve session data", http.StatusInternalServerError)
			return
		}
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var userID int
		err = database.DB.QueryRow(`SELECT UserID FROM User WHERE Username = ?`, username).Scan(&userID)
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

		_, err = database.DB.Exec(`INSERT INTO Post (PostID, Title, Content, UserID, CategoryID) VALUES (?, ?, ?, ?, ?)`,
			postID, title, content, userID, categoryID)
		if err != nil {
			http.Error(w, "Could not create post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/post", http.StatusSeeOther)
	} else {
		tmpl, err := template.New("create_post.html").Funcs(template.FuncMap{
			"add": add,
			"sub": sub,
		}).ParseFiles("static/html/create_post.html")
		if err != nil {
			http.Error(w, "Template parsing error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "create_post.html", nil); err != nil {
			http.Error(w, "Template execution error", http.StatusInternalServerError)
			return
		}
	}
}

func ListPosts(w http.ResponseWriter, r *http.Request) {

	// Extract page number from query parameters
	pageParam := r.URL.Query().Get("page")
	currentPage := 1
	if pageParam != "" {
		var err error
		currentPage, err = strconv.Atoi(pageParam)
		if err != nil || currentPage < 1 {
			currentPage = 1
		}
	}

	// Set up pagination parameters
	postsPerPage := 10
	offset := (currentPage - 1) * postsPerPage

	// Fetch total number of posts
	var totalPosts int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM Post").Scan(&totalPosts)
	if err != nil {
		http.Error(w, "Could not retrieve total post count", http.StatusInternalServerError)
		log.Printf("Error retrieving total post count: %v", err)
		return
	}

	// Fetch posts for the current page
	rows, err := database.DB.Query(`
        SELECT PostID, Title, Content, CategoryID, UserID
        FROM Post
        LIMIT ? OFFSET ?`, postsPerPage, offset)
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
			username = "Unknown"
		}
		post.Username = username

		posts = append(posts, post)
	}

	// Calculate pagination details
	totalPages := (totalPosts + postsPerPage - 1) / postsPerPage // Ceiling division

	// Check for session cookie
	sessionCookie, err := r.Cookie("session_id")
	authenticated, _ := r.Context().Value("Authenticated").(bool)
	username, _ := r.Context().Value("Username").(string)

	if err == nil {
		sessionData, exists := session.GetSession(sessionCookie.Value)
		if exists {
			authenticated = sessionData.Authenticated
			username = sessionData.Username
		}
	}

	// Prepare data for template
	data := PageData{
		Posts:         posts,
		Authenticated: authenticated,
		Username:      username,
		Pagination:    Pagination{CurrentPage: currentPage, TotalPages: totalPages},
	}

	tmpl, err := template.New("post.html").Funcs(template.FuncMap{
		"add": add,
		"sub": sub,
	}).ParseFiles("static/html/post.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		log.Printf("Template parsing error: %v", err)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "post.html", data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
		return
	}
}

func AddReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusInternalServerError)
		return
	}

	postID := r.FormValue("post_id")
	content := r.FormValue("content")

	if postID == "" || content == "" {
		http.Error(w, "Invalid input: post_id and content are required", http.StatusBadRequest)
		return
	}

	authenticated, _, userID, err := getSessionData(r)
	if err != nil {
		http.Error(w, "Could not retrieve session data", http.StatusInternalServerError)
		return
	}
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var username string
	err = database.DB.QueryRow(`SELECT Username FROM User WHERE UserID = ?`, userID).Scan(&username)
	if err != nil {
		log.Printf("Error retrieving username for UserID %d: %v", userID, err)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Could not retrieve username: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Adding reply for postID %s by user %s", postID, username)

	_, err = database.DB.Exec(`INSERT INTO Comment (PostID, UserID, Content) VALUES (?, ?, ?)`, postID, userID, content)
	if err != nil {
		log.Printf("Error adding reply: %v", err)
		http.Error(w, "Could not add reply: "+err.Error(), http.StatusInternalServerError)
		return
	}

	redirectURL := "/post/view?id=" + url.QueryEscape(postID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
