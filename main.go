package main

import (
	"lions/database"
	"lions/handle"
	"lions/like"
	"lions/post"
	"lions/session"
	"log"
	"net/http"
)

func main() {
	// Initialize the database connection.
	database.Init()

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))

	// Apply session middleware to the uploads route
	http.Handle("/uploads/", session.SessionMiddleware(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads")))))

	// Apply session middleware to all routes
	http.Handle("/", session.SessionMiddleware(http.HandlerFunc(handle.MainPageHandler)))
	http.Handle("/register", session.SessionMiddleware(http.HandlerFunc(handle.RegisterHandler)))
	http.Handle("/login", session.SessionMiddleware(http.HandlerFunc(handle.LoginHandler)))
	http.Handle("/logout", session.SessionMiddleware(http.HandlerFunc(handle.LogoutHandler)))
	http.Handle("/profile", session.SessionMiddleware(http.HandlerFunc(handle.ProfileHandler)))

	http.Handle("/post/create", session.SessionMiddleware(http.HandlerFunc(post.CreatePost)))
	http.Handle("/post/view", session.SessionMiddleware(http.HandlerFunc(post.ViewPost)))

    http.HandleFunc("/post/confirm_delete", handle.ConfirmDeleteHandler)

	http.Handle("/post", session.SessionMiddleware(http.HandlerFunc(post.ListPosts)))
	http.Handle("/post/reply", session.SessionMiddleware(http.HandlerFunc(post.AddReply)))
	http.Handle("/post/delete", session.SessionMiddleware(http.HandlerFunc(post.DeletePostHandler)))
	http.Handle("/like", session.SessionMiddleware(http.HandlerFunc(like.LikeHandler)))

	// Define routes that do not use session middleware
	http.HandleFunc("/confirm", handle.ConfirmEmailHandler)
	http.HandleFunc("/password-reset-request", handle.PasswordResetRequestHandler)
	http.HandleFunc("/reset-password", handle.ResetPasswordHandler)
	http.HandleFunc("/delete-account", handle.DeleteAccountHandler)

	http.Handle("/filter", session.SessionMiddleware(http.HandlerFunc(post.FilterPostHandler)))

	// Start the HTTP server
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
