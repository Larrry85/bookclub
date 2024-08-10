package session

import (
	"log"
	"sync"
)

// SessionData holds the session information
type SessionData struct {
	Username      string
	Authenticated bool
}

var (
	sessions = make(map[string]SessionData)
)

var mu sync.Mutex // Ensure that this is defined at a package level

func SetSession(sessionID string, data SessionData) {
	mu.Lock()
	defer mu.Unlock()
	sessions[sessionID] = data
	log.Printf("Session set: %s -> %+v", sessionID, data)
}

func GetSession(sessionID string) (SessionData, bool) {
	mu.Lock()
	defer mu.Unlock()
	data, exists := sessions[sessionID]
	if !exists {
		log.Printf("Session not found for ID: %s", sessionID)
	}
	log.Printf("Session get: %s -> %+v, exists: %v", sessionID, data, exists)
	return data, exists
}

// DeleteSession deletes session data
func DeleteSession(sessionID string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessions, sessionID)
}
