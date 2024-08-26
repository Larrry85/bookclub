package comment

import (
	"database/sql"
	"lions/database"
	"lions/session"
	"log"
	"net/http"
)

// CommentLikeHandler handles like/dislike requests for comments
func CommentLikeHandler(w http.ResponseWriter, r *http.Request) {
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
	commentIDStr := r.FormValue("comment_id")
	isLike := r.FormValue("is_like") == "true"
	userID, ok := ctx.Value(session.UserID).(int) // Get userID from session context

	if !ok {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Call function to handle the like/dislike action
	err = handleCommentLikeDislike(userID, commentIDStr, isLike)
	if err != nil {
		http.Error(w, "Error processing like/dislike", http.StatusInternalServerError)
		return
	}

	// Redirect back to the post view to show updated like/dislike count
	http.Redirect(w, r, "/post/view?id="+r.FormValue("post_id"), http.StatusSeeOther)
}

func handleCommentLikeDislike(userID int, commentIDStr string, isLike bool) error {
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

	// Check if the user has already liked/disliked the comment
	existingAction, err := getUserCommentAction(userID, commentIDStr)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error retrieving user action: %v", err)
		return err
	}

	// Handle new action or change of action
	if existingAction == "" {
		// New like/dislike action
		err = insertCommentLikeDislikeTx(tx, userID, commentIDStr, isLike)
		if err != nil {
			log.Printf("Error inserting like/dislike: %v", err)
			return err
		}
		// Update comment counters (increment)
		err = updateCommentCountersTx(tx, commentIDStr, isLike, true)
	} else {
		// User is switching from like to dislike or vice versa
		if (existingAction == "like" && !isLike) || (existingAction == "dislike" && isLike) {
			err = updateCommentLikeDislikeTx(tx, userID, commentIDStr, isLike)
			if err != nil {
				log.Printf("Error updating like/dislike: %v", err)
				return err
			}
			// Update comment counters accordingly
			err = updateCommentCountersTx(tx, commentIDStr, isLike, true) // Increment the new action
			if err != nil {
				log.Printf("Error incrementing comment counter: %v", err)
				return err
			}
			err = updateCommentCountersTx(tx, commentIDStr, !isLike, false) // Decrement the old action
		}
	}

	return err
}

func insertCommentLikeDislikeTx(tx *sql.Tx, userID int, commentID string, isLike bool) error {
	_, err := tx.Exec("INSERT INTO CommentLikes (UserID, CommentID, IsLike) VALUES (?, ?, ?)", userID, commentID, isLike)
	return err
}

func updateCommentLikeDislikeTx(tx *sql.Tx, userID int, commentID string, isLike bool) error {
	_, err := tx.Exec("UPDATE CommentLikes SET IsLike = ? WHERE UserID = ? AND CommentID = ?", isLike, userID, commentID)
	return err
}

func updateCommentCountersTx(tx *sql.Tx, commentID string, isLike bool, increment bool) error {
	var operation string
	if increment {
		if isLike {
			operation = "CommentLikesCount = CommentLikesCount + 1"
		} else {
			operation = "CommentDislikesCount = CommentDislikesCount + 1"
		}
	} else {
		if isLike {
			operation = "CommentLikesCount = CommentLikesCount - 1"
		} else {
			operation = "CommentDislikesCount = CommentDislikesCount - 1"
		}
	}

	_, err := tx.Exec("UPDATE Comment SET "+operation+" WHERE CommentID = ?", commentID)
	return err
}

func getUserCommentAction(userID int, commentID string) (string, error) {
	var action bool
	err := database.DB.QueryRow("SELECT IsLike FROM CommentLikes WHERE UserID = ? AND CommentID = ?", userID, commentID).Scan(&action)
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
