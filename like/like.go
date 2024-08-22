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

	// No need to convert postID to integer if it's a UUID
	postID := postIDStr // Use postIDStr directly if it's a UUID

	var commentID *int = nil
	if commentIDStr != "" {
		commentIDVal, err := strconv.Atoi(commentIDStr)
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}
		commentID = &commentIDVal
	}

	// Call function to handle the like/dislike action
	err = handleLikeDislike(userID, postID, commentID, isLike)
	if err != nil {
		http.Error(w, "Error processing like/dislike", http.StatusInternalServerError)
		return
	}

	// Redirect back to the post view
	http.Redirect(w, r, "/post/view?id="+postIDStr, http.StatusSeeOther)
}

func handleLikeDislike(userID int, postID string, commentID *int, isLike bool) error {
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

	existingAction, err := getUserPostCommentAction(userID, postID, commentID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error retrieving user action: %v", err)
		return err
	}

	var increment, update bool
	if existingAction == "" {
		// New action
		increment = true
		err = insertLikeDislikeTx(tx, userID, postID, commentID, isLike)
	} else {
		// Existing action
		if (existingAction == "like" && !isLike) || (existingAction == "dislike" && isLike) {
			update = true
			err = updateLikeDislikeTx(tx, userID, postID, commentID, isLike)
		}
	}

	if err != nil {
		log.Printf("Error inserting/updating like/dislike: %v", err)
		return err
	}

	if increment || update {
		err = updatePostCountersTx(tx, postID, isLike, increment)
		if err != nil {
			log.Printf("Error updating post counters: %v", err)
			return err
		}
	}

	return nil
}

func insertLikeDislikeTx(tx *sql.Tx, userID int, postID string, commentID *int, isLike bool) error {
	_, err := tx.Exec("INSERT INTO PostLikes (UserID, PostID, CommentID, IsLike) VALUES (?, ?, ?, ?)", userID, postID, commentID, isLike)
	return err
}

func updateLikeDislikeTx(tx *sql.Tx, userID int, postID string, commentID *int, isLike bool) error {
	var commentIDQuery string
	if commentID != nil {
		commentIDQuery = "= ?"
	} else {
		commentIDQuery = "IS NULL"
	}

	_, err := tx.Exec("UPDATE PostLikes SET IsLike = ? WHERE UserID = ? AND PostID = ? AND CommentID "+commentIDQuery, isLike, userID, postID, commentID)
	return err
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

func getUserPostCommentAction(userID int, postID string, commentID *int) (string, error) {
	var action bool
	var commentIDQuery string
	if commentID != nil {
		commentIDQuery = "= ?"
	} else {
		commentIDQuery = "IS NULL"
	}

	query := "SELECT IsLike FROM PostLikes WHERE UserID = ? AND PostID = ? AND CommentID " + commentIDQuery
	err := database.DB.QueryRow(query, userID, postID, commentID).Scan(&action)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if err == sql.ErrNoRows {
		return "", nil
	}
	if action {
		return "like", nil
	} else {
		return "dislike", nil
	}
}
