// main.go
package main

import (
	"lions/database"
	"lions/handle"
	"log"
	"net/http"
		"html/template"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	// Initialize the database connection
	database.Init()

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Route handlers
	http.HandleFunc("/", handle.MainPageHandler)
	http.HandleFunc("/register", handle.RegisterHandler)
	http.HandleFunc("/login", handle.LoginHandler)
	http.HandleFunc("/logout", handle.LogoutHandler)
	http.HandleFunc("/posts", handle.PostsHandler)
	http.HandleFunc("/mainpage", handle.PostsHandler)

	// Serve static HTML files
	http.HandleFunc("/post", servePage("static/html/post.html"))
	http.HandleFunc("/general", servePage("static/html/general.html"))
	http.HandleFunc("/genres", servePage("static/html/genres.html"))
	http.HandleFunc("/book_specific", servePage("static/html/book_specific.html"))


	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// servePage returns a handler function that serves the static HTML file at the given path
func servePage(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
