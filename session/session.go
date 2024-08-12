package session

import (
	"context"
	"net/http"
	"sync"
)

var sessionStore = struct {
	sync.RWMutex
	m map[string]SessionData
}{m: make(map[string]SessionData)}

type contextKey string

const (
	Username      = contextKey("Username")
	Authenticated = contextKey("Authenticated")
)

type SessionData struct {
	Username      string
	Authenticated bool
	UserID        int
}

func GetSession(sessionID string) (SessionData, bool) {
	sessionStore.RLock()
	defer sessionStore.RUnlock()
	data, exists := sessionStore.m[sessionID]
	return data, exists
}

func SetSession(sessionID string, data SessionData) {
	sessionStore.Lock()
	defer sessionStore.Unlock()
	sessionStore.m[sessionID] = data
}

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var authenticated bool
		var sessionData SessionData

		// Try to retrieve the session cookie
		sessionID, err := r.Cookie("session_id")
		if err == nil {
			// Retrieve session data if cookie is present
			sessionData, authenticated = GetSession(sessionID.Value)
		} else {
			// If there's an error retrieving the cookie, set authenticated to false
			authenticated = false
		}

		// Add session data and authentication status to the request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, Username, sessionData.Username)
		ctx = context.WithValue(ctx, Authenticated, authenticated)
		r = r.WithContext(ctx)

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}
