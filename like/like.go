package like

import (
	"database/sql"
	"lions/database"
	"lions/session"
	"log"
	"net/http"
	"strconv"
)

// LikeHandler handles like/dislike requests for posts or comments
func LikeHandler(w http.ResponseWriter, r *http.Request) {
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

	// Extract form values
	postIDStr := r.FormValue("post_id")
	commentIDStr := r.FormValue("comment_id") // Optional for comments
	isLike := r.FormValue("is_like") == "true"
	userID, ok := ctx.Value(session.UserID).(int) // Get userID from session context

	if !ok {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// If commentID is present, handle comment like, else handle post like
	if commentIDStr == "" {
		// This is a like/dislike on a post
		err = handleLikeDislikePost(userID, postIDStr, isLike)
	} else {
		// This is a like/dislike on a comment
		commentID, err := strconv.Atoi(commentIDStr)
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}
		err = handleLikeDislikeComment(userID, commentID, isLike)
	}

	if err != nil {
		http.Error(w, "Error processing like/dislike", http.StatusInternalServerError)
		return
	}

	// Redirect back to the post view
	http.Redirect(w, r, "/post/view?id="+postIDStr, http.StatusSeeOther)
}

func handleLikeDislikePost(userID int, postID string, isLike bool) error {
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

	// Check if there's an existing action
	existingAction, err := getUserPostAction(tx, userID, postID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error retrieving user action: %v", err)
		return err
	}

	var increment, update bool
	if existingAction == "" {
		increment = true
		err = insertLikeDislikePostTx(tx, userID, postID, isLike)
	} else if (existingAction == "like" && !isLike) || (existingAction == "dislike" && isLike) {
		update = true
		err = updateLikeDislikePostTx(tx, userID, postID, isLike)
	}

	if err != nil {
		log.Printf("Error inserting/updating like/dislike: %v", err)
		return err
	}

	// Update post counters if necessary
	if increment || update {
		err = updatePostCountersTx(tx, postID, isLike, increment)
		if err != nil {
			log.Printf("Error updating post counters: %v", err)
			return err
		}
	}

	return nil
}

func insertLikeDislikePostTx(tx *sql.Tx, userID int, postID string, isLike bool) error {
	_, err := tx.Exec("INSERT INTO PostLikes (UserID, PostID, IsLike) VALUES (?, ?, ?)", userID, postID, isLike)
	return err
}

func updateLikeDislikePostTx(tx *sql.Tx, userID int, postID string, isLike bool) error {
	_, err := tx.Exec("UPDATE PostLikes SET IsLike = ? WHERE UserID = ? AND PostID = ?", isLike, userID, postID)
	return err
}

func getUserPostAction(tx *sql.Tx, userID int, postID string) (string, error) {
	var action bool
	err := tx.QueryRow("SELECT IsLike FROM PostLikes WHERE UserID = ? AND PostID = ?", userID, postID).Scan(&action)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if err == sql.ErrNoRows {
		return "", nil
	}
	if action {
		return "like", nil
	}
	return "dislike", nil
}

func updatePostCountersTx(tx *sql.Tx, postID string, isLike bool, increment bool) error {
	var operation string
	if increment {
		if isLike {
			operation = "LikesCount = LikesCount + 1"
		} else {
			operation = "DislikesCount = DislikesCount + 1"
		}
	} else {
		if isLike {
			operation = "LikesCount = LikesCount - 1"
		} else {
			operation = "DislikesCount = DislikesCount - 1"
		}
	}

	_, err := tx.Exec("UPDATE Post SET "+operation+" WHERE PostID = ?", postID)
	return err
}

func handleLikeDislikeComment(userID int, commentID int, isLike bool) error {
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

	// Check if there's an existing action
	existingAction, err := getUserCommentAction(tx, userID, commentID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error retrieving user action: %v", err)
		return err
	}

	// Insert or update the like/dislike
	if existingAction == "" {
		// Insert new like/dislike
		err = insertLikeDislikeCommentTx(tx, userID, commentID, isLike)
	} else if (existingAction == "like" && !isLike) || (existingAction == "dislike" && isLike) {
		// Update existing like/dislike
		err = updateLikeDislikeCommentTx(tx, userID, commentID, isLike)
	}

	if err != nil {
		log.Printf("Error inserting/updating like/dislike: %v", err)
		return err
	}

	return nil
}

func insertLikeDislikeCommentTx(tx *sql.Tx, userID int, commentID int, isLike bool) error {
	_, err := tx.Exec("INSERT INTO CommentLikes (UserID, CommentID, IsLike) VALUES (?, ?, ?)", userID, commentID, isLike)
	return err
}

func updateLikeDislikeCommentTx(tx *sql.Tx, userID int, commentID int, isLike bool) error {
	_, err := tx.Exec("UPDATE CommentLikes SET IsLike = ? WHERE UserID = ? AND CommentID = ?", isLike, userID, commentID)
	return err
}

func getUserCommentAction(tx *sql.Tx, userID int, commentID int) (string, error) {
	var action bool
	err := tx.QueryRow("SELECT IsLike FROM CommentLikes WHERE UserID = ? AND CommentID = ?", userID, commentID).Scan(&action)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if err == sql.ErrNoRows {
		return "", nil
	}
	if action {
		return "like", nil
	}
	return "dislike", nil
}
