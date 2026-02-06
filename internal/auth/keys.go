package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateAPIKey returns (rawKey, hashedKey)
// rawKey is given to the client ONCE. hashedKey is stored in MongoDB.
func GenerateAPIKey() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	key := "gh_" + base64.URLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(key))

	return key, hex.EncodeToString(hash[:]), nil
}

// HashAPIKey hashes a raw API key for comparison with stored hash
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// VerifyAPIKey checks if the provided key's hash matches the database hash
func VerifyAPIKey(providedKey, storedHash string) bool {
	return HashAPIKey(providedKey) == storedHash
}
