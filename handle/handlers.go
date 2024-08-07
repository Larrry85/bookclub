package handle

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"lions/database"
	"lions/email"
	"lions/post"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var (
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

// Middleware to check session and set user data
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		username, usernameOk := session.Values["username"].(string)
		authenticated, authOk := session.Values["authenticated"].(bool)

		if !usernameOk {
			username = ""
		}
		if !authOk {
			authenticated = false
		}

		ctx := context.WithValue(r.Context(), "Username", username)
		ctx = context.WithValue(ctx, "Authenticated", authenticated)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	username, _ := r.Context().Value("Username").(string)
	authenticated, _ := r.Context().Value("Authenticated").(bool)

	data := map[string]interface{}{
		"Username":      username,
		"Authenticated": authenticated,
	}

	tmpl, err := template.ParseFiles("static/html/mainpage.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ConfirmEmailHandler(w http.ResponseWriter, r *http.Request) {
	emailAddr := r.URL.Query().Get("email")
	if emailAddr == "" {
		http.Error(w, "Email not provided", http.StatusBadRequest)
		return
	}

	// Update the user's status to confirmed in the database
	_, err := database.DB.Exec(`UPDATE User SET Confirmed = 1 WHERE Email = ?`, emailAddr)
	if err != nil {
		log.Println("Error confirming email:", err)
		http.Error(w, "Failed to confirm email", http.StatusInternalServerError)
		return
	}

	log.Printf("Email confirmed: %s", emailAddr)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("static/html/register.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	} else if r.Method == "POST" {
		name := r.FormValue("username")
		emailAddr := r.FormValue("email")
		password := r.FormValue("password")

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = database.DB.Exec(`INSERT INTO User (Username, Email, Password) VALUES (?, ?, ?)`, name, emailAddr, hashedPassword)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send welcome email
		subject := "Welcome to Our Service"
		body := fmt.Sprintf("Hello %s,\n\nWelcome to our literary-lions task!\n\nThank you for registering.\n\nBest regards\nLaura and Jonathan", name)
		err = email.SendEmail(emailAddr, subject, body)
		if err != nil {
			log.Println("Error sending welcome email:", err)
			http.Error(w, "Failed to send welcome email", http.StatusInternalServerError)
			return
		}

		log.Printf("User registered: username=%s, email=%s", name, emailAddr)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func PostsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		post.CreatePost(w, r)
	case http.MethodGet:
		post.ListPosts(w, r)
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")

		log.Printf("Login attempt with email: %s", email)

		var dbPassword, username string
		err := database.DB.QueryRow(`SELECT Password, Username FROM User WHERE Email = ?`, email).Scan(&dbPassword, &username)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Email not found: %s", email)
				http.Error(w, "Invalid email or password", http.StatusUnauthorized)
				return
			}
			log.Println("Database error:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Fetched hashed password for email: %s", email)

		// Compare the hashed password with the plain text password
		err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
		if err != nil {
			log.Printf("Invalid password for email: %s", email)
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		session, _ := store.Get(r, "session")
		session.Values["authenticated"] = true
		session.Values["username"] = username
		session.Save(r, w)

		http.Redirect(w, r, "/mainpage", http.StatusSeeOther)
		return
	} else {
		tmpl, err := template.ParseFiles("static/html/login.html")
		if err != nil {
			log.Println("Template parsing error:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["authenticated"] = false
	session.Values["username"] = nil
	session.Save(r, w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
