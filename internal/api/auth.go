package api

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"github.com/akua/bsv-broadcaster/internal/auth"
	"github.com/gofiber/fiber/v2"
)

// RegisterPublicKeyRequest represents a self-service public key registration
type RegisterPublicKeyRequest struct {
	PublicKey string `json:"public_key"`
}

// RotatePublicKeyRequest represents a key rotation request
type RotatePublicKeyRequest struct {
	NewPublicKey string `json:"new_public_key"`
}

// HandleRegisterPublicKey allows a client to bind their ECDSA public key
func (s *Server) HandleRegisterPublicKey(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "X-API-Key header required",
		})
	}

	// Hash the API key to look up client
	apiKeyHash := hashAPIKey(apiKey)

	var req RegisterPublicKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.PublicKey == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "public_key is required",
		})
	}

	// Validate public key format (should be 65 bytes hex = 130 chars)
	if len(req.PublicKey) != 130 {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid public key format (expected 130 hex characters)",
		})
	}

	// Attempt to bind the public key to this API key
	err := s.db.BindPublicKeyToClient(c.Context(), apiKeyHash, req.PublicKey)
	if err != nil {
		log.Printf("❌ Public key registration failed: %v", err)
		return c.Status(401).JSON(fiber.Map{
			"error": "unauthorized or public key already registered",
		})
	}

	log.Printf("✅ Public key registered for client with API key hash: %s", apiKeyHash[:8])

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Public key registered successfully. Signature verification is now required for all requests.",
	})
}

// HandleRotatePublicKey allows a client to rotate their ECDSA key with grace period
func (s *Server) HandleRotatePublicKey(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "X-API-Key header required",
		})
	}

	// Hash the API key to look up client
	apiKeyHash := hashAPIKey(apiKey)

	// Get client to verify current signature
	client, err := s.db.GetClientByAPIKey(c.Context(), apiKeyHash)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// Verify the request is signed with the CURRENT public key
	signature := c.Get("X-Signature")
	timestamp := c.Get("X-Timestamp")
	nonce := c.Get("X-Nonce")

	if signature == "" || timestamp == "" || nonce == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "signature headers required for key rotation (X-Signature, X-Timestamp, X-Nonce)",
		})
	}

	// Get request body for signature verification
	var req RotatePublicKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.NewPublicKey == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "new_public_key is required",
		})
	}

	// Validate new public key format
	if len(req.NewPublicKey) != 130 {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid public key format (expected 130 hex characters)",
		})
	}

	// Verify signature with CURRENT public key
	payload := timestamp + nonce + req.NewPublicKey
	valid, err := auth.VerifySignature(client.PublicKey, payload, signature)
	if err != nil || !valid {
		log.Printf("❌ Key rotation failed - invalid signature from client: %s", client.Name)
		return c.Status(401).JSON(fiber.Map{
			"error": "invalid signature - must be signed with current public key",
		})
	}

	// Perform key rotation
	err = s.db.RotateClientPublicKey(c.Context(), apiKeyHash, req.NewPublicKey)
	if err != nil {
		log.Printf("❌ Key rotation failed: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to rotate public key",
		})
	}

	gracePeriod := client.GracePeriodHours
	if gracePeriod == 0 {
		gracePeriod = 24 // Default
	}

	log.Printf("🔄 Public key rotated for client: %s (grace period: %dh)", client.Name, gracePeriod)

	return c.JSON(fiber.Map{
		"success":             true,
		"message":             "Public key rotated successfully",
		"grace_period_hours":  gracePeriod,
		"old_key_valid_until": time.Now().Add(time.Duration(gracePeriod) * time.Hour),
	})
}

// HandleKeyStatus returns the current key status for a client
func (s *Server) HandleKeyStatus(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "X-API-Key header required",
		})
	}

	// Hash the API key to look up client
	apiKeyHash := hashAPIKey(apiKey)

	client, err := s.db.GetClientByAPIKey(c.Context(), apiKeyHash)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// Calculate public key hash for audit purposes (don't expose full key)
	var publicKeyHash string
	if client.PublicKey != "" {
		hash := sha256.Sum256([]byte(client.PublicKey))
		publicKeyHash = hex.EncodeToString(hash[:])
	}

	// Check if grace period is active
	gracePeriodActive := false
	var gracePeriodEndsAt *time.Time
	if client.KeyRotatedAt != nil && client.OldPublicKey != "" {
		gracePeriod := client.GracePeriodHours
		if gracePeriod == 0 {
			gracePeriod = 24
		}
		expiresAt := client.KeyRotatedAt.Add(time.Duration(gracePeriod) * time.Hour)
		if time.Now().Before(expiresAt) {
			gracePeriodActive = true
			gracePeriodEndsAt = &expiresAt
		}
	}

	return c.JSON(fiber.Map{
		"client_name":          client.Name,
		"tier":                 client.Tier,
		"has_public_key":       client.PublicKey != "",
		"require_signature":    client.RequireSignature,
		"public_key_hash":      publicKeyHash,
		"grace_period_active":  gracePeriodActive,
		"grace_period_ends_at": gracePeriodEndsAt,
		"key_rotated_at":       client.KeyRotatedAt,
	})
}

// hashAPIKey creates a SHA-256 hash of the API key for database lookup
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
