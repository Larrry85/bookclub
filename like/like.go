package like

import (
	"lions/session"
	"net/http"
	"sync"
	"text/template"
)

var (
	count int        // Counter
	mu    sync.Mutex // Mutex to handle concurrent requests
)

// Template rendering function
func renderTemplate(w http.ResponseWriter, count int) {
	tmpl := template.Must(template.ParseFiles("static/html/view_post.html"))
	tmpl.Execute(w, count)
}

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

	mu.Lock()
	defer mu.Unlock()

	if r.Method == http.MethodPost {
		count++
	}

	// Render the template with the current count value
	renderTemplate(w, count)

}
