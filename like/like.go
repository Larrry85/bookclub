package like

import (
	"database/sql"
	"lions/database"
	"lions/session"
	"log"
	"net/http"
)

// LikeHandler handles like/dislike requests for posts
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
	isLike := r.FormValue("is_like") == "true"
	userID, ok := ctx.Value(session.UserID).(int) // Get userID from session context

	if !ok {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Call function to handle the like/dislike action
	err = handleLikeDislike(userID, postIDStr, isLike)
	if err != nil {
		http.Error(w, "Error processing like/dislike", http.StatusInternalServerError)
		return
	}

	// Redirect back to the post view to show updated like/dislike count
	http.Redirect(w, r, "/post/view?id="+postIDStr, http.StatusSeeOther)
}

func handleLikeDislike(userID int, postID string, isLike bool) error {
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

	// Check if the user has already liked/disliked the post
	existingAction, err := getUserPostAction(userID, postID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error retrieving user action: %v", err)
		return err
	}

	// Handle new action or change of action
	if existingAction == "" {
		// New like/dislike action
		err = insertLikeDislikeTx(tx, userID, postID, isLike)
		if err != nil {
			log.Printf("Error inserting like/dislike: %v", err)
			return err
		}
		// Update post counters (increment)
		err = updatePostCountersTx(tx, postID, isLike, true)
	} else {
		// User is switching from like to dislike or vice versa
		if (existingAction == "like" && !isLike) || (existingAction == "dislike" && isLike) {
			err = updateLikeDislikeTx(tx, userID, postID, isLike)
			if err != nil {
				log.Printf("Error updating like/dislike: %v", err)
				return err
			}
			// Update post counters accordingly
			err = updatePostCountersTx(tx, postID, isLike, true) // Increment the new action
			if err != nil {
				log.Printf("Error incrementing post counter: %v", err)
				return err
			}
			err = updatePostCountersTx(tx, postID, !isLike, false) // Decrement the old action
		}
	}

	return err
}

func insertLikeDislikeTx(tx *sql.Tx, userID int, postID string, isLike bool) error {
	_, err := tx.Exec("INSERT INTO PostLikes (UserID, PostID, IsLike) VALUES (?, ?, ?)", userID, postID, isLike)
	return err
}

func updateLikeDislikeTx(tx *sql.Tx, userID int, postID string, isLike bool) error {
	_, err := tx.Exec("UPDATE PostLikes SET IsLike = ? WHERE UserID = ? AND PostID = ?", isLike, userID, postID)
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

func getUserPostAction(userID int, postID string) (string, error) {
	var action bool
	err := database.DB.QueryRow("SELECT IsLike FROM PostLikes WHERE UserID = ? AND PostID = ?", userID, postID).Scan(&action)
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
