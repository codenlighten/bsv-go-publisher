package api

import (
	"log"

	"github.com/akua/bsv-broadcaster/internal/admin"
	"github.com/akua/bsv-broadcaster/internal/models"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterAdminRoutes sets up all admin endpoints
func (s *Server) RegisterAdminRoutes(clientMgr *admin.ClientManager, sweeper *admin.Sweeper, adminPassword string) {
	adminAuth := AdminAuthMiddleware(adminPassword)

	// Client management endpoints
	clients := s.app.Group("/admin/clients", adminAuth)

	clients.Post("/register", func(c *fiber.Ctx) error {
		var req struct {
			Name       string   `json:"name"`
			PublicKey  string   `json:"public_key"` // NOW OPTIONAL for pilot tier
			SiteOrigin string   `json:"site_origin"`
			MaxDailyTx int      `json:"max_daily_tx"`
			Tier       string   `json:"tier"`        // NEW: "pilot", "enterprise", "government"
			AllowedIPs []string `json:"allowed_ips"` // NEW: IP whitelist for pilot tier
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "name is required",
			})
		}

		// Default tier to "enterprise" if not specified
		if req.Tier == "" {
			req.Tier = "enterprise"
		}

		// Validate tier
		if req.Tier != "pilot" && req.Tier != "enterprise" && req.Tier != "government" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "tier must be 'pilot', 'enterprise', or 'government'",
			})
		}

		// For enterprise/government tiers, public key is required
		if (req.Tier == "enterprise" || req.Tier == "government") && req.PublicKey == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "public_key is required for enterprise and government tiers",
			})
		}

		// Default max daily tx if not provided
		if req.MaxDailyTx == 0 {
			req.MaxDailyTx = 1000
		}

		// Register client with legacy method
		rawKey, client, err := clientMgr.RegisterClient(
			c.Context(),
			req.Name,
			req.PublicKey,
			req.SiteOrigin,
			req.MaxDailyTx,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Apply tier-based security defaults
		var requireSignature bool
		var gracePeriodHours int

		switch req.Tier {
		case "pilot":
			requireSignature = false
			gracePeriodHours = 0 // Not applicable for pilot
		case "enterprise":
			requireSignature = true
			gracePeriodHours = 24 // 24 hours
		case "government":
			requireSignature = true
			gracePeriodHours = 168 // 7 days
		}

		// Update client with tier settings
		err = s.db.UpdateClientSecurity(
			c.Context(),
			client.ID,
			req.Tier,
			requireSignature,
			req.AllowedIPs,
			gracePeriodHours,
		)
		if err != nil {
			log.Printf("⚠️ Failed to apply tier settings: %v", err)
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Client registered successfully. Save the API key - it will only be shown once!",
			"api_key": rawKey,
			"client":  client,
			"tier":    req.Tier,
			"security": fiber.Map{
				"require_signature":  requireSignature,
				"grace_period_hours": gracePeriodHours,
				"allowed_ips":        req.AllowedIPs,
			},
		})
	})

	clients.Get("/list", func(c *fiber.Ctx) error {
		clientList, err := clientMgr.ListClients(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"clients": clientList,
		})
	})

	clients.Post("/:id/activate", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid client ID",
			})
		}

		if err := clientMgr.ActivateClient(c.Context(), objID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Client activated",
		})
	})

	clients.Post("/:id/deactivate", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid client ID",
			})
		}

		if err := clientMgr.DeactivateClient(c.Context(), objID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Client deactivated",
		})
	})

	// PHASE 5: Runtime Security Management
	clients.Patch("/:id/security", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid client ID",
			})
		}

		var req struct {
			Tier             string   `json:"tier"`               // "pilot", "enterprise", "government"
			RequireSignature *bool    `json:"require_signature"`  // Pointer to distinguish false from unset
			AllowedIPs       []string `json:"allowed_ips"`        // IP whitelist
			GracePeriodHours *int     `json:"grace_period_hours"` // Pointer to allow 0 as valid value
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Fetch current client to merge updates
		currentClient, err := clientMgr.GetClientByID(c.Context(), objID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Client not found",
			})
		}

		// Apply updates (use current values if not provided)
		tier := currentClient.Tier
		if req.Tier != "" {
			// Validate tier
			if req.Tier != "pilot" && req.Tier != "enterprise" && req.Tier != "government" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "tier must be 'pilot', 'enterprise', or 'government'",
				})
			}
			tier = req.Tier
		}

		requireSignature := currentClient.RequireSignature
		if req.RequireSignature != nil {
			requireSignature = *req.RequireSignature
		}

		allowedIPs := currentClient.AllowedIPs
		if req.AllowedIPs != nil {
			allowedIPs = req.AllowedIPs
		}

		gracePeriodHours := currentClient.GracePeriodHours
		if req.GracePeriodHours != nil {
			gracePeriodHours = *req.GracePeriodHours
		}

		// Update database
		err = s.db.UpdateClientSecurity(
			c.Context(),
			objID,
			tier,
			requireSignature,
			allowedIPs,
			gracePeriodHours,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		log.Printf("✅ Updated security for client %s: tier=%s, require_signature=%v", currentClient.Name, tier, requireSignature)

		return c.JSON(fiber.Map{
			"success":   true,
			"client_id": objID.Hex(),
			"security": fiber.Map{
				"tier":               tier,
				"require_signature":  requireSignature,
				"allowed_ips":        allowedIPs,
				"grace_period_hours": gracePeriodHours,
			},
			"message": "Security settings updated. Changes effective immediately.",
		})
	})

	// Maintenance endpoints
	maintenance := s.app.Group("/admin/maintenance", adminAuth)

	maintenance.Post("/sweep", func(c *fiber.Ctx) error {
		var req struct {
			DestAddress string `json:"dest_address"`
			MaxInputs   int    `json:"max_inputs"`
			UTXOType    string `json:"utxo_type"` // "publishing" or "funding"
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.DestAddress == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "dest_address is required",
			})
		}

		if req.MaxInputs == 0 {
			req.MaxInputs = 100
		}

		utxoType := models.UTXOTypePublishing
		if req.UTXOType == "funding" {
			utxoType = models.UTXOTypeFunding
		}

		txID, amount, err := sweeper.SweepUTXOs(c.Context(), req.DestAddress, req.MaxInputs, utxoType)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"txid":    txID,
			"amount":  amount,
			"message": "UTXOs consolidated successfully",
		})
	})

	maintenance.Post("/consolidate-dust", func(c *fiber.Ctx) error {
		var req struct {
			FundingAddress string `json:"funding_address"`
			MaxInputs      int    `json:"max_inputs"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.FundingAddress == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "funding_address is required",
			})
		}

		if req.MaxInputs == 0 {
			req.MaxInputs = 100
		}

		txID, amount, err := sweeper.ConsolidateDust(c.Context(), req.FundingAddress, req.MaxInputs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"txid":    txID,
			"amount":  amount,
			"message": "Dust UTXOs consolidated successfully",
		})
	})

	maintenance.Get("/estimate-sweep", func(c *fiber.Ctx) error {
		utxoType := c.Query("utxo_type", "publishing")
		maxInputs := c.QueryInt("max_inputs", 100)

		typ := models.UTXOTypePublishing
		if utxoType == "funding" {
			typ = models.UTXOTypeFunding
		}

		total, count, err := sweeper.EstimateSweepValue(c.Context(), typ, maxInputs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success":    true,
			"utxo_type":  utxoType,
			"count":      count,
			"total_sats": total,
			"total_bsv":  float64(total) / 100000000,
		})
	})

	// Emergency endpoints
	emergency := s.app.Group("/admin/emergency", adminAuth)

	emergency.Post("/stop-train", func(c *fiber.Ctx) error {
		s.train.Stop()
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Train worker stopped. Restart server to resume.",
		})
	})

	emergency.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"running": s.train.IsRunning(),
		})
	})
}
