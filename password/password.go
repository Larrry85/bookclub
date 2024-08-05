//password.go
package password

import (
    "crypto/sha256"
    "encoding/base64"
)

// Hashes passwords using SHA256 (consider using a better approach in production)
func HashPassword(password string) string {
    hash := sha256.Sum256([]byte(password))
    return base64.URLEncoding.EncodeToString(hash[:])
}

