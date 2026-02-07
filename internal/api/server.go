package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akua/bsv-broadcaster/internal/arc"
	"github.com/akua/bsv-broadcaster/internal/bsv"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
	"github.com/akua/bsv-broadcaster/internal/train"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Server handles HTTP API requests
type Server struct {
	db            *database.Database
	train         *train.Train
	publishingKey *bsv.KeyPair
	splitter      *bsv.Splitter
	arcClient     *arc.Client
	app           *fiber.App
}

// NewServer creates a new API server
func NewServer(db *database.Database, trainWorker *train.Train, publishingKey *bsv.KeyPair, splitter *bsv.Splitter, arcClient *arc.Client) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "BSV AKUA Broadcaster",
		DisableStartupMessage: true,
	})

	s := &Server{
		db:            db,
		train:         trainWorker,
		publishingKey: publishingKey,
		splitter:      splitter,
		arcClient:     arcClient,
		app:           app,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures HTTP endpoints
func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", s.handleHealth)

	// Main endpoints
	s.app.Post("/publish", s.handlePublish)
	s.app.Get("/status/:uuid", s.handleStatus)

	// Admin endpoints (should be protected in production)
	s.app.Get("/admin/stats", s.handleStats)
	s.app.Post("/admin/split", s.handleSplit)
	s.app.Post("/admin/split-phase2", s.handleSplitPhase2)
}

// PublishRequest represents a request to publish an OP_RETURN transaction
type PublishRequest struct {
	Data string `json:"data"` // Hex-encoded data for OP_RETURN
}

// PublishResponse contains the UUID for tracking
type PublishResponse struct {
	UUID       string `json:"uuid"`
	Message    string `json:"message"`
	QueueDepth int    `json:"queueDepth"`
}

// handlePublish processes a new broadcast request
func (s *Server) handlePublish(c *fiber.Ctx) error {
	var req PublishRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Data == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "data field is required",
		})
	}

	// Validate hex
	dataBytes, err := hex.DecodeString(req.Data)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "data must be valid hex",
		})
	}

	// Get an available publishing UTXO
	utxo, err := s.db.FindAndLockUTXO(c.Context(), models.UTXOTypePublishing)
	if err != nil {
		log.Printf("❌ No publishing UTXOs available: %v", err)
		return c.Status(503).JSON(fiber.Map{
			"error": "no publishing UTXOs available, try again later",
		})
	}

	// Create the OP_RETURN transaction
	rawHex, err := s.createOPReturnTx(utxo, dataBytes)
	if err != nil {
		s.db.UnlockUTXO(c.Context(), utxo.Outpoint) // Release UTXO
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to create transaction: %v", err),
		})
	}

	// Generate UUID for tracking
	requestUUID := uuid.New().String()

	// Check if client wants synchronous wait
	waitForResult := c.Query("wait") == "true"
	queueSize := s.train.QueueSize()

	// If queue is already full (>1000), fall back to async even if wait=true
	if waitForResult && queueSize >= 1000 {
		log.Printf("⚠️  Queue is full (%d), falling back to async mode", queueSize)
		waitForResult = false
	}

	// Save to database
	broadcastReq := &models.BroadcastRequest{
		UUID:     requestUUID,
		RawTxHex: rawHex,
		UTXOUsed: utxo.Outpoint,
		Status:   models.RequestStatusPending,
	}

	// Create response channel if synchronous mode
	if waitForResult {
		broadcastReq.ResponseChan = make(chan models.BroadcastResult, 1)
	}

	if err := s.db.InsertBroadcastRequest(c.Context(), broadcastReq); err != nil {
		s.db.UnlockUTXO(c.Context(), utxo.Outpoint)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to save request",
		})
	}

	// Enqueue for the next train
	work := train.TxWork{
		UUID:         requestUUID,
		RawTxHex:     rawHex,
		UTXOUsed:     utxo.Outpoint,
		ResponseChan: broadcastReq.ResponseChan,
	}

	if err := s.train.Enqueue(work); err != nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "queue is full, try again",
		})
	}

	// If synchronous mode, wait for result
	if waitForResult {
		// Get timeout from env var (default 5 seconds)
		timeoutStr := os.Getenv("SYNC_WAIT_TIMEOUT")
		timeout := 5 * time.Second
		if timeoutStr != "" {
			if duration, err := time.ParseDuration(timeoutStr); err == nil {
				timeout = duration
			}
		}

		select {
		case result := <-broadcastReq.ResponseChan:
			if result.Error != nil {
				// Determine appropriate HTTP status based on error
				statusCode := 500 // Default to internal server error
				if strings.Contains(result.Error.Error(), "ARC") {
					statusCode = 502 // Bad Gateway for ARC errors
				} else if strings.Contains(result.Error.Error(), "malformed") ||
					strings.Contains(result.Error.Error(), "fee") {
					statusCode = 400 // Bad Request for client errors
				}

				return c.Status(statusCode).JSON(fiber.Map{
					"error": result.Error.Error(),
					"uuid":  requestUUID,
				})
			}

			// Success - return 201 Created with txid
			return c.Status(201).JSON(fiber.Map{
				"success":    true,
				"txid":       result.TXID,
				"uuid":       requestUUID,
				"arc_status": result.ARCStatus,
				"message":    "Transaction broadcasted successfully",
			})

		case <-time.After(timeout):
			// Timeout - fall back to async response
			log.Printf("⚠️  Sync wait timeout for %s, falling back to async", requestUUID)
			return c.Status(202).JSON(fiber.Map{
				"success": true,
				"uuid":    requestUUID,
				"status":  "queued",
				"message": "Queue busy, poll /status/" + requestUUID + " for result",
			})
		}
	}

	// Default async response (202 Accepted)
	return c.Status(202).JSON(PublishResponse{
		UUID:       requestUUID,
		Message:    "Transaction queued for broadcast",
		QueueDepth: queueSize,
	})
}

// createOPReturnTx constructs a raw OP_RETURN transaction
func (s *Server) createOPReturnTx(utxo *models.UTXO, data []byte) (string, error) {
	tx := transaction.NewTransaction()

	// Add input (the 100-sat publishing UTXO)
	utxoScript, err := hex.DecodeString(utxo.ScriptPubKey)
	if err != nil {
		return "", fmt.Errorf("invalid script: %w", err)
	}

	// Create P2PKH unlocker
	unlocker, err := p2pkh.Unlock(s.publishingKey.PrivateKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create unlocker: %w", err)
	}

	err = tx.AddInputFrom(
		utxo.TxID,
		utxo.Vout,
		hex.EncodeToString(utxoScript),
		utxo.Satoshis,
		unlocker,
	)
	if err != nil {
		return "", fmt.Errorf("failed to add input: %w", err)
	}

	// Add OP_RETURN output
	// Construct OP_RETURN script manually: OP_FALSE OP_RETURN <data>
	opReturnHex := "006a" // OP_FALSE OP_RETURN

	// Push data length (varint encoding)
	dataLen := len(data)
	var lenBytes []byte
	if dataLen < 76 {
		lenBytes = []byte{byte(dataLen)}
	} else if dataLen < 256 {
		lenBytes = []byte{76, byte(dataLen)}
	} else {
		lenBytes = []byte{77, byte(dataLen), byte(dataLen >> 8)}
	}

	opReturnHex += hex.EncodeToString(lenBytes) + hex.EncodeToString(data)

	// Create script from hex
	opReturnBytes, _ := hex.DecodeString(opReturnHex)
	opReturnScript := script.Script(opReturnBytes)

	tx.AddOutput(&transaction.TransactionOutput{
		Satoshis:      0,
		LockingScript: &opReturnScript,
	})

	// Calculate fee (should be ~0.5 sats/byte)
	// For 100 sat input, this should consume all sats as fee
	// No change output needed

	// Sign the transaction
	if err := tx.Sign(); err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Return raw hex
	return tx.String(), nil
}

// StatusResponse contains transaction status information
type StatusResponse struct {
	UUID      string `json:"uuid"`
	Status    string `json:"status"`
	TxID      string `json:"txid,omitempty"`
	ARCStatus string `json:"arcStatus,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// handleStatus checks the status of a broadcast request
func (s *Server) handleStatus(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	req, err := s.db.GetRequestByUUID(c.Context(), uuid)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "request not found",
		})
	}

	return c.JSON(StatusResponse{
		UUID:      req.UUID,
		Status:    string(req.Status),
		TxID:      req.TxID,
		ARCStatus: req.ARCStatus,
		Error:     req.Error,
		CreatedAt: req.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: req.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleHealth returns server health status
func (s *Server) handleHealth(c *fiber.Ctx) error {
	stats, err := s.db.GetUTXOStats(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":     "healthy",
		"queueDepth": s.train.QueueSize(),
		"utxos":      stats,
	})
}

// handleStats returns detailed UTXO statistics
func (s *Server) handleStats(c *fiber.Ctx) error {
	stats, err := s.db.GetUTXOStats(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"utxos":      stats,
		"queueDepth": s.train.QueueSize(),
	})
}

// SplitRequest triggers manual UTXO splitting
type SplitRequest struct {
	Count int `json:"count"`
}

// handleSplit manually triggers UTXO splitting (admin only)
func (s *Server) handleSplit(c *fiber.Ctx) error {
	var req SplitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	if req.Count <= 0 {
		req.Count = 1000 // Default
	}

	ctx := context.Background()

	// Define ARC broadcast function for dependency injection
	arcBroadcastFunc := func(ctx context.Context, rawHex string) (string, error) {
		responses, err := s.arcClient.BroadcastBatch(ctx, []string{rawHex})
		if err != nil {
			return "", fmt.Errorf("ARC broadcast failed: %w", err)
		}
		if len(responses) == 0 {
			return "", fmt.Errorf("ARC returned no responses")
		}

		// Check response status - only accept if truly successful
		resp := responses[0]

		// Reject if status code indicates error (400+)
		if strings.Contains(resp.Title, "Malformed") || strings.Contains(resp.Title, "error") {
			return "", fmt.Errorf("transaction rejected: %s - %s", resp.Title, resp.ExtraInfo)
		}

		// Only accept if transaction was received and stored (not orphaned)
		if resp.TxStatus == arc.TxStatusRejected {
			return "", fmt.Errorf("transaction rejected: %s", resp.ExtraInfo)
		}

		// Reject orphan mempool status - means parent tx not found
		if strings.Contains(string(resp.TxStatus), "ORPHAN") {
			return "", fmt.Errorf("transaction orphaned - parent transaction not found or not confirmed")
		}

		// Must be at least RECEIVED or STORED
		if resp.TxStatus != arc.TxStatusReceived &&
			resp.TxStatus != arc.TxStatusStored &&
			resp.TxStatus != arc.TxStatusAnnounced &&
			resp.TxStatus != arc.TxStatusSent &&
			resp.TxStatus != arc.TxStatusSeenOnNetwork &&
			resp.TxStatus != arc.TxStatusAccepted &&
			resp.TxStatus != arc.TxStatusMined {
			return "", fmt.Errorf("unexpected transaction status: %s", resp.TxStatus)
		}

		return resp.TxID, nil
	}

	// Execute Phase 1: Split into 50 branches
	result, err := s.splitter.SplitIntoFiftyBranches(ctx, arcBroadcastFunc)
	if err != nil {
		log.Printf("❌ Split failed: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("✅ Phase 1 split complete: %d branch UTXOs created", result.TotalUTXOs)

	return c.JSON(fiber.Map{
		"message":      "Phase 1 split complete",
		"branch_txids": result.BranchTxIDs,
		"total_utxos":  result.TotalUTXOs,
	})
}

// handleSplitPhase2 splits all branch UTXOs into publishing UTXOs
func (s *Server) handleSplitPhase2(c *fiber.Ctx) error {
	ctx := context.Background()

	// Define ARC broadcast function
	arcBroadcastFunc := func(ctx context.Context, rawHex string) (string, error) {
		responses, err := s.arcClient.BroadcastBatch(ctx, []string{rawHex})
		if err != nil {
			return "", fmt.Errorf("ARC broadcast failed: %w", err)
		}
		if len(responses) == 0 {
			return "", fmt.Errorf("ARC returned no responses")
		}

		resp := responses[0]

		// Reject malformed transactions
		if strings.Contains(resp.Title, "Malformed") || strings.Contains(resp.Title, "error") {
			return "", fmt.Errorf("transaction rejected: %s - %s", resp.Title, resp.ExtraInfo)
		}

		if resp.TxStatus == arc.TxStatusRejected {
			return "", fmt.Errorf("transaction rejected: %s", resp.ExtraInfo)
		}

		// Reject orphan mempool status
		if strings.Contains(string(resp.TxStatus), "ORPHAN") {
			return "", fmt.Errorf("transaction orphaned - parent transaction not found or not confirmed")
		}

		// Must be at least RECEIVED or STORED
		if resp.TxStatus != arc.TxStatusReceived &&
			resp.TxStatus != arc.TxStatusStored &&
			resp.TxStatus != arc.TxStatusAnnounced &&
			resp.TxStatus != arc.TxStatusSent &&
			resp.TxStatus != arc.TxStatusSeenOnNetwork &&
			resp.TxStatus != arc.TxStatusAccepted &&
			resp.TxStatus != arc.TxStatusMined {
			return "", fmt.Errorf("unexpected transaction status: %s", resp.TxStatus)
		}

		return resp.TxID, nil
	}

	// Execute Phase 2: Split branches into publishing leaves
	result, err := s.splitter.SplitBranchesIntoLeaves(ctx, arcBroadcastFunc)
	if err != nil {
		log.Printf("❌ Phase 2 split failed: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("✅ Phase 2 split complete: %d publishing UTXOs created", result.TotalUTXOs)

	return c.JSON(fiber.Map{
		"message":      "Phase 2 split complete",
		"leaf_txids":   result.LeafTxIDs,
		"total_utxos":  result.TotalUTXOs,
		"transactions": len(result.LeafTxIDs),
	})
}

// Start begins serving HTTP requests
func (s *Server) Start(addr string) error {
	log.Printf("🌐 API server listening on %s", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}
