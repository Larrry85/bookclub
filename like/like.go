//like.go
package like

import (
	"net/http"
	"strconv"
	"github.com/gorilla/sessions"
	"lions/database"
)

var (
	// Replace with your own secret key
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func LikePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, _ := store.Get(r, "session")
		userID, _ := session.Values["userID"].(int)

		postID, _ := strconv.Atoi(r.FormValue("post_id"))
		isLike := r.FormValue("is_like") == "true"

		_, err := database.DB.Exec(`INSERT INTO Like (UserID, PostID, IsLike) VALUES (?, ?, ?) ON CONFLICT(UserID, PostID) DO UPDATE SET IsLike = ?`, userID, postID, isLike, isLike)
		if err != nil {
			http.Error(w, "Could not update like", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
	}
}

