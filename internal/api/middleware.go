package api

import (
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/akua/bsv-broadcaster/internal/admin"
	"github.com/akua/bsv-broadcaster/internal/auth"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates API key and adaptively enforces ECDSA signature based on client tier
func AuthMiddleware(db *database.Database, clientMgr *admin.ClientManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract API key
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing X-API-Key header",
			})
		}

		// Get client by API key
		client, err := clientMgr.GetClientByAPIKey(c.Context(), apiKey)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API key",
			})
		}

		// Check if client is active
		if !client.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Client account is disabled",
			})
		}

		// ADAPTIVE TIER LOGIC
		// Pilot Tier: API Key only (optional IP whitelist)
		if !client.RequireSignature {
			log.Printf("[PILOT] Legacy request from: %s (tier: %s)", client.Name, client.Tier)

			// Optional IP whitelist for pilot tier
			if len(client.AllowedIPs) > 0 && !isIPWhitelisted(c.IP(), client.AllowedIPs) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "IP not whitelisted for pilot tier",
					"tier":  client.Tier,
				})
			}

			// Check rate limit
			if client.TxCount >= client.MaxDailyTx {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "Daily transaction limit exceeded",
				})
			}

			// Increment transaction count
			if err := clientMgr.IncrementClientTxCount(c.Context(), client.ID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to update transaction count",
				})
			}

			// Store client in context and proceed
			c.Locals("client", client)
			return c.Next()
		}

		// Secure/Government Tier: Enforce ECDSA Signature
		signature := c.Get("X-Signature")
		timestamp := c.Get("X-Timestamp")
		nonce := c.Get("X-Nonce")

		if signature == "" || timestamp == "" || nonce == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "ECDSA signature headers (X-Signature, X-Timestamp, X-Nonce) are required for this tier",
				"tier":  client.Tier,
			})
		}

		// Parse request body
		var payload struct {
			Data string `json:"data"`
		}
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Verify signature with grace period support
		isValid, err := verifyAdaptiveSignature(client, payload.Data, signature, timestamp, nonce)
		if !isValid || err != nil {
			log.Printf("❌ Signature verification failed for client: %s (tier: %s)", client.Name, client.Tier)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid cryptographic signature",
			})
		}

		// Check rate limit
		if client.TxCount >= client.MaxDailyTx {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Daily transaction limit exceeded",
			})
		}

		// Increment transaction count
		if err := clientMgr.IncrementClientTxCount(c.Context(), client.ID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update transaction count",
			})
		}

		// Store client in context for downstream handlers
		c.Locals("client", client)

		return c.Next()
	}
}

// verifyAdaptiveSignature verifies ECDSA signature with grace period support for key rotation
func verifyAdaptiveSignature(client *models.Client, data, signature, timestamp, nonce string) (bool, error) {
	// Construct signature payload: timestamp + nonce + data
	payload := timestamp + nonce + data

	// Try current public key first
	if client.PublicKey != "" {
		valid, err := auth.VerifySignature(client.PublicKey, payload, signature)
		if err == nil && valid {
			return true, nil
		}
	}

	// If current key fails, check old key if within grace period
	if client.OldPublicKey != "" && client.KeyRotatedAt != nil {
		gracePeriod := client.GracePeriodHours
		if gracePeriod == 0 {
			gracePeriod = 24 // Default
		}

		expiresAt := client.KeyRotatedAt.Add(time.Duration(gracePeriod) * time.Hour)
		if time.Now().Before(expiresAt) {
			log.Printf("🔄 Trying old public key (grace period active until %s)", expiresAt.Format(time.RFC3339))
			valid, err := auth.VerifySignature(client.OldPublicKey, payload, signature)
			if err == nil && valid {
				log.Printf("✅ Signature verified with old public key during grace period")
				return true, nil
			}
		}
	}

	return false, nil
}

// isIPWhitelisted checks if the client IP is in the allowed list
func isIPWhitelisted(clientIP string, allowedIPs []string) bool {
	// Extract actual IP (remove port if present)
	ip := clientIP
	if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
		ip = clientIP[:idx]
	}

	for _, allowedIP := range allowedIPs {
		if ip == allowedIP || clientIP == allowedIP {
			return true
		}
	}
	return false
}

// AdminAuthMiddleware validates admin password
func AdminAuthMiddleware(adminPassword string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		password := c.Get("X-Admin-Password")
		if password == "" || password != adminPassword {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or missing admin password",
			})
		}
		return c.Next()
	}
}

// DecodeHexData extracts the data field from requests and decodes it
func DecodeHexData(c *fiber.Ctx) ([]byte, error) {
	var payload struct {
		Data string `json:"data"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return nil, err
	}

	data, err := hex.DecodeString(payload.Data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
