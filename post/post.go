// post.go
package post

import (
	"database/sql"
	"fmt"
	"lions/database"
	"lions/session"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
	"time"

	"github.com/google/uuid"

	"io"
	"os"
)

// Post represents a blog post with various details.
type Post struct {
	ID              string // Unique identifier for the post
	Title           string // Title of the post
	Content         string // Content of the post
	Username        string // Username of the post author
	Category        string // Category of the post
	CommentLikes    int
	CommentDislikes int
	Likes           int            // Number of likes on the post
	Dislikes        int            // Number of dislikes on the post
	CategoryID      int            // ID of the category
	UserID          int            // ID of the user who created the post
	RepliesCount    int            // Number of replies to the post
	Views           int            // Number of views of the post
	LastReplyDate   sql.NullTime   // Date of the last reply
	LastReplyUser   sql.NullString // Username of the user who made the last reply
	CreatedAt       time.Time      // Timestamp when the post was created
	Images          []PostImage    // List of images associated with the post
}

type Category struct {
	Name  string
	Posts []Post
}

// Pagination represents pagination data for listing posts.
type Pagination struct {
	CurrentPage int // Current page number
	TotalPages  int // Total number of pages
	PageSize    int
}

// PageData holds data for rendering post list or details pages.
type PageData struct {
	Authenticated bool       // Whether the user is authenticated
	Username      string     // Username of the authenticated user
	Posts         []Post     // List of posts to display
	Post          Post       // Single post for detailed view
	Replies       []Reply    // List of replies to a post
	Pagination    Pagination // Pagination data
	CurrentPage   int
	TotalPages    int
	Filter        FilterParams
	Categories    []Category
}

type FilterParams struct {
	Category   string
	SortOrder  string
	LikesOrder string
}

// Reply represents a reply to a post with user information.
type Reply struct {
	ID         string
	Content    string
	Username   string
	CreatedAt  time.Time
	TaggedUser string
}

type FormattedReply struct {
	Reply              Reply
	FormattedCreatedAt string
	LikesCount         int
	DislikesCount      int
	TaggedUser         string
}

// PostViewData holds data for rendering a single post with its replies.
type PostViewData struct {
	Post                   Post
	Replies                []FormattedReply
	Authenticated          bool
	Username               string
	FormattedCreatedAt     string
	LastReplyDateFormatted string
	SameUser               bool
	Users                  []string
}

// PostImage represents an image associated with a blog post.
type PostImage struct {
	ID        string       // Unique identifier for the post image
	PostID    string       // ID of the associated post
	UserID    int          // ID of the user who uploaded the image
	ImagePath string       // Path to the image file
	CreatedAt sql.NullTime // Timestamp when the image was uploaded
}

// ImageData holds data for rendering the image upload page.
type ImageData struct {
	Authenticated bool   // Whether the user is authenticated
	Username      string // Username of the authenticated user
}

// Define template functions for use in HTML templates.
func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

///////////////SessionMiddleware ////////////////////

func CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")
		file, handler, err := r.FormFile("image")
		if err != nil && err != http.ErrMissingFile {
			http.Error(w, "Error retrieving the file", http.StatusBadRequest)
			return
		}

		if title == "" || content == "" || category == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		authenticated := r.Context().Value(session.Authenticated).(bool)
		username := r.Context().Value(session.Username).(string)
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

		_, err = database.DB.Exec(`INSERT INTO Post (Title, Content, UserID, CategoryID, CreatedAt) VALUES (?, ?, ?, ?, ?)`,
			title, content, userID, categoryID, time.Now().Format(time.RFC3339))
		if err != nil {
			log.Printf("Error creating post: %v", err)
			http.Error(w, "Could not create post", http.StatusInternalServerError)
			return
		}

		if file != nil {
			fileName := uuid.New().String() + "_" + handler.Filename
			filePath := "uploads/" + fileName

			if err := os.MkdirAll("uploads", os.ModePerm); err != nil {
				http.Error(w, "Unable to create uploads directory", http.StatusInternalServerError)
				return
			}

			dst, err := os.Create(filePath)
			if err != nil {
				log.Println("Error creating file:", err)
				http.Error(w, "Unable to create the file", http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			_, err = io.Copy(dst, file)
			if err != nil {
				log.Println("Error saving file:", err)
				http.Error(w, "Unable to save the file", http.StatusInternalServerError)
				return
			}

			postID := r.FormValue("post_id")

			postImage := PostImage{
				ID:        uuid.New().String(),
				PostID:    postID,
				UserID:    userID,
				ImagePath: filePath,
				CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			}

			err = savePostImageToDB(postImage)
			if err != nil {
				log.Println("Error saving image to database:", err)
				http.Error(w, "Unable to save image information", http.StatusInternalServerError)
				return
			}
			log.Println("Image saved to database:", postImage)
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

func ViewPost(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("id")
	if postID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	var post Post
	err := database.DB.QueryRow(`
        SELECT p.PostID, p.Title, p.Content, p.CreatedAt, p.LastReplyDate, p.LastReplyUser, 
               u.Username, c.CategoryName, 
               (SELECT COUNT(*) FROM Comment WHERE PostID = p.PostID) AS RepliesCount,
               p.LikesCount, p.DislikesCount
        FROM Post p
        JOIN User u ON p.UserID = u.UserID
        JOIN Category c ON p.CategoryID = c.CategoryID
        WHERE p.PostID = ?`, postID).Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.LastReplyDate,
		&post.LastReplyUser,
		&post.Username,
		&post.Category,
		&post.RepliesCount,
		&post.Likes,
		&post.Dislikes,
	)
	if err != nil {
		log.Printf("Error fetching post: %v", err)
		http.Error(w, "Could not fetch post", http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query(`
        SELECT c.CommentID, c.Content, c.CreatedAt, u.Username, c.TaggedUser,
               (SELECT COUNT(*) FROM CommentLikes WHERE CommentID = c.CommentID AND IsLike = 1) AS LikesCount,
               (SELECT COUNT(*) FROM CommentLikes WHERE CommentID = c.CommentID AND IsLike = 0) AS DislikesCount
        FROM Comment c
        JOIN User u ON c.UserID = u.UserID
        WHERE c.PostID = ?
        ORDER BY c.CreatedAt DESC`, postID)
	if err != nil {
		log.Printf("Error fetching replies: %v", err)
		http.Error(w, "Could not fetch replies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var replies []FormattedReply
	for rows.Next() {
		var reply Reply
		var likesCount, dislikesCount int
		if err := rows.Scan(&reply.ID, &reply.Content, &reply.CreatedAt, &reply.Username, &reply.TaggedUser, &likesCount, &dislikesCount); err != nil {
			log.Printf("Error scanning reply: %v", err)
			continue
		}
		formattedReply := FormattedReply{
			Reply:              reply,
			FormattedCreatedAt: reply.CreatedAt.Format("January 2, 2006 at 3:04pm"),
			LikesCount:         likesCount,
			DislikesCount:      dislikesCount,
		}
		replies = append(replies, formattedReply)
	}

	var lastReplyDateFormatted string
	if post.LastReplyDate.Valid {
		lastReplyDateFormatted = post.LastReplyDate.Time.Format("January 2, 2006 at 3:04pm")
	} else {
		lastReplyDateFormatted = "No replies yet"
	}

	currentUsername := r.Context().Value(session.Username).(string)
	sameUser := currentUsername == post.Username

	users, err := fetchUsers()
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Could not fetch users", http.StatusInternalServerError)
		return
	}

	data := PostViewData{
		Post:                   post,
		Replies:                replies,
		Authenticated:          r.Context().Value(session.Authenticated).(bool),
		Username:               currentUsername,
		FormattedCreatedAt:     post.CreatedAt.Format("January 2, 2006 at 3:04pm"),
		LastReplyDateFormatted: lastReplyDateFormatted,
		SameUser:               sameUser,
		Users:                  users,
	}

	tmpl, err := template.New("view_post.html").Funcs(template.FuncMap{
		"add": add,
		"sub": sub,
	}).ParseFiles("static/html/view_post.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Could not load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Could not render template", http.StatusInternalServerError)
	}
}

// ListPosts handles displaying a paginated list of posts.
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

	// Fetch posts for the current page along with likes, dislikes, and comments count
	rows, err := database.DB.Query(`
        SELECT Post.PostID, Post.Title, Post.Content, Post.CategoryID, Post.UserID, Post.LastReplyUser, 
               Post.LastReplyDate, Post.CreatedAt,
               COALESCE(SUM(CASE WHEN PostLikes.IsLike = 1 THEN 1 ELSE 0 END), 0) AS Likes,
               COALESCE(SUM(CASE WHEN PostLikes.IsLike = 0 THEN 1 ELSE 0 END), 0) AS Dislikes,
               COALESCE((SELECT COUNT(*) FROM Comment WHERE Comment.PostID = Post.PostID), 0) AS NumComments
        FROM Post
        LEFT JOIN PostLikes ON Post.PostID = Postlikes.PostID
        GROUP BY Post.PostID
        ORDER BY Post.CreatedAt DESC
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
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.CategoryID, &post.UserID, &post.LastReplyUser,
			&post.LastReplyDate, &post.CreatedAt, &post.Likes, &post.Dislikes, &post.RepliesCount)
		if err != nil {
			http.Error(w, "Could not scan post", http.StatusInternalServerError)
			log.Printf("Error scanning post: %v", err)
			return
		}

		// Fetch the category name for each post
		var categoryName string
		err = database.DB.QueryRow(`SELECT CategoryName FROM Category WHERE CategoryID = ?`, post.CategoryID).Scan(&categoryName)
		if err != nil {
			http.Error(w, "Could not retrieve category", http.StatusInternalServerError)
			log.Printf("Error retrieving category: %v", err)
			return
		}
		post.Category = categoryName

		// Fetch the username for each post
		var username string
		err = database.DB.QueryRow(`SELECT Username FROM User WHERE UserID = ?`, post.UserID).Scan(&username)
		if err != nil {
			http.Error(w, "Could not retrieve username", http.StatusInternalServerError)
			log.Printf("Error retrieving username: %v", err)
			return
		}
		post.Username = username

		posts = append(posts, post)
	}

	// Calculate the total number of pages
	totalPages := (totalPosts + postsPerPage - 1) / postsPerPage

	// Set up the pagination data
	pagination := Pagination{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
	}

	// Use session data from the request context
	authenticated := r.Context().Value(session.Authenticated).(bool)
	username := r.Context().Value(session.Username).(string)

	// Prepare data for rendering the template
	data := PageData{
		Posts:         posts,
		Pagination:    pagination,
		Authenticated: authenticated,
		Username:      username,
	}

	tmpl, err := template.New("post.html").Funcs(template.FuncMap{
		"add": add,
		"sub": sub,
	}).ParseFiles("static/html/post.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "post.html", data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func AddReply(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Check if the user is authenticated
	authenticated := r.Context().Value(session.Authenticated).(bool)
	if !authenticated {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get the username from session
	username := r.Context().Value(session.Username).(string)

	// Get the post ID and content from the form data
	postID := r.FormValue("postID")
	content := r.FormValue("content")
	taggedUser := r.FormValue("tagged_user")

	// Validate input
	if postID == "" || content == "" {
		http.Error(w, "Post ID and content are required", http.StatusBadRequest)
		return
	}

	// Get the user ID
	var userID int
	err := database.DB.QueryRow(`SELECT UserID FROM User WHERE Username = ?`, username).Scan(&userID)
	if err != nil {
		log.Printf("Error retrieving user ID for username %s: %v", username, err)
		http.Error(w, "Could not retrieve user ID", http.StatusInternalServerError)
		return
	}

	// Get the current time
	now := time.Now()

	// Insert the reply into the database
	_, err = database.DB.Exec(`INSERT INTO Comment (PostID, UserID, Content, TaggedUser, CreatedAt) VALUES (?, ?, ?, ?, ?)`,
		postID, userID, content, taggedUser, now)
	if err != nil {
		log.Printf("Error inserting reply into database: %v", err)
		http.Error(w, "Could not add reply", http.StatusInternalServerError)
		return
	}

	// Update the last reply info in the post
	_, err = database.DB.Exec(`UPDATE Post SET LastReplyDate = ?, LastReplyUser = ? WHERE PostID = ?`,
		now, username, postID)
	if err != nil {
		log.Printf("Error updating post last reply: %v", err)
		http.Error(w, "Could not update post last reply", http.StatusInternalServerError)
		return
	}

	// Redirect to the post view page
	redirectURL := "/post/view?id=" + url.QueryEscape(postID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

///////////////SessionMiddleware END////////////////////

func fetchUsers() ([]string, error) {
	rows, err := database.DB.Query(`SELECT Username FROM User`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		users = append(users, username)
	}
	return users, nil
}

///////////////Filter posts ////////////////////

func FilterPostHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve filter, sort, and pagination parameters from query
	category := r.URL.Query().Get("category")
	sortOrder := r.URL.Query().Get("sort")
	likesOrder := r.URL.Query().Get("likes")
	pageParam := r.URL.Query().Get("page")
	pageSizeParam := r.URL.Query().Get("pageSize")

	// Default values for sorting if not provided
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if likesOrder == "" {
		likesOrder = "desc"
	}

	// Default values for pagination if not provided
	currentPage := 1
	if pageParam != "" {
		var err error
		currentPage, err = strconv.Atoi(pageParam)
		if err != nil || currentPage < 1 {
			currentPage = 1
		}
	}
	pageSize := 10
	if pageSizeParam != "" {
		pageSize, _ = strconv.Atoi(pageSizeParam)
	}
	offset := (currentPage - 1) * pageSize

	// Prepare category condition
	var categoryCondition string
	var args []interface{}

	if category == "all" || category == "" {
		categoryCondition = "1=1" // No filter
	} else {
		categoryCondition = "p.CategoryID = (SELECT CategoryID FROM Category WHERE CategoryName = ?)"
		args = append(args, category)
	}

	// Fetch total number of posts matching the filter criteria
	var totalPosts int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM Post p WHERE "+categoryCondition, args...).Scan(&totalPosts)
	if err != nil {
		http.Error(w, "Could not retrieve total post count", http.StatusInternalServerError)
		log.Printf("Error retrieving total post count: %v", err)
		return
	}

	// Construct SQL query with filtering, sorting, and pagination
	query := `
    SELECT p.PostID, p.Title, p.Content, p.UserID, p.CategoryID, 
           COALESCE(l.Likes, 0) AS Likes, 
           COALESCE(l.Dislikes, 0) AS Dislikes, 
           p.CreatedAt
    FROM Post p
    LEFT JOIN (
        SELECT PostID, 
               SUM(CASE WHEN IsLike = 1 THEN 1 ELSE 0 END) AS Likes,
               SUM(CASE WHEN IsLike = 0 THEN 1 ELSE 0 END) AS Dislikes
        FROM PostLikes
        GROUP BY PostID
    ) l ON p.PostID = l.PostID
    WHERE ` + categoryCondition + `
    ORDER BY 
        p.CreatedAt ` + sortOrder + `,
        l.Likes ` + likesOrder + `
    LIMIT ? OFFSET ?`

	// Append limit and offset to args
	args = append(args, pageSize, offset)

	// Execute the query
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Could not retrieve posts", http.StatusInternalServerError)
		log.Printf("Error retrieving posts: %v", err)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CategoryID, &post.Likes, &post.Dislikes, &post.CreatedAt)
		if err != nil {
			http.Error(w, "Could not scan post", http.StatusInternalServerError)
			log.Printf("Error scanning post: %v", err)
			return
		}

		// Fetch the category name for each post
		var categoryName string
		err = database.DB.QueryRow(`SELECT CategoryName FROM Category WHERE CategoryID = ?`, post.CategoryID).Scan(&categoryName)
		if err != nil {
			http.Error(w, "Could not retrieve category", http.StatusInternalServerError)
			log.Printf("Error retrieving category: %v", err)
			return
		}
		post.Category = categoryName

		// Fetch the username for each post
		var username string
		err = database.DB.QueryRow(`SELECT Username FROM User WHERE UserID = ?`, post.UserID).Scan(&username)
		if err != nil {
			http.Error(w, "Could not retrieve username", http.StatusInternalServerError)
			log.Printf("Error retrieving username: %v", err)
			return
		}
		post.Username = username

		posts = append(posts, post)
	}

	// Calculate the total number of pages
	totalPages := (totalPosts + pageSize - 1) / pageSize

	// Set up the pagination data
	pagination := Pagination{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		PageSize:    pageSize,
	}

	// Use session data from the request context
	authenticated := r.Context().Value(session.Authenticated).(bool)
	username := r.Context().Value(session.Username).(string)

	// Prepare data for rendering the template
	data := PageData{
		Posts:         posts,
		Pagination:    pagination,
		Authenticated: authenticated,
		Username:      username,
		Filter: FilterParams{
			Category:   category,
			SortOrder:  sortOrder,
			LikesOrder: likesOrder,
		},
	}

	tmpl, err := template.New("post.html").Funcs(template.FuncMap{
		"add": add,
		"sub": sub,
	}).ParseFiles("static/html/post.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		log.Printf("Error details: %v", err)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "post.html", data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Printf("Error details: %v", err)
		return
	}
}

func savePostImageToDB(postImage PostImage) error {
	_, err := database.DB.Exec(`
        INSERT INTO PostImage (ID, PostID, UserID, ImagePath, CreatedAt)
        VALUES (?, ?, ?, ?, ?)`,
		postImage.ID, postImage.PostID, postImage.UserID, postImage.ImagePath, postImage.CreatedAt)
	return err
}

// DeletePostHandler handles requests to delete a post
func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated from the context
	ctx := r.Context()
	authenticated, ok := ctx.Value(session.Authenticated).(bool)
	if !ok || !authenticated {
		http.Error(w, "Unauthorized: User not logged in", http.StatusUnauthorized)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract post ID
	postIDStr := r.FormValue("post_id")
	if postIDStr == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID := postIDStr // Use postIDStr directly if it's a UUID

	userID, ok := ctx.Value(session.UserID).(int)
	if !ok {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Call function to handle the post deletion
	err = deletePost(userID, postID)
	if err != nil {
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Redirect back to the posts list or home page
	http.Redirect(w, r, "/post", http.StatusSeeOther)
}

// deletePost deletes a post from the database
func deletePost(userID int, postID string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Check if the user is the owner of the post or has permissions to delete it
	err = checkPostOwnership(tx, userID, postID)
	if err != nil {
		return err
	}

	// Delete likes and comments related to the post
	err = deletePostLikesAndCommentsTx(tx, postID)
	if err != nil {
		log.Printf("Error deleting post likes and comments: %v", err)
		return err
	}

	// Delete the post
	_, err = tx.Exec("DELETE FROM Post WHERE PostID = ?", postID)
	if err != nil {
		log.Printf("Error deleting post: %v", err)
		return err
	}

	return nil
}

// checkPostOwnership checks if the user owns the post or has permissions to delete it
func checkPostOwnership(tx *sql.Tx, userID int, postID string) error {
	var ownerID int
	err := tx.QueryRow("SELECT UserID FROM Post WHERE PostID = ?", postID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Post not found with ID: %s", postID)
			return fmt.Errorf("Post not found")
		}
		log.Printf("Error querying post ownership: %v", err)
		return err
	}

	if ownerID != userID {
		log.Printf("User %d is not authorized to delete post %s owned by %d", userID, postID, ownerID)
		return fmt.Errorf("unauthorized: User does not own the post")
	}

	return nil
}

// deletePostLikesAndCommentsTx deletes likes and comments associated with the post
func deletePostLikesAndCommentsTx(tx *sql.Tx, postID string) error {
	// Delete post likes
	_, err := tx.Exec("DELETE FROM PostLikes WHERE PostID = ?", postID)
	if err != nil {
		log.Printf("Error deleting post likes for post ID: %s", postID)
		return err
	}

	// Delete post comments (corrected table name)
	_, err = tx.Exec("DELETE FROM Comment WHERE PostID = ?", postID)
	if err != nil {
		log.Printf("Error deleting post comments for post ID: %s", postID)
		return err
	}

	return nil
}

func CategoriesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticated, ok := ctx.Value(session.Authenticated).(bool)
	if !ok || !authenticated {
		http.Error(w, "Unauthorized: User not logged in", http.StatusUnauthorized)
		return
	}

	// Fetch the session data (e.g., user ID)
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Unable to retrieve session", http.StatusUnauthorized)
		return
	}
	sessionData, authenticated := session.GetSession(sessionCookie.Value)
	if !authenticated {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Fetch posts created by the user
	myPosts, err := fetchUserPosts(sessionData.UserID)
	if err != nil {
		http.Error(w, "Unable to fetch user posts", http.StatusInternalServerError)
		return
	}

	// Fetch posts liked by the user
	likedPosts, err := fetchLikedPosts(sessionData.UserID)
	if err != nil {
		http.Error(w, "Unable to fetch liked posts", http.StatusInternalServerError)
		return
	}

	// Prepare data for rendering the template
	data := struct {
		Authenticated bool
		Username      string
		MyPosts       []Post
		LikedPosts    []Post
	}{
		Authenticated: sessionData.Authenticated,
		Username:      sessionData.Username,
		MyPosts:       myPosts,
		LikedPosts:    likedPosts,
	}

	// Parse and execute the template
	tmpl, err := template.ParseFiles("static/html/postlist.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to render template", http.StatusInternalServerError)
	}
}

// fetchUserPosts retrieves posts created by the user.
func fetchUserPosts(userID int) ([]Post, error) {
	rows, err := database.DB.Query(`
        SELECT p.PostID, p.Title, p.Content, u.Username
        FROM Post p
        JOIN User u ON p.UserID = u.UserID
        WHERE p.UserID = ?
        ORDER BY p.CreatedAt DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func fetchLikedPosts(userID int) ([]Post, error) {
	rows, err := database.DB.Query(`
        SELECT p.PostID, p.Title, p.Content, u.Username
        FROM Post p
        JOIN PostLikes l ON p.PostID = l.PostID
        JOIN User u ON p.UserID = u.UserID
        WHERE l.UserID = ?
        ORDER BY p.CreatedAt DESC
    `, userID)
	if err != nil {
		log.Printf("Error fetching liked posts for user %d: %v", userID, err) // Add logging
		return nil, err
	}
	defer rows.Close()

	var likedPosts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username)
		if err != nil {
			log.Printf("Error scanning liked post for user %d: %v", userID, err) // Add logging
			return nil, err
		}
		likedPosts = append(likedPosts, post)
	}

	return likedPosts, nil
}
