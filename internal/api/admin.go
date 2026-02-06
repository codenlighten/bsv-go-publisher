package api

import (
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
			Name       string `json:"name"`
			PublicKey  string `json:"public_key"`
			SiteOrigin string `json:"site_origin"`
			MaxDailyTx int    `json:"max_daily_tx"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if req.Name == "" || req.PublicKey == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "name and public_key are required",
			})
		}

		// Default max daily tx if not provided
		if req.MaxDailyTx == 0 {
			req.MaxDailyTx = 1000
		}

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

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Client registered successfully. Save the API key - it will only be shown once!",
			"api_key": rawKey,
			"client":  client,
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
