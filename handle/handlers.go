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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var (
	sessions = make(map[string]SessionData) // SessionID -> SessionData
)

type SessionData struct {
	Username      string
	Authenticated bool
}

type contextKey string

const (
	UsernameKey      = contextKey("Username")
	AuthenticatedKey = contextKey("Authenticated")
)

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			if err == http.ErrNoCookie {
				// If no session cookie, create a new one
				cookie = &http.Cookie{
					Name:     "session_id",
					Value:    uuid.New().String(),
					HttpOnly: true,
				}
				http.SetCookie(w, cookie)
			} else {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
		}

		sessionID := cookie.Value
		sessionData, ok := sessions[sessionID]
		if !ok {
			sessionData = SessionData{}
		}

		ctx := context.WithValue(r.Context(), UsernameKey, sessionData.Username)
		ctx = context.WithValue(ctx, AuthenticatedKey, sessionData.Authenticated)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	username, _ := r.Context().Value(UsernameKey).(string)
	authenticated, _ := r.Context().Value(AuthenticatedKey).(bool)

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
			var errorMessage string
			if sqliteErr, ok := err.(sqlite3.Error); ok {
				if sqliteErr.Code == sqlite3.ErrConstraint {
					if strings.Contains(sqliteErr.Error(), "User.Username") {
						errorMessage = "The username is already taken."
					} else if strings.Contains(sqliteErr.Error(), "User.Email") {
						errorMessage = "The email is already registered."
					}
				}
			}
			renderRegister(w, errorMessage)
			return
		}

		// Send welcome email
		subject := "Welcome to Literary Lions Forum!"
		body := fmt.Sprintf("Hello %s,\n\nThank you for registering at Literary Lions Forum!\n\nBest regards,\nThe Literary Lions Team", name)
		err = email.SendEmail(emailAddr, subject, body)
		if err != nil {
			log.Printf("Failed to send email: %v", err)
			http.Error(w, "Failed to send email", http.StatusInternalServerError)
			return
		}

		log.Printf("User registered: username=%s, email=%s", name, emailAddr)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func renderRegister(w http.ResponseWriter, errorMessage string) {
	tmpl, err := template.ParseFiles("static/html/register.html")
	if err != nil {
		log.Println("Template parsing error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"ErrorMessage": errorMessage,
	}

	tmpl.Execute(w, data)
}

func PostsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		post.ListPosts(w, r)
	case http.MethodPost:
		post.CreatePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
				renderLogin(w, "Invalid email or password")
				return
			}
			log.Println("Database error:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Fetched hashed password for email: %s", email)

		err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
		if err != nil {
			log.Printf("Invalid password for email: %s", email)
			renderLogin(w, "Invalid email or password")
			return
		}

		sessionID := uuid.New().String()
		sessions[sessionID] = SessionData{Username: username, Authenticated: true}
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sessionID})

		http.Redirect(w, r, "/mainpage", http.StatusSeeOther)
		return
	} else {
		renderLogin(w, "")
	}
}

func renderLogin(w http.ResponseWriter, errorMessage string) {
	tmpl, err := template.ParseFiles("static/html/login.html")
	if err != nil {
		log.Println("Template parsing error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"ErrorMessage": errorMessage,
	}

	tmpl.Execute(w, data)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Clear session data and delete cookie
		delete(sessions, cookie.Value)
		http.SetCookie(w, &http.Cookie{
			Name:   "session_id",
			Value:  "",
			MaxAge: -1,
			Path:   "/",
		})
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, err := r.Cookie("session_id")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionData, ok := sessions[sessionID.Value]
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := sessionData.Username

	var email, hashedPassword string
	var userID int

	// Fetch user details
	err = database.DB.QueryRow(`
        SELECT UserID, Email, Password 
        FROM User 
        WHERE Username = ?`, username).Scan(&userID, &email, &hashedPassword)
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

	showPassword := r.URL.Query().Get("show") == "true"
	var password string
	if showPassword {
		password = "ActualPlainTextPassword" // Replace with actual password retrieval if needed.
	}

	data := map[string]interface{}{
		"Username":     username,
		"Email":        email,
		"NumPosts":     numPosts,
		"NumComments":  numComments,
		"ShowPassword": showPassword,
		"Password":     password,
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

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, err := r.Cookie("session_id")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionData, ok := sessions[sessionID.Value]
	if !ok || !sessionData.Authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := sessionData.Username
	_, err = database.DB.Exec(`DELETE FROM User WHERE Username = ?`, username)
	if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	delete(sessions, sessionID.Value)
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func PasswordResetRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		emailAddr := r.FormValue("email")

		// Check if the email exists in the database
		var username string
		err := database.DB.QueryRow(`SELECT Username FROM User WHERE Email = ?`, emailAddr).Scan(&username)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Email not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Generate password reset token
		token := uuid.New().String()
		expiry := time.Now().Add(1 * time.Hour)

		_, err = database.DB.Exec(`INSERT INTO PasswordReset (Email, Token, Expiry) VALUES (?, ?, ?)`, emailAddr, token, expiry)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Send reset email
		resetURL := fmt.Sprintf("http://localhost:8080/reset-password?token=%s", token)
		subject := "Password Reset Request"
		body := fmt.Sprintf("Hello %s,\n\nTo reset your password, click the following link: %s\n\nIf you did not request this, please ignore this email.\n\nBest regards,\nThe Literary Lions Team", username, resetURL)
		err = email.SendEmail(emailAddr, subject, body)
		if err != nil {
			http.Error(w, "Failed to send email", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		tmpl, err := template.ParseFiles("static/html/password-reset-request.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	}
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		token := r.FormValue("token")
		newPassword := r.FormValue("password")

		var emailAddr string
		var expiry time.Time

		err := database.DB.QueryRow(`SELECT Email, Expiry FROM PasswordReset WHERE Token = ?`, token).Scan(&emailAddr, &expiry)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Invalid or expired token", http.StatusBadRequest)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if time.Now().After(expiry) {
			http.Error(w, "Token has expired", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = database.DB.Exec(`UPDATE User SET Password = ? WHERE Email = ?`, hashedPassword, emailAddr)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		_, err = database.DB.Exec(`DELETE FROM PasswordReset WHERE Token = ?`, token)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Token is required", http.StatusBadRequest)
			return
		}

		tmpl, err := template.ParseFiles("static/html/reset-password.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"Token": token,
		}

		tmpl.Execute(w, data)
	}
}
