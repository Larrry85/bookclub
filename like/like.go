// like.go
package like

import (
	"net/http"
	"strconv"
	"github.com/gorilla/sessions"
	"lions/database"
	"log"
)

var (
	// Replace with your own secret key for cookie encryption
	key   = []byte("super-secret-key")
	// Create a new session store with the provided secret key
	store = sessions.NewCookieStore(key)
)

// LikePostHandler handles the liking and disliking of posts.
// It processes POST requests to update the like/dislike status of a post for the current user.
func LikePostHandler(w http.ResponseWriter, r *http.Request) {
	// Handle only POST requests
	if r.Method == http.MethodPost {
		// Retrieve the session for the current request
		session, _ := store.Get(r, "session")
		// Get the user ID from the session values
		userID, _ := session.Values["userID"].(int)

		// Retrieve the post ID and like status from the form data
		postID, _ := strconv.Atoi(r.FormValue("post_id"))
		isLike := r.FormValue("is_like") == "true"

		// Insert or update the like/dislike status in the database
		_, err := database.DB.Exec(`
		INSERT INTO PostLikes (UserID, PostID, IsLike) 
		VALUES (?, ?, ?) 
		ON CONFLICT(UserID, PostID) 
		DO UPDATE SET IsLike = ?`, userID, postID, isLike, isLike)
	if err != nil {
		log.Printf("Error updating like/dislike: %v", err) // Log detailed error
		http.Error(w, "Could not update like: "+err.Error(), http.StatusInternalServerError)
		return
	}

		// Redirect the user to the posts page after updating
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
	}
}
