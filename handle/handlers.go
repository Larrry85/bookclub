package handle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"html/template"
	"lions/database"
	"lions/email"
	"lions/post"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var (
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

var (
	DB *sql.DB
)

type contextKey string

const (
	UsernameKey      = contextKey("Username")
	AuthenticatedKey = contextKey("Authenticated")
)

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

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)

	var email string
	var userID int

	// Fetch user details
	err := database.DB.QueryRow(`
        SELECT UserID, Email 
        FROM User 
        WHERE Username = ?`, username).Scan(&userID, &email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Println("Database error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get number of posts and comments
	numPosts, numComments, err := database.GetUserStats(userID)
	if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Username":    username,
		"Email":       email,
		"NumPosts":    numPosts,
		"NumComments": numComments,
	}

	tmpl, err := template.ParseFiles("static/html/profile.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func GenerateResetToken(userID int) (string, error) {
	token := make([]byte, 32) // Generate a 32-byte token
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}
	tokenStr := fmt.Sprintf("%x", token)

	expiration := time.Now().Add(1 * time.Hour) // Token valid for 1 hour

	_, err = database.DB.Exec("INSERT INTO PasswordResetToken (UserID, Token, Expiration) VALUES (?, ?, ?)", userID, tokenStr, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return tokenStr, nil
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Parse the form
		r.ParseForm()
		token := r.FormValue("token")
		newPassword := r.FormValue("password")

		// Validate the token and get the associated user
		var userID int
		var expiration time.Time
		err := database.DB.QueryRow("SELECT UserID, Expiration FROM PasswordResetToken WHERE Token = ?", token).Scan(&userID, &expiration)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Invalid or expired token", http.StatusBadRequest)
				return
			}
			log.Printf("Error fetching token: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check if the token is expired
		if time.Now().After(expiration) {
			http.Error(w, "Token expired", http.StatusBadRequest)
			return
		}

		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing new password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Update the user's password
		_, err = database.DB.Exec("UPDATE User SET Password = ? WHERE UserID = ?", hashedPassword, userID)
		if err != nil {
			log.Printf("Error updating password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Delete the token after successful password reset
		_, err = database.DB.Exec("DELETE FROM PasswordResetToken WHERE Token = ?", token)
		if err != nil {
			log.Printf("Error deleting password reset token: %v", err)
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		// Render the reset password page if method is GET
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		tmpl, err := template.ParseFiles("static/html/reset_password.html")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]interface{}{
			"Token": token,
		})
	}
}

func PasswordResetRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sent := r.URL.Query().Get("sent") == "true"

		tmpl, err := template.ParseFiles("static/html/password_reset_request.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, map[string]interface{}{
			"Sent": sent,
		})
	} else if r.Method == http.MethodPost {
		emailAddr := r.FormValue("email")

		// Verify the email exists in the database
		var userID int
		err := database.DB.QueryRow("SELECT UserID FROM User WHERE Email = ?", emailAddr).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "No user found with that email address", http.StatusBadRequest)
				return
			}
			log.Println("Database error:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Generate a reset token
		token, err := GenerateResetToken(userID)
		if err != nil {
			http.Error(w, "Failed to generate reset token", http.StatusInternalServerError)
			return
		}

		// Send reset email
		resetURL := fmt.Sprintf("http://%s/reset-password?token=%s", r.Host, token)
		subject := "Password Reset Request"
		body := fmt.Sprintf("Click the following link to reset your password: %s", resetURL)

		err = email.SendEmail(emailAddr, subject, body)
		if err != nil {
			http.Error(w, "Failed to send email", http.StatusInternalServerError)
			return
		}

		// Redirect to the same page with the "sent" query parameter
		http.Redirect(w, r, "/password-reset-request?sent=true", http.StatusSeeOther)
	}
}
