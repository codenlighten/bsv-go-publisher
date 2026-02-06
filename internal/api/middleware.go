package api

import (
	"encoding/hex"

	"github.com/akua/bsv-broadcaster/internal/admin"
	"github.com/akua/bsv-broadcaster/internal/auth"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates API key and ECDSA signature
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

		// Extract and verify signature
		signature := c.Get("X-Signature")
		if signature == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing X-Signature header",
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

		// Verify ECDSA signature
		valid, err := auth.VerifySignature(client.PublicKey, payload.Data, signature)
		if err != nil || !valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid signature",
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
