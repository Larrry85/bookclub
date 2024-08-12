package main

import (
	"lions/database"
	"lions/handle"
	"lions/post"
	"lions/session"
	"log"
	"net/http"
)

func main() {

	// Initialize the database connection
	database.Init()

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Apply session middleware to all routes
	http.Handle("/", session.SessionMiddleware(http.HandlerFunc(handle.MainPageHandler)))
	http.Handle("/register", session.SessionMiddleware(http.HandlerFunc(handle.RegisterHandler)))
	http.Handle("/login", session.SessionMiddleware(http.HandlerFunc(handle.LoginHandler)))
	http.Handle("/logout", session.SessionMiddleware(http.HandlerFunc(handle.LogoutHandler)))
	http.Handle("/post/create", session.SessionMiddleware(http.HandlerFunc(post.CreatePost)))
	http.Handle("/post/view", session.SessionMiddleware(http.HandlerFunc(post.ViewPost)))

	http.HandleFunc("/confirm", handle.ConfirmEmailHandler)
	http.Handle("/post", session.SessionMiddleware(http.HandlerFunc(post.ListPosts)))
	http.Handle("/post/reply", session.SessionMiddleware(http.HandlerFunc(post.AddReply)))
	http.HandleFunc("/password-reset-request", handle.PasswordResetRequestHandler)
	http.HandleFunc("/reset-password", handle.ResetPasswordHandler)
	http.HandleFunc("/delete-account", handle.DeleteAccountHandler)
	http.Handle("/profile", session.SessionMiddleware(http.HandlerFunc(handle.ProfileHandler)))

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/*/ servePage returns a handler function that serves the static HTML file at the given path
func servePage(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		authenticated := session.Values["authenticated"] == true
		username, _ := session.Values["username"].(string)

		data := map[string]interface{}{
			"Username":      username,
			"Authenticated": authenticated,
		}

		tmpl, err := template.ParseFiles(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}*/
